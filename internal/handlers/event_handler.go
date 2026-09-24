package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// CreateEventRequest структура для фиксации события
type CreateEventRequest struct {
	JobID     string `json:"job_id" binding:"required"`
	EventType string `json:"event_type" binding:"required,oneof=photo_uploaded measurement_taken status_changed material_scanned"`
	Payload   string `json:"payload"` // JSON строка с данными (base64 фото, координаты и т.д.)
}

// EventHandler обработчик событий (Dynamic State)
type EventHandler struct {
	eventService domain.EventService
	jobService   domain.JobService
}

// NewEventHandler создает новый обработчик
func NewEventHandler(es domain.EventService, js domain.JobService) *EventHandler {
	return &EventHandler{eventService: es, jobService: js}
}

// Create фиксирует событие в цепочке SHA-256
// Матрица динамических изменений:
// - Каждое действие (фото, замер, смена статуса) порождает событие
// - События связаны хешем (current_hash зависит от previous_hash)
// - Это обеспечивает неизменяемость истории (End-to-End Traceability)
func (h *EventHandler) Create(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем существование наряда (статичное состояние)
	jobID, err := uuid.Parse(req.JobID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job_id"})
		return
	}

	_, err = h.jobService.Get(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Получаем пользователя
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// Преобразуем payload в map для валидации
	var payloadMap map[string]interface{}
	if err := json.Unmarshal([]byte(req.Payload), &payloadMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload json"})
		return
	}

	// Создаем событие через сервис (с проверкой цепочки хешей)
	event, err := h.eventService.CreateEvent(c.Request.Context(), domain.EventInput{
		JobID:     jobID,
		EventType: domain.EventType(req.EventType),
		Payload:   payloadMap,
	}, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Если это событие смены статуса - обновляем статичное состояние наряда
	if req.EventType == "status_changed" {
		if status, ok := payloadMap["status"].(string); ok {
			_, err = h.jobService.UpdateStatus(c.Request.Context(), jobID, &domain.JobUpdate{
				Status:    domain.JobStatus(status),
				UpdatedBy: userID,
				UpdatedAt: time.Now().UTC(),
			})
			if err != nil {
				// Логируем, но не падаем - событие уже сохранено
				c.Error(err)
			}
		}
	}

	c.JSON(http.StatusCreated, event)
}

// GetChain возвращает полную историю событий для наряда
// Позволяет найти цепочку событий хоть через 100 лет
func (h *EventHandler) GetChain(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job_id"})
		return
	}

	events, err := h.eventService.GetEventChain(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// VerifyChain проверяет целостность цепочки (для аудита)
func (h *EventHandler) VerifyChain(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job_id"})
		return
	}

	isValid, err := h.eventService.VerifyChain(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":    jobID.String(),
		"is_valid":  isValid,
		"message":   "Цепочка хешей валидна",
		"audit_at":  time.Now().UTC().Format(time.RFC3339),
		"audited_by": c.GetString("user_id"),
	})
}
