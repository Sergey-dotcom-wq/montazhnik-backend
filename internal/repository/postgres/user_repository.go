package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// UserRepositoryPostgres репозиторий пользователей (PostgreSQL)
type UserRepositoryPostgres struct {
	db *pgxpool.Pool
}

// NewUserRepositoryPostgres конструктор
func NewUserRepositoryPostgres(db *pgxpool.Pool) *UserRepositoryPostgres {
	return &UserRepositoryPostgres{db: db}
}

func (r *UserRepositoryPostgres) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (id, username, full_name, phone, email, role, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), $6, $7, $8, $8)
		RETURNING id, username, COALESCE(full_name,''), COALESCE(phone,''), COALESCE(email,''), role, password_hash, created_at, updated_at`
	err := r.db.QueryRow(ctx, query,
		u.ID, u.Username, u.FullName, u.Phone, u.Email, u.Role, u.PasswordHash, u.CreatedAt,
	).Scan(&u.ID, &u.Username, &u.FullName, &u.Phone, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (r *UserRepositoryPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u := &domain.User{}
	query := `
		SELECT id, username, COALESCE(full_name,''), COALESCE(phone,''), COALESCE(email,''), role, password_hash, created_at, updated_at
		FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&u.ID, &u.Username, &u.FullName, &u.Phone, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepositoryPostgres) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	u := &domain.User{}
	query := `
		SELECT id, username, COALESCE(full_name,''), COALESCE(phone,''), COALESCE(email,''), role, password_hash, created_at, updated_at
		FROM users WHERE username = $1`
	err := r.db.QueryRow(ctx, query, username).Scan(&u.ID, &u.Username, &u.FullName, &u.Phone, &u.Email, &u.Role, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// SetReferralPartner привязывает пользователя к партнёру (по реферальному коду).
func (r *UserRepositoryPostgres) SetReferralPartner(ctx context.Context, userID, partnerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET referral_partner_id = $2 WHERE id = $1`, userID, partnerID)
	return err
}

// ActivateUser помечает пользователя активированным (триал/подписка).
func (r *UserRepositoryPostgres) ActivateUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET activated_at = CASE WHEN activated_at IS NULL THEN $2 ELSE activated_at END
		 WHERE id = $1`, userID, time.Now().UnixMilli())
	return err
}

// PartnerIDByReferralCode возвращает ID партнёра по реферальному коду.
func (r *UserRepositoryPostgres) PartnerIDByReferralCode(ctx context.Context, code string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM partners WHERE referral_code = $1 AND status = 'active'`, code).Scan(&id)
	return id, err
}

// SetOrg привязывает пользователя к организации (мультиаренда).
func (r *UserRepositoryPostgres) SetOrg(ctx context.Context, userID, orgID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET org_id = $2 WHERE id = $1`, userID, orgID)
	return err
}

// SetRole обновляет роль пользователя.
func (r *UserRepositoryPostgres) SetRole(ctx context.Context, userID uuid.UUID, role string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET role = $2 WHERE id = $1`, userID, role)
	return err
}

// AnonymizeAccount удаляет персональные данные пользователя (152-ФЗ).
// Сохраняет только username (не ПДн по 152-ФЗ), UUID и org_id.
// full_name, phone, password_hash очищаются (phone=NULL — nullable,
// чтобы не конфликтовать с UNIQUE при множественных анонимах).
func (r *UserRepositoryPostgres) AnonymizeAccount(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET
			full_name = 'Аноним',
			phone = NULL,
			password_hash = '',
			updated_at = now()
		 WHERE id = $1`, userID)
	return err
}

// BindHwID привязывает устройство к пользователю (hw_id).
func (r *UserRepositoryPostgres) BindHwID(ctx context.Context, userID uuid.UUID, hwID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET hw_id = COALESCE(hw_id, $2) WHERE id = $1`, userID, hwID)
	return err
}

// GetOrgID возвращает org_id пользователя (uuid.Nil, если не привязан).
func (r *UserRepositoryPostgres) GetOrgID(ctx context.Context, userID uuid.UUID) uuid.UUID {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT org_id FROM users WHERE id = $1`, userID).Scan(&id)
	if err != nil {
		return uuid.Nil
	}
	return id
}