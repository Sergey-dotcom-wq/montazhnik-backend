package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
	"github.com/yourorg/montazhnik/backend/internal/repository/postgres"
)

// BindToolRequest привязка инструмента к инженеру.
type BindToolRequest struct {
	Name string `json:"name" binding:"required"`
	Mac  string `json:"mac" binding:"required"`
}

// HeartbeatToolRequest обновление состояния инструмента.
type HeartbeatToolRequest struct {
	Battery int `json:"battery"`
}

// ReportLostRequest координаты потери инструмента.
type ReportLostRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ToolHandler обрабатывает инструменты «виртуальной сумки» инженера.
type ToolHandler struct {
	db   *pgxpool.Pool
	repo domain.ToolRepository
}

func NewToolHandler(db *pgxpool.Pool) *ToolHandler {
	return &ToolHandler{
		db:   db,
		repo: postgres.NewToolRepositoryPostgres(db),
	}
}

// List возвращает инструменты текущего пользователя (GET /tools).
func (h *ToolHandler) List(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	tools, err := h.repo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tools": tools})
}

// Bind привязывает инструмент (POST /tools/bind).
func (h *ToolHandler) Bind(c *gin.Context) {
	var req BindToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Mac) != 17 || !containsColon(req.Mac) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mac address"})
		return
	}
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	orgID, err := uuid.Parse(OrgIDFromCtx(c, h.db))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	now := time.Now().UTC()
	tool, err := h.repo.Bind(c.Request.Context(), &domain.Tool{
		ID:        uuid.New(),
		OrgID:     orgID,
		UserID:    userID,
		Name:      req.Name,
		Mac:       req.Mac,
		Status:    domain.ToolBound,
		Battery:   -1,
		LastSeen:  now,
		BoundAt:   now,
		UpdatedAt: now,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tool)
}

// Unbind отвязывает инструмент (POST /tools/:id/unbind).
func (h *ToolHandler) Unbind(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tool id"})
		return
	}
	if err := h.repo.Unbind(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "unbound"})
}

// ReportLost помечает инструмент как потерянный (POST /tools/:id/lost).
// Field Guide: Hard Block + фиксация GPS + алерт бригадиру.
func (h *ToolHandler) ReportLost(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tool id"})
		return
	}
	var req ReportLostRequest
	_ = c.ShouldBindJSON(&req)

	// Берём инструмент, чтобы узнать организацию и имя.
	tool, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tool not found"})
		return
	}

	if err := h.repo.MarkLost(c.Request.Context(), id, req.Lat, req.Lng); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Критический алерт бригадиру/коллективу: «Потеряна связь с инструментом».
	alertRepo := postgres.NewAlertRepositoryPostgres(h.db)
	_, _ = alertRepo.Create(c.Request.Context(), &domain.CriticalAlert{
		ID:        uuid.New(),
		OrgID:     tool.OrgID,
		Title:     "Потерян инструмент: " + tool.Name,
		Message:   "Связь потеряна. GPS: " + formatCoord(req.Lat) + ", " + formatCoord(req.Lng),
		Severity:  domain.AlertSeverityCritical,
		Status:    domain.AlertStatusPending,
		CreatedBy: tool.UserID,
		CreatedAt: time.Now().UTC(),
	})

	c.JSON(http.StatusOK, gin.H{"status": "lost", "id": id.String()})
}

func formatCoord(v float64) string {
	if v == 0 {
		return "—"
	}
	return fmt.Sprintf("%.6f", v)
}

// Heartbeat обновляет last_seen и батарею (POST /tools/:id/heartbeat).
func (h *ToolHandler) Heartbeat(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tool id"})
		return
	}
	var req HeartbeatToolRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.repo.Heartbeat(c.Request.Context(), id, req.Battery); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func containsColon(s string) bool {
	for _, r := range s {
		if r == ':' {
			return true
		}
	}
	return false
}