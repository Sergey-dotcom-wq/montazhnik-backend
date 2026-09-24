package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

// FuelHandler обработчик учёта топлива (POST /api/v1/fuel)
type FuelHandler struct {
	db *pgxpool.Pool
}

func NewFuelHandler(db *pgxpool.Pool) *FuelHandler {
	return &FuelHandler{db: db}
}

// FuelRecord входные данные записи о топливе (формат Android-клиента)
type FuelRecord struct {
	ID            string   `json:"id"`
	OdometerKm    float64  `json:"odometerKm"`
	Liters        float64  `json:"liters"`
	PricePerLiter float64  `json:"pricePerLiter"`
	TimestampMs   int64    `json:"timestampMs"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
}

// Add принимает запись о топливе в очередь и фиксирует в БД
func (h *FuelHandler) Add(c *gin.Context) {
	var rec FuelRecord
	if err := c.ShouldBindJSON(&rec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	// Изоляция коллективов: запись привязывается к организации пользователя.
	orgID := c.GetString("org_id")
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	_, err := h.db.Exec(c.Request.Context(),
		`INSERT INTO fuel_records (id, odometer_km, liters, price_per_liter, timestamp_ms, latitude, longitude, created_by, org_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, rec.OdometerKm, rec.Liters, rec.PricePerLiter, rec.TimestampMs, rec.Latitude, rec.Longitude, nil, orgID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save fuel record"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok", "id": id.String()})
}