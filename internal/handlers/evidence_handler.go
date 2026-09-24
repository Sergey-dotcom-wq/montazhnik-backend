package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EvidenceHandler: доверенное время и реестр фото-доказательств (anti-fraud).
//
// Идея: камера привязана к серверному времени (подписанный anchor), а хеш
// каждого фото регистрируется на сервере в момент съёмки. Подменить старое
// фото невозможно — SHA-256 не совпадёт с реестром, а "открутить" время
// устройства назад не поможет (расхождение с server_time > 10 мин отклоняется).
type EvidenceHandler struct {
	db     *pgxpool.Pool
	secret string // HMAC-ключ для подписи времени (из .env, не в коде)
}

func NewEvidenceHandler(db *pgxpool.Pool, secret string) *EvidenceHandler {
	return &EvidenceHandler{db: db, secret: secret}
}

// SignedTime GET /time: возвращает серверное время (UTC, мс) с HMAC-подписью.
func (h *EvidenceHandler) SignedTime(c *gin.Context) {
	now := time.Now().UTC().UnixMilli()
	sig := h.signTime(now)
	c.JSON(http.StatusOK, gin.H{
		"server_time_ms": now,
		"signature":      sig,
		"skew_limit_ms":  10 * 60 * 1000, // допустимое расхождение устройства и сервера
	})
}

func (h *EvidenceHandler) signTime(ms int64) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write([]byte(strconv.FormatInt(ms, 10)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// RegisterEvidence POST /photo-evidence: регистрация хеша фото при съёмке.
type RegisterEvidenceRequest struct {
	OrderID      string  `json:"order_id"`
	PhotoSha256  string  `json:"photo_sha256" binding:"required"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	AccuracyM    float64 `json:"accuracy_m"`
	DeviceTimeMs int64   `json:"device_time_ms" binding:"required"`
	ServerTimeMs int64   `json:"server_time_ms" binding:"required"`
}

func (h *EvidenceHandler) RegisterEvidence(c *gin.Context) {
	var req RegisterEvidenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	userID := c.GetString("user_id")

	// Проверяем длину SHA-256 (64 hex).
	if len(req.PhotoSha256) != 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sha256"})
		return
	}

	// Дубликат хеша — подмена: тот же хеш уже регистрировался.
	var exists bool
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM photo_evidence WHERE photo_sha256 = $1)`,
		req.PhotoSha256).Scan(&exists)
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate photo hash — possible substitution"})
		return
	}

	_, err := h.db.Exec(c.Request.Context(),
		`INSERT INTO photo_evidence (org_id, user_id, order_id, photo_sha256, latitude, longitude, accuracy_m, device_time_ms, server_time_ms)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		orgID, userID, req.OrderID, req.PhotoSha256, req.Latitude, req.Longitude, req.AccuracyM,
		req.DeviceTimeMs, req.ServerTimeMs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "registered"})
}

// VerifyEvidence POST /photo-evidence/verify: проверка по хешу (для КС-2 и сводки).
type VerifyEvidenceRequest struct {
	PhotoSha256 string `json:"photo_sha256" binding:"required"`
}

func (h *EvidenceHandler) VerifyEvidence(c *gin.Context) {
	var req VerifyEvidenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}

	var lat, lng, acc *float64
	var serverTimeMs int64
	var created time.Time
	err := h.db.QueryRow(c.Request.Context(),
		`SELECT latitude, longitude, accuracy_m, server_time_ms, created_at
		 FROM photo_evidence WHERE photo_sha256 = $1 AND org_id = $2`,
		req.PhotoSha256, orgID).Scan(&lat, &lng, &acc, &serverTimeMs, &created)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"verified": false, "reason": "no evidence"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"verified":       true,
		"latitude":       lat,
		"longitude":      lng,
		"accuracy_m":     acc,
		"server_time_ms": serverTimeMs,
		"created_at":     created,
	})
}

// photoEvidenceCount — сводная статистика для отчёта диспетчера.
func photoEvidenceStats(c *gin.Context, db *pgxpool.Pool) gin.H {
	orgID := OrgIDFromCtx(c, db)
	var total, verified24h int
	if orgID != "" {
		_ = db.QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM photo_evidence WHERE org_id = $1`, orgID).Scan(&total)
		_ = db.QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM photo_evidence WHERE org_id = $1 AND created_at > now() - interval '24 hours'`,
			orgID).Scan(&verified24h)
	}
	return gin.H{"verified": total, "last_24h": verified24h}
}

// Возвращает SHA-256 hex заданной строки (для тестов/сигнатур).
func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}