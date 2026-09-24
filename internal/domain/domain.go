package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// JobStatus статусы наряда (Матрица постоянных состояний)
type JobStatus string

const (
	StatusNew        JobStatus = "new"
	StatusAssigned   JobStatus = "assigned"
	StatusInProgress JobStatus = "in_progress"
	StatusCompleted  JobStatus = "completed"
	StatusCancelled  JobStatus = "cancelled"
)

// Job сущность наряда (Static State)
type Job struct {
	ID            uuid.UUID
	ExternalID    string
	Address       string
	Description   string
	Status        JobStatus
	AssignedTo    uuid.UUID
	PlannedStart  time.Time
	PlannedEnd    time.Time
	ActualStart   *time.Time
	ActualEnd     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// JobUpdate данные для обновления динамического состояния
type JobUpdate struct {
	Status      JobStatus
	ActualStart *time.Time
	ActualEnd   *time.Time
	Location    string // "lat,lon"
	UpdatedBy   uuid.UUID
	UpdatedAt   time.Time
}

// EventType типы событий (Матрица динамических изменений)
type EventType string

const (
	EventPhotoUploaded EventType = "photo_uploaded"
	EventMeasurement   EventType = "measurement_taken"
	EventStatusChanged EventType = "status_changed"
	EventMaterialScan  EventType = "material_scanned"
)

// EventInput входные данные для создания события
type EventInput struct {
	JobID     uuid.UUID
	EventType EventType
	Payload   map[string]interface{}
}

// Event сущность события в цепочке SHA-256
type Event struct {
	ID           uuid.UUID
	JobID        uuid.UUID
	EventType    EventType
	Payload      string // JSON
	PreviousHash string
	CurrentHash  string
	CreatedAt    time.Time
	CreatedBy    uuid.UUID
}

// JobService интерфейс для работы с нарядами
type JobService interface {
	Create(ctx context.Context, job *Job) (*Job, error)
	Get(ctx context.Context, id uuid.UUID) (*Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, update *JobUpdate) (*Job, error)
	List(ctx context.Context, filter *JobFilter, page, pageSize int) ([]*Job, int64, error)
}

// JobFilter параметры фильтрации
type JobFilter struct {
	Status      *JobStatus
	AssignedTo  *uuid.UUID
	PlannedGTE  *time.Time
	PlannedLTE  *time.Time
	ActualStart *time.Time
}

// EventService интерфейс для работы с событиями
type EventService interface {
	CreateEvent(ctx context.Context, input EventInput, userID uuid.UUID) (*Event, error)
	GetEventChain(ctx context.Context, jobID uuid.UUID) ([]*Event, error)
	VerifyChain(ctx context.Context, jobID uuid.UUID) (bool, error)
}

// Material сущность материала (Справочник / статичное состояние)
type Material struct {
	ID        uuid.UUID
	SKU       string
	Name      string
	Unit      string
	Price     float64
	VAT       float64
	Category  string
	CreatedAt time.Time
}

// JobMaterial расход материала по наряду (динамическое состояние)
type JobMaterial struct {
	ID         uuid.UUID
	JobID      uuid.UUID
	MaterialID uuid.UUID
	Quantity   float64
	Action     string // consumed | returned
	ScannedAt  time.Time
	ScannedBy  uuid.UUID
}

// MaterialRepository интерфейс для справочника материалов
type MaterialRepository interface {
	Create(ctx context.Context, m *Material) (*Material, error)
	GetBySKU(ctx context.Context, sku string) (*Material, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Material, error)
	List(ctx context.Context, category string) ([]*Material, error)
}

// JobMaterialRepository интерфейс для расхода материалов
type JobMaterialRepository interface {
	Create(ctx context.Context, jm *JobMaterial) (*JobMaterial, error)
	ListByJob(ctx context.Context, jobID uuid.UUID) ([]*JobMaterial, error)
}

// MaterialService сервис материалов
type MaterialService interface {
	Create(ctx context.Context, m *Material) (*Material, error)
	GetBySKU(ctx context.Context, sku string) (*Material, error)
	List(ctx context.Context, category string) ([]*Material, error)
	ScanMaterial(ctx context.Context, jobID uuid.UUID, sku string, qty float64, action string, userID uuid.UUID) (*JobMaterial, error)
	ListByJob(ctx context.Context, jobID uuid.UUID) ([]*JobMaterial, error)
}

// User сущность пользователя системы
type User struct {
	ID           uuid.UUID
	FullName     string
	Username     string // логин: ≤10 символов, латиница+цифры; ед. идентификатор
	Phone        string // необязательное поле профиля (не идентификатор)
	Email        string // для подтверждения/отправки кода (не идентификатор)
	Role         string // admin | dispatcher | installer
	PasswordHash string
	OrgID        uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Organization сущность коллектива (мультиаренда).
type Organization struct {
	ID         uuid.UUID
	Name       string
	CreatorID  uuid.UUID
	InviteCode string
	CreatedAt  time.Time
}

// OrganizationRepository — работа с коллективами.
type OrganizationRepository interface {
	Create(ctx context.Context, name string, creatorID uuid.UUID, inviteCode string) (uuid.UUID, error)
	FindByInvite(ctx context.Context, inviteCode string) (uuid.UUID, error)
	FindByName(ctx context.Context, name string) (uuid.UUID, error)
}

// UserRepository интерфейс для пользователей
type UserRepository interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	SetReferralPartner(ctx context.Context, userID, partnerID uuid.UUID) error
	ActivateUser(ctx context.Context, userID uuid.UUID) error
	PartnerIDByReferralCode(ctx context.Context, code string) (uuid.UUID, error)
	BindHwID(ctx context.Context, userID uuid.UUID, hwID string) error
	GetOrgID(ctx context.Context, userID uuid.UUID) uuid.UUID
	SetOrg(ctx context.Context, userID, orgID uuid.UUID) error
	SetRole(ctx context.Context, userID uuid.UUID, role string) error
	AnonymizeAccount(ctx context.Context, userID uuid.UUID) error
}

// AuthService сервис аутентификации
type AuthService interface {
	// Register: username — логин (≤10 латиница+цифры, уникальный), orgName — коллектив
	// (по желанию; существующий — присоединение, новый — создание, пользователь admin).
	// phone, email — для подтверждения и отправки кода (не идентификаторы).
	Register(ctx context.Context, fullName, username, password, role, orgName, phone, email, referralCode string) (*User, error)
	Login(ctx context.Context, username, password string) (string, *User, error) // token, user
	ValidateToken(tokenString string) (uuid.UUID, string, uuid.UUID, error)      // userID, role, orgID, err
	IssueToken(userID uuid.UUID) (string, *User, error)
	BindHwID(ctx context.Context, userID uuid.UUID, hwID string)
	AnonymizeAccount(ctx context.Context, userID uuid.UUID) error
}

// AlertSeverity уровни критичности оповещений.
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus статусы жизненного цикла оповещения.
type AlertStatus string

const (
	AlertStatusPending  AlertStatus = "pending"
	AlertStatusActive   AlertStatus = "active"
	AlertStatusAcked    AlertStatus = "acknowledged"
	AlertStatusExpired  AlertStatus = "expired"
)

// CriticalAlert критическое оповещение диспетчера: показывается на весь экран,
// не может быть закрыто без подтверждения получения (ack). Статус pending → active,
// когда клиент «подхватил» алерт; active → acknowledged после «Подтвердить получение».
type CriticalAlert struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	Title           string
	Message         string
	Severity        AlertSeverity
	Status          AlertStatus
	CreatedBy       uuid.UUID
	CreatedAt       time.Time
	AcknowledgedBy  *uuid.UUID
	AcknowledgedAt  *time.Time
}

// CriticalAlertRepository работа с критическими оповещениями.
type CriticalAlertRepository interface {
	Create(ctx context.Context, a *CriticalAlert) (*CriticalAlert, error)
	// ListPending возвращает оповещения коллектива со статусами pending/active
	// (не подтверждённые и уже «подхваченные»).
	ListPending(ctx context.Context, orgID uuid.UUID, limit int) ([]*CriticalAlert, error)
	// Acknowledge подтверждает получение алерта пользователем.
	Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	// MarkActive переводит pending → active (клиент получил алерт).
	MarkActive(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*CriticalAlert, error)
}

// ShiftStatus статусы рабочей смены инженера.
type ShiftStatus string

const (
	ShiftNone    ShiftStatus = "none"          // смены нет
	ShiftActive  ShiftStatus = "active"        // смена идёт
	ShiftPaused  ShiftStatus = "paused"        // смена на паузе (инженер отошёл)
	ShiftEnded   ShiftStatus = "ended"         // смена завершена
	ShiftLocked  ShiftStatus = "locked"        // утренняя блокировка: «Начать смену» до звонка диспетчеру
)

// WorkShift рабочая смена инженера (Field Guide: геозонирование смены).
// Пауза «без возврата»: 60 c → экран тускнеет, 5 мин → смена приостановлена.
// Утром кнопка «Начать смену» заблокирована до разблокировки диспетчером.
type WorkShift struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
	Status     ShiftStatus
	StartedAt  time.Time
	PausedAt   *time.Time
	ResumedAt  *time.Time
	EndedAt    *time.Time
	LastActive time.Time // последняя активность (heartbeat)
	// SleepUntil — момент, до которого смена заблокирована (утренняя блокировка).
	SleepUntil *time.Time
	// UnlockRequested — инженер нажал «Начать смену» при блокировке → диспетчеру
	// ушёл запрос на разблокировку; клиент показывает «позвоните диспетчеру».
	UnlockRequested bool
	UpdatedAt       time.Time
}

// WorkShiftRepository работа со сменами.
type WorkShiftRepository interface {
	// GetActive возвращает текущую (не завершённую) смену пользователя.
	GetActive(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
	// Start открывает новую смену пользователя.
	Start(ctx context.Context, s *WorkShift) (*WorkShift, error)
	// UpdateStatus меняет статус (paused/resumed/ended) и временные метки.
	UpdateStatus(ctx context.Context, id uuid.UUID, status ShiftStatus, ts time.Time) (*WorkShift, error)
	// SetSleep блокирует начало смены до указанного времени (утренняя блокировка).
	SetSleep(ctx context.Context, userID uuid.UUID, until time.Time) error
	// RequestUnlock фиксирует запрос инженера на разблокировку.
	RequestUnlock(ctx context.Context, id uuid.UUID) error
	// UnlockNow снимает блокировку (диспетчер).
	UnlockNow(ctx context.Context, userID uuid.UUID) error
	// Heartbeat обновляет время последней активности.
	Heartbeat(ctx context.Context, id uuid.UUID, ts time.Time) error
	// GetByUserLast возвращает последнюю смену пользователя (для статуса блокировки).
	GetByUserLast(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
}

// ShiftService сервис рабочих смен.
type ShiftService interface {
	StartShift(ctx context.Context, userID, orgID uuid.UUID) (*WorkShift, error)
	PauseShift(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
	ResumeShift(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
	EndShift(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
	GetCurrent(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
	RequestUnlock(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
	DispatchUnlock(ctx context.Context, userID uuid.UUID) (*WorkShift, error)
}

// ToolStatus состояние инструмента в сумке инженера.
type ToolStatus string

const (
	ToolBound   ToolStatus = "bound"   // привязан, норма
	ToolLost    ToolStatus = "lost"     // потерян
	ToolRemoved ToolStatus = "removed" // отвязан/снят с учёта
)

// Tool инвентарная единица инструмента в «виртуальной сумке» инженера.
type Tool struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	UserID     uuid.UUID
	Name       string // название (например, «OTDR EXFO MaxTester 730C»)
	Mac        string // BLE MAC-адрес
	Status     ToolStatus
	Battery    int // 0–100, -1 если неизвестно
	LastSeen   time.Time
	LostLat    *float64
	LostLng    *float64
	LostAt     *time.Time
	BoundAt    time.Time
	UpdatedAt  time.Time
}

// ToolRepository работа с инструментами.
type ToolRepository interface {
	// ListByUser возвращает инструменты пользователя (привязанные и потерянные).
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Tool, error)
	// Bind создаёт/привязывает инструмент к пользователю.
	Bind(ctx context.Context, t *Tool) (*Tool, error)
	// Unbind отвязывает инструмент (status = removed).
	Unbind(ctx context.Context, id uuid.UUID) error
	// MarkLost помечает инструмент как потерянный с координатами.
	MarkLost(ctx context.Context, id uuid.UUID, lat, lng float64) error
	// Heartbeat обновляет last_seen и батарею.
	Heartbeat(ctx context.Context, id uuid.UUID, battery int) error
	// GetByMac возвращает инструмент по MAC в рамках организации.
	GetByMac(ctx context.Context, orgID uuid.UUID, mac string) (*Tool, error)
	// GetByID возвращает инструмент по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*Tool, error)
}

// UserMaterial позиция в «виртуальной сумке» инженера (Field Guide §3):
// материалы не списываются окончательно, а попадают в персональную ячейку,
// физически они в багажнике, юридически — закреплены за нарядом.
type UserMaterial struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
	ItemID     uuid.UUID // ссылка на warehouse_items
	Name       string
	Unit       string
	Quantity   float64
	AssignedToJob string // номер наряда, за которым закреплён материал
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// UserMaterialRepository работа с сумкой материалов инженера.
type UserMaterialRepository interface {
	// ListByUser возвращает содержимое сумки пользователя.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*UserMaterial, error)
	// Add кладёт материал в сумку (увеличивает количество, если уже есть).
	Add(ctx context.Context, m *UserMaterial) (*UserMaterial, error)
	// Take убирает/списывает количество из сумки (для возврата на склад).
	Take(ctx context.Context, id uuid.UUID, quantity float64) error
	// GetByItem возвращает запись сумки по позиции склада.
	GetByItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*UserMaterial, error)
	// ReturnToWarehouse списывает количество из сумки и возвращает на склад.
	ReturnToWarehouse(ctx context.Context, id uuid.UUID, quantity float64) error
}
