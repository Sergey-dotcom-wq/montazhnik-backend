package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PartnerHandler обработчик партнёрской программы
type PartnerHandler struct {
	db *pgxpool.Pool
}

func NewPartnerHandler(db *pgxpool.Pool) *PartnerHandler {
	return &PartnerHandler{db: db}
}

// CheckCode проверяет реферальный код (GET /api/v1/partners/check-code?code=...)
func (h *PartnerHandler) CheckCode(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code required"})
		return
	}
	var id string
	var name string
	err := h.db.QueryRow(c.Request.Context(),
		`SELECT id, full_name FROM partners WHERE referral_code = $1 AND status = 'active'`, code).Scan(&id, &name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"valid": false, "error": "invalid or inactive code"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "partner_id": id, "partner_name": name})
}

// ListPartners возвращает список партнёров (GET /api/v1/partners)
// Доступ: admin, dispatcher, accountant, partner.
func (h *PartnerHandler) ListPartners(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id, inn, full_name, referral_code, payout_percent, bank_account, status, created_at
		 FROM partners ORDER BY full_name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type partnerRow struct {
		ID            string  `json:"id"`
		INN           string  `json:"inn"`
		FullName      string  `json:"full_name"`
		ReferralCode  string  `json:"referral_code"`
		PayoutPercent float64 `json:"payout_percent"`
		BankAccount   string  `json:"bank_account,omitempty"`
		Status        string  `json:"status"`
	}

	items := make([]partnerRow, 0)
	for rows.Next() {
		var p partnerRow
		var created interface{}
		if err := rows.Scan(&p.ID, &p.INN, &p.FullName, &p.ReferralCode, &p.PayoutPercent, &p.BankAccount, &p.Status, &created); err != nil {
			continue
		}
		items = append(items, p)
	}
	c.JSON(http.StatusOK, gin.H{"partners": items})
}

// ListReferrals возвращает рефералов (GET /api/v1/partners/referrals)
// Права: partner видит только рефералов своего партнёрского профиля,
// admin/accountant/dispatcher — всех (или по partner_id=...).
func (h *PartnerHandler) ListReferrals(c *gin.Context) {
	role := c.GetString("role")
	partnerID := c.Query("partner_id")

	var args []interface{}
	query := `
		SELECT r.id, r.partner_id, p.full_name AS partner_name, r.subscription_id,
		       r.amount, r.payout_status, r.created_at
		FROM referrals r
		JOIN partners p ON p.id = r.partner_id
		WHERE 1=1`

	// Партнёр видит только свои рефералы: определяем partner_id по user_id.
	if role == "partner" {
		userID := c.GetString("user_id")
		query += " AND r.partner_id = (SELECT id FROM partners WHERE user_id = $1)"
		args = append(args, userID)
	} else if partnerID != "" {
		query += " AND r.partner_id = $1"
		args = append(args, partnerID)
	}
	query += " ORDER BY r.created_at DESC"

	rows, err := h.db.Query(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type referralRow struct {
		ID             string  `json:"id"`
		PartnerID      string  `json:"partner_id"`
		PartnerName    string  `json:"partner_name"`
		SubscriptionID string  `json:"subscription_id"`
		Amount         float64 `json:"amount"`
		PayoutStatus   string  `json:"payout_status"`
		CreatedAt      string  `json:"created_at"`
	}

	items := make([]referralRow, 0)
	for rows.Next() {
		var r referralRow
		var created interface{}
		if err := rows.Scan(&r.ID, &r.PartnerID, &r.PartnerName, &r.SubscriptionID, &r.Amount, &r.PayoutStatus, &created); err != nil {
			continue
		}
		items = append(items, r)
	}
	c.JSON(http.StatusOK, gin.H{"referrals": items})
}