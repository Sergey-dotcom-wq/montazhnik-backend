package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// UserMaterialRepositoryPostgres репозиторий «виртуальной сумки» материалов.
type UserMaterialRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewUserMaterialRepositoryPostgres(db *pgxpool.Pool) *UserMaterialRepositoryPostgres {
	return &UserMaterialRepositoryPostgres{db: db}
}

const userMaterialColumns = `id, user_id, org_id, item_id, name, unit, quantity, assigned_to_job, created_at, updated_at`

func scanUserMaterial(row interface{ Scan(...interface{}) error }, m *domain.UserMaterial) error {
	return row.Scan(
		&m.ID, &m.UserID, &m.OrgID, &m.ItemID, &m.Name, &m.Unit, &m.Quantity,
		&m.AssignedToJob, &m.CreatedAt, &m.UpdatedAt,
	)
}

func (r *UserMaterialRepositoryPostgres) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserMaterial, error) {
	query := `SELECT ` + userMaterialColumns + ` FROM user_materials
		WHERE user_id = $1 AND quantity > 0
		ORDER BY name ASC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.UserMaterial
	for rows.Next() {
		m := &domain.UserMaterial{}
		if err := scanUserMaterial(rows, m); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *UserMaterialRepositoryPostgres) Add(ctx context.Context, m *domain.UserMaterial) (*domain.UserMaterial, error) {
	query := `
		INSERT INTO user_materials (id, user_id, org_id, item_id, name, unit, quantity, assigned_to_job)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, item_id) DO UPDATE SET
			quantity = user_materials.quantity + EXCLUDED.quantity,
			name = EXCLUDED.name,
			unit = EXCLUDED.unit,
			assigned_to_job = EXCLUDED.assigned_to_job,
			updated_at = now()
		RETURNING ` + userMaterialColumns
	err := scanUserMaterial(r.db.QueryRow(ctx, query,
		m.ID, m.UserID, m.OrgID, m.ItemID, m.Name, m.Unit, m.Quantity, m.AssignedToJob,
	), m)
	return m, err
}

func (r *UserMaterialRepositoryPostgres) Take(ctx context.Context, id uuid.UUID, quantity float64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_materials SET quantity = GREATEST(quantity - $2, 0), updated_at = now() WHERE id = $1`,
		id, quantity)
	return err
}

func (r *UserMaterialRepositoryPostgres) GetByItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*domain.UserMaterial, error) {
	m := &domain.UserMaterial{}
	query := `SELECT ` + userMaterialColumns + ` FROM user_materials WHERE user_id = $1 AND item_id = $2`
	err := scanUserMaterial(r.db.QueryRow(ctx, query, userID, itemID), m)
	return m, err
}

// ReturnToWarehouse: списываем из сумки и возвращаем количество на склад.
// Всё в одной транзакции, чтобы не было расхождений.
func (r *UserMaterialRepositoryPostgres) ReturnToWarehouse(ctx context.Context, id uuid.UUID, quantity float64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Узнаём item_id и текущее количество.
	var itemID uuid.UUID
	var current float64
	if err := tx.QueryRow(ctx,
		`SELECT item_id, quantity FROM user_materials WHERE id = $1`, id).Scan(&itemID, &current); err != nil {
		return err
	}

	// Списываем из сумки.
	if _, err := tx.Exec(ctx,
		`UPDATE user_materials SET quantity = GREATEST(quantity - $2, 0), updated_at = now() WHERE id = $1`,
		id, quantity); err != nil {
		return err
	}

	// Возвращаем на склад.
	if _, err := tx.Exec(ctx,
		`UPDATE warehouse_items SET quantity = quantity + $2, updated_at = now() WHERE id = $1`,
		itemID, quantity); err != nil {
		return err
	}

	return tx.Commit(ctx)
}