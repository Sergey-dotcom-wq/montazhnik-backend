package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoadHandler обработчик дорожной обстановки
type RoadHandler struct {
	db *pgxpool.Pool
}

func NewRoadHandler(db *pgxpool.Pool) *RoadHandler {
	return &RoadHandler{db: db}
}

// CurrentRoute возвращает текущий маршрут бригады (GET /api/v1/road/route)
func (h *RoadHandler) CurrentRoute(c *gin.Context) {
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	var id, name, description, note string
	var extraMinutes int

	err := h.db.QueryRow(c.Request.Context(),
		`SELECT id, name, description, extra_minutes, note FROM road_routes
		 WHERE org_id = $1
		 ORDER BY created_at LIMIT 1`, orgID).Scan(&id, &name, &description, &extraMinutes, &note)
	if err != nil {
		// Маршрут не назначен — возвращаем пустой ответ
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           id,
		"name":         name,
		"description":  description,
		"extraMinutes": extraMinutes,
		"note":         note,
	})
}

// Conditions возвращает дорожные события (GET /api/v1/road/conditions?routeId=...)
func (h *RoadHandler) Conditions(c *gin.Context) {
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	routeID := c.Query("routeId")
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT rc.id, rc.route_id, rc.type, rc.severity, rc.description, rc.source, rc.latitude, rc.longitude, rc.reported_at_ms
		 FROM road_conditions rc
		 JOIN road_routes rr ON rr.id = rc.route_id
		 WHERE rc.route_id = $1 AND rr.org_id = $2`, routeID, orgID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"conditions": []interface{}{}})
		return
	}
	defer rows.Close()

	var conditions []map[string]interface{}
	for rows.Next() {
		var id, routeId, typ, severity, description, source string
		var lat, lon *float64
		var reportedAtMs int64
		if err := rows.Scan(&id, &routeId, &typ, &severity, &description, &source, &lat, &lon, &reportedAtMs); err != nil {
			continue
		}
		cond := map[string]interface{}{
			"id":           id,
			"routeId":      routeId,
			"type":         typ,
			"severity":     severity,
			"description":  description,
			"source":       source,
			"latitude":     lat,
			"longitude":    lon,
			"reportedAtMs": reportedAtMs,
		}
		conditions = append(conditions, cond)
	}
	if conditions == nil {
		conditions = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"conditions": conditions})
}

// Alternatives возвращает альтернативные маршруты (GET /api/v1/road/alternatives?routeId=...)
func (h *RoadHandler) Alternatives(c *gin.Context) {
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}
	routeID := c.Query("routeId")
	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id, name, description, extra_minutes, note FROM road_routes
		 WHERE id <> $1 AND org_id = $2 ORDER BY extra_minutes`, routeID, orgID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"routes": []interface{}{}})
		return
	}
	defer rows.Close()

	var routes []map[string]interface{}
	for rows.Next() {
		var id, name, description, note string
		var extraMinutes int
		if err := rows.Scan(&id, &name, &description, &extraMinutes, &note); err != nil {
			continue
		}
		routes = append(routes, map[string]interface{}{
			"id":           id,
			"name":         name,
			"description":  description,
			"extraMinutes": extraMinutes,
			"note":         note,
		})
	}
	if routes == nil {
		routes = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"routes": routes})
}