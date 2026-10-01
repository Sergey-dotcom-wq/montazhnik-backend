package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// GigaChatClient — клиент бесплатной LLM GigaChat (Сбер).
//
// Получает токен по OAuth 2.0 (client_credentials) из CLIENT_ID/CLIENT_SECRET
// (см. developers.sber.ru, платформа GigaChat), затем шлёт запросы в
// /api/v1/chat/completions. Ключи хранятся в .env, наружу не отдаются:
// приложение дергает наш endpoint /api/v1/ai/ask, мы проксируем в GigaChat.
type GigaChatClient struct {
	baseURL      string
	clientID     string
	clientSecret string

	mu        sync.Mutex
	token     string
	tokenExp  time.Time
	endpoints map[string]string // GigaChat контуры: api/scope и auth URL могут различаться
}

// ChatMessage — сообщение в чат (роль: system / user / assistant).
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest — тело запроса к GigaChat.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	// необязательные: max_tokens, temperature, stream
}

// chatResponse — ответ GigaChat.
type chatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// NewGigaChatClient создаёт клиент из env.
func NewGigaChatClient() *GigaChatClient {
	return &GigaChatClient{
		baseURL:      envOr("GIGACHAT_BASE_URL", "https://gigachat.devices.sberbank.ru/api/v1"),
		clientID:     os.Getenv("GIGACHAT_CLIENT_ID"),
		clientSecret: os.Getenv("GIGACHAT_CLIENT_SECRET"),
		endpoints: map[string]string{
			"auth": envOr("GIGACHAT_AUTH_URL", "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"),
		},
	}
}

// Ask отправляет сообщения в GigaChat и возвращает текст ответа.
func (c *GigaChatClient) Ask(messages []ChatMessage, model string) (string, error) {
	if model == "" {
		model = envOr("GIGACHAT_MODEL", "GigaChat-ULTRA")
	}
	token, err := c.authToken()
	if err != nil {
		return "", fmt.Errorf("gigachat auth: %w", err)
	}

	body, _ := json.Marshal(chatRequest{Model: model, Messages: messages})
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Токен мог истечь на середине сессии — сбрасываем и ретраим один раз.
		c.invalidateToken()
		return "", fmt.Errorf("gigachat unauthorized (token refreshed on next call)")
	}

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("gigachat status %d: %s", resp.StatusCode, string(b))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("gigachat: no choices in response")
	}
	return out.Choices[0].Message.Content, nil
}

// authToken возвращает действующий токен, получая новый при необходимости.
func (c *GigaChatClient) authToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExp.Add(-time.Minute)) {
		return c.token, nil
	}

	form := fmt.Sprintf("scope=%s&grant_type=client_credentials",
		// scope различается для контуров: GIGACHAT_API_PERS для промышленного.
		envOr("GIGACHAT_SCOPE", "GIGACHAT_API_PERS"))
	req, err := http.NewRequest(http.MethodPost, c.endpoints["auth"], bytes.NewBufferString(form))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", randomUUID())
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("gigachat auth status %d: %s", resp.StatusCode, string(b))
	}

	var auth struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"` // Unix ms
	}
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return "", err
	}
	c.token = auth.AccessToken
	// GigaChat отдаёт expires_at в мс; на всякий случай ограничим 30 минутами.
	expSec := time.Duration(30) * time.Minute
	if auth.ExpiresAt > 0 {
		expSec = time.Until(time.UnixMilli(auth.ExpiresAt))
		if expSec <= 0 {
			expSec = time.Minute
		}
	}
	c.tokenExp = time.Now().Add(expSec)
	return c.token, nil
}

func (c *GigaChatClient) invalidateToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.tokenExp = time.Time{}
}

// RegisterRoutes регистрирует AI-эндпоинты нашего сервиса.
func (c *GigaChatClient) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/ai/ask", c.GinAsk)
}

// GinAsk — POST /api/v1/ai/ask (gin-хендлер).
// Тело: {"prompt":"...", "role":"системная подсказка (опц.)", "model":"(опц.)"}
func (c *GigaChatClient) GinAsk(ctx *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
		Role   string `json:"role"`
		Model  string `json:"model"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad json: " + err.Error()})
		return
	}
	if req.Prompt == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "prompt required"})
		return
	}

	reply, err := c.genericAIRoleModel(
		req.Role, req.Prompt, req.Model,
		"Ты — ассистент сервиса монтажа связи и авто-диагностики.\n"+
			"Отвечай кратко, понятно, по существу. Если вопрос не по теме — честно скажи, что это вне твоей области.",
	)
	if err != nil {
		log.Printf("gigachat error: %v", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "ai unavailable"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"reply": reply})
}

func (c *GigaChatClient) genericAIRoleModel(role, prompt, model, defaultSystem string) (string, error) {
	sys := role
	if sys == "" {
		sys = defaultSystem
	}
	msgs := []ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: prompt}}
	return c.Ask(msgs, model)
}

// helpers (используются и в других файлах — не дублируем, а защищаем).
func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// randomUUID — быстрый UUID для заголовка RqUID (уникален на запрос).
// на практике можно использовать github.com/google/uuid; здесь — без внеш. dep.
func randomUUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}