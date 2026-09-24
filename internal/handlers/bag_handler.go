package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/montazhnik/backend/internal/domain"
	"github.com/yourorg/montazhnik/backend/internal/repository/postgres"
)

// BagHandler управляет «виртуальной сумкой» материалов инженера (Field Guide §3).
// Материалы не списываются окончательно, а попадают в персональную ячейку.
type BagHandler struct {
	db   *pgxpool.Pool
	repo domain.UserMaterialRepository
}

func NewBagHandler(db *pgxpool.Pool) *BagHandler {
	return &BagHandler{
		db:   db,
		repo: postgres.NewUserMaterialRepositoryPostgres(db),
	}
}

// AddToBagRequest добавляет материал в сумку.
type AddToBagRequest struct {
	ItemID       string  `json:"item_id" binding:"required"`
	Quantity     float64 `json:"quantity" binding:"required,gt=0"`
	AssignedToJob string `json:"assigned_to_job"`
}

// ListBag возвращает содержимое сумки (GET /bag).
func (h *BagHandler) ListBag(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	items, err := h.repo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// AddToBag кладёт материал в сумку (POST /bag/add).
// Если позиция уже есть — увеличивает количество.
func (h *BagHandler) AddToBag(c *gin.Context) {
	var req AddToBagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item_id"})
		return
	}

	// Берём название/единицу из warehouse_items (если есть).
	var name, unit string
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT name, unit FROM warehouse_items WHERE id = $1`, itemID).Scan(&name, &unit)

	now := time.Now().UTC()
	mat, err := h.repo.Add(c.Request.Context(), &domain.UserMaterial{
		ID:            uuid.New(),
		UserID:        userID,
		OrgID:         orgID,
		ItemID:        itemID,
		Name:          name,
		Unit:          unit,
		Quantity:      req.Quantity,
		AssignedToJob: req.AssignedToJob,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mat)
}

// TakeFromBag списывает/возвращает материал из сумки (POST /bag/take).
type TakeFromBagRequest struct {
	ItemID   string  `json:"item_id" binding:"required"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
}

func (h *BagHandler) TakeFromBag(c *gin.Context) {
	var req TakeFromBagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item_id"})
		return
	}
	mat, err := h.repo.GetByItem(c.Request.Context(), userID, itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not in bag"})
		return
	}
	if err := h.repo.Take(c.Request.Context(), mat.ID, req.Quantity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "taken", "id": mat.ID.String()})
}