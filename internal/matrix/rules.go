package matrix

// ПРАВИЛА ЯДРА: неизменяемый свод инвариантов.
// Каждый пункт - одна сота (правило), которое обязано выполняться всегда.
// Нарушение → deny + блокировка + аудит (см. Hive.Check).

// Rule — пункт Правил ядра (сота).
type Rule struct {
	Code  string // A1, B2, C1, E4...
	Title string
	Check func(ctx RuleContext) bool
}

// RuleContext — контекст проверки (устройство, субъект, действие, данные).
type RuleContext struct {
	DeviceCertOK  bool
	HasJWT        bool
	Role          string
	Action        string
	MaterialState string
	InGeozone     bool
	InTimeWindow  bool
	KnownAction   bool
	AllowedRoles  []string
}

// RuleSet — набор сот. Неизменяем после построения.
type RuleSet struct {
	Version string
	rules   []Rule
}

func NewRuleSet(version string) *RuleSet {
	return &RuleSet{Version: version}
}

func (r *RuleSet) Add(rule Rule) *RuleSet {
	r.rules = append(r.rules, rule)
	return r
}

func (r *RuleSet) Rules() []Rule { return r.rules }

// DefaultRuleSet — свод ядра (v1.0).
func DefaultRuleSet() *RuleSet {
	return NewRuleSet("1.0").
		Add(Rule{"A1", "Подключение только по mTLS с сертификатом нашего CA", func(c RuleContext) bool { return c.DeviceCertOK }}).
		Add(Rule{"A2", "Каждое действие требует действующий JWT", func(c RuleContext) bool { return c.HasJWT }}).
		Add(Rule{"B1", "Разрешено только явно разрешённое (allowlist)", func(c RuleContext) bool { return c.KnownAction }}).
		Add(Rule{"B2", "Право проверяется для каждой операции (роль)", func(c RuleContext) bool {
			for _, r := range c.AllowedRoles {
				if r == c.Role {
					return true
				}
			}
			return false
		}}).
		Add(Rule{"C1", "Модель не толкует правила расширительно", func(c RuleContext) bool { return c.KnownAction }}).
		Add(Rule{"C3", "Модель не инициирует действия вне политики", func(c RuleContext) bool { return c.KnownAction }}).
		Add(Rule{"F1", "Работа только в допустимой геозоне и времени", func(c RuleContext) bool { return c.InGeozone && c.InTimeWindow }})
}