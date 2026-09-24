package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// ShiftRepositoryPostgres репозиторий рабочих смен.
type ShiftRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewShiftRepositoryPostgres(db *pgxpool.Pool) *ShiftRepositoryPostgres {
	return &ShiftRepositoryPostgres{db: db}
}

const shiftColumns = `id, user_id, org_id, status, started_at, paused_at, resumed_at, ended_at, last_active, sleep_until, unlock_requested, updated_at`

func scanShift(row interface{ Scan(...interface{}) error }, s *domain.WorkShift) error {
	return row.Scan(
		&s.ID, &s.UserID, &s.OrgID, &s.Status,
		&s.StartedAt, &s.PausedAt, &s.ResumedAt, &s.EndedAt,
		&s.LastActive, &s.SleepUntil, &s.UnlockRequested, &s.UpdatedAt,
	)
}

// GetActive возвращает не завершённую смену пользователя.
func (r *ShiftRepositoryPostgres) GetActive(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	s := &domain.WorkShift{}
	query := `SELECT ` + shiftColumns + ` FROM work_shifts
		WHERE user_id = $1 AND status IN ('active','paused','locked')
		ORDER BY started_at DESC LIMIT 1`
	err := scanShift(r.db.QueryRow(ctx, query, userID), s)
	return s, err
}

// GetByUserLast возвращает последнюю смену пользователя (в т.ч. завершённую).
func (r *ShiftRepositoryPostgres) GetByUserLast(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	s := &domain.WorkShift{}
	query := `SELECT ` + shiftColumns + ` FROM work_shifts
		WHERE user_id = $1 ORDER BY started_at DESC LIMIT 1`
	err := scanShift(r.db.QueryRow(ctx, query, userID), s)
	return s, err
}

func (r *ShiftRepositoryPostgres) Start(ctx context.Context, s *domain.WorkShift) (*domain.WorkShift, error) {
	query := `
		INSERT INTO work_shifts (id, user_id, org_id, status, started_at, last_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + shiftColumns
	err := scanShift(r.db.QueryRow(ctx, query,
		s.ID, s.UserID, s.OrgID, s.Status, s.StartedAt, s.LastActive, s.UpdatedAt,
	), s)
	return s, err
}

func (r *ShiftRepositoryPostgres) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ShiftStatus, ts time.Time) (*domain.WorkShift, error) {
	s := &domain.WorkShift{}
	query := `
		UPDATE work_shifts
		SET status = $2, updated_at = $3, last_active = $3,
		    paused_at = CASE WHEN $2 = 'paused' THEN $3 ELSE paused_at END,
		    resumed_at = CASE WHEN $2 = 'active' AND status = 'paused' THEN $3 ELSE resumed_at END,
		    ended_at   = CASE WHEN $2 = 'ended' THEN $3 ELSE ended_at END
		WHERE id = $1
		RETURNING ` + shiftColumns
	err := scanShift(r.db.QueryRow(ctx, query, id, status, ts), s)
	return s, err
}

func (r *ShiftRepositoryPostgres) SetSleep(ctx context.Context, userID uuid.UUID, until time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE work_shifts SET status = 'locked', sleep_until = $2, unlock_requested = false, updated_at = now(), last_active = now()
		 WHERE user_id = $1 AND status IN ('active','paused')`,
		userID, until)
	return err
}

func (r *ShiftRepositoryPostgres) RequestUnlock(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE work_shifts SET unlock_requested = true, updated_at = now() WHERE id = $1`, id)
	return err
}

func (r *ShiftRepositoryPostgres) UnlockNow(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE work_shifts SET status = 'active', sleep_until = NULL, unlock_requested = false,
		        resumed_at = now(), updated_at = now(), last_active = now()
		 WHERE user_id = $1 AND status = 'locked'`,
		userID)
	return err
}

func (r *ShiftRepositoryPostgres) Heartbeat(ctx context.Context, id uuid.UUID, ts time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE work_shifts SET last_active = $2, updated_at = $2 WHERE id = $1`, id, ts)
	return err
}