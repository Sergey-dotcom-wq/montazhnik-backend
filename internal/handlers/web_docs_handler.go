package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WebDocsHandler генерация исполнительных документов (КС-2/КС-3) в вебе.
type WebDocsHandler struct {
	db *pgxpool.Pool
}

func NewWebDocsHandler(db *pgxpool.Pool) *WebDocsHandler {
	return &WebDocsHandler{db: db}
}

// GenerateKs2 POST /admin/generate-ks2 {job_id}
// Берёт наряд из БД и формирует текст акта КС-2.
func (h *WebDocsHandler) GenerateKs2(c *gin.Context) {
	var req struct {
		JobID string `json:"job_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job, err := h.getJob(c, req.JobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	now := time.Now().Format("02.01.2006")
	sb := fmt.Sprintf(
		"АКТ № %s\nо приёмке выполненных работ (КС-2)\n\n"+
			"Дата: %s\nОбъект: %s\nВид работ: %s\nСтатус: %s\n\n"+
			"Сдал: _______________\nПринял: _______________\n\n"+
			"Подпись УКЭП: [сформировано веб-диспетчером]\n",
		job.ID, now, job.Address, job.Description, job.Status)

	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"ks2_text": sb,
		"job_id":   job.ID,
	})
}

// GenerateKs3 POST /admin/generate-ks3 {job_id, total_cost}
func (h *WebDocsHandler) GenerateKs3(c *gin.Context) {
	var req struct {
		JobID     string  `json:"job_id" binding:"required"`
		TotalCost float64 `json:"total_cost"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	job, err := h.getJob(c, req.JobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	if req.TotalCost <= 0 {
		req.TotalCost = 100000
	}

	now := time.Now().Format("02.01.2006")
	sb := fmt.Sprintf(
		"СПРАВКА № %s\nо стоимости выполненных работ (КС-3)\n\n"+
			"Дата: %s\nОбъект: %s\n\n"+
			"1. Монтажные работы: %.2f руб.\n"+
			"2. Материалы:        %.2f руб.\n"+
			"3. Транспорт:        %.2f руб.\n"+
			"ИТОГО:               %.2f руб.\n\n"+
			"Руководитель: _______________\nБухгалтер: _______________\n",
		job.ID, now, job.Address,
		req.TotalCost*0.6, req.TotalCost*0.3, req.TotalCost*0.1, req.TotalCost)

	c.JSON(http.StatusOK, gin.H{
		"ok":       true,
		"ks3_text": sb,
		"job_id":   job.ID,
	})
}

type webJob struct {
	ID          string `json:"id"`
	ExternalID  string `json:"external_id"`
	Address     string `json:"address"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func (h *WebDocsHandler) getJob(c *gin.Context, id string) (*webJob, error) {
	var j webJob
	var createdAt time.Time
	err := h.db.QueryRow(c.Request.Context(),
		`SELECT id::text, COALESCE(external_id,''), COALESCE(address,''), COALESCE(description,''),
		        status::text, created_at
		 FROM jobs WHERE id = $1`, id).Scan(
		&j.ID, &j.ExternalID, &j.Address, &j.Description, &j.Status, &createdAt)
	if err != nil {
		return nil, err
	}
	j.CreatedAt = createdAt.Format("02.01.2006 15:04")
	return &j, nil
}