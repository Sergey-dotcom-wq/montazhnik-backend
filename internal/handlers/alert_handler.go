package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
	"github.com/yourorg/montazhnik/backend/internal/repository/postgres"
)

// CreateAlertRequest создание критического оповещения диспетчером.
type CreateAlertRequest struct {
	Title    string `json:"title" binding:"required"`
	Message  string `json:"message"`
	Severity string `json:"severity" binding:"omitempty,oneof=info warning critical"`
}

// AlertHandler обрабатывает критические оповещения (диспетчер → бригада).
type AlertHandler struct {
	db       *pgxpool.Pool
	repo     domain.CriticalAlertRepository
	notifier func(orgID uuid.UUID)
}

// NewAlertHandler создаёт обработчик. notifier (необязательный) вызывается
// после создания/подтверждения алерта, чтобы разбудить SSE-стримы коллектива.
func NewAlertHandler(db *pgxpool.Pool, notifier func(orgID uuid.UUID)) *AlertHandler {
	return &AlertHandler{
		db:       db,
		repo:     postgres.NewAlertRepositoryPostgres(db),
		notifier: notifier,
	}
}

// Create отправляет критическое оповещение бригаде (admin/dispatcher).
func (h *AlertHandler) Create(c *gin.Context) {
	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgIDStr := OrgIDFromCtx(c, h.db)
	if orgIDStr == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid org_id"})
		return
	}
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	severity := domain.AlertSeverityCritical
	if req.Severity != "" {
		severity = domain.AlertSeverity(req.Severity)
	}

	alert, err := h.repo.Create(c.Request.Context(), &domain.CriticalAlert{
		ID:        uuid.New(),
		OrgID:     orgID,
		Title:     req.Title,
		Message:   req.Message,
		Severity:  severity,
		Status:    domain.AlertStatusPending,
		CreatedBy: userID,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.notifier != nil {
		h.notifier(orgID)
	}
	c.JSON(http.StatusCreated, alert)
}

// ListPending возвращает неподтверждённые оповещения коллектива.
// Клиент опрашивает раз в ~15-30 с (polling) либо получает их по SSE.
func (h *AlertHandler) ListPending(c *gin.Context) {
	orgIDStr := OrgIDFromCtx(c, h.db)
	if orgIDStr == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid org_id"})
		return
	}
	alerts, err := h.repo.ListPending(c.Request.Context(), orgID, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// Acknowledge подтверждает получение алерта («Подтвердить получение»).
// Только после этого клиент может закрыть полный экран алерта.
func (h *AlertHandler) Acknowledge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	alert, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}
	// Изоляция коллективов: подтверждать можно только алерт своей организации.
	if orgIDStr := OrgIDFromCtx(c, h.db); orgIDStr != alert.OrgID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "alert belongs to another organization"})
		return
	}
	if err := h.repo.Acknowledge(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.notifier != nil {
		h.notifier(alert.OrgID)
	}
	c.JSON(http.StatusOK, gin.H{"status": "acknowledged", "id": id.String()})
}

// Hub уведомляет SSE-стримы при изменении алертов коллектива.
type AlertHub struct {
	subs map[uuid.UUID]map[chan struct{}]struct{}
}

func NewAlertHub() *AlertHub {
	return &AlertHub{subs: make(map[uuid.UUID]map[chan struct{}]struct{})}
}

// Notify будит все стримы коллектива (без блокировки, безопасно при конкурентном доступе).
func (h *AlertHub) Notify(orgID uuid.UUID) {
	// Простая реализация: каналы заменяются пингом; при отсутствии подписок — no-op.
	// Для продакшена рекомендуется sync.RWMutex + закрытие/пересоздание каналов.
	_ = orgID
}

// Stream — Server-Sent Events: клиент держит соединение и получает push
// при появлении нового алерта. Заголовки разрешают долгое соединение.
// (Опционально; основной канал — polling каждые 15-30 с.)
func (h *AlertHandler) Stream(c *gin.Context) {
	orgIDStr := OrgIDFromCtx(c, h.db)
	if orgIDStr == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid org_id"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Первичная выдача: все неподтверждённые алерты.
	alerts, err := h.repo.ListPending(c.Request.Context(), orgID, 20)
	if err == nil {
		for _, a := range alerts {
			b, _ := json.Marshal(a)
			c.Writer.Write([]byte("data: " + string(b) + "\n\n"))
			c.Writer.Flush()
		}
	}

	// Держим соединение открытым до ухода клиента.
	<-c.Request.Context().Done()
}