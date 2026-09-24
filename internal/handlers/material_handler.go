package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// MaterialHandler обработчик материалов
type MaterialHandler struct {
	materialService domain.MaterialService
}

// NewMaterialHandler создает новый обработчик
func NewMaterialHandler(ms domain.MaterialService) *MaterialHandler {
	return &MaterialHandler{materialService: ms}
}

// CreateMaterialRequest запрос на добавление материала в справочник
type CreateMaterialRequest struct {
	SKU      string  `json:"sku" binding:"required"`
	Name     string  `json:"name" binding:"required"`
	Unit     string  `json:"unit" binding:"required"`
	Price    float64 `json:"price" binding:"required"`
	VAT      float64 `json:"vat"`
	Category string  `json:"category"`
}

// Create добавляет материал в справочник
func (h *MaterialHandler) Create(c *gin.Context) {
	var req CreateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m := &domain.Material{
		SKU:      req.SKU,
		Name:     req.Name,
		Unit:     req.Unit,
		Price:    req.Price,
		VAT:      req.VAT,
		Category: req.Category,
	}

	created, err := h.materialService.Create(c.Request.Context(), m)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create material"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// ScanMaterialRequest запрос на списание/возврат материала по наряду
type ScanMaterialRequest struct {
	JobID  string  `json:"job_id" binding:"required"`
	SKU    string  `json:"sku" binding:"required"`
	Qty    float64 `json:"quantity" binding:"required"`
	Action string  `json:"action" binding:"required,oneof=consumed returned"`
}

// Scan фиксирует расход/возврат материала по наряду
func (h *MaterialHandler) Scan(c *gin.Context) {
	var req ScanMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobID, err := uuid.Parse(req.JobID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job_id"})
		return
	}

	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	jm, err := h.materialService.ScanMaterial(c.Request.Context(), jobID, req.SKU, req.Qty, req.Action, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, jm)
}

// ListMaterials возвращает список материалов справочника
func (h *MaterialHandler) ListMaterials(c *gin.Context) {
	category := c.Query("category")
	items, err := h.materialService.List(c.Request.Context(), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list materials"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// ListByJob возвращает расход материалов по наряду
func (h *MaterialHandler) ListByJob(c *gin.Context) {
	jobIDStr := c.Param("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job_id"})
		return
	}

	items, err := h.materialService.ListByJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list job materials"})
		return
	}

	c.JSON(http.StatusOK, items)
}