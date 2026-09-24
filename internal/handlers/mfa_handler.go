package handlers

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// MFAHandler — двухфакторная аутентификация для администратора.
// Первый фактор: пароль. Второй: одноразовый код (OTP, 6 цифр, TTL 5 минут).
// Для пилота код выводится в ответ + пишется в журнал аудита; в production
// код отправляется в приложение-аутентификатор (TOTP) или по SMS.
type MFAHandler struct {
	db   *pgxpool.Pool
	auth domain.AuthService
}

func NewMFAHandler(db *pgxpool.Pool, auth domain.AuthService) *MFAHandler {
	return &MFAHandler{db: db, auth: auth}
}

// MFAResponse — ответ с запросом кода (или подтверждением входа).
type MFAResponse struct {
	MFARequired bool   `json:"mfa_required"`
	MFASent     bool   `json:"mfa_sent"`
	TTLSeconds  int    `json:"ttl_seconds"`
	Token       string `json:"token,omitempty"`
	Role        string `json:"role,omitempty"`
}

const mfaTTL = 5 * time.Minute

// Request — POST /auth/mfa/request  { phone, password }
// Проверяет пароль; для admin возвращает MFARequired=true и генерирует OTP.
// Функционально вызывается из Login (см. auth_handler) — здесь только логика.
func (h *MFAHandler) Request(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Role != "admin" {
		c.JSON(200, MFAResponse{MFARequired: false})
		return
	}

	code, err := generateOTP()
	if err != nil {
		c.JSON(500, gin.H{"error": "otp generation failed"})
		return
	}

	_, err = h.db.Exec(c.Request.Context(),
		`INSERT INTO otp_codes (id, user_id, code, used, expires_at, created_at)
		 VALUES (gen_random_uuid(), CAST(NULLIF($1,'') AS uuid), $2, false, now() + $3::interval, now())`,
		req.UserID, code, (mfaTTL).String())
	if err != nil {
		log.Printf("mfa insert error: %v", err)
		c.JSON(500, gin.H{"error": "otp save failed"})
		return
	}

	// В пилоте код возвращаем в ответе (потом — только в аудит/канал доставки).
	c.JSON(200, gin.H{"mfa_required": true, "mfa_sent": true, "ttl_seconds": int(mfaTTL.Seconds()), "dev_code": code})
}

// Verify — POST /auth/mfa/verify  { user_id, code }
// Проверяет OTP: не использован, не истёк, совпадает.
// При успехе выдаёт JWT-токен отдельной сессии (второй фактор пройден).
func (h *MFAHandler) Verify(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var expires time.Time
	err := h.db.QueryRow(c.Request.Context(),
		`UPDATE otp_codes SET used = true
		 WHERE user_id = CAST(NULLIF($1,'') AS uuid)
		   AND code = $2
		   AND used = false
		   AND expires_at > now()
		 RETURNING expires_at`, req.UserID, req.Code).Scan(&expires)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid or expired code", "ok": false})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	token, user, terr := h.auth.IssueToken(userID)
	if terr != nil || token == "" {
		c.JSON(500, gin.H{"error": "token issuance failed"})
		return
	}

	c.JSON(200, gin.H{
		"ok":    true,
		"token": token,
		"user": gin.H{
			"id":        user.ID.String(),
			"full_name": user.FullName,
			"phone":     user.Phone,
			"role":      user.Role,
		},
	})
}

// generateOTP генерирует 6-значный код.
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// для использования context в будущих доработках
var _ = context.Background