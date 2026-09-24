package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// AuthHandler обработчик аутентификации
type AuthHandler struct {
	authService domain.AuthService
	limiter     *LoginRateLimiter
	db          *pgxpool.Pool
}

// NewAuthHandler создает новый обработчик
func NewAuthHandler(as domain.AuthService, db *pgxpool.Pool, limiter *LoginRateLimiter) *AuthHandler {
	return &AuthHandler{authService: as, db: db, limiter: limiter}
}

// RegisterRequest запрос на регистрацию.
// Идентификатор — username (≤10, латиница+цифры).
// email, phone — для подтверждения и отправки кода (необязательны, не ПДн-логин).
// orgName — коллектив (по желанию): существующий — присоединение, новый — создание.
type RegisterRequest struct {
	FullName string `json:"full_name"`
	Username string `json:"username" binding:"required,min=3,max=10"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"`
	OrgName  string `json:"org_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	ReferralCode string `json:"referral_code"`
}

// Register регистрирует нового пользователя (создаёт или присоединяет к коллективу)
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Валидация: username только латиница+цифры.
	if !isValidUsername(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username: only latin letters and digits"})
		return
	}

	// Роль по умолчанию — installer (если не указана клиентом).
	role := req.Role
	if role == "" {
		role = "installer"
	}

	user, err := h.authService.Register(c.Request.Context(), req.FullName, req.Username, req.Password, role, req.OrgName, req.Phone, req.Email, req.ReferralCode)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        user.ID.String(),
		"username":  user.Username,
		"full_name": user.FullName,
		"role":      user.Role,
	})
}

// LoginRequest запрос на вход по логину (username)
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	HwID     string `json:"hw_id"`
}

// Login выполняет вход и выдаёт JWT-токен
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := ClientIP(c)

	// Защита от брутфорса: если IP или логин заблокирован — отказ.
	if h.limiter.IsBlocked(ip) || h.limiter.IsBlocked(req.Username) {
		h.auditLogin(req.Username, ip, "blocked_request", false)
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts, try later"})
		return
	}

	token, user, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		// Неудачная попытка — фиксируем и ограничиваем.
		if h.limiter.RegisterFailed(ip, req.Username) {
			h.auditLogin(req.Username, ip, "blocked", false)
		} else {
			h.auditLogin(req.Username, ip, "failed", false)
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":         err.Error(),
			"attempts_left": h.limiter.LoginAttempts(ip, req.Username),
		})
		return
	}

	// Успешный вход — сброс счётчиков и аудит.
	h.limiter.RegisterSuccess(ip, req.Username)

	// Привязываем устройство (hw_id) к пользователю при входе.
	if req.HwID != "" {
		h.authService.BindHwID(c.Request.Context(), user.ID, req.HwID)
	}

	h.auditLogin(req.Username, ip, "success", true, user.ID.String())

	// MFA: если включено и роль admin — токен не выдаём сразу,
	// возвращаем запрос второго фактора (клиент должен пройти /auth/mfa).
	if os.Getenv("MFA_ENABLED") == "1" && user.Role == "admin" {
		c.JSON(http.StatusOK, gin.H{
			"mfa_required": true,
			"user_id":      user.ID.String(),
			"role":         user.Role,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":        user.ID.String(),
			"username":  user.Username,
			"full_name": user.FullName,
			"role":      user.Role,
		},
	})
}

// auditLogin пишет запись в журнал аудита с SHA-256 хеш-цепочкой:
// each record current_hash = sha256(prev_hash + data). Позволяет проверить
// целостность журнала (ни одна запись не изменена и не удалена).
// В журнал попадает только логин (username), ПДн — нет.
func (h *AuthHandler) auditLogin(username, ip, result string, ok bool, userID ...string) {
	if h.db == nil {
		return
	}
	uid := ""
	if len(userID) > 0 {
		uid = userID[0]
	}

	// Берём hash последней записи (предыдущий в цепочке).
	var prevHash string
	if err := h.db.QueryRow(context.Background(),
		`SELECT COALESCE(current_hash, '') FROM audit_log ORDER BY created_at DESC, id DESC LIMIT 1`).Scan(&prevHash); err != nil {
		prevHash = ""
	}

	// current_hash = sha256(prev_hash + action + entity + detail + ip)
	raw := prevHash + "|auth_login|" + username + "|" + result + "|" + ip
	curHash := sha256Hex(raw)

	if _, err := h.db.Exec(context.Background(),
		`INSERT INTO audit_log (id, user_id, action, entity_id, detail, ip, prev_hash, current_hash, created_at)
		 VALUES (gen_random_uuid(), CAST(NULLIF($1,'') AS uuid), 'auth_login', $2, $3, $4, $5, $6, now())`,
		uid, username, result, ip, prevHash, curHash); err != nil {
		log.Printf("audit_login error: %v", err)
	}

	// Уведомляем watchdog об аномалиях (блокировки, неудачные входы).
	if result == "blocked" || result == "blocked_request" || result == "failed" {
		_, _ = h.db.Exec(context.Background(),
			`SELECT pg_notify('audit_events', $1)`,
			"auth_login:"+result+":"+username+":"+ip)
	}
}

// isValidUsername: только латиница и цифры, длина 3–10.
func isValidUsername(s string) bool {
	if len(s) < 3 || len(s) > 10 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// sha256Hex возвращает SHA-256 хеш строки в hex.
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// VerifyAuditChain проверяет целостность журнала аудита (SHA-256 цепочка).
// GET /api/v1/admin/audit/verify  (JWT, роль admin)
func (h *AuthHandler) VerifyAuditChain(c *gin.Context) {
	type row struct {
		id          string
		action      string
		entityID    string
		detail      string
		ip          string
		prevHash    string
		currentHash string
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id::text, action, COALESCE(entity_id,''), COALESCE(detail,''), COALESCE(ip,''),
		        COALESCE(prev_hash,''), COALESCE(current_hash,'')
		 FROM audit_log ORDER BY created_at ASC, id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	var records []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.action, &r.entityID, &r.detail, &r.ip, &r.prevHash, &r.currentHash); err != nil {
			continue
		}
		records = append(records, r)
	}

	valid := true
	brokenAt := ""
	prevChainHash := ""
	for i := range records {
		r := records[i]
		expected := sha256Hex(prevChainHash + "|" + r.action + "|" + r.entityID + "|" + r.detail + "|" + r.ip)
		if r.prevHash != prevChainHash || (r.currentHash != "" && r.currentHash != expected) {
			if r.currentHash != "" {
				valid = false
				brokenAt = r.id
				break
			}
		}
		if r.currentHash != "" {
			prevChainHash = r.currentHash
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":     valid,
		"total":     len(records),
		"broken_at": brokenAt,
	})
}

// GetMe возвращает профиль текущего пользователя (по токену)
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID := c.GetString("user_id")
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"role":    c.GetString("role"),
	})
}

// DeleteAccount анонимизирует персональные данные пользователя (152-ФЗ).
// POST /auth/delete-account  (JWT; user_id из токена)
// Телефон, ФИО и пароль очищаются; username и UUID остаются
// для юридически значимых журналов (история не удаляется).
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
		return
	}

	if err := h.authService.AnonymizeAccount(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to anonymize"})
		return
	}

	h.auditLogin(userIDStr, ClientIP(c), "account_deleted", true, userIDStr)
	c.JSON(http.StatusOK, gin.H{"status": "account anonymized"})
}

// ExtractBearerToken достает токен из заголовка Authorization
func ExtractBearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// MaskPhone больше не нужен (в аудите username, а не телефон).
// Оставлен для совместимости, если где-то ещё вызывается.
func MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}
	if len(phone) <= 4 {
		return "***"
	}
	return phone[:1] + "***" + phone[len(phone)-4:]
}

// ListAudit возвращает журнал аудита (admin/audit) — последние записи с хешами.
// GET /api/v1/admin/audit?limit=20
func (h *AuthHandler) ListAudit(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id::text, action, COALESCE(entity_id,''), COALESCE(detail,''),
		        COALESCE(ip,''), created_at::text,
		        COALESCE(prev_hash,''), COALESCE(current_hash,'')
		 FROM audit_log ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type entry struct {
		ID       string `json:"id"`
		Action   string `json:"action"`
		EntityID string `json:"entity_id"`
		Detail   string `json:"detail"`
		IP       string `json:"ip"`
		Time     string `json:"time"`
		PrevHash string `json:"prev_hash"`
		CurrHash string `json:"current_hash"`
	}
	var items []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.Action, &e.EntityID, &e.Detail, &e.IP, &e.Time, &e.PrevHash, &e.CurrHash); err != nil {
			continue
		}
		items = append(items, e)
	}
	if items == nil {
		items = []entry{}
	}
	c.JSON(http.StatusOK, gin.H{"audit": items})
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}