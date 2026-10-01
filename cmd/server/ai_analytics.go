package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalyticsService — ИИ-аналитик по бизнес-данным.
//
// Собирает агрегаты из БД (наряды, смены, топливо, пользователи, алерты),
// формирует короткую сводку и передаёт её в GigaChat вместе с вопросом
// пользователя. Если GigaChat недоступен/не настроен — отвечает сам
// текстовой сводкой из цифр (фолбэк без LLM).
type AnalyticsService struct {
	db   *pgxpool.Pool
	giga *GigaChatClient
}

// analyticsSnapshot — собранные агрегаты для подмешивания в промпт.
type analyticsSnapshot struct {
	TotalJobs      int      `json:"total_jobs"`
	OpenJobs       int      `json:"open_jobs"`
	ActiveUsers    int      `json:"active_users"`
	WorkShifts     int      `json:"work_shifts"`
	FuelRecords    int      `json:"fuel_records"`
	CriticalAlerts int      `json:"critical_alerts"`
	TopStatuses    []string `json:"top_statuses,omitempty"`
}

func NewAnalyticsService(db *pgxpool.Pool, giga *GigaChatClient) *AnalyticsService {
	return &AnalyticsService{db: db, giga: giga}
}

// gather собирает агрегаты. Все запросы безопасны (таблицы есть в миграциях),
// при ошибке конкретного счётчика — ставим 0, не роняя весь ответ.
func (s *AnalyticsService) gather(ctx context.Context) analyticsSnapshot {
	snap := analyticsSnapshot{}
	q := func(sql string) int {
		var n int
		_ = s.db.QueryRow(ctx, sql).Scan(&n)
		return n
	}
	snap.TotalJobs = q(`SELECT count(*) FROM jobs`)
	snap.OpenJobs = q(`SELECT count(*) FROM jobs WHERE status <> 'DONE' AND status <> 'CANCELLED'`)
	snap.ActiveUsers = q(`SELECT count(*) FROM users WHERE is_active = true`)
	snap.WorkShifts = q(`SELECT count(*) FROM work_shifts`)
	snap.FuelRecords = q(`SELECT count(*) FROM fuel_records`)
	snap.CriticalAlerts = q(`SELECT count(*) FROM critical_alerts`)
	// Топ-3 статуса нарядов (если колонка status есть — она есть).
	if rows, err := s.db.Query(ctx,
		`SELECT status, count(*) FROM jobs GROUP BY status ORDER BY count(*) DESC LIMIT 3`); err == nil {
		defer rows.Close()
		for rows.Next() {
			var st string
			var n int
			if rows.Scan(&st, &n) == nil {
				snap.TopStatuses = append(snap.TopStatuses, fmt.Sprintf("%s=%d", st, n))
			}
		}
	}
	return snap
}

// buildPrompt формирует контекст для LLM из агрегатов + вопроса.
func (s *AnalyticsService) buildPrompt(snap analyticsSnapshot, question string) string {
	var b strings.Builder
	b.WriteString("Ты — аналитик данных службы связи. Ниже — реальные агрегаты из БД за всё время:\n")
	fmt.Fprintf(&b, "- всего нарядов: %d\n", snap.TotalJobs)
	fmt.Fprintf(&b, "- открытых нарядов: %d\n", snap.OpenJobs)
	fmt.Fprintf(&b, "- активных пользователей: %d\n", snap.ActiveUsers)
	fmt.Fprintf(&b, "- рабочих смен: %d\n", snap.WorkShifts)
	fmt.Fprintf(&b, "- записей топлива: %d\n", snap.FuelRecords)
	fmt.Fprintf(&b, "- критических алертов: %d\n", snap.CriticalAlerts)
	if len(snap.TopStatuses) > 0 {
		fmt.Fprintf(&b, "- наряды по статусам: %s\n", strings.Join(snap.TopStatuses, ", "))
	}
	b.WriteString("\nОтветь кратко и по делу на вопрос пользователя, ссылаясь на эти цифры.\n")
	b.WriteString("Вопрос: " + question)
	return b.String()
}

// RegisterRoutes регистрирует аналитический endpoint.
func (s *AnalyticsService) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/ai/analytics", s.GinAnalytics)
}

// GinAnalytics — POST /api/v1/ai/analytics. Тело: {"question":"..."}.
// Возвращает {"reply": "...", "digest": {...цифры для виджета...}}.
func (s *AnalyticsService) GinAnalytics(ctx *gin.Context) {
	var req struct {
		Question string `json:"question"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Question == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "question required"})
		return
	}

	snap := s.gather(ctx.Request.Context())
	prompt := s.buildPrompt(snap, req.Question)

	reply := s.fallbackReply(snap)
	// Пробуем GigaChat; при ошибке — остаёмся на фолбэке (сводка без LLM).
	if s.giga != nil {
		if out, err := s.giga.Ask([]ChatMessage{
			{
				Role: "system",
				Content: "Ты — аналитик данных сервиса монтажа связи.\n" +
					"Правила:\n" +
					"- говори кратко, по делу, цифрами из предоставленного контекста;\n" +
					"- если цифры не позволяют ответить — прямо скажи «по этим данным ответить нельзя»;\n" +
					"- не выдумывай показатели, которых нет в контексте;\n" +
					"- для вопроса о тенденции сравнивай то, что есть, и отмечай «нужна более детальная выборка»;\n" +
					"- русский язык, деловой тон, без воды.",
			},
			{Role: "user", Content: prompt},
		}, ""); err == nil && out != "" {
			reply = out
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"reply": reply, "digest": snap})
}

// fallbackReply — самодостаточная текстовая сводка без внешней LLM.
func (s *AnalyticsService) fallbackReply(snap analyticsSnapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Всего нарядов: %d (открыто %d). Активных пользователей: %d. ",
		snap.TotalJobs, snap.OpenJobs, snap.ActiveUsers)
	fmt.Fprintf(&b, "Рабочих смен: %d. Записей топлива: %d. Критических алертов: %d.",
		snap.WorkShifts, snap.FuelRecords, snap.CriticalAlerts)
	if len(snap.TopStatuses) > 0 {
		b.WriteString(" Статусы: " + strings.Join(snap.TopStatuses, ", ") + ".")
	}
	return b.String()
}