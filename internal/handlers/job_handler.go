package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/montazhnik/backend/internal/domain"
)

// CreateJobRequest структура для создания наряда
type CreateJobRequest struct {
	Address      string `json:"address" binding:"required"`
	Description  string `json:"description"`
	PlannedStart int64  `json:"planned_start"` // Unix timestamp
	PlannedEnd   int64  `json:"planned_end"`   // Unix timestamp
}

// CreateJobResponse структура ответа
type CreateJobResponse struct {
	ID           string `json:"id"`
	ExternalID   string `json:"external_id"`
	Address      string `json:"address"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	PlannedStart int64  `json:"planned_start"`
	PlannedEnd   int64  `json:"planned_end"`
}

// JobHandler обработчик нарядов
type JobHandler struct {
	jobService domain.JobService
}

// NewJobHandler создает новый обработчик
func NewJobHandler(jobService domain.JobService) *JobHandler {
	return &JobHandler{jobService: jobService}
}

// Create создает новый наряд
// Матрица постоянных состояний (Static State):
// - Адрес объекта (не меняется)
// - Плановое время (базовая норма)
// - Ответственный (назначается при создании)
func (h *JobHandler) Create(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем пользователя из контекста (middleware)
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// Преобразуем в доменную модель
	job := &domain.Job{
		Address:      req.Address,
		Description:  req.Description,
		Status:       "new",
		AssignedTo:   userID,
		PlannedStart: time.Unix(req.PlannedStart, 0),
		PlannedEnd:   time.Unix(req.PlannedEnd, 0),
		CreatedAt:    time.Now().UTC(),
	}

	// Сохраняем в БД (статичное состояние)
	createdJob, err := h.jobService.Create(c.Request.Context(), job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
		return
	}

	// Формируем ответ
	resp := CreateJobResponse{
		ID:           createdJob.ID.String(),
		ExternalID:   createdJob.ExternalID,
		Address:      createdJob.Address,
		Status:       string(createdJob.Status),
		CreatedAt:    createdJob.CreatedAt.Unix(),
		PlannedStart: createdJob.PlannedStart.Unix(),
		PlannedEnd:   createdJob.PlannedEnd.Unix(),
	}

	c.JSON(http.StatusCreated, resp)
}

// Get возвращает данные наряда
func (h *JobHandler) Get(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	job, err := h.jobService.Get(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	resp := CreateJobResponse{
		ID:           job.ID.String(),
		ExternalID:   job.ExternalID,
		Address:      job.Address,
		Status:       string(job.Status),
		CreatedAt:    job.CreatedAt.Unix(),
		PlannedStart: job.PlannedStart.Unix(),
		PlannedEnd:   job.PlannedEnd.Unix(),
	}

	c.JSON(http.StatusOK, resp)
}

// List возвращает список нарядов с фильтрами (пагинация)
func (h *JobHandler) List(c *gin.Context) {
	filter := &domain.JobFilter{}

	// Фильтр по статусу
	if status := c.Query("status"); status != "" {
		s := domain.JobStatus(status)
		filter.Status = &s
	}

	// Пагинация (по умолчанию: страница 1, размер 20)
	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	jobs, total, err := h.jobService.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}

	// Формируем ответ
	items := make([]CreateJobResponse, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, CreateJobResponse{
			ID:           job.ID.String(),
			ExternalID:   job.ExternalID,
			Address:      job.Address,
			Status:       string(job.Status),
			CreatedAt:    job.CreatedAt.Unix(),
			PlannedStart: job.PlannedStart.Unix(),
			PlannedEnd:   job.PlannedEnd.Unix(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// UpdateStatus обновляет статус наряда
// Матрица динамических изменений (Dynamic State):
// - Текущий статус (new -> assigned -> in_progress -> completed)
// - Фактическое время (ActualStart/End)
// - Геолокация фиксации
func (h *JobHandler) UpdateStatus(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	var req struct {
		Status      string `json:"status" binding:"required,oneof=new assigned in_progress completed cancelled"`
		ActualStart int64  `json:"actual_start,omitempty"`
		ActualEnd   int64  `json:"actual_end,omitempty"`
		Latitude    string `json:"latitude,omitempty"`
		Longitude   string `json:"longitude,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем пользователя
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	// Обновляем динамическое состояние
	update := &domain.JobUpdate{
		Status:    domain.JobStatus(req.Status),
		Location:  req.Latitude + "," + req.Longitude,
		UpdatedBy: userID,
		UpdatedAt: time.Now().UTC(),
	}

	if req.ActualStart != 0 {
		t := time.Unix(req.ActualStart, 0)
		update.ActualStart = &t
	}
	if req.ActualEnd != 0 {
		t := time.Unix(req.ActualEnd, 0)
		update.ActualEnd = &t
	}

	updatedJob, err := h.jobService.UpdateStatus(c.Request.Context(), jobID, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "status updated",
		"job_id":  jobID.String(),
		"status":  updatedJob.Status,
	})
}

// WorkOrderResponse — формат наряда, ожидаемый Android-клиентом
type WorkOrderResponse struct {
	ID            string `json:"id"`
	Address       string `json:"address"`
	WorkType      string `json:"workType"`
	Status        string `json:"status"`
	CreatedAtMs   int64  `json:"createdAtMs"`
}

// ListWorkOrders возвращает наряды в формате Android-клиента (GET /api/v1/work-orders)
func (h *JobHandler) ListWorkOrders(c *gin.Context) {
	filter := &domain.JobFilter{}
	if status := c.Query("status"); status != "" {
		s := domain.JobStatus(status)
		filter.Status = &s
	}
	page := 1
	pageSize := 100
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 200 {
		pageSize = ps
	}
	jobs, _, err := h.jobService.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list work orders"})
		return
	}
	orders := make([]WorkOrderResponse, 0, len(jobs))
	for _, job := range jobs {
		wt := job.Description
		if wt == "" {
			wt = "Монтажные работы"
		}
		orders = append(orders, WorkOrderResponse{
			ID:          job.ID.String(),
			Address:     job.Address,
			WorkType:    wt,
			Status:      string(job.Status),
			CreatedAtMs: job.CreatedAt.UnixMilli(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}
