package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/aliwatters/ginla/ginla-api/internal/model"
	"github.com/aliwatters/ginla/ginla-api/internal/repository"
)

// TaskHandler holds dependencies for the tasks endpoints.
type TaskHandler struct {
	repo *repository.TaskRepository
}

// NewTaskHandler creates a TaskHandler.
func NewTaskHandler(repo *repository.TaskRepository) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// ---- request/response types ------------------------------------------------

// createTaskRequest is the JSON body for POST /v1/tasks.
type createTaskRequest struct {
	Title       string                `json:"title"       binding:"required"`
	Description string                `json:"description"`
	Checklist   []model.ChecklistItem `json:"checklist"`
	Tag         *model.Tag            `json:"tag"`
	HandlerID   *bson.ObjectID        `json:"handler_id"`
	Priority    *model.Priority       `json:"priority"`
	Position    *float64              `json:"position"`
	Due         *time.Time            `json:"due"`
	Source      *model.Source         `json:"source"`
	Meta        map[string]any        `json:"meta"`
	ParentID    *bson.ObjectID        `json:"parent_id"`
	Recurrence  *model.Recurrence     `json:"recurrence"`
	Attachments []model.Attachment    `json:"attachments"`
}

// updateTaskRequest is the JSON body for PATCH /v1/tasks/:id.
type updateTaskRequest struct {
	Title       *string               `json:"title"`
	Description *string               `json:"description"`
	Checklist   []model.ChecklistItem `json:"checklist"`
	Tag         *model.Tag            `json:"tag"`
	HandlerID   *bson.ObjectID        `json:"handler_id"`
	Status      *model.Status         `json:"status"`
	Priority    *model.Priority       `json:"priority"`
	Position    *float64              `json:"position"`
	Due         *time.Time            `json:"due"`
	Source      *model.Source         `json:"source"`
	Meta        map[string]any        `json:"meta"`
	ParentID    *bson.ObjectID        `json:"parent_id"`
	Recurrence  *model.Recurrence     `json:"recurrence"`
	Attachments []model.Attachment    `json:"attachments"`
}

// ---- helpers ---------------------------------------------------------------

func parseObjectID(s string) (bson.ObjectID, bool) {
	oid, err := bson.ObjectIDFromHex(s)
	return oid, err == nil
}

// ---- handlers --------------------------------------------------------------

// Create handles POST /v1/tasks.
func (h *TaskHandler) Create(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// validate enums
	if req.Tag != nil && !model.ValidTags[*req.Tag] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag"})
		return
	}
	if req.Priority != nil && !model.ValidPriorities[*req.Priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority"})
		return
	}

	now := time.Now().UTC()

	priority := model.PriorityNormal
	if req.Priority != nil {
		priority = *req.Priority
	}

	checklist := req.Checklist
	if checklist == nil {
		checklist = []model.ChecklistItem{}
	}
	attachments := req.Attachments
	if attachments == nil {
		attachments = []model.Attachment{}
	}

	task := &model.Task{
		Title:       req.Title,
		Description: req.Description,
		Checklist:   checklist,
		Tag:         req.Tag,
		HandlerID:   req.HandlerID,
		Status:      model.StatusInbox,
		Priority:    priority,
		Position:    req.Position,
		Due:         req.Due,
		Source:      req.Source,
		Meta:        req.Meta,
		ParentID:    req.ParentID,
		Recurrence:  req.Recurrence,
		Attachments: attachments,
		Activity: []model.ActivityEntry{
			{Action: "created", By: "api", At: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.repo.Create(ctx, task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// List handles GET /v1/tasks.
func (h *TaskHandler) List(c *gin.Context) {
	f := repository.TaskFilter{}

	if v := c.Query("status"); v != "" {
		s := model.Status(v)
		if !model.ValidStatuses[s] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		f.Status = &s
	}
	if v := c.Query("tag"); v != "" {
		t := model.Tag(v)
		if !model.ValidTags[t] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag"})
			return
		}
		f.Tag = &t
	}
	if v := c.Query("handler_id"); v != "" {
		oid, ok := parseObjectID(v)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid handler_id"})
			return
		}
		f.HandlerID = &oid
	}
	if v := c.Query("priority"); v != "" {
		p := model.Priority(v)
		if !model.ValidPriorities[p] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority"})
			return
		}
		f.Priority = &p
	}
	if v := c.Query("due_before"); v != "" {
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_before (use YYYY-MM-DD)"})
			return
		}
		f.DueBefore = &t
	}
	if v := c.Query("due_after"); v != "" {
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid due_after (use YYYY-MM-DD)"})
			return
		}
		f.DueAfter = &t
	}
	if v := c.Query("parent_id"); v != "" {
		oid, ok := parseObjectID(v)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		f.ParentID = &oid
	}

	f.Sort = c.DefaultQuery("sort", "created_at")

	if v := c.Query("limit"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		f.Limit = n
	}
	if v := c.Query("offset"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		f.Offset = n
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tasks, err := h.repo.List(ctx, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// Get handles GET /v1/tasks/:id.
func (h *TaskHandler) Get(c *gin.Context) {
	oid, ok := parseObjectID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	task, err := h.repo.GetByID(ctx, oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get task"})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// Update handles PATCH /v1/tasks/:id.
func (h *TaskHandler) Update(c *gin.Context) {
	oid, ok := parseObjectID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// validate enums
	if req.Tag != nil && !model.ValidTags[*req.Tag] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag"})
		return
	}
	if req.Status != nil && !model.ValidStatuses[*req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	if req.Priority != nil && !model.ValidPriorities[*req.Priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority"})
		return
	}

	now := time.Now().UTC()

	// build $set fields from non-nil request fields
	set := bson.D{}
	if req.Title != nil {
		set = append(set, bson.E{Key: "title", Value: *req.Title})
	}
	if req.Description != nil {
		set = append(set, bson.E{Key: "description", Value: *req.Description})
	}
	if req.Checklist != nil {
		set = append(set, bson.E{Key: "checklist", Value: req.Checklist})
	}
	if req.Tag != nil {
		set = append(set, bson.E{Key: "tag", Value: *req.Tag})
	}
	if req.HandlerID != nil {
		set = append(set, bson.E{Key: "handler_id", Value: *req.HandlerID})
	}
	if req.Status != nil {
		set = append(set, bson.E{Key: "status", Value: *req.Status})
		if *req.Status == model.StatusDone {
			set = append(set, bson.E{Key: "done_at", Value: now})
		}
	}
	if req.Priority != nil {
		set = append(set, bson.E{Key: "priority", Value: *req.Priority})
	}
	if req.Position != nil {
		set = append(set, bson.E{Key: "position", Value: *req.Position})
	}
	if req.Due != nil {
		set = append(set, bson.E{Key: "due", Value: *req.Due})
	}
	if req.Source != nil {
		set = append(set, bson.E{Key: "source", Value: *req.Source})
	}
	if req.Meta != nil {
		set = append(set, bson.E{Key: "meta", Value: req.Meta})
	}
	if req.ParentID != nil {
		set = append(set, bson.E{Key: "parent_id", Value: *req.ParentID})
	}
	if req.Recurrence != nil {
		set = append(set, bson.E{Key: "recurrence", Value: req.Recurrence})
	}
	if req.Attachments != nil {
		set = append(set, bson.E{Key: "attachments", Value: req.Attachments})
	}

	if len(set) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	action := "updated"
	if req.Status != nil {
		action = string(*req.Status)
	}
	entry := model.ActivityEntry{Action: action, By: "api", At: now}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	updated, err := h.repo.UpdateFields(ctx, oid, set, entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
		return
	}
	if updated == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// Delete handles DELETE /v1/tasks/:id.
func (h *TaskHandler) Delete(c *gin.Context) {
	oid, ok := parseObjectID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	deleted, err := h.repo.Delete(ctx, oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete task"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
