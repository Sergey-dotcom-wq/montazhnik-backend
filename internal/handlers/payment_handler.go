package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Плановая стоимость подписки pro (₽).
const planPrice = 3000.00

// PaymentHandler обработчик платежей (приём через СБП + выплаты партнёрам).
type PaymentHandler struct {
	db       *pgxpool.Pool
	provider PaymentProvider
}

func NewPaymentHandler(db *pgxpool.Pool, provider PaymentProvider) *PaymentHandler {
	return &PaymentHandler{db: db, provider: provider}
}

// CreatePayment создаёт платёж и возвращает ссылку/QR для оплаты по СБП.
// POST /api/v1/payments/create  (публичный; user_id опционален:
// берётся из JWT, если пользователь уже вошёл; иначе платёж анонимный
// и активация произойдёт после входа по этому же device/платежу).
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID := uuid.Nil
	if userIDStr != "" {
		userID, _ = uuid.Parse(userIDStr)
	}

	var req struct {
		Product string `json:"product"` // subscription_month | subscription_year
	}
	_ = c.ShouldBindJSON(&req)
	product := req.Product
	if product == "" {
		product = "subscription_month"
	}

	paymentID := "PAY-" + uuid.NewString()[:13] + "-" + time.Now().Format("060102")
	created, err := h.provider.CreatePayment(c.Request.Context(), PaymentRequest{
		PaymentID: paymentID,
		UserID:    userID.String(),
		Product:   product,
		Amount:    planPrice,
	})
	if err != nil {
		log.Printf("create payment error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "payment provider error"})
		return
	}

	// Сохраняем черновик платежа (pending; станет paid по вебхуку).
	var userIDAny any
	if userID != uuid.Nil {
		userIDAny = userID
	}
	_, err = h.db.Exec(c.Request.Context(),
		`INSERT INTO payments (id, payment_id, user_id, amount, status, method, product, paid_at, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, 'pending', $4, $5, NULL, now())
		 ON CONFLICT (payment_id) DO NOTHING`,
		paymentID, userIDAny, planPrice, "sbp", product)
	if err != nil {
		log.Printf("create payment insert error: %v", err)
	}

	c.JSON(http.StatusOK, created)
}

// Webhook принимает уведомления от платёжного провайдера (Сбер СБП).
// POST /api/v1/webhook/payment  (публичный; подпись проверяет провайдер)
func (h *PaymentHandler) Webhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	signature := c.GetHeader("X-Signature")
	if signature == "" {
		signature = c.Query("signature")
	}

	// Проверка подписи у провайдера (mock — всегда ок).
	ok, err := h.provider.VerifyWebhook(c.Request.Context(), body, signature)
	if err != nil || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	status, err := h.provider.ParseWebhook(c.Request.Context(), body)
	if err != nil || status == nil || status.PaymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot parse webhook"})
		return
	}

	// Проверяем текущее состояние платежа: черновик (pending) — подтверждаем,
	// уже оплаченный (paid) — дубль, пропускаем.
	var currentStatus string
	var userIDStr string
	err = h.db.QueryRow(c.Request.Context(),
		`SELECT status, COALESCE(user_id::text, '') FROM payments WHERE payment_id = $1`,
		status.PaymentID).Scan(&currentStatus, &userIDStr)
	if err != nil {
		// Платёж не найден вовсе — пришёл без черновика CreatePayment.
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown payment"})
		return
	}
	if currentStatus == "paid" {
		c.JSON(http.StatusOK, gin.H{"status": "duplicate, skipped"})
		return
	}

	if status.Status != "paid" {
		// Неуспешный платёж фиксируем без активации.
		h.db.Exec(c.Request.Context(),
			`UPDATE payments SET status=$2 WHERE payment_id=$1`,
			status.PaymentID, "failed")
		c.JSON(http.StatusOK, gin.H{"status": "not paid"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil || userID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown payment"})
		return
	}

	nowMs := time.Now().UnixMilli()
	_, err = h.db.Exec(c.Request.Context(),
		`UPDATE payments SET status='paid', paid_at=$2 WHERE payment_id=$1`,
		status.PaymentID, nowMs)
	if err != nil {
		log.Printf("payment update error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	// Запускаем обработку в фоне: активация + начисления партнёру.
	go h.processPayment(status.PaymentID, userID)

	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// processPayment обрабатывает успешный платёж: активирует пользователя,
// при наличии партнёра начисляет 20% от каждого платежа (пожизненно).
func (h *PaymentHandler) processPayment(paymentID string, userID uuid.UUID) {
	ctx := context.Background()

	// Помечаем пользователя активированным (триал → активен).
	_, _ = h.db.Exec(ctx,
		`UPDATE users SET activated_at = COALESCE(activated_at, $2) WHERE id = $1`,
		userID, time.Now().UnixMilli())

	// Ищем партнёра пользователя.
	var partnerID uuid.UUID
	err := h.db.QueryRow(ctx,
		`SELECT referral_partner_id FROM users WHERE id = $1 AND referral_partner_id IS NOT NULL`,
		userID).Scan(&partnerID)
	if err != nil {
		return // без партнёра — ничего не начисляем
	}

	// 20% от каждого платежа (пожизненно).
	amount := planPrice * 20.0 / 100.0 // 600 ₽

	_, _ = h.db.Exec(ctx,
		`INSERT INTO referrals (id, partner_id, client_id, subscription_id, amount, payout_status, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, 'pending', now())`,
		partnerID, userID, paymentID, amount)
}

// ListPayments возвращает список платежей (GET /api/v1/admin/payments)
func (h *PaymentHandler) ListPayments(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT p.payment_id, COALESCE(u.full_name, '—'), p.amount, p.status, p.method, p.paid_at
		 FROM payments p LEFT JOIN users u ON u.id = p.user_id
		 ORDER BY p.created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type pRow struct {
		PaymentID string  `json:"payment_id"`
		UserName  string  `json:"user_name"`
		Amount    float64 `json:"amount"`
		Status    string  `json:"status"`
		Method    string  `json:"method"`
		PaidAt    *int64  `json:"paid_at"`
	}
	items := make([]pRow, 0)
	for rows.Next() {
		var r pRow
		if err := rows.Scan(&r.PaymentID, &r.UserName, &r.Amount, &r.Status, &r.Method, &r.PaidAt); err != nil {
			continue
		}
		items = append(items, r)
	}
	c.JSON(http.StatusOK, gin.H{"payments": items})
}

// ProcessPayouts выполняет автоматические СБП-выплаты партнёрам по порогу.
// GET /api/v1/admin/payments/process-payouts  (JWT, роль admin/dispatcher)
// Начисления ≥ 1000 ₽ переводятся партнёру на карту по номеру телефона
// (СБП-выплата через провайдера; в mock-режиме — сразу paid).
func (h *PaymentHandler) ProcessPayouts(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT r.partner_id, p.payout_phone, SUM(r.amount) AS total
		 FROM referrals r JOIN partners p ON p.id = r.partner_id
		 WHERE r.payout_status = 'pending'
		 GROUP BY r.partner_id, p.payout_phone
		 HAVING SUM(r.amount) >= 1000`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type payoutRow struct {
		PartnerID string  `json:"partner_id"`
		Phone     string  `json:"phone"`
		Total     float64 `json:"total"`
		Status    string  `json:"status"`
	}
	items := make([]payoutRow, 0)
	for rows.Next() {
		var r payoutRow
		if err := rows.Scan(&r.PartnerID, &r.Phone, &r.Total); err != nil {
			continue
		}

		res, perr := h.provider.Payout(c.Request.Context(), PayoutRequest{
			PartnerID:   r.PartnerID,
			Amount:      r.Total,
			PhoneNumber: r.Phone,
			Comment:     "Партнёрское вознаграждение Монтажник Связи",
		})
		if perr != nil {
			log.Printf("payout error partner=%s: %v", r.PartnerID, perr)
			r.Status = "failed"
			items = append(items, r)
			continue
		}

		r.Status = res.Status
		items = append(items, r)

		nowMs := time.Now().UnixMilli()
		h.db.Exec(c.Request.Context(),
			`INSERT INTO payouts (id, partner_id, amount, status, bank_account, paid_at, created_at)
			 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, now())`,
			r.PartnerID, r.Total, res.Status, r.Phone, nowMs)

		if res.Status == "paid" {
			h.db.Exec(c.Request.Context(),
				`UPDATE referrals SET payout_status = 'paid' WHERE partner_id = $1 AND payout_status = 'pending'`,
				r.PartnerID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"payouts": items})
}

// WebhookPayload — обратная совместимость: старый формат вебхука (для тестов).
type WebhookPayload struct {
	PaymentID string `json:"payment_id"`
	UserID    string `json:"user_id"`
	Amount    float64 `json:"amount"`
	Method    string `json:"method"`
	Product   string `json:"product"`
	Status    string `json:"status"`
}

// mockParseWebhook разбирает старый формат тестового вебхука в PaymentStatus.
func mockParseWebhook(raw []byte) (*PaymentStatus, error) {
	var p WebhookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	status := "pending"
	if p.Status == "paid" || p.Status == "" {
		status = "paid"
	}
	return &PaymentStatus{
		PaymentID: p.PaymentID,
		Status:    status,
		Amount:    p.Amount,
	}, nil
}