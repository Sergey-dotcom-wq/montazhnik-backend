package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// JobRepositoryPostgres реализация JobRepository через PostgreSQL
type JobRepositoryPostgres struct {
	db *pgxpool.Pool
}

// NewJobRepositoryPostgres конструктор
func NewJobRepositoryPostgres(db *pgxpool.Pool) *JobRepositoryPostgres {
	return &JobRepositoryPostgres{db: db}
}

// Create создает новый наряд
func (r *JobRepositoryPostgres) Create(ctx context.Context, job *domain.Job) (*domain.Job, error) {
	query := `
		INSERT INTO jobs (
			id, external_id, address, description, status, assigned_to,
			planned_start, planned_end, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, external_id, address, description, status, assigned_to,
			planned_start, planned_end, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		job.ID, job.ExternalID, job.Address, job.Description, job.Status, job.AssignedTo,
		job.PlannedStart, job.PlannedEnd, job.CreatedAt, job.UpdatedAt,
	).Scan(
		&job.ID, &job.ExternalID, &job.Address, &job.Description, &job.Status, &job.AssignedTo,
		&job.PlannedStart, &job.PlannedEnd, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return job, nil
}

// Get получает наряд по ID
func (r *JobRepositoryPostgres) Get(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	job := &domain.Job{}
	query := `
		SELECT id, external_id, address, description, status, assigned_to,
			planned_start, planned_end, actual_start, actual_end, created_at, updated_at
		FROM jobs WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID, &job.ExternalID, &job.Address, &job.Description, &job.Status, &job.AssignedTo,
		&job.PlannedStart, &job.PlannedEnd, &job.ActualStart, &job.ActualEnd, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return job, nil
}

// UpdateStatus обновляет статус наряда (динамическое состояние)
func (r *JobRepositoryPostgres) UpdateStatus(ctx context.Context, id uuid.UUID, update *domain.JobUpdate) (*domain.Job, error) {
	query := `
		UPDATE jobs 
		SET status = $2, actual_start = $3, actual_end = $4, location = $5, 
			updated_at = $6, updated_by = $7
		WHERE id = $1
		RETURNING id, external_id, address, description, status, assigned_to,
			planned_start, planned_end, actual_start, actual_end, created_at, updated_at`

	job := &domain.Job{}
	err := r.db.QueryRow(ctx, query,
		id, update.Status, update.ActualStart, update.ActualEnd, update.Location,
		update.UpdatedAt, update.UpdatedBy,
	).Scan(
		&job.ID, &job.ExternalID, &job.Address, &job.Description, &job.Status, &job.AssignedTo,
		&job.PlannedStart, &job.PlannedEnd, &job.ActualStart, &job.ActualEnd, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return job, nil
}

// List список нарядов с фильтрацией
func (r *JobRepositoryPostgres) List(ctx context.Context, filter *domain.JobFilter, page, pageSize int) ([]*domain.Job, int64, error) {
	var jobs []*domain.Job
	baseQuery := "SELECT id, external_id, address, description, status, assigned_to, planned_start, planned_end, created_at, updated_at FROM jobs WHERE 1=1"
	var args []interface{}
	var conditions []string
	argPos := 1

	if filter != nil {
		if filter.Status != nil {
			conditions = append(conditions, "status = $"+fmt.Sprintf("%d", argPos))
			args = append(args, *filter.Status)
			argPos++
		}
		if filter.AssignedTo != nil {
			conditions = append(conditions, "assigned_to = $"+fmt.Sprintf("%d", argPos))
			args = append(args, *filter.AssignedTo)
			argPos++
		}
		if filter.PlannedGTE != nil {
			conditions = append(conditions, "planned_start >= $"+fmt.Sprintf("%d", argPos))
			args = append(args, *filter.PlannedGTE)
			argPos++
		}
		if filter.PlannedLTE != nil {
			conditions = append(conditions, "planned_start <= $"+fmt.Sprintf("%d", argPos))
			args = append(args, *filter.PlannedLTE)
			argPos++
		}
	}

	for _, cond := range conditions {
		baseQuery += " AND " + cond
	}

	// Подсчет общего количества
	countQuery := "SELECT COUNT(*) FROM (" + baseQuery + ") AS sub"
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Пагинация
	baseQuery += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", argPos) + " OFFSET $" + fmt.Sprintf("%d", argPos+1)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, total, err
	}
	defer rows.Close()

	for rows.Next() {
		job := &domain.Job{}
		err := rows.Scan(
			&job.ID, &job.ExternalID, &job.Address, &job.Description, &job.Status, &job.AssignedTo,
			&job.PlannedStart, &job.PlannedEnd, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, total, err
		}
		jobs = append(jobs, job)
	}

	return jobs, total, nil
}

// EventRepositoryPostgres реализация EventRepository
type EventRepositoryPostgres struct {
	db *pgxpool.Pool
}

// NewEventRepositoryPostgres конструктор
func NewEventRepositoryPostgres(db *pgxpool.Pool) *EventRepositoryPostgres {
	return &EventRepositoryPostgres{db: db}
}

// CreateEvent создает событие в цепочке SHA-256
func (r *EventRepositoryPostgres) CreateEvent(ctx context.Context, event *domain.Event) (*domain.Event, error) {
	query := `
		INSERT INTO job_events (
			id, job_id, event_type, payload, previous_hash, current_hash, created_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, job_id, event_type, payload, previous_hash, current_hash, created_at, created_by`

	err := r.db.QueryRow(ctx, query,
		event.ID, event.JobID, event.EventType, event.Payload, event.PreviousHash, event.CurrentHash,
		event.CreatedAt, event.CreatedBy,
	).Scan(
		&event.ID, &event.JobID, &event.EventType, &event.Payload, &event.PreviousHash, &event.CurrentHash,
		&event.CreatedAt, &event.CreatedBy,
	)

	if err != nil {
		return nil, err
	}
	return event, nil
}

// GetChain возвращает полную цепочку событий
func (r *EventRepositoryPostgres) GetChain(ctx context.Context, jobID uuid.UUID) ([]*domain.Event, error) {
	var events []*domain.Event
	query := `
		SELECT id, job_id, event_type, payload, previous_hash, current_hash, created_at, created_by
		FROM job_events 
		WHERE job_id = $1 
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		event := &domain.Event{}
		err := rows.Scan(
			&event.ID, &event.JobID, &event.EventType, &event.Payload, &event.PreviousHash, &event.CurrentHash,
			&event.CreatedAt, &event.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

// VerifyChain проверяет целостность цепочки хешей
func (r *EventRepositoryPostgres) VerifyChain(ctx context.Context, jobID uuid.UUID) (bool, error) {
	events, err := r.GetChain(ctx, jobID)
	if err != nil {
		return false, err
	}

	if len(events) == 0 {
		return true, nil // Пустая цепочка валидна
	}

	// Проверяем первый элемент (у него должен быть genesis hash или пустой previous_hash)
	if events[0].PreviousHash != "" && events[0].PreviousHash != "0000000000000000000000000000000000000000000000000000000000000000" {
		return false, nil
	}

	// Проверяем связность цепочки
	for i := 1; i < len(events); i++ {
		if events[i].PreviousHash != events[i-1].CurrentHash {
			return false, nil
		}
	}

	return true, nil
}
