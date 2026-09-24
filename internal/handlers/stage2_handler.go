package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Stage2Handler синхронизация результатов Stage 2 (граф сварок, термо, акустика).
type Stage2Handler struct {
	db *pgxpool.Pool
}

func NewStage2Handler(db *pgxpool.Pool) *Stage2Handler {
	return &Stage2Handler{db: db}
}

// ---- Граф сварок ----

func (h *Stage2Handler) ListSplices(c *gin.Context) {
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	routeID := c.Query("route_id")
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id, route_id, sequence, sleeve_id, loss_db, lat, lng, note, created_at
		 FROM splice_points WHERE org_id = $1 AND ($2 = '' OR route_id = $2)
		 ORDER BY route_id, sequence`, orgID, routeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []gin.H
	for rows.Next() {
		var id, route, sleeve, note string
		var seq int
		var loss float64
		var lat, lng *float64
		var createdAt interface{}
		if err := rows.Scan(&id, &route, &seq, &sleeve, &loss, &lat, &lng, &note, &createdAt); err != nil {
			continue
		}
		items = append(items, gin.H{
			"id": id, "route_id": route, "sequence": seq,
			"sleeve_id": sleeve, "loss_db": loss,
			"lat": lat, "lng": lng, "note": note,
			"created_at": createdAt,
		})
	}
	if items == nil {
		items = []gin.H{}
	}
	c.JSON(http.StatusOK, gin.H{"splices": items})
}

type AddSpliceRequest struct {
	RouteID  string   `json:"route_id" binding:"required"`
	Sequence int      `json:"sequence"`
	SleeveID string   `json:"sleeve_id"`
	LossDb   float64  `json:"loss_db"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	Note     string   `json:"note"`
}

func (h *Stage2Handler) AddSplice(c *gin.Context) {
	var req AddSpliceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgIDStr := OrgIDFromCtx(c, h.db)
	if orgIDStr == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	userID := c.GetString("user_id")

	_, err := h.db.Exec(c.Request.Context(),
		`INSERT INTO splice_points (org_id, user_id, route_id, sequence, sleeve_id, loss_db, lat, lng, note)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		orgIDStr, userID, req.RouteID, req.Sequence, req.SleeveID, req.LossDb, req.Lat, req.Lng, req.Note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "saved"})
}

// ---- Тепловые замеры ----

type AddThermalRequest struct {
	ContactTempC float64 `json:"contact_temp_c" binding:"required"`
	AmbientTempC float64 `json:"ambient_temp_c" binding:"required"`
	DeltaC       float64 `json:"delta_c"`
	Location     string  `json:"location"`
}

func (h *Stage2Handler) AddThermal(c *gin.Context) {
	var req AddThermalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgIDStr := OrgIDFromCtx(c, h.db)
	if orgIDStr == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	userID := c.GetString("user_id")
	delta := req.DeltaC
	if delta == 0 {
		delta = req.ContactTempC - req.AmbientTempC
		if delta < 0 {
			delta = 0
		}
	}

	_, err := h.db.Exec(c.Request.Context(),
		`INSERT INTO thermal_readings (org_id, user_id, contact_temp_c, ambient_temp_c, delta_c, location)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		orgIDStr, userID, req.ContactTempC, req.AmbientTempC, delta, req.Location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "saved"})
}

func (h *Stage2Handler) ListThermal(c *gin.Context) {
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT contact_temp_c, ambient_temp_c, delta_c, location, created_at
		 FROM thermal_readings WHERE org_id = $1 ORDER BY created_at DESC LIMIT 50`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []gin.H
	for rows.Next() {
		var ct, at, dt float64
		var loc string
		var createdAt interface{}
		if err := rows.Scan(&ct, &at, &dt, &loc, &createdAt); err != nil {
			continue
		}
		items = append(items, gin.H{"contact_temp_c": ct, "ambient_temp_c": at, "delta_c": dt, "location": loc, "created_at": createdAt})
	}
	if items == nil {
		items = []gin.H{}
	}
	c.JSON(http.StatusOK, gin.H{"thermal": items})
}

// ---- Акустические замеры ----

type AddAcousticRequest struct {
	LevelDb float64 `json:"level_db" binding:"required"`
}

func (h *Stage2Handler) AddAcoustic(c *gin.Context) {
	var req AddAcousticRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orgIDStr := OrgIDFromCtx(c, h.db)
	if orgIDStr == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	userID := c.GetString("user_id")

	_, err := h.db.Exec(c.Request.Context(),
		`INSERT INTO acoustic_readings (org_id, user_id, level_db) VALUES ($1, $2, $3)`,
		orgIDStr, userID, req.LevelDb)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "saved"})
}