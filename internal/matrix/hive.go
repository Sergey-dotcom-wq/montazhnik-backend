package matrix

// СОТОВАЯ МАТРИЦА: решётка проверок прав доступа.
//
// Любое действие проходит через соты (пункты Правил ядра). Хоть одна сота
// даёт "нет" → действие запрещено, устройство блокируется, пишется аудит.
// Модель не может обойти матрицу: она не выбирает, какие соты применять.

// Action — действие, запрошенное клиентом.
type Action struct {
	Name     string // warehouse.issue, inventory.check, obd.diagnose...
	Resource string
}

// Subject — субъект действия (устройство + пользователь).
type Subject struct {
	DeviceCertOK bool
	HasJWT       bool
	DeviceBound  bool
	Role         string
	InGeozone    bool
	InTimeWindow bool
}

// Decision — результат прохождения матрицы.
type Decision struct {
	Allow      bool
	Block      bool   // блокировать устройство (стоп-кран)
	FailedCode string // код соты, которая отклонила
	Reason     string
}

// policy — таблица разрешённых действий (allowlist) и ролей.
// Всё, чего здесь нет, запрещено (B1, C1, C3).
var policy = map[string][]string{
	"warehouse.issue":  {"engineer", "foreman"},
	"warehouse.intake": {"engineer", "foreman", "storekeeper"},
	"inventory.check":  {"engineer", "foreman"},
	"obd.diagnose":     {"engineer", "foreman", "driver"},
	"tool.report_lost": {"engineer", "foreman"},
	"shift.start":      {"engineer", "foreman"},
}

// Hive — сотовая матрица.
type Hive struct {
	cons   *RuleSet
	policy map[string][]string
}

func NewHive(c *RuleSet) *Hive { return &Hive{cons: c, policy: policy} }

// NewHiveWithPolicy создаёт Hive с таблицей действий (полключ+роли),
// применимой к шлюзу API (маршруты вида "GET /api/v1/jobs/"). Если переданная
// политика nil, используется встроенная таблица (для внутренних действий).
func NewHiveWithPolicy(c *RuleSet, p map[string][]string) *Hive {
	if p == nil {
		return NewHive(c)
	}
	cp := make(map[string][]string, len(p))
	for k, v := range p {
		cp[k] = v
	}
	return &Hive{cons: c, policy: cp}
}

// Check прогоняет действие через все соты Правил ядра.
func (h *Hive) Check(s Subject, a Action) Decision {
	// Устройство и токен проверяются всегда, независимо от действия.
	if !s.DeviceCertOK {
		return Decision{Allow: false, Block: true, FailedCode: "A1",
			Reason: "нет действительного клиентского сертификата"}
	}
	if !s.HasJWT {
		return Decision{Allow: false, Block: true, FailedCode: "A2",
			Reason: "нет действующего токена"}
	}
	if !s.DeviceBound {
		return Decision{Allow: false, Block: true, FailedCode: "A3",
			Reason: "сертификат не привязан к устройству"}
	}

	// B1/C1/C3: действие должно быть в политике. Нет — не догадываемся (C2).
	allowed, known := h.policy[a.Name]
	if !known {
		return Decision{Allow: false, Block: true, FailedCode: "C2",
			Reason: "действие вне политики — требуется решение человека"}
	}

	// B2: роль должна иметь право на это действие.
	roleOK := false
	for _, r := range allowed {
		if r == s.Role {
			roleOK = true
			break
		}
	}
	if !roleOK {
		return Decision{Allow: false, Block: true, FailedCode: "B2",
			Reason: "у роли нет права на действие"}
	}

	// F1: геозона и время.
	if !s.InGeozone || !s.InTimeWindow {
		return Decision{Allow: false, Block: true, FailedCode: "F1",
			Reason: "действие вне допустимой геозоны или времени"}
	}

	return Decision{Allow: true, Block: false, Reason: "ok"}
}
