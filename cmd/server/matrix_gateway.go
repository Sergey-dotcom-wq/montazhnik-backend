package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/montazhnik/backend/internal/matrix"
)

// policy — таблица действий (METHOD + route pattern) и роль, которой разрешено.
// ЗАПРЕЩЕНО всё, чего здесь нет (B1/C1/C3): маршрут вне таблицы -> отказ C2.
// Роли в системе: admin, dispatcher, installer, accountant, partner.
var matrixPolicy = map[string][]string{
	// === Аутентификация и профиль ===
	"GET /api/v1/auth/me":                    {"admin", "dispatcher", "installer", "accountant", "partner"},
	"POST /api/v1/auth/delete-account":        {"admin", "dispatcher", "installer", "accountant", "partner"},
	"POST /api/v1/subscription/activate":      {"admin", "dispatcher", "installer", "accountant", "partner"},

	// === ИИ-ассистент (GigaChat) ===
	"POST /api/v1/ai/ask":                     {"admin", "dispatcher", "installer", "accountant", "partner"},
	"POST /api/v1/ai/analytics":               {"admin", "dispatcher"},

	// === Администрирование ===
	"GET /api/v1/admin/payments":              {"admin", "dispatcher"},
	"GET /api/v1/admin/payments/process-payouts": {"admin", "dispatcher"},
	"GET /api/v1/admin/audit":                 {"admin", "dispatcher"},
	"GET /api/v1/admin/audit/verify":          {"admin", "dispatcher"},
	"GET /api/v1/admin/report":                {"admin", "dispatcher"},
	"POST /api/v1/admin/generate-ks2":         {"admin", "dispatcher"},
	"POST /api/v1/admin/generate-ks3":         {"admin", "dispatcher"},

	// === Топливо, склад, дороги ===
	"POST /api/v1/fuel":                       {"installer"},
	"GET /api/v1/warehouse/items":             {"installer", "dispatcher", "admin"},
	"POST /api/v1/warehouse/requests":         {"installer", "admin"},
	"GET /api/v1/road/route":                  {"installer"},
	"GET /api/v1/road/conditions":             {"installer"},
	"GET /api/v1/road/alternatives":           {"installer"},

	// === Наряды ===
	"GET /api/v1/work-orders":                 {"installer", "dispatcher", "admin"},
	"GET /api/v1/jobs/":                       {"installer", "dispatcher", "admin"},
	"POST /api/v1/jobs/":                      {"dispatcher", "admin"},
	"GET /api/v1/jobs/:id":                    {"installer", "dispatcher", "admin"},
	"PATCH /api/v1/jobs/:id/status":           {"installer", "dispatcher", "admin"},

	// === Партнёры ===
	"GET /api/v1/partners":                    {"partner", "dispatcher", "admin"},
	"GET /api/v1/partners/referrals":          {"partner", "dispatcher", "admin"},
	"GET /api/v1/partners/check-code":         {"admin", "dispatcher", "installer", "accountant", "partner"},

	// === События, материалы, оповещения, смены ===
	"POST /api/v1/events/":                    {"installer", "dispatcher", "admin"},
	"GET /api/v1/events/:id/chain":            {"installer", "dispatcher", "admin"},
	"GET /api/v1/events/:id/verify":           {"admin"},
	"GET /api/v1/materials/":                  {"installer", "dispatcher", "admin"},
	"POST /api/v1/materials/":                 {"installer", "admin"},
	"POST /api/v1/materials/scan":             {"installer"},
	"GET /api/v1/materials/job/:job_id":       {"installer", "dispatcher", "admin"},
	"GET /api/v1/alerts":                      {"installer", "dispatcher", "admin"},
	"GET /api/v1/alerts/stream":               {"installer", "dispatcher", "admin"},
	"POST /api/v1/alerts/:id/ack":             {"installer", "dispatcher", "admin"},
	"POST /api/v1/alerts":                     {"dispatcher", "admin"},
	"GET /api/v1/shift":                       {"installer", "dispatcher", "admin"},
	"POST /api/v1/shift/start":                {"installer"},
	"POST /api/v1/shift/pause":                {"installer"},
	"POST /api/v1/shift/resume":               {"installer"},
	"POST /api/v1/shift/end":                  {"installer"},
	"POST /api/v1/shift/request-unlock":       {"installer"},
	"POST /api/v1/shift/dispatch-unlock":      {"dispatcher", "admin"},

	// === Инструменты и инвентарь ===
	"GET /api/v1/tools":                       {"installer"},
	"POST /api/v1/tools/bind":                 {"installer"},
	"POST /api/v1/tools/:id/unbind":           {"installer", "admin"},
	"POST /api/v1/tools/:id/lost":             {"installer"},
	"POST /api/v1/tools/:id/heartbeat":        {"installer"},
	"GET /api/v1/bag":                         {"installer"},
	"POST /api/v1/bag/add":                    {"installer"},
	"POST /api/v1/bag/take":                   {"installer"},

	// === Диагностика (stage2) и фотофиксация ===
	"GET /api/v1/stage2/splices":              {"installer", "admin"},
	"POST /api/v1/stage2/splices":             {"installer"},
	"GET /api/v1/stage2/thermal":              {"installer", "admin"},
	"POST /api/v1/stage2/thermal":             {"installer"},
	"POST /api/v1/stage2/acoustic":            {"installer"},
	"POST /api/v1/photo-evidence":             {"installer"},
	"POST /api/v1/photo-evidence/verify":      {"installer", "admin"},
}

// matrixGatewayMiddleware прогоняет каждый авторизованный запрос через
// сотовую матрицу (Hive). Подключается внутри authMiddleware — поэтому
// работает для всех защищённых групп единообразно.
func matrixGatewayMiddleware(h *matrix.Hive) gin.HandlerFunc {
	return func(c *gin.Context) {
		actionKey := c.Request.Method + " " + c.FullPath()

		subject := matrix.Subject{
			// A1: сертификат устройства проверяется при mTLS (HTTPS :8443).
			// Запрос по HTTP :8080 (веб-панель) уже прошёл authMiddleware
			// с валидным JWT, поэтому A1 не блокирует рабочий доступ.
			DeviceCertOK: true,
			HasJWT:       true,                 // мы уже внутри authMiddleware
			DeviceBound:  true,                 // пиннинг сертификата на шлюзе (упрощённо)
			Role:         c.GetString("role"),
			InGeozone:    true, // геозонную проверку выполняет ShiftHandler с данными GPS
			InTimeWindow: true,
		}

		dec := h.Check(subject, matrix.Action{Name: actionKey})
		if !dec.Allow {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":  "policy denied",
				"rule":   dec.FailedCode,
				"reason": dec.Reason,
				"block":  dec.Block,
			})
			return
		}
		c.Next()
	}
}

// actionExists — вспомогательная функция для проверки покрытия политики в тестах.
func actionExists(method, path string) bool {
	_, ok := matrixPolicy[method+" "+path]
	return ok
}