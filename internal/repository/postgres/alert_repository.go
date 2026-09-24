package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// AlertRepositoryPostgres репозиторий критических оповещений.
type AlertRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewAlertRepositoryPostgres(db *pgxpool.Pool) *AlertRepositoryPostgres {
	return &AlertRepositoryPostgres{db: db}
}

func scanAlert(row interface{ Scan(...interface{}) error }, a *domain.CriticalAlert) error {
	return row.Scan(
		&a.ID, &a.OrgID, &a.Title, &a.Message, &a.Severity, &a.Status,
		&a.CreatedBy, &a.CreatedAt, &a.AcknowledgedBy, &a.AcknowledgedAt,
	)
}

func (r *AlertRepositoryPostgres) Create(ctx context.Context, a *domain.CriticalAlert) (*domain.CriticalAlert, error) {
	query := `
		INSERT INTO critical_alerts (id, org_id, title, message, severity, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, org_id, title, message, severity, status, created_by, created_at, acknowledged_by, acknowledged_at`
	err := scanAlert(r.db.QueryRow(ctx, query,
		a.ID, a.OrgID, a.Title, a.Message, a.Severity, a.Status, a.CreatedBy,
	), a)
	return a, err
}

func (r *AlertRepositoryPostgres) ListPending(ctx context.Context, orgID uuid.UUID, limit int) ([]*domain.CriticalAlert, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := `
		SELECT id, org_id, title, message, severity, status, created_by, created_at, acknowledged_by, acknowledged_at
		FROM critical_alerts
		WHERE org_id = $1 AND status IN ('pending','active')
		ORDER BY created_at DESC
		LIMIT $2`
	rows, err := r.db.Query(ctx, query, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.CriticalAlert
	for rows.Next() {
		a := &domain.CriticalAlert{}
		if err := scanAlert(rows, a); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *AlertRepositoryPostgres) Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE critical_alerts
		SET status = 'acknowledged', acknowledged_by = $2, acknowledged_at = now()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, userID)
	return err
}

func (r *AlertRepositoryPostgres) MarkActive(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE critical_alerts SET status = 'active' WHERE id = $1 AND status = 'pending'`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *AlertRepositoryPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.CriticalAlert, error) {
	a := &domain.CriticalAlert{}
	query := `
		SELECT id, org_id, title, message, severity, status, created_by, created_at, acknowledged_by, acknowledged_at
		FROM critical_alerts
		WHERE id = $1`
	err := scanAlert(r.db.QueryRow(ctx, query, id), a)
	return a, err
}