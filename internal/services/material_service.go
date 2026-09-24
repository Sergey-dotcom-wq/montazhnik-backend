package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// MaterialServiceImpl реализация MaterialService
type MaterialServiceImpl struct {
	materialRepo    domain.MaterialRepository
	jobMaterialRepo domain.JobMaterialRepository
}

// NewMaterialService конструктор
func NewMaterialService(mr domain.MaterialRepository, jmr domain.JobMaterialRepository) *MaterialServiceImpl {
	return &MaterialServiceImpl{materialRepo: mr, jobMaterialRepo: jmr}
}

func (s *MaterialServiceImpl) Create(ctx context.Context, m *domain.Material) (*domain.Material, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	m.CreatedAt = time.Now().UTC()
	return s.materialRepo.Create(ctx, m)
}

func (s *MaterialServiceImpl) GetBySKU(ctx context.Context, sku string) (*domain.Material, error) {
	return s.materialRepo.GetBySKU(ctx, sku)
}

func (s *MaterialServiceImpl) List(ctx context.Context, category string) ([]*domain.Material, error) {
	return s.materialRepo.List(ctx, category)
}

func (s *MaterialServiceImpl) ScanMaterial(ctx context.Context, jobID uuid.UUID, sku string, qty float64, action string, userID uuid.UUID) (*domain.JobMaterial, error) {
	m, err := s.materialRepo.GetBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	jm := &domain.JobMaterial{
		ID:         uuid.New(),
		JobID:      jobID,
		MaterialID: m.ID,
		Quantity:   qty,
		Action:     action,
		ScannedAt:  time.Now().UTC(),
		ScannedBy:  userID,
	}

	return s.jobMaterialRepo.Create(ctx, jm)
}

func (s *MaterialServiceImpl) ListByJob(ctx context.Context, jobID uuid.UUID) ([]*domain.JobMaterial, error) {
	return s.jobMaterialRepo.ListByJob(ctx, jobID)
}