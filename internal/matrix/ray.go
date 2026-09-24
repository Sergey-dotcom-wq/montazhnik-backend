package matrix

import (
	"fmt"
	"sync"
	"time"
)

// ЛУЧЕВАЯ МАТРИЦА: движение материала во времени.
//
// Луч = перемещение материальной единицы: склад → сумка → объект → уложено.
// Каждый луч фиксирует направление (откуда-куда), кто и когда.
// Лучи меняют состояние сот: например, выдача разрешена только для того,
// что ещё на складе; инвентаризация — только для того, что «в сумке».

type MaterialState string

const (
	StateWarehouse MaterialState = "warehouse" // на складе
	StateBag       MaterialState = "bag"       // в сумке инженера
	StateObject    MaterialState = "object"    // доставлено на объект
	StateInstalled MaterialState = "installed" // уложено/смонтировано
)

// Разрешённые переходы (E1). Всё, чего нет в таблице, — запрещено.
var allowedTransitions = map[MaterialState]map[MaterialState]bool{
	StateWarehouse: {StateBag: true},
	StateBag:       {StateObject: true, StateWarehouse: true},
	StateObject:    {StateInstalled: true, StateBag: true},
	StateInstalled: {},
}

// Ray — зафиксированное перемещение материала (E2).
type Ray struct {
	MaterialID string
	From       MaterialState
	To         MaterialState
	By         string // кто выполнил
	At         time.Time
}

type RayMatrix struct {
	mu    sync.Mutex
	state map[string]MaterialState
	log   []Ray
}

func NewRayMatrix() *RayMatrix {
	return &RayMatrix{state: make(map[string]MaterialState)}
}

// State — текущее состояние материала.
func (r *RayMatrix) State(id string) (MaterialState, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.state[id]
	return s, ok
}

// Move фиксирует луч: проверяет допустимость перехода и записывает историю.
func (r *RayMatrix) Move(id string, to MaterialState, by string) (Ray, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	from, known := r.state[id]
	if !known {
		// Новый материал по умолчанию числится на складе.
		from = StateWarehouse
	}
	if from == to {
		return Ray{}, fmt.Errorf("E1: материал уже в состоянии %q", to)
	}
	if !allowedTransitions[from][to] {
		return Ray{}, fmt.Errorf("E1: переход %q → %q запрещён", from, to)
	}

	ray := Ray{MaterialID: id, From: from, To: to, By: by, At: time.Now()}
	r.state[id] = to
	r.log = append(r.log, ray)
	return ray, nil
}

// History — история лучей по материалу.
func (r *RayMatrix) History(id string) []Ray {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Ray
	for _, ray := range r.log {
		if ray.MaterialID == id {
			out = append(out, ray)
		}
	}
	return out
}

// Allowed — можно ли выполнить действие с материалом в текущем состоянии (E4/E5).
func (r *RayMatrix) Allowed(id string, action string) bool {
	st, ok := r.State(id)
	switch action {
	case "inventory.check":
		// инвентаризация — только то, что «в сумке» (E4)
		return ok && st == StateBag
	case "install":
		// укладка — только то, что доставлено на объект
		return ok && st == StateObject
	case "write_off":
		// списание — только то, что выдано (E3)
		return ok && (st == StateBag || st == StateObject || st == StateInstalled)
	}
	return false
}