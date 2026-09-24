package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Watchdog — сторожевой процесс для Go-сервера «Монтажник Связи».
//
// Роль:
//  1. Постоянное наблюдение: health-check HTTP запроса к API каждые 10 секунд.
//  2. Аномалии: подписан на LISTEN/NOTIFY канал audit_events в PostgreSQL —
//     получает события журнала аудита и реагирует только на значимые
//     (множественные неудачные входы, blocked_request) — шум игнорирует.
//  3. Восстановление: если API не отвечает 3 раза подряд — убивает процесс
//     сервера и запускает его заново (тот же бинарник).
//
// Ресурсы: почти нулевые — работает по событиям и редким проверкам,
// а не крутит цикл с пингами каждую миллисекунду.

const (
	healthURL     = "http://127.0.0.1:8080/api/v1/subscription/status?hw_id=watchdog"
	healthEvery   = 10 * time.Second
	maxMisses     = 3
	serverBin     = "server.exe"
	recoverWait   = 3 * time.Second
	serverRestart = 2 * time.Second
)

func main() {
	log.SetOutput(os.Stdout)
	log.Printf("watchdog: started, watching %s", healthURL)

	// Загружаем секреты из .env (рядом с бинарником), если файл есть.
	loadDotEnv(".env")
	loadDotEnv("watchdog.env")

	// Фон: health-check + восстановление сервера.
	go healthLoop()

	// Основной цикл: события аудита из PostgreSQL.
	watchAudit()

	// Если watchAudit упал (нет БД) — держим health-check активным.
	select {}
}

// loadDotEnv читает файл KEY=VALUE в окружение (не перезаписывая установленные).
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

// healthLoop проверяет API и при отказе перезапускает сервер.
func healthLoop() {
	misses := 0
	for {
		ok := ping()
		if ok {
			misses = 0
		} else {
			misses++
			log.Printf("watchdog: health miss #%d", misses)
			if misses >= maxMisses {
				log.Printf("watchdog: server unhealthy (%d misses), restarting...", misses)
				restartServer()
				misses = 0
			}
		}
		time.Sleep(healthEvery)
	}
}

// ping делает HTTP-запрос к API и возвращает true при 200.
func ping() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// restartServer убивает и заново запускает сервер (тот же каталог).
func restartServer() {
	// Убиваем старый процесс.
	cmd := exec.Command("taskkill", "/IM", serverBin, "/F")
	_ = cmd.Run()
	time.Sleep(recoverWait)

	// Запускаем заново в текущем рабочем каталоге (где лежит бинарник).
	dir, _ := os.Getwd()
	starter := exec.Command(".\\" + serverBin)
	starter.Dir = dir
	starter.Stdout = os.Stdout
	starter.Stderr = os.Stderr
	if err := starter.Start(); err != nil {
		log.Printf("watchdog: restart failed: %v", err)
		return
	}
	log.Printf("watchdog: server restarted")
	go func() {
		_ = starter.Wait()
	}()
}

// watchAudit подписывается на события аудита PostgreSQL и реагирует на аномалии.
func watchAudit() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Printf("watchdog: DATABASE_URL not set, audit watcher disabled")
		return
	}
	for {
		if err := listenAndAnalyze(dsn); err != nil {
			log.Printf("watchdog: audit watcher error: %v (retry in 5s)", err)
			time.Sleep(5 * time.Second)
		}
	}
}

// listenAndAnalyze держит соединение LISTEN и анализирует события.
// Аномалия = блокировка входа (blocked_request / blocked) — фиксируем как alert.
func listenAndAnalyze(dsn string) error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "LISTEN audit_events"); err != nil {
		return err
	}
	log.Printf("watchdog: LISTEN audit_events")

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		payload := strings.ToLower(notification.Payload)
		// Реагируем только на значимые события — шум (failed, success) игнорируем.
		if strings.Contains(payload, "blocked") {
			log.Printf("watchdog: ALERT — blocked request: %s", notification.Payload)
			writeAlert(conn, notification.Payload)
		}
	}
}

// writeAlert записывает тревогу в таблицу alerts.
func writeAlert(conn *pgx.Conn, msg string) {
	_, _ = conn.Exec(context.Background(),
		`INSERT INTO alerts (id, source, message, created_at)
		 VALUES (gen_random_uuid(), 'watchdog', $1, now())`, msg)
}