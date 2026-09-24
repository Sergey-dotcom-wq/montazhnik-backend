package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// MaterialRepositoryPostgres репозиторий материалов (PostgreSQL)
type MaterialRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewMaterialRepositoryPostgres(db *pgxpool.Pool) *MaterialRepositoryPostgres {
	return &MaterialRepositoryPostgres{db: db}
}

func (r *MaterialRepositoryPostgres) Create(ctx context.Context, m *domain.Material) (*domain.Material, error) {
	query := `
		INSERT INTO materials (id, sku, name, unit, price, vat, category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, sku, name, unit, price, vat, category, created_at`
	err := r.db.QueryRow(ctx, query,
		m.ID, m.SKU, m.Name, m.Unit, m.Price, m.VAT, m.Category, m.CreatedAt,
	).Scan(&m.ID, &m.SKU, &m.Name, &m.Unit, &m.Price, &m.VAT, &m.Category, &m.CreatedAt)
	return m, err
}

func (r *MaterialRepositoryPostgres) GetBySKU(ctx context.Context, sku string) (*domain.Material, error) {
	m := &domain.Material{}
	query := `SELECT id, sku, name, unit, price, vat, category, created_at FROM materials WHERE sku = $1`
	err := r.db.QueryRow(ctx, query, sku).Scan(&m.ID, &m.SKU, &m.Name, &m.Unit, &m.Price, &m.VAT, &m.Category, &m.CreatedAt)
	return m, err
}

func (r *MaterialRepositoryPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.Material, error) {
	m := &domain.Material{}
	query := `SELECT id, sku, name, unit, price, vat, category, created_at FROM materials WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&m.ID, &m.SKU, &m.Name, &m.Unit, &m.Price, &m.VAT, &m.Category, &m.CreatedAt)
	return m, err
}

func (r *MaterialRepositoryPostgres) List(ctx context.Context, category string) ([]*domain.Material, error) {
	query := `SELECT id, sku, name, unit, price, vat, category, created_at FROM materials`
	var args []interface{}

	if category != "" {
		query += " WHERE category = $1"
		args = append(args, category)
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.Material
	for rows.Next() {
		m := &domain.Material{}
		if err := rows.Scan(&m.ID, &m.SKU, &m.Name, &m.Unit, &m.Price, &m.VAT, &m.Category, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, nil
}

// JobMaterialRepositoryPostgres репозиторий расхода материалов
type JobMaterialRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewJobMaterialRepositoryPostgres(db *pgxpool.Pool) *JobMaterialRepositoryPostgres {
	return &JobMaterialRepositoryPostgres{db: db}
}

func (r *JobMaterialRepositoryPostgres) Create(ctx context.Context, jm *domain.JobMaterial) (*domain.JobMaterial, error) {
	query := `
		INSERT INTO job_materials (id, job_id, material_id, quantity, action, scanned_at, scanned_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, job_id, material_id, quantity, action, scanned_at, scanned_by`
	err := r.db.QueryRow(ctx, query,
		jm.ID, jm.JobID, jm.MaterialID, jm.Quantity, jm.Action, jm.ScannedAt, jm.ScannedBy,
	).Scan(&jm.ID, &jm.JobID, &jm.MaterialID, &jm.Quantity, &jm.Action, &jm.ScannedAt, &jm.ScannedBy)
	return jm, err
}

func (r *JobMaterialRepositoryPostgres) ListByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.JobMaterial, error) {
	query := `
		SELECT jm.id, jm.job_id, jm.material_id, jm.quantity, jm.action, jm.scanned_at, jm.scanned_by,
		       m.sku, m.name, m.unit
		FROM job_materials jm
		JOIN materials m ON m.id = jm.material_id
		WHERE jm.job_id = $1
		ORDER BY jm.scanned_at ASC`
	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.JobMaterial
	for rows.Next() {
		jm := &domain.JobMaterial{}
		var sku, name, unit string
		if err := rows.Scan(&jm.ID, &jm.JobID, &jm.MaterialID, &jm.Quantity, &jm.Action, &jm.ScannedAt, &jm.ScannedBy,
			&sku, &name, &unit); err != nil {
			return nil, err
		}
		items = append(items, jm)
	}
	return items, nil
}