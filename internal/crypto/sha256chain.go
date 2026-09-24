package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// GenesisHash - стартовый хеш для первой записи в цепочке
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// ChainHasher вычисляет SHA-256 хеши для цепочки событий
type ChainHasher struct {
	// Дополнительные параметры (если нужны)
}

// NewChainHasher создает новый ChainHasher
func NewChainHasher() *ChainHasher {
	return &ChainHasher{}
}

// ComputeHash вычисляет SHA-256 хеш для события
// data: строка для хеширования (JSON payload + предыдущий хеш)
func (h *ChainHasher) ComputeHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ComputeEventHash вычисляет хеш для события
func (h *ChainHasher) ComputeEventHash(
	eventID string,
	jobID string,
	eventType string,
	payload string,
	previousHash string,
	createdAt string,
	userID string,
) string {
	// Каноническая сериализация: конкатенация в строгом порядке
	data := eventID + "|" +
		jobID + "|" +
		eventType + "|" +
		payload + "|" +
		previousHash + "|" +
		createdAt + "|" +
		userID
	return h.ComputeHash(data)
}

// VerifyChainLink проверяет связность двух соседних событий
func (h *ChainHasher) VerifyChainLink(prevCurrentHash, nextPreviousHash string) bool {
	return prevCurrentHash == nextPreviousHash
}