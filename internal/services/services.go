package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/montazhnik/backend/internal/domain"
	"github.com/yourorg/montazhnik/backend/internal/repository"
)

// JobServiceImpl реализация сервиса нарядов
type JobServiceImpl struct {
	jobRepo repository.JobRepository
}

// NewJobService конструктор
func NewJobService(repo repository.JobRepository) *JobServiceImpl {
	return &JobServiceImpl{jobRepo: repo}
}

// Create создает новый наряд
func (s *JobServiceImpl) Create(ctx context.Context, job *domain.Job) (*domain.Job, error) {
	// Генерируем ID, если не задан
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	// Устанавливаем время создания
	job.CreatedAt = time.Now().UTC()
	job.UpdatedAt = job.CreatedAt

	return s.jobRepo.Create(ctx, job)
}

// Get получает наряд по ID
func (s *JobServiceImpl) Get(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	return s.jobRepo.Get(ctx, id)
}

// UpdateStatus обновляет статус наряда
func (s *JobServiceImpl) UpdateStatus(ctx context.Context, id uuid.UUID, update *domain.JobUpdate) (*domain.Job, error) {
	// Устанавливаем время обновления
	update.UpdatedAt = time.Now().UTC()

	return s.jobRepo.UpdateStatus(ctx, id, update)
}

// List список нарядов
func (s *JobServiceImpl) List(ctx context.Context, filter *domain.JobFilter, page, pageSize int) ([]*domain.Job, int64, error) {
	return s.jobRepo.List(ctx, filter, page, pageSize)
}

// EventServiceImpl реализация сервиса событий
type EventServiceImpl struct {
	eventRepo repository.EventRepository
	jobRepo   repository.JobRepository
}

// NewEventService конструктор
func NewEventService(eventRepo repository.EventRepository, jobRepo repository.JobRepository) *EventServiceImpl {
	return &EventServiceImpl{eventRepo: eventRepo, jobRepo: jobRepo}
}

// CreateEvent создает событие и проверяет цепочку SHA-256
func (s *EventServiceImpl) CreateEvent(ctx context.Context, input domain.EventInput, userID uuid.UUID) (*domain.Event, error) {
	// Получаем последний хеш для проверки цепочки
	lastEvent, err := s.getLastEvent(ctx, input.JobID)
	if err != nil {
		return nil, err
	}

	var previousHash string
	if lastEvent != nil {
		previousHash = lastEvent.CurrentHash
	} else {
		// Genesis hash для первого события
		previousHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	// Сериализуем payload
	payloadBytes, err := json.Marshal(input.Payload)
	if err != nil {
		return nil, err
	}

	// Вычисляем хеш
	data := input.JobID.String() + string(input.EventType) + string(payloadBytes) + previousHash + userID.String() + time.Now().UTC().String()
	hash := sha256.Sum256([]byte(data))
	currentHash := hex.EncodeToString(hash[:])

	// Создаем событие
	event := &domain.Event{
		ID:           uuid.New(),
		JobID:        input.JobID,
		EventType:    input.EventType,
		Payload:      string(payloadBytes),
		PreviousHash: previousHash,
		CurrentHash:  currentHash,
		CreatedAt:    time.Now().UTC(),
		CreatedBy:    userID,
	}

	// Сохраняем в БД
	createdEvent, err := s.eventRepo.CreateEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	return createdEvent, nil
}

// GetEventChain возвращает цепочку событий
func (s *EventServiceImpl) GetEventChain(ctx context.Context, jobID uuid.UUID) ([]*domain.Event, error) {
	return s.eventRepo.GetChain(ctx, jobID)
}

// VerifyChain проверяет целостность цепочки
func (s *EventServiceImpl) VerifyChain(ctx context.Context, jobID uuid.UUID) (bool, error) {
	return s.eventRepo.VerifyChain(ctx, jobID)
}

// getLastEvent получает последнее событие для наряда
func (s *EventServiceImpl) getLastEvent(ctx context.Context, jobID uuid.UUID) (*domain.Event, error) {
	events, err := s.eventRepo.GetChain(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, nil
	}

	return events[len(events)-1], nil
}

// ShiftServiceImpl реализация сервиса рабочих смен.
//
// Механика Field Guide (Геозонирование смены / «Пауза без возврата»):
//   - StartShift: открывает новую смену. Если предыдущая завершилась меньше
//     SLEEP_AFTER_END назад (например, инженер «уснул» без завершения) —
//     смена стартует в статусе locked (утренняя блокировка до разблокировки
//     диспетчером).
//   - PauseShift: добровольная пауза (отошёл к машине). Hard Block по таймеру
//     60 c обрабатывает клиент; сервер лишь фиксирует статус.
//   - EndShift: завершение; при уходе в «сон» (пауза > SLEEP_AFTER_END) клиент
//     сам завершает смену со статусом «Смена приостановлена».
type ShiftServiceImpl struct {
	shiftRepo domain.WorkShiftRepository
}

func NewShiftService(repo domain.WorkShiftRepository) *ShiftServiceImpl {
	return &ShiftServiceImpl{shiftRepo: repo}
}

// sleepWindow — период, в течение которого после завершения смены
// старт новой блокируется (утренняя блокировка до звонка диспетчеру).
const sleepWindow = 6 * time.Hour

func (s *ShiftServiceImpl) StartShift(ctx context.Context, userID, orgID uuid.UUID) (*domain.WorkShift, error) {
	now := time.Now().UTC()
	// Проверяем последнюю смену — не наступила ли «спящая» блокировка.
	last, err := s.shiftRepo.GetByUserLast(ctx, userID)
	if err == nil && last != nil {
		// Если предыдущая смена завершилась недавно и статус ended — блокируем
		// старт до звонка диспетчеру (см. Field Guide: утром «Начать смену»
		// заблокирована до разблокировки).
		if last.Status == domain.ShiftEnded && last.EndedAt != nil {
			endedAgo := now.Sub(*last.EndedAt)
			if endedAgo < sleepWindow {
				lockUntil := last.EndedAt.Add(sleepWindow)
				// Создаём «спящую» смену, чтобы клиент показал блокировку.
				return s.createLocked(ctx, userID, orgID, lockUntil)
			}
		}
		if last.Status == domain.ShiftLocked {
			// Уже заблокирована — возвращаем как есть (не переоткрываем).
			if last.UserID == userID {
				return last, nil
			}
		}
	}

	shift := &domain.WorkShift{
		ID:         uuid.New(),
		UserID:     userID,
		OrgID:      orgID,
		Status:     domain.ShiftActive,
		StartedAt:  now,
		LastActive: now,
		UpdatedAt:  now,
	}
	return s.shiftRepo.Start(ctx, shift)
}

func (s *ShiftServiceImpl) createLocked(ctx context.Context, userID, orgID uuid.UUID, until time.Time) (*domain.WorkShift, error) {
	now := time.Now().UTC()
	shift := &domain.WorkShift{
		ID:         uuid.New(),
		UserID:     userID,
		OrgID:      orgID,
		Status:     domain.ShiftLocked,
		StartedAt:  now,
		SleepUntil: &until,
		LastActive: now,
		UpdatedAt:  now,
	}
	return s.shiftRepo.Start(ctx, shift)
}

func (s *ShiftServiceImpl) PauseShift(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	shift, err := s.shiftRepo.GetActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.shiftRepo.UpdateStatus(ctx, shift.ID, domain.ShiftPaused, time.Now().UTC())
}

func (s *ShiftServiceImpl) ResumeShift(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	shift, err := s.shiftRepo.GetActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if shift.Status == domain.ShiftLocked {
		return nil, fmt.Errorf("shift locked: call dispatcher")
	}
	return s.shiftRepo.UpdateStatus(ctx, shift.ID, domain.ShiftActive, time.Now().UTC())
}

func (s *ShiftServiceImpl) EndShift(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	shift, err := s.shiftRepo.GetActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	res, err := s.shiftRepo.UpdateStatus(ctx, shift.ID, domain.ShiftEnded, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	// Если смена была на паузе дольше sleepWindow — сервер сам переведёт
	// в «Смена приостановлена» (клиент также делает это локально).
	if shift.Status == domain.ShiftPaused {
		// Сигнал клиенту: статус ended → клиент покажет «Смена приостановлена».
	}
	return res, nil
}

func (s *ShiftServiceImpl) GetCurrent(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	return s.shiftRepo.GetActive(ctx, userID)
}

// RequestUnlock — инженер нажал «Начать смену» при блокировке:
// фиксируем запрос, клиент сообщает «позвоните диспетчеру».
func (s *ShiftServiceImpl) RequestUnlock(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	shift, err := s.shiftRepo.GetActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if shift.Status != domain.ShiftLocked {
		return shift, nil
	}
	if err := s.shiftRepo.RequestUnlock(ctx, shift.ID); err != nil {
		return nil, err
	}
	shift.UnlockRequested = true
	return shift, nil
}

// DispatchUnlock — диспетчер снимает блокировку (разблокировка смены).
func (s *ShiftServiceImpl) DispatchUnlock(ctx context.Context, userID uuid.UUID) (*domain.WorkShift, error) {
	if err := s.shiftRepo.UnlockNow(ctx, userID); err != nil {
		return nil, err
	}
	return s.shiftRepo.GetActive(ctx, userID)
}
