package matrix

import "testing"

func engineerOK() Subject {
	return Subject{
		DeviceCertOK: true, HasJWT: true, DeviceBound: true,
		Role: "engineer", InGeozone: true, InTimeWindow: true,
	}
}

func TestHive_AllowValidAction(t *testing.T) {
	h := NewHive(DefaultRuleSet())
	d := h.Check(engineerOK(), Action{Name: "warehouse.issue"})
	if !d.Allow {
		t.Fatalf("ожидалось разрешение, получено: %+v", d)
	}
}

func TestHive_DenyWithoutCert(t *testing.T) {
	h := NewHive(DefaultRuleSet())
	s := engineerOK()
	s.DeviceCertOK = false
	d := h.Check(s, Action{Name: "warehouse.issue"})
	if d.Allow || d.FailedCode != "A1" {
		t.Fatalf("ожидался отказ A1, получено: %+v", d)
	}
	if !d.Block {
		t.Fatalf("ожидалась блокировка устройства")
	}
}

func TestHive_DenyWrongRole(t *testing.T) {
	h := NewHive(DefaultRuleSet())
	s := engineerOK()
	s.Role = "driver"
	// obd.diagnose разрешён driver, а warehouse.issue — нет
	d := h.Check(s, Action{Name: "warehouse.issue"})
	if d.Allow || d.FailedCode != "B2" {
		t.Fatalf("ожидался отказ B2 по роли, получено: %+v", d)
	}
}

func TestHive_UnknownAction_NoGuess(t *testing.T) {
	h := NewHive(DefaultRuleSet())
	d := h.Check(engineerOK(), Action{Name: "warehouse.give_everything_away"})
	if d.Allow || d.FailedCode != "C2" {
		t.Fatalf("неизвестное действие должно блокироваться по C2, получено: %+v", d)
	}
}

func TestHive_DenyOutsideGeozone(t *testing.T) {
	h := NewHive(DefaultRuleSet())
	s := engineerOK()
	s.InGeozone = false
	d := h.Check(s, Action{Name: "warehouse.issue"})
	if d.Allow || d.FailedCode != "F1" {
		t.Fatalf("ожидался отказ F1 (геозона), получено: %+v", d)
	}
}

func TestRay_AllowedPath(t *testing.T) {
	r := NewRayMatrix()
	if _, err := r.Move("cable-1", StateBag, "engineer"); err != nil {
		t.Fatalf("склад→сумка должен быть разрешён: %v", err)
	}
	if _, err := r.Move("cable-1", StateObject, "engineer"); err != nil {
		t.Fatalf("сумка→объект должен быть разрешён: %v", err)
	}
	if _, err := r.Move("cable-1", StateInstalled, "engineer"); err != nil {
		t.Fatalf("объект→уложено должен быть разрешён: %v", err)
	}
	if len(r.History("cable-1")) != 3 {
		t.Fatalf("ожидалось 3 луча в истории")
	}
}

func TestRay_ForbiddenJump(t *testing.T) {
	r := NewRayMatrix()
	// склад → сразу уложено: запрещено (E1)
	if _, err := r.Move("cable-2", StateInstalled, "engineer"); err == nil {
		t.Fatalf("переход склад→уложено должен быть запрещён")
	}
}

func TestRay_InventoryOnlyInBag(t *testing.T) {
	r := NewRayMatrix()
	r.Move("tool-1", StateBag, "engineer")
	if !r.Allowed("tool-1", "inventory.check") {
		t.Fatalf("инвентаризация инструмента в сумке должна быть разрешена (E4)")
	}
	// неизвестный материал — инвентаризация запрещена
	if r.Allowed("tool-999", "inventory.check") {
		t.Fatalf("инвентаризация неизвестного материала должна быть запрещена")
	}
}
