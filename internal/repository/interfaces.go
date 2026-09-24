package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// JobRepository интерфейс для работы с нарядами (статичное состояние)
type JobRepository interface {
	Create(ctx context.Context, job *domain.Job) (*domain.Job, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, update *domain.JobUpdate) (*domain.Job, error)
	List(ctx context.Context, filter *domain.JobFilter, page, pageSize int) ([]*domain.Job, int64, error)
}

// EventRepository интерфейс для работы с цепочкой событий (динамическое состояние)
type EventRepository interface {
	CreateEvent(ctx context.Context, event *domain.Event) (*domain.Event, error)
	GetChain(ctx context.Context, jobID uuid.UUID) ([]*domain.Event, error)
	VerifyChain(ctx context.Context, jobID uuid.UUID) (bool, error)
}