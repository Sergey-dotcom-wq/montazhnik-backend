package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
	"github.com/yourorg/montazhnik/backend/internal/services"
)

// ShiftHandler управляет рабочими сменами инженеров.
type ShiftHandler struct {
	db     *pgxpool.Pool
	svc    *services.ShiftServiceImpl
}

func NewShiftHandler(db *pgxpool.Pool, svc *services.ShiftServiceImpl) *ShiftHandler {
	return &ShiftHandler{db: db, svc: svc}
}

// Start открывает смену (POST /shift/start).
func (h *ShiftHandler) Start(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	orgIDStr := OrgIDFromCtx(c, h.db)
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	shift, err := h.svc.StartShift(c.Request.Context(), userID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	statusCode := http.StatusCreated
	if shift.Status == domain.ShiftLocked {
		statusCode = http.StatusForbidden
	}
	c.JSON(statusCode, shift)
}

// Pause ставит смену на паузу (POST /shift/pause).
func (h *ShiftHandler) Pause(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	shift, err := h.svc.PauseShift(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shift)
}

// Resume возобновляет смену (POST /shift/resume).
func (h *ShiftHandler) Resume(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	shift, err := h.svc.ResumeShift(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shift)
}

// End завершает смену (POST /shift/end).
func (h *ShiftHandler) End(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	shift, err := h.svc.EndShift(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shift)
}

// Current возвращает текущую смену (GET /shift).
func (h *ShiftHandler) Current(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	shift, err := h.svc.GetCurrent(c.Request.Context(), userID)
	if err != nil {
		// pgx.ErrNoRows: нет смены — не ошибка.
		c.JSON(http.StatusOK, gin.H{"shift": nil, "status": string(domain.ShiftNone)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shift": shift, "status": string(shift.Status)})
}

// RequestUnlock — инженер запрашивает разблокировку (POST /shift/request-unlock).
func (h *ShiftHandler) RequestUnlock(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	shift, err := h.svc.RequestUnlock(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shift)
}

// DispatchUnlock — диспетчер разблокирует смену (POST /shift/dispatch-unlock).
// Требует роль admin/dispatcher.
type DispatchUnlockRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *ShiftHandler) DispatchUnlock(c *gin.Context) {
	var req DispatchUnlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	shift, err := h.svc.DispatchUnlock(c.Request.Context(), targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shift)
}