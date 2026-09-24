package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Тарифы и доли партнёрской программы.
const (
	// PlanPriceRub — стоимость подписки pro (₽/мес).
	PlanPriceRub = 3000.0
	// ReferrerPercent — доля реферера от каждого платежа реферала (20%, пожизненно).
	ReferrerPercent = 20.0
	// ReferralDiscountPercent — скидка реферала на первый месяц (30%).
	ReferralDiscountPercent = 30.0
)

// SubscriptionHandler обработчик подписки
type SubscriptionHandler struct {
	db *pgxpool.Pool
}

func NewSubscriptionHandler(db *pgxpool.Pool) *SubscriptionHandler {
	return &SubscriptionHandler{db: db}
}

// Длительность учебного (бесплатного) триала в днях.
const TrialDays = 14

// SubscriptionStatus возвращает статус подписки (GET /api/v1/subscription/status?hw_id=...)
// Состояние считается по данным пользователя в БД:
//   - активирован (activated_at есть, т.е. была оплата) → active
//   - триал не истёк (<= 14 дней с регистрации) → trial
//   - триал истёк и оплаты нет → trial_expired (вход только после оплаты)
func (h *SubscriptionHandler) Status(c *gin.Context) {
	hwID := c.Query("hw_id")
	if hwID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hw_id required"})
		return
	}

	var (
		activatedAt *int64
		createdAtMs int64
	)
	nowMs := time.Now().UnixMilli()

	err := h.db.QueryRow(c.Request.Context(),
		`SELECT activated_at, COALESCE(trial_start_ms, EXTRACT(EPOCH FROM created_at)*1000)
		 FROM users WHERE hw_id = $1`, hwID).Scan(&activatedAt, &createdAtMs)
	if err != nil {
		// Устройство неизвестно — триал ещё не начался (регистрацию не прошли),
		// для UI это то же самое, что и активный триал.
		c.JSON(http.StatusOK, gin.H{
			"state": "trial", "plan": "pro", "holder": hwID,
			"days_left": TrialDays,
			"expires_at_ms": time.UnixMilli(createdAtMs).AddDate(0, 0, TrialDays).UnixMilli(),
			"trial_days": TrialDays,
		})
		return
	}

	if activatedAt != nil {
		// Оплата была — подписка активна.
		c.JSON(http.StatusOK, gin.H{
			"state": "active", "plan": "pro", "holder": hwID,
			"days_left": 365,
			"expires_at_ms": time.UnixMilli(*activatedAt).AddDate(1, 0, 0).UnixMilli(),
			"trial_days": TrialDays,
		})
		return
	}

	// Триал: сколько дней прошло с регистрации.
	trialEndMs := time.UnixMilli(createdAtMs).AddDate(0, 0, TrialDays).UnixMilli()
	daysLeft := int((trialEndMs - nowMs) / (24 * 60 * 60 * 1000))
	if nowMs > trialEndMs {
		// Триал истёк, оплаты нет — доступ только после оплаты.
		c.JSON(http.StatusOK, gin.H{
			"state": "trial_expired", "plan": "pro", "holder": hwID,
			"days_left": 0,
			"expires_at_ms": trialEndMs,
			"trial_days": TrialDays,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"state": "trial", "plan": "pro", "holder": hwID,
		"days_left": daysLeft,
		"expires_at_ms": trialEndMs,
		"trial_days": TrialDays,
	})
}

// Activate активирует подписку (POST /api/v1/subscription/activate)
// JWT-защищён: user_id берётся из токена. Если пользователь зарегистрирован
// по реферальному коду партнёра — создаётся запись в referrals с начислением
// 30% от первого платежа партнёру, а реферал получает скидку 10%.
func (h *SubscriptionHandler) Activate(c *gin.Context) {
	var req struct {
		HwID string `json:"hw_id"`
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// Помечаем пользователя активированным (триал завершён, подписка активна).
	_, _ = h.db.Exec(c.Request.Context(),
		`UPDATE users SET activated_at = COALESCE(activated_at, $2) WHERE id = $1`,
		userID, time.Now().UnixMilli())

	// Находим привязку к партнёру (если регистрировались по реферальному коду).
	var partnerID uuid.UUID
	hasPartner := false
	err = h.db.QueryRow(c.Request.Context(),
		`SELECT referral_partner_id FROM users WHERE id = $1 AND referral_partner_id IS NOT NULL`,
		userID).Scan(&partnerID)
	if err == nil {
		hasPartner = true
	}

	// Считаем итоговую цену: реферал платит 70% в первый месяц (скидка 30%).
	price := PlanPriceRub
	discount := 0.0
	if hasPartner {
		discount = PlanPriceRub * ReferralDiscountPercent / 100.0
		price = PlanPriceRub - discount
	}

	// Если есть партнёр — создаём реферальную выплату 20% от полной цены.
	referralCreated := false
	partnerEarn := 0.0
	if hasPartner {
		partnerEarn = PlanPriceRub * ReferrerPercent / 100.0
		subscriptionID := "SUB-" + req.HwID
		if subscriptionID == "SUB-" {
			subscriptionID = "SUB-" + userID.String()
		}
		_, err = h.db.Exec(c.Request.Context(),
			`INSERT INTO referrals (id, partner_id, client_id, subscription_id, amount, payout_status, created_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, $4, 'pending', now())`,
			partnerID, userID, subscriptionID, partnerEarn)
		if err == nil {
			referralCreated = true
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            "activated",
		"plan":              "pro",
		"price":             price,
		"discount":          discount,
		"referral_discount": discount > 0,
		"partner_earn":      partnerEarn,
		"referral_created":  referralCreated,
	})
}