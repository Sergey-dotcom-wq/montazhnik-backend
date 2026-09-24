package handlers

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LoginRateLimiter — защита от перебора пароля (брутфорс).
// Держит счётчики неудачных попыток по IP и по номеру телефона.
// После N неудач — блокировка на 15 минут (429 Too Many Requests).
type LoginRateLimiter struct {
	mu       sync.Mutex
	byIP     map[string]*rlBucket
	byPhone  map[string]*rlBucket
	maxFail  int
	window   time.Duration
}

type rlBucket struct {
	fails    int
	blockedUntil time.Time
}

// NewLoginRateLimiter создаёт лимитер: maxFail неудачных попыток за окно.
func NewLoginRateLimiter(maxFail int, window time.Duration) *LoginRateLimiter {
	if maxFail <= 0 {
		maxFail = 5
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &LoginRateLimiter{
		byIP:    make(map[string]*rlBucket),
		byPhone: make(map[string]*rlBucket),
		maxFail: maxFail,
		window:  window,
	}
}

// IsBlocked возвращает true, если ключ (IP или телефон) заблокирован.
func (l *LoginRateLimiter) IsBlocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.byIP[key]
	if b == nil {
		b = l.byPhone[key]
	}
	if b == nil {
		return false
	}
	return time.Now().Before(b.blockedUntil)
}

// RegisterFailed фиксирует неудачную попытку; возвращает true, если наступила блокировка.
func (l *LoginRateLimiter) RegisterFailed(ip, phone string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()

	b1 := l.byIP[ip]
	if b1 == nil {
		b1 = &rlBucket{}
		l.byIP[ip] = b1
	}
	b1.fails++
	if now.After(b1.blockedUntil) && b1.fails >= l.maxFail {
		b1.blockedUntil = now.Add(l.window)
		b1.fails = 0
	}

	if phone != "" {
		b2 := l.byPhone[phone]
		if b2 == nil {
			b2 = &rlBucket{}
			l.byPhone[phone] = b2
		}
		b2.fails++
		if now.After(b2.blockedUntil) && b2.fails >= l.maxFail {
			b2.blockedUntil = now.Add(l.window)
			b2.fails = 0
		}
	}
	return l.isBlockedLocked(ip, phone)
}

// RegisterSuccess сбрасывает счётчики после успешного входа.
func (l *LoginRateLimiter) RegisterSuccess(ip, phone string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.byIP, ip)
	if phone != "" {
		delete(l.byPhone, phone)
	}
}

func (l *LoginRateLimiter) isBlockedLocked(ip, phone string) bool {
	now := time.Now()
	if b := l.byIP[ip]; b != nil && now.Before(b.blockedUntil) {
		return true
	}
	if phone != "" {
		if b := l.byPhone[phone]; b != nil && now.Before(b.blockedUntil) {
			return true
		}
	}
	return false
}

// LoginAttempts — состояние для ответа клиенту (сколько осталось попыток).
func (l *LoginRateLimiter) LoginAttempts(ip, phone string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	remaining := l.maxFail
	if b := l.byIP[ip]; b != nil {
		if left := l.maxFail - b.fails; left < remaining {
			remaining = left
		}
	}
	if phone != "" {
		if b := l.byPhone[phone]; b != nil {
			if left := l.maxFail - b.fails; left < remaining {
				remaining = left
			}
		}
	}
	if remaining < 1 {
		remaining = 1
	}
	return remaining
}

// ClientIP извлекает IP клиента (прокси-заголовки учитываются).
func ClientIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = c.GetHeader("X-Real-IP")
	}
	if ip == "" {
		ip = c.ClientIP()
	}
	return ip
}

// OrgIDFromCtx возвращает org_id текущего пользователя для изоляции коллективов.
// Если в JWT org нет (старый токен) — достаёт из БД по user_id (fallback).
// Пустая строка = нет организации (доступ к данным своего коллектива запрещён).
func OrgIDFromCtx(c *gin.Context, db *pgxpool.Pool) string {
	if v := c.GetString("org_id"); v != "" {
		return v
	}
	userID := c.GetString("user_id")
	if userID == "" {
		return ""
	}
	var orgID string
	err := db.QueryRow(c.Request.Context(),
		`SELECT COALESCE(org_id::text, '') FROM users WHERE id = $1`, userID).Scan(&orgID)
	if err != nil {
		return ""
	}
	if orgID != "" {
		c.Set("org_id", orgID)
	}
	return orgID
}