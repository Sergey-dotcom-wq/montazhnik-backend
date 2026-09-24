package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// ToolRepositoryPostgres репозиторий инструментов «виртуальной сумки».
type ToolRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewToolRepositoryPostgres(db *pgxpool.Pool) *ToolRepositoryPostgres {
	return &ToolRepositoryPostgres{db: db}
}

const toolColumns = `id, org_id, user_id, name, mac, status, battery, last_seen, lost_lat, lost_lng, lost_at, bound_at, updated_at`

func scanTool(row interface{ Scan(...interface{}) error }, t *domain.Tool) error {
	return row.Scan(
		&t.ID, &t.OrgID, &t.UserID, &t.Name, &t.Mac, &t.Status,
		&t.Battery, &t.LastSeen, &t.LostLat, &t.LostLng, &t.LostAt, &t.BoundAt, &t.UpdatedAt,
	)
}

func (r *ToolRepositoryPostgres) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Tool, error) {
	query := `SELECT ` + toolColumns + ` FROM tools
		WHERE user_id = $1 AND status != 'removed'
		ORDER BY status DESC, name ASC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.Tool
	for rows.Next() {
		t := &domain.Tool{}
		if err := scanTool(rows, t); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (r *ToolRepositoryPostgres) Bind(ctx context.Context, t *domain.Tool) (*domain.Tool, error) {
	query := `
		INSERT INTO tools (id, org_id, user_id, name, mac, status, battery, last_seen, bound_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (org_id, mac) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			name = EXCLUDED.name,
			status = 'bound',
			updated_at = now(),
			bound_at = now()
		RETURNING ` + toolColumns
	err := scanTool(r.db.QueryRow(ctx, query,
		t.ID, t.OrgID, t.UserID, t.Name, t.Mac, t.Status, t.Battery, t.LastSeen, t.BoundAt, t.UpdatedAt,
	), t)
	return t, err
}

func (r *ToolRepositoryPostgres) Unbind(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tools SET status = 'removed', updated_at = now() WHERE id = $1`, id)
	return err
}

func (r *ToolRepositoryPostgres) MarkLost(ctx context.Context, id uuid.UUID, lat, lng float64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tools SET status = 'lost', lost_lat = $2, lost_lng = $3, lost_at = now(), updated_at = now()
		 WHERE id = $1`, id, lat, lng)
	return err
}

func (r *ToolRepositoryPostgres) Heartbeat(ctx context.Context, id uuid.UUID, battery int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tools SET battery = $2, last_seen = now(), updated_at = now() WHERE id = $1`,
		id, battery)
	return err
}

func (r *ToolRepositoryPostgres) GetByMac(ctx context.Context, orgID uuid.UUID, mac string) (*domain.Tool, error) {
	t := &domain.Tool{}
	query := `SELECT ` + toolColumns + ` FROM tools WHERE org_id = $1 AND mac = $2`
	err := scanTool(r.db.QueryRow(ctx, query, orgID, mac), t)
	return t, err
}

func (r *ToolRepositoryPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tool, error) {
	t := &domain.Tool{}
	query := `SELECT ` + toolColumns + ` FROM tools WHERE id = $1`
	err := scanTool(r.db.QueryRow(ctx, query, id), t)
	return t, err
}