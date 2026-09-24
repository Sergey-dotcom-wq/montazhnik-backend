package crypto

import (
	"fmt"
	"testing"
)

func TestChainHasher_ComputeEventHash(t *testing.T) {
	h := NewChainHasher()

	firstHash := h.ComputeEventHash(
		"evt-1", "job-1", "status_changed",
		`{"status":"new"}`, GenesisHash, "2026-01-01T00:00:00Z", "user-1",
	)

	if firstHash == "" {
		t.Fatal("hash должен быть непустым")
	}
	if len(firstHash) != 64 {
		t.Fatalf("длина SHA-256 хеша должна быть 64 символа, получили %d", len(firstHash))
	}
}

func TestChainHasher_Deterministic(t *testing.T) {
	h := NewChainHasher()

	hash1 := h.ComputeEventHash("e", "j", "t", `{"a":1}`, GenesisHash, "2026-01-01", "u")
	hash2 := h.ComputeEventHash("e", "j", "t", `{"a":1}`, GenesisHash, "2026-01-01", "u")

	if hash1 != hash2 {
		t.Fatal("хеш должен быть детерминированным для одинаковых входных данных")
	}
}

func TestChainHasher_ChangesOnPayload(t *testing.T) {
	h := NewChainHasher()

	hash1 := h.ComputeEventHash("e", "j", "t", `{"a":1}`, GenesisHash, "2026-01-01", "u")
	hash2 := h.ComputeEventHash("e", "j", "t", `{"a":2}`, GenesisHash, "2026-01-01", "u")

	if hash1 == hash2 {
		t.Fatal("изменение payload обязано менять хеш")
	}
}

func TestChainHasher_VerifySingleLink(t *testing.T) {
	h := NewChainHasher()

	evt1Hash := h.ComputeEventHash("e1", "j1", "status_changed", `{"status":"new"}`, GenesisHash, "2026-01-01", "u1")
	evt2PrevHash := evt1Hash // событие 2 ссылается на хеш события 1

	if !h.VerifyChainLink(evt1Hash, evt2PrevHash) {
		t.Fatal("связность цепочки должна быть подтверждена, когда хеши совпадают")
	}
	if h.VerifyChainLink(evt1Hash, "wrong-hash") {
		t.Fatal("неверная связность не должна проходить проверку")
	}
}

func TestChainHasher_FullChainIntegrity(t *testing.T) {
	h := NewChainHasher()

	type event struct {
		id         string
		payload    string
		current    string
	}

	// Строим цепочку из 3 событий
	steps := []event{
		{id: "e1", payload: `{"status":"new"}`},
		{id: "e2", payload: `{"status":"in_progress"}`},
		{id: "e3", payload: `{"status":"completed"}`},
	}

	prevHash := GenesisHash
	for i := range steps {
		steps[i].current = h.ComputeEventHash(
			steps[i].id,
			"job-x",
			"status_changed",
			steps[i].payload,
			prevHash,
			fmt.Sprintf("2026-01-0%d", i+1),
			fmt.Sprintf("u%d", i+1),
		)

		// Проверяем связность с предыдущим звеном
		if i > 0 {
			if !h.VerifyChainLink(steps[i-1].current, prevHash) {
				t.Fatalf("разрыв цепочки между %s и %s: prevHash=%s, steps[i-1].current=%s",
					steps[i-1].id, steps[i].id, prevHash, steps[i-1].current)
			}
		}

		prevHash = steps[i].current
	}

	// Проверяем, что последний хеш не равен genesis
	if prevHash == GenesisHash {
		t.Fatal("последний хеш не должен совпадать с genesis")
	}

	// Проверяем, что все хеши уникальны
	for i := 0; i < len(steps); i++ {
		for j := i + 1; j < len(steps); j++ {
			if steps[i].current == steps[j].current {
				t.Fatalf("хеши событий %s и %s не должны совпадать", steps[i].id, steps[j].id)
			}
		}
	}
}