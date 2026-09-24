package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportHandler сводка по коллективу для диспетчера/админа.
type ReportHandler struct {
	db *pgxpool.Pool
}

func NewReportHandler(db *pgxpool.Pool) *ReportHandler {
	return &ReportHandler{db: db}
}

// DispatchReport возвращает оперативную сводку: смены, инструменты,
// неподтверждённые алерты, объём сумок материалов (GET /admin/report).
func (h *ReportHandler) DispatchReport(c *gin.Context) {
	orgID := OrgIDFromCtx(c, h.db)
	if orgID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no organization"})
		return
	}

	// Сколько пользователей в коллективе.
	var usersCount int
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM users WHERE org_id = $1`, orgID).Scan(&usersCount)

	// Активные/на паузе смены.
	var activeShifts, pausedShifts, lockedShifts int
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM work_shifts WHERE org_id = $1 AND status = 'active'`, orgID).Scan(&activeShifts)
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM work_shifts WHERE org_id = $1 AND status = 'paused'`, orgID).Scan(&pausedShifts)
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM work_shifts WHERE org_id = $1 AND status = 'locked'`, orgID).Scan(&lockedShifts)

	// Инструменты: всего привязанных и потерянных.
	var toolsBound, toolsLost int
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM tools WHERE org_id = $1 AND status = 'bound'`, orgID).Scan(&toolsBound)
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM tools WHERE org_id = $1 AND status = 'lost'`, orgID).Scan(&toolsLost)

	// Неподтверждённые критические алерты.
	var pendingAlerts int
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM critical_alerts WHERE org_id = $1 AND status IN ('pending','active')`, orgID).Scan(&pendingAlerts)

	// Объём виртуальных сумок: количество позиций и единиц.
	var bagPositions, bagUnits float64
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*), COALESCE(SUM(quantity),0) FROM user_materials WHERE org_id = $1`, orgID).Scan(&bagPositions, &bagUnits)

	// Stage 2: сварки, тепловые и акустические замеры (за 30 дней).
	var splicesCount, thermalCount, acousticCount int
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM splice_points WHERE org_id = $1 AND created_at > now() - interval '30 days'`, orgID).Scan(&splicesCount)
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM thermal_readings WHERE org_id = $1 AND created_at > now() - interval '30 days'`, orgID).Scan(&thermalCount)
	_ = h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM acoustic_readings WHERE org_id = $1 AND created_at > now() - interval '30 days'`, orgID).Scan(&acousticCount)

	// Последние активные смены (топ-5) для ленты.
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT ws.status, ws.started_at, ws.last_active, u.username
		FROM work_shifts ws
		JOIN users u ON u.id = ws.user_id
		WHERE ws.org_id = $1 AND ws.status IN ('active','paused','locked')
		ORDER BY ws.last_active DESC
		LIMIT 5`, orgID)
	if err == nil {
		defer rows.Close()
	}

	type shiftLine struct {
		Username   string     `json:"username"`
		Status     string     `json:"status"`
		StartedAt  time.Time  `json:"started_at"`
		LastActive time.Time  `json:"last_active"`
	}
	var recent []shiftLine
	if rows != nil {
		for rows.Next() {
			var s shiftLine
			if err := rows.Scan(&s.Status, &s.StartedAt, &s.LastActive, &s.Username); err != nil {
				continue
			}
			recent = append(recent, s)
		}
	}
	if recent == nil {
		recent = []shiftLine{}
	}

	c.JSON(http.StatusOK, gin.H{
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"users_count":   usersCount,
		"shifts": gin.H{
			"active": activeShifts,
			"paused": pausedShifts,
			"locked": lockedShifts,
		},
		"tools": gin.H{
			"bound": toolsBound,
			"lost":  toolsLost,
		},
		"pending_alerts": pendingAlerts,
		"bag": gin.H{
			"positions": bagPositions,
			"units":     bagUnits,
		},
		"stage2": gin.H{
			"splices":  splicesCount,
			"thermal":  thermalCount,
			"acoustic": acousticCount,
		},
		"photo_evidence": photoEvidenceStats(c, h.db),
		"recent_shifts": recent,
	})
}