package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Partner сущность партнера
type Partner struct {
	ID            uuid.UUID
	INN           string
	FullName      string
	ReferralCode  string
	PayoutPercent float64
	BankAccount   string
	Status        string
	CreatedAt     time.Time
}

// Referral сущность реферала
type Referral struct {
	ID             uuid.UUID
	PartnerID      uuid.UUID
	ClientID       uuid.UUID
	SubscriptionID string
	Amount         float64
	PayoutStatus   string
	PayoutDate     *time.Time
	CreatedAt      time.Time
}

// PartnerRepositoryPostgres репозиторий партнеров
type PartnerRepositoryPostgres struct {
	db *pgxpool.Pool
}

// NewPartnerRepositoryPostgres конструктор
func NewPartnerRepositoryPostgres(db *pgxpool.Pool) *PartnerRepositoryPostgres {
	return &PartnerRepositoryPostgres{db: db}
}

// Create создает партнера
func (r *PartnerRepositoryPostgres) Create(ctx context.Context, p *Partner) (*Partner, error) {
	query := `
		INSERT INTO partners (id, inn, full_name, referral_code, payout_percent, bank_account, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, inn, full_name, referral_code, payout_percent, bank_account, status, created_at`

	err := r.db.QueryRow(ctx, query,
		p.ID, p.INN, p.FullName, p.ReferralCode, p.PayoutPercent, p.BankAccount, p.Status, p.CreatedAt,
	).Scan(
		&p.ID, &p.INN, &p.FullName, &p.ReferralCode, &p.PayoutPercent, &p.BankAccount, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetByINN получает партнера по ИНН
func (r *PartnerRepositoryPostgres) GetByINN(ctx context.Context, inn string) (*Partner, error) {
	p := &Partner{}
	query := `
		SELECT id, inn, full_name, referral_code, payout_percent, bank_account, status, created_at
		FROM partners WHERE inn = $1`

	err := r.db.QueryRow(ctx, query, inn).Scan(
		&p.ID, &p.INN, &p.FullName, &p.ReferralCode, &p.PayoutPercent, &p.BankAccount, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetByReferralCode получает партнера по реферальному коду
func (r *PartnerRepositoryPostgres) GetByReferralCode(ctx context.Context, code string) (*Partner, error) {
	p := &Partner{}
	query := `
		SELECT id, inn, full_name, referral_code, payout_percent, bank_account, status, created_at
		FROM partners WHERE referral_code = $1`

	err := r.db.QueryRow(ctx, query, code).Scan(
		&p.ID, &p.INN, &p.FullName, &p.ReferralCode, &p.PayoutPercent, &p.BankAccount, &p.Status, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// AddReferral создает запись о привлеченном клиенте
func (r *PartnerRepositoryPostgres) AddReferral(ctx context.Context, ref *Referral) (*Referral, error) {
	query := `
		INSERT INTO referrals (id, partner_id, client_id, subscription_id, amount, payout_status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, partner_id, client_id, subscription_id, amount, payout_status, payout_date, created_at`

	err := r.db.QueryRow(ctx, query,
		ref.ID, ref.PartnerID, ref.ClientID, ref.SubscriptionID, ref.Amount, ref.PayoutStatus, ref.CreatedAt,
	).Scan(
		&ref.ID, &ref.PartnerID, &ref.ClientID, &ref.SubscriptionID, &ref.Amount, &ref.PayoutStatus, &ref.PayoutDate, &ref.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return ref, nil
}