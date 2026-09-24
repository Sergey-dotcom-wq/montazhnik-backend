package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

// WarehouseHandler обработчик склада
type WarehouseHandler struct {
	db *pgxpool.Pool
}

func NewWarehouseHandler(db *pgxpool.Pool) *WarehouseHandler {
	return &WarehouseHandler{db: db}
}

type WarehouseItem struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Unit                string  `json:"unit"`
	Quantity            float64 `json:"quantity"`
	MinThreshold        float64 `json:"minThreshold"`
	Category            string  `json:"category"`
	CanBeRequested      bool    `json:"canBeRequested"`
	ExpectedDeliveryDays *int   `json:"expectedDeliveryDays,omitempty"`
	Code                string  `json:"code"`
	Local               bool    `json:"local"`
}

// ListItems возвращает каталог остатков (GET /api/v1/warehouse/items)
// Изоляция коллективов: показываем только позиции своей организации.
func (h *WarehouseHandler) ListItems(c *gin.Context) {
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id, name, unit, quantity, min_threshold, category, can_be_requested, expected_delivery_days, code
		 FROM warehouse_items
		 WHERE org_id = $1
		 ORDER BY category, name`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	var items []WarehouseItem
	for rows.Next() {
		var it WarehouseItem
		err := rows.Scan(&it.ID, &it.Name, &it.Unit, &it.Quantity, &it.MinThreshold, &it.Category, &it.CanBeRequested, &it.ExpectedDeliveryDays, &it.Code)
		if err != nil {
			continue
		}
		items = append(items, it)
	}
	if items == nil {
		items = []WarehouseItem{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// CreateRequest создаёт заявку на позицию (POST /api/v1/warehouse/requests)
func (h *WarehouseHandler) CreateRequest(c *gin.Context) {
	var req struct {
		Item     WarehouseItem `json:"item"`
		Quantity float64       `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	_, err := h.db.Exec(c.Request.Context(),
		`INSERT INTO warehouse_requests (id, item_id, quantity, status) VALUES ($1,$2,$3,'pending')`,
		id, req.Item.ID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok", "id": id.String()})
}