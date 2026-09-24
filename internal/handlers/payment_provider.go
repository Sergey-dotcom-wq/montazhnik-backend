package handlers

import (
	"context"
	"os"
	"sync"
)

// PaymentRequest — запрос на создание платежа.
type PaymentRequest struct {
	PaymentID string
	UserID    string
	Product   string // subscription_month | subscription_year
	Amount    float64
}

// PaymentCreated — результат создания платежа (QR/ссылка для оплаты).
type PaymentCreated struct {
	PaymentID   string `json:"payment_id"`
	PayURL      string `json:"pay_url,omitempty"`      // платёжная ссылка СБП
	QRPayload   string `json:"qr_payload,omitempty"`   // payload для QR (Банк России формат)
	QRBase64    string `json:"qr_base64,omitempty"`    // готовое QR-изображение (base64 PNG)
	ProviderRef string `json:"provider_ref,omitempty"` // ID заказа в платёжной системе
}

// PaymentStatus — ответ провайдера о статусе платежа.
type PaymentStatus struct {
	PaymentID string
	Status    string // paid | pending | failed
	Amount    float64
}

// PayoutRequest — запрос на выплату партнёру (СБП: на карту по номеру телефона).
type PayoutRequest struct {
	PartnerID   string
	Amount      float64
	PhoneNumber string // номер телефона получателя (карта СБП)
	Comment     string
}

// PayoutResult — результат выплаты.
type PayoutResult struct {
	PayoutID string
	Status   string // paid | processing | failed
	Message  string
}

// PaymentProvider — абстракция платёжного провайдера.
// Две реализации: MockProvider (тест, без реальных денег) и
// SberSbpProvider (СБП через Сбер: QR/ссылка для оплаты,
// вебхуки, СБП-выплаты на карты по номеру телефона).
type PaymentProvider interface {
	// CreatePayment создаёт платёж и возвращает ссылку/QR для оплаты.
	CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentCreated, error)
	// VerifyWebhook проверяет подпись входящего вебхука от провайдера.
	VerifyWebhook(ctx context.Context, raw []byte, signature string) (bool, error)
	// ParseWebhook разбирает вебхук в статус платежа.
	ParseWebhook(ctx context.Context, raw []byte) (*PaymentStatus, error)
	// Payout выполняет выплату партнёру (СБП-перевод на карту по телефону).
	Payout(ctx context.Context, req PayoutRequest) (*PayoutResult, error)
	// Name возвращает имя провайдера (mock|sber).
	Name() string
}

// NewPaymentProvider создаёт провайдера по имени из конфига
// (переменная окружения PAYMENT_PROVIDER: mock по умолчанию, sber — для Сбера).
func NewPaymentProvider() PaymentProvider {
	name := os.Getenv("PAYMENT_PROVIDER")
	if name == "sber" {
		return NewSberSbpProvider(SberConfig{
			MerchantID:     os.Getenv("SBER_MERCHANT_ID"),
			APIUsername:    os.Getenv("SBER_API_USERNAME"),
			APIPassword:    os.Getenv("SBER_API_PASSWORD"),
			WebhookSecret:  os.Getenv("SBER_WEBHOOK_SECRET"),
			PayoutAPIUser:  os.Getenv("SBER_PAYOUT_USER"),
			PayoutAPIPass:  os.Getenv("SBER_PAYOUT_PASSWORD"),
			BaseURL:        os.Getenv("SBER_API_URL"),
			PaymentTimeout: 0,
		})
	}
	return newMockPaymentProvider()
}

// --- Mock провайдер (тест без реальных денег) ---

// mockProvider имитирует СБП: любая оплата сразу paid,
// выплата сразу успешна. Используется до подключения Сбера.
type mockProvider struct {
	mu sync.Mutex
}

func newMockPaymentProvider() *mockProvider { return &mockProvider{} }

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentCreated, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Имитация СБП-ссылки: в проде здесь был бы запрос к Сберу.
	return &PaymentCreated{
		PaymentID:   req.PaymentID,
		PayURL:      "https://securepayments.sberbank.ru/pay/" + req.PaymentID,
		ProviderRef: "MOCK-" + req.PaymentID,
	}, nil
}

func (m *mockProvider) VerifyWebhook(ctx context.Context, raw []byte, signature string) (bool, error) {
	// Для mock-режима любая подпись принимается (в тестах).
	return true, nil
}

func (m *mockProvider) ParseWebhook(ctx context.Context, raw []byte) (*PaymentStatus, error) {
	// В mock-режиме разбираем прежний формат тестового вебхука.
	return mockParseWebhook(raw)
}

func (m *mockProvider) Payout(ctx context.Context, req PayoutRequest) (*PayoutResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Имитация выплаты: в проде — вызов API СБП-выплат Сбера.
	return &PayoutResult{
		PayoutID: "MOCK-PAYOUT-" + req.PartnerID,
		Status:   "paid",
		Message:  "mock payout ok",
	}, nil
}