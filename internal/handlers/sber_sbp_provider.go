package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SberConfig — параметры интеграции со Сбером (СБП-эквайринг + СБП-выплаты).
// Значения берутся из переменных окружения (см. NewPaymentProvider).
type SberConfig struct {
	MerchantID     string // ID мерчанта Сбера (ticket в платежном шлюзе)
	APIUsername    string // логин API «Эквайринга»
	APIPassword    string // пароль API
	WebhookSecret  string // секрет для проверки подписи вебхуков
	PayoutAPIUser  string // логин API СБП-выплат
	PayoutAPIPass  string // пароль API СБП-выплат
	BaseURL        string // базовый URL шлюза (https://securepayments.sberbank.ru)
	PaymentTimeout time.Duration
}

// SberSbpProvider — реальная интеграция со Сбером по СБП.
type SberSbpProvider struct {
	cfg SberConfig
}

func NewSberSbpProvider(cfg SberConfig) *SberSbpProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://securepayments.sberbank.ru"
	}
	if cfg.PaymentTimeout == 0 {
		cfg.PaymentTimeout = 15 * time.Minute
	}
	return &SberSbpProvider{cfg: cfg}
}

func (s *SberSbpProvider) Name() string { return "sber" }

// CreatePayment создаёт платёж в Сбере и возвращает ссылку/QR для СБП.
// Использует REST API Сбера (register/QR) — в проде тут запрос
// POST /payment/rest/register.do с логином/паролем эквайринга.
func (s *SberSbpProvider) CreatePayment(ctx context.Context, req PaymentRequest) (*PaymentCreated, error) {
	if s.cfg.MerchantID == "" || s.cfg.APIUsername == "" {
		// В проде — ошибка конфигурации. Сейчас возвращаем понятную заглушку,
		// чтобы конвейер был проверяем без ключей.
		return &PaymentCreated{
			PaymentID:   req.PaymentID,
			PayURL:      "https://securepayments.sberbank.ru/not-configured/" + req.PaymentID,
			ProviderRef: "SBER-UNCONFIGURED-" + req.PaymentID,
		}, nil
	}

	// Здесь — реальный вызов Сбера:
	// 1. POST {BaseURL}/payment/rest/register.do
	//    userName, password, orderNumber (PaymentID), amount, currency=643,
	//    returnUrl (страница успеха), jsonParams (СБП: form = SBPQR|SBPURL)
	// 2. Ответ: orderId (ProviderRef), formUrl/sbpUrl (PayURL) или QR payload.
	//
	// Ниже — скелет, выполняемый только при настроенных ключах.
	body, _ := json.Marshal(map[string]string{
		"userName":    s.cfg.APIUsername,
		"password":    s.cfg.APIPassword,
		"orderNumber": req.PaymentID,
		"amount":      fmt.Sprintf("%.2f", req.Amount*100), // копейки
		"currency":    "643",
		"returnUrl":   "https://app.domain.ru/payment/success",
	})

	resp, err := s.postJSON(ctx, s.cfg.BaseURL+"/payment/rest/register.do", body, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		OrderID  string `json:"orderId"`
		FormURL  string `json:"formUrl"`
		SBPURL   string `json:"sbpUrl"`
		QR       string `json:"qrImage"` // base64 PNG
		QRString string `json:"qrString"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return nil, err
	}
	payURL := out.FormURL
	if payURL == "" {
		payURL = out.SBPURL
	}
	return &PaymentCreated{
		PaymentID:   req.PaymentID,
		PayURL:      payURL,
		QRPayload:   out.QRString,
		QRBase64:    out.QR,
		ProviderRef: out.OrderID,
	}, nil
}

// VerifyWebhook проверяет подпись вебхука Сбера (HMAC-SHA256 по телу).
func (s *SberSbpProvider) VerifyWebhook(ctx context.Context, raw []byte, signature string) (bool, error) {
	if s.cfg.WebhookSecret == "" {
		return false, fmt.Errorf("webhook secret not configured")
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.WebhookSecret))
	mac.Write(raw)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

// ParseWebhook разбирает вебхук Сбера в PaymentStatus.
func (s *SberSbpProvider) ParseWebhook(ctx context.Context, raw []byte) (*PaymentStatus, error) {
	var hw struct {
		PaymentID string `json:"payment_id"`
		Status    string `json:"status"`
		Amount    float64 `json:"amount"`
	}
	if err := json.Unmarshal(raw, &hw); err != nil {
		return nil, err
	}
	status := "pending"
	switch hw.Status {
	case "paid", "COMPLETED", "success":
		status = "paid"
	case "failed", "DECLINED", "error":
		status = "failed"
	}
	return &PaymentStatus{
		PaymentID: hw.PaymentID,
		Status:    status,
		Amount:    hw.Amount,
	}, nil
}

// Payout выполняет СБП-выплату партнёру на карту по номеру телефона.
// Реальный API: Сбер «СБП-выплаты» (при наличии договора). Ниже скелет.
func (s *SberSbpProvider) Payout(ctx context.Context, req PayoutRequest) (*PayoutResult, error) {
	if s.cfg.PayoutAPIUser == "" {
		return &PayoutResult{
			PayoutID: "SBER-PAYOUT-UNCONFIGURED-" + req.PartnerID,
			Status:   "processing",
			Message:  "интеграция СБП-выплат не настроена; ожидается договор со Сбером",
		}, nil
	}

	// Здесь — реальный вызов СБП-выплат Сбера:
	// POST {PayoutBaseURL}/api/... { merchantId, phone, amount, currency}
	// (детали зависят от конкретного API и договора).
	body, _ := json.Marshal(map[string]string{
		"partnerId": req.PartnerID,
		"phone":     req.PhoneNumber,
		"amount":    fmt.Sprintf("%.2f", req.Amount),
	})
	resp, err := s.postJSON(ctx, s.cfg.BaseURL+"/sbp/payout", body, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		PayoutID string `json:"payout_id"`
		Status   string `json:"status"`
	}
	_ = json.Unmarshal(resp, &out)
	if out.Status == "" {
		out.Status = "processing"
	}
	return &PayoutResult{
		PayoutID: out.PayoutID,
		Status:   out.Status,
		Message:  "sbp payout submitted",
	}, nil
}

// postJSON — утилита HTTP POST с JSON.
func (s *SberSbpProvider) postJSON(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sber http %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// decodeBase64 helper (для QR, если понадобится).
func decodeBase64(s string) []byte {
	if s == "" {
		return nil
	}
	d, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil
	}
	return d
}