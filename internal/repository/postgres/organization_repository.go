package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrganizationRepositoryPostgres — коллективы (мультиаренда) в PostgreSQL.
type OrganizationRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewOrganizationRepositoryPostgres(db *pgxpool.Pool) *OrganizationRepositoryPostgres {
	return &OrganizationRepositoryPostgres{db: db}
}

// Create создаёт коллектив и возвращает его id.
func (r *OrganizationRepositoryPostgres) Create(ctx context.Context, name string, creatorID uuid.UUID, inviteCode string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`INSERT INTO organizations (id, name, creator_id, invite_code, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, now())
		 RETURNING id`,
		name, creatorID, inviteCode).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// FindByName ищет коллектив по точному имени.
func (r *OrganizationRepositoryPostgres) FindByName(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM organizations WHERE name = $1`, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errors.New("not found")
	}
	return id, err
}

// FindByInvite ищет коллектив по коду приглашения.
func (r *OrganizationRepositoryPostgres) FindByInvite(ctx context.Context, inviteCode string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT id FROM organizations WHERE invite_code = $1`, inviteCode).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errors.New("not found")
	}
	return id, err
}