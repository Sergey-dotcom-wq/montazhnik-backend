package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/handlers"
	"github.com/yourorg/montazhnik/backend/internal/matrix"
	"github.com/yourorg/montazhnik/backend/internal/repository/postgres"
	"github.com/yourorg/montazhnik/backend/internal/services"
	"github.com/yourorg/montazhnik/backend/internal/web"
)

func newGatewayHive() *matrix.Hive {
	return matrix.NewHiveWithPolicy(matrix.DefaultRuleSet(), matrixPolicy)
}

// App структура приложения
type App struct {
	Router          *gin.Engine
	DB              *pgxpool.Pool
	AuthService     *services.AuthServiceImpl
	JobService      *services.JobServiceImpl
	EventService    *services.EventServiceImpl
	MaterialService *services.MaterialServiceImpl
	AuthHandler     *handlers.AuthHandler
	JobHandler      *handlers.JobHandler
	EventHandler    *handlers.EventHandler
	MaterialHandler *handlers.MaterialHandler
}

// NewApp создает экземпляр приложения и связывает зависимости (Wiring)
func NewApp(ctx context.Context, cfg Config) (*App, error) {
	// Подключение к БД
	dbpool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Инициализация репозиториев (Матрица состояний: Static + Dynamic)
	jobRepo := postgres.NewJobRepositoryPostgres(dbpool)
	eventRepo := postgres.NewEventRepositoryPostgres(dbpool)
	materialRepo := postgres.NewMaterialRepositoryPostgres(dbpool)
	jobMaterialRepo := postgres.NewJobMaterialRepositoryPostgres(dbpool)
	userRepo := postgres.NewUserRepositoryPostgres(dbpool)
	orgRepo := postgres.NewOrganizationRepositoryPostgres(dbpool)

	// Инициализация сервисов (Бизнес-логика)
	authService := services.NewAuthService(userRepo, orgRepo, cfg.JWTSecret)
	jobService := services.NewJobService(jobRepo)
	eventService := services.NewEventService(eventRepo, jobRepo)
	materialService := services.NewMaterialService(materialRepo, jobMaterialRepo)

	// Инициализация хендлеров (Транспортный слой)
	// Защита от брутфорса: 5 неудачных попыток входа → блокировка 15 минут.
	loginLimiter := handlers.NewLoginRateLimiter(5, 15*time.Minute)
	authHandler := handlers.NewAuthHandler(authService, dbpool, loginLimiter)
	jobHandler := handlers.NewJobHandler(jobService)
	eventHandler := handlers.NewEventHandler(eventService, jobService)
	materialHandler := handlers.NewMaterialHandler(materialService)
	fuelHandler := handlers.NewFuelHandler(dbpool)
	warehouseHandler := handlers.NewWarehouseHandler(dbpool)
	subscriptionHandler := handlers.NewSubscriptionHandler(dbpool)
	roadHandler := handlers.NewRoadHandler(dbpool)
	partnerHandler := handlers.NewPartnerHandler(dbpool)
	paymentProvider := handlers.NewPaymentProvider()
	paymentHandler := handlers.NewPaymentHandler(dbpool, paymentProvider)
	mfaHandler := handlers.NewMFAHandler(dbpool, authService)
	alertHandler := handlers.NewAlertHandler(dbpool, nil)
	shiftRepo := postgres.NewShiftRepositoryPostgres(dbpool)
	shiftService := services.NewShiftService(shiftRepo)
	shiftHandler := handlers.NewShiftHandler(dbpool, shiftService)
	toolHandler := handlers.NewToolHandler(dbpool)
	bagHandler := handlers.NewBagHandler(dbpool)
	reportHandler := handlers.NewReportHandler(dbpool)
	stage2Handler := handlers.NewStage2Handler(dbpool)
	evidenceHandler := handlers.NewEvidenceHandler(dbpool, cfg.JWTSecret)
	webDocsHandler := handlers.NewWebDocsHandler(dbpool)

	// Настройка роутера
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Сотовая матрица доступа (Правила ядра): каждый авторизованный запрос
	// проверяется на все защищённые маршруты единообразно.
	hive := newGatewayHive()

	api := router.Group("/api/v1")
	{
		// Аутентификация (публичные маршруты)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", authMiddleware(authService), matrixGatewayMiddleware(hive), authHandler.GetMe)
			// MFA для администратора: запрос кода и проверка.
			auth.POST("/mfa/request", mfaHandler.Request)
			auth.POST("/mfa/verify", mfaHandler.Verify)
			// Удаление аккаунта: анонимизация ПДн (152-ФЗ). Под JWT.
			auth.POST("/delete-account", authMiddleware(authService), matrixGatewayMiddleware(hive), authHandler.DeleteAccount)
		}

		// Публичные эндпоинты: аутентификация, статус подписки и создание
		// платежа по СБП (стартовая страница позволяет оплатить ДО входа:
		// user_id в этом случае берётся из черновика платежа при вебхуке или
		// при активации после входа).
		api.GET("/subscription/status", subscriptionHandler.Status)
		api.POST("/payments/create", paymentHandler.CreatePayment)
		// Доверенное время (подписанное сервером) — для привязки фото.
		api.GET("/time", evidenceHandler.SignedTime)

		// Приём краш-репортов с телефона (GET, чтобы не было CORS/body).
		// Стек приходит в query: /api/v1/crash?msg=...&stack=...&ver=1.3.0
		api.GET("/crash", func(c *gin.Context) {
			msg := c.Query("msg")
			stack := c.Query("stack")
			ver := c.Query("ver")
			if len(stack) > 4000 {
				stack = stack[:4000]
			}
			log.Printf("CRASH from %s ver=%s: %s | %s", c.ClientIP(), ver, msg, stack)
			c.JSON(200, gin.H{"ok": true})
		})

		// Вебхук платёжного провайдера — публичный (платёжка шлёт без JWT).
		// Подпись проверяется внутри (HMAC-SHA256).
		api.POST("/webhook/payment", paymentHandler.Webhook)

		// Админские операции — JWT + роль.
		adminPay := api.Group("/admin")
		adminPay.Use(authMiddleware(authService), requireRole("admin", "dispatcher"), matrixGatewayMiddleware(hive))
		{
			adminPay.GET("/payments", paymentHandler.ListPayments)
			adminPay.GET("/payments/process-payouts", paymentHandler.ProcessPayouts)
			// Проверка целостности журнала аудита (SHA-256 цепочка).
			adminPay.GET("/audit/verify", authHandler.VerifyAuditChain)
			// Сводка диспетчера (смены, инструменты, алерты, сумки).
			adminPay.GET("/report", reportHandler.DispatchReport)
			// Журнал аудита (список с хешами) и генерация актов КС-2/КС-3.
			adminPay.GET("/audit", authHandler.ListAudit)
			adminPay.POST("/generate-ks2", webDocsHandler.GenerateKs2)
			adminPay.POST("/generate-ks3", webDocsHandler.GenerateKs3)
		}

		// Защищённые маршруты (JWT). Все мобильные данные — только по токену,
		// который приложение получает при входе/регистрации.
protected := api.Group("")
	protected.Use(authMiddleware(authService), matrixGatewayMiddleware(hive))
		{
			// Активация подписки идёт под JWT: привязываем пользователя из токена
			// к реферальной выплате партнёру (см. SubscriptionHandler.Activate).
			protected.POST("/subscription/activate", subscriptionHandler.Activate)

			// Мобильные эндпоинты (Android-клиент)
			protected.POST("/fuel", fuelHandler.Add)
			protected.GET("/warehouse/items", warehouseHandler.ListItems)
			protected.POST("/warehouse/requests", warehouseHandler.CreateRequest)
			protected.GET("/road/route", roadHandler.CurrentRoute)
			protected.GET("/road/conditions", roadHandler.Conditions)
			protected.GET("/road/alternatives", roadHandler.Alternatives)
			protected.GET("/work-orders", jobHandler.ListWorkOrders)

			// Партнёрская программа: partner видит только свои рефералы
			// (логика в хендлере), остальные роли — по правам ниже.
			partnerOffice := protected.Group("")
			partnerOffice.Use(requireRole("admin", "dispatcher", "accountant", "partner"))
			{
				partnerOffice.GET("/partners", partnerHandler.ListPartners)
				partnerOffice.GET("/partners/referrals", partnerHandler.ListReferrals)
			}

			// Проверка реферального кода — доступна любому авторизованному
			// (нужна на экране регистрации до завершения).
			protected.GET("/partners/check-code", partnerHandler.CheckCode)

			// Наряды (Static State)
			jobs := protected.Group("/jobs")
			jobs.Use(requireRole("admin", "dispatcher", "installer"))
			{
				jobs.GET("/", jobHandler.List)
				jobs.POST("/", jobHandler.Create)
				jobs.GET("/:id", jobHandler.Get)
				jobs.PATCH("/:id/status", jobHandler.UpdateStatus)
			}

			// События (Dynamic State / SHA-256 Chain)
			events := protected.Group("/events")
			{
				events.POST("/", eventHandler.Create)
				events.GET("/:id/chain", eventHandler.GetChain)
				events.GET("/:id/verify", eventHandler.VerifyChain)
			}

			// Материалы (Static + Dynamic)
			materials := protected.Group("/materials")
			{
				materials.GET("/", materialHandler.ListMaterials)
				materials.POST("/", materialHandler.Create)
				materials.POST("/scan", materialHandler.Scan)
				materials.GET("/job/:job_id", materialHandler.ListByJob)
			}

			// Критические оповещения (диспетчер → бригада).
			// GET /alerts — неподтверждённые алерты коллектива (polling);
			// POST /alerts — создание (admin/dispatcher);
			// POST /alerts/:id/ack — «Подтвердить получение»;
			// GET /alerts/stream — SSE (опциональный real-time push).
			// Пути регистрируются БЕЗ trailing slash, чтобы клиент не ловил
			// 307-редирект (иначе заголовок Authorization может теряться).
			alerts := protected.Group("/alerts")
			alerts.Use(requireRole("admin", "dispatcher", "installer"))
			{
				alerts.GET("", alertHandler.ListPending)
				alerts.GET("/stream", alertHandler.Stream)
				alerts.POST("/:id/ack", alertHandler.Acknowledge)
			}
			alertSend := protected.Group("/alerts")
			alertSend.Use(requireRole("admin", "dispatcher"))
			{
				alertSend.POST("", alertHandler.Create)
			}

			// Рабочие смены (Field Guide: геозонирование смены, утренняя
			// блокировка до звонка диспетчеру).
			shift := protected.Group("/shift")
			{
				shift.GET("", shiftHandler.Current)
				shift.POST("/start", shiftHandler.Start)
				shift.POST("/pause", shiftHandler.Pause)
				shift.POST("/resume", shiftHandler.Resume)
				shift.POST("/end", shiftHandler.End)
				// Инженер запрашивает разблокировку («Начать смену» заблокирована).
				shift.POST("/request-unlock", shiftHandler.RequestUnlock)
			}
			// Разблокировка смены диспетчером — только admin/dispatcher.
			shiftDispatch := protected.Group("/shift")
			shiftDispatch.Use(requireRole("admin", "dispatcher"))
			{
				shiftDispatch.POST("/dispatch-unlock", shiftHandler.DispatchUnlock)
			}

			// Инструменты «виртуальной сумки» инженера (Anti-Theft / Anti-Loss).
			// Field Guide: привязка BLE по MAC, потеря → Hard Block + GPS + алерт.
			tools := protected.Group("/tools")
			{
				tools.GET("", toolHandler.List)
				tools.POST("/bind", toolHandler.Bind)
				tools.POST("/:id/unbind", toolHandler.Unbind)
				tools.POST("/:id/lost", toolHandler.ReportLost)
				tools.POST("/:id/heartbeat", toolHandler.Heartbeat)
			}

			// «Виртуальная сумка» материалов (Field Guide §3): материалы не
			// списываются окончательно, а закрепляются за инженером/нарядом.
			bag := protected.Group("/bag")
			{
				bag.GET("", bagHandler.ListBag)
				bag.POST("/add", bagHandler.AddToBag)
				bag.POST("/take", bagHandler.TakeFromBag)
			}

			// Stage 2: граф сварок, тепловые и акустические замеры
			// (синхронизация диагностики с сервером → отчёт диспетчера).
			stage2 := protected.Group("/stage2")
			{
				stage2.GET("/splices", stage2Handler.ListSplices)
				stage2.POST("/splices", stage2Handler.AddSplice)
				stage2.GET("/thermal", stage2Handler.ListThermal)
				stage2.POST("/thermal", stage2Handler.AddThermal)
				stage2.POST("/acoustic", stage2Handler.AddAcoustic)
			}

			// Фото-доказательства (anti-fraud): регистрация хеша фото при съёмке
			// и проверка по хешу (для КС-2 и сводки диспетчера).
			evidence := protected.Group("/photo-evidence")
			{
				evidence.POST("", evidenceHandler.RegisterEvidence)
				evidence.POST("/verify", evidenceHandler.VerifyEvidence)
			}
		}
	}

	// Статические страницы
	// Корень "/" — новый дашборд коллектива (стиль десктоп-клиента).
	// "/admin/*" — старая админка (наряды/материалы/цепочки/партнёры).
	// Презентация (оригинал) вынесена на отдельный порт 8081 — см. cmd/present.
	router.GET("/", serveDashboard)
	router.GET("/admin/*filepath", serveAdminPage)

	// Презентация «Монтажник Связи 1.2» (та же, что в мобильном приложении).
	// Отдаётся на основном порту, чтобы кнопка в дашборде работала без 8081.
	router.GET("/presentation", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, web.AppPresentationHTML)
	})

	return &App{
		Router:          router,
		DB:              dbpool,
		AuthService:     authService,
		JobService:      jobService,
		EventService:    eventService,
		MaterialService: materialService,
		AuthHandler:     authHandler,
		JobHandler:      jobHandler,
		EventHandler:    eventHandler,
		MaterialHandler: materialHandler,
	}, nil
}

// serveDashboard отдает главную страницу — дашборд коллектива.
func serveDashboard(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, web.DashboardHTML)
}

// serveAdminPage отдает встроенную HTML-страницу веб-админки
func serveAdminPage(c *gin.Context) {
	// Запрет кэширования: админка обновляется вместе с бинарником,
	// чтобы браузер телефона не показывал старую версию.
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, web.AdminHTML)
}

// authMiddleware проверяет JWT-токен из заголовка Authorization
func authMiddleware(authService *services.AuthServiceImpl) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := handlers.ExtractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
			return
		}

		userID, role, orgID, err := authService.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("user_id", userID.String())
		c.Set("role", role)
		// Изоляция коллективов: org_id из токена доступен всем хендлерам,
		// данные фильтруются по нему (своя организация — свои данные).
		if orgID != uuid.Nil {
			c.Set("org_id", orgID.String())
		}
		c.Next()
	}
}

// requireRole ограничивает доступ к маршруту набором ролей.
func requireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	}
}

// Config конфигурация
type Config struct {
	Port        string
	BindAddr    string
	DatabaseURL string
	JWTSecret   string

	// S3/MinIO резервное копирование (опционально)
	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3EncKeyHex string // ключ AES-256-GCM (64 hex), клиентское шифрование
	S3CA        string // путь к корневому CA для проверки сервера (пусто = useTSL)
	S3UseTLS    bool
}

func loadConfig() Config {
	// Загружаем секреты из .env (рядом с бинарником), если файл есть.
	loadDotEnv(".env")
	loadDotEnv("backend.env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	bind := os.Getenv("BIND_ADDR")
	if bind == "" {
		bind = "0.0.0.0"
	}
	// Пароль БД и JWT-секрет — только из окружения (или .env),
	// в коде дефолтов нет. При отсутствии — сервер не стартует.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set. Set env or .env file.")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set. Set env or .env file.")
	}
	return Config{
		Port:        port,
		BindAddr:    bind,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
		S3EncKeyHex: os.Getenv("S3_ENC_KEY"),
		S3CA:        os.Getenv("S3_CA"),
		S3UseTLS:    os.Getenv("S3_USE_TLS") == "1" || os.Getenv("S3_USE_TLS") == "true",
	}
}

// loadDotEnv читает файл KEY=VALUE (без кавычек, комментарии #) в окружение
// процесса. Не перезаписывает уже установленные переменные.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if key == "" {
			continue
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func main() {
	ctx := context.Background()
	cfg := loadConfig()

	// Логи в файл и в консоль, чтобы видеть запросы с телефона
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
		gin.DefaultWriter = mw
		gin.DefaultErrorWriter = mw
		defer logFile.Close()
	}

	app, err := NewApp(ctx, cfg)
	if err != nil {
		log.Fatal("Failed to initialize app:", err)
	}

	// Авто-резервное копирование БД: pg_dump раз в 24 часа (hardening).
	// Бэкапы складываются в каталог, который не трогает очистка приложения.
	go func() {
		doBackupAndS3(cfg)
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			doBackupAndS3(cfg)
		}
	}()

	srv := &http.Server{
		Addr:    cfg.BindAddr + ":" + cfg.Port,
		Handler: app.Router,
	}

	// HTTPS с mTLS на отдельном порту (8443): TLS 1.2+ с серверным сертификатом
	// от нашего корневого CA и ОБЯЗАТЕЛЬНОЙ проверкой клиентских сертификатов
	// (mTLS). Сертификаты генерирует cmd/gen_mtls в backend/pki.
	// HTTP остаётся для локальной сети/разработки.
	go func() {
		log.Printf("Server starting on http://%s:%s", cfg.BindAddr, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	go func() {
		// Загружаем корневой CA и серверный сертификат для mTLS.
		// Пути по умолчанию — backend/pki; можно переопределить через окружение.
		caPath := os.Getenv("MONT_CA_CRT")
		if caPath == "" {
			caPath = "pki/ca.crt"
		}
		serverCrt := os.Getenv("MONT_SERVER_CRT")
		if serverCrt == "" {
			serverCrt = "pki/server.crt"
		}
		serverKey := os.Getenv("MONT_SERVER_KEY")
		if serverKey == "" {
			serverKey = "pki/server.key"
		}

		caPEM, err := os.ReadFile(caPath)
		if err != nil {
			log.Printf("mTLS: ca read failed: %v", err)
			return
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			log.Printf("mTLS: ca parse failed")
			return
		}

		tlsSrv := &http.Server{
			Addr:    cfg.BindAddr + ":8443",
			Handler: app.Router,
			TLSConfig: &tls.Config{
				ClientCAs:  pool,
				ClientAuth: tls.RequireAndVerifyClientCert,
				MinVersion: tls.VersionTLS12,
			},
		}
		log.Printf("Server starting on https://%s:8443 (TLS + mTLS, require client cert)", cfg.BindAddr)
		if err := tlsSrv.ListenAndServeTLS(serverCrt, serverKey); err != nil && err != http.ErrServerClosed {
			log.Printf("tls listen: %v", err)
		}
	}()

	// Graceful Shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down...")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	app.DB.Close()
}

// backupDB делает резервную копию БД через pg_dump.
// Хранит последние 7 дампов; старые удаляет (ротация).
func backupDB(databaseURL string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("backupDB panic: %v", r)
		}
	}()
	const backupPath = "C:/1.2 MS/postgres/backups"
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		log.Printf("backup: mkdir: %v", err)
		return
	}
	// Парсим host/port/user/pass/db из URL (простой разбор для нашего формата).
	// Формат: postgres://user:pass@host:port/db?sslmode=disable
	u := strings.TrimPrefix(databaseURL, "postgres://")
	at := strings.LastIndex(u, "@")
	colon := strings.Index(u, ":")
	if at <= 0 || colon <= 0 {
		log.Printf("backup: cannot parse db url")
		return
	}
	userPass := u[:at]
	hostDB := u[at+1:]
	user := userPass[:colon]
	pass := userPass[colon+1:]
	slash := strings.Index(hostDB, "/")
	hostPort := hostDB
	dbName := "montazhnik"
	if slash > 0 {
		hostPort = hostDB[:slash]
		dbName = strings.TrimSuffix(hostDB[slash+1:], "?sslmode=disable")
	}

	ts := time.Now().Format("20060102_150405")
	outFile := filepath.Join(backupPath, "montazhnik_"+ts+".dump")
	cmd := exec.Command("C:/1.2 MS/postgres/pgsql/bin/pg_dump.exe",
		"-h", strings.Split(hostPort, ":")[0],
		"-p", func() string { if len(strings.Split(hostPort, ":")) > 1 { return strings.Split(hostPort, ":")[1] }; return "5432" }(),
		"-U", user, "-d", dbName, "-F", "c", "-f", outFile)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	if err := cmd.Run(); err != nil {
		log.Printf("backup: pg_dump failed: %v", err)
		return
	}
	log.Printf("backup: saved %s", outFile)

	// Ротация: оставляем последние 7 файлов.
	files, _ := filepath.Glob(filepath.Join(backupPath, "montazhnik_*.dump"))
	sort.Strings(files)
	for len(files) > 7 {
		os.Remove(files[0])
		files = files[1:]
	}
}
