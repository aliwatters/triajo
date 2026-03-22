package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliwatters/ginla/ginla-api/internal/model"
	"github.com/aliwatters/ginla/ginla-api/internal/repository"
)

// EmailHandler holds dependencies for the email-to-task endpoint.
type EmailHandler struct {
	repo *repository.TaskRepository
}

// NewEmailHandler creates an EmailHandler.
func NewEmailHandler(repo *repository.TaskRepository) *EmailHandler {
	return &EmailHandler{repo: repo}
}

// fromEmailRequest is the JSON body for POST /v1/tasks/from-email.
type fromEmailRequest struct {
	From       string `json:"from"        binding:"required"`
	Subject    string `json:"subject"     binding:"required"`
	Body       string `json:"body"`
	ReceivedAt string `json:"received_at"` // RFC3339
}

// priorityKeywords maps uppercase subject keywords to priorities.
var emailPriorityHints = map[string]model.Priority{
	"[URGENT]":   model.PriorityUrgent,
	"[HIGH]":     model.PriorityHigh,
	"[NORMAL]":   model.PriorityNormal,
	"[LOW]":      model.PriorityLow,
	"URGENT:":    model.PriorityUrgent,
	"HIGH:":      model.PriorityHigh,
	"[CRITICAL]": model.PriorityUrgent,
}

// emailTagHints maps uppercase subject keywords to tags.
var emailTagHints = map[string]model.Tag{
	"[AI]":          model.TagAI,
	"[VA]":          model.TagVA,
	"[FAMILY]":      model.TagFamily,
	"[HOUSEKEEPER]": model.TagHousekeeper,
	"[DELEGATE]":    model.TagDelegate,
	"[ME]":          model.TagMe,
}

// FromEmail handles POST /v1/tasks/from-email.
func (h *EmailHandler) FromEmail(c *gin.Context) {
	var req fromEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subjectUpper := strings.ToUpper(req.Subject)

	// Parse priority from subject
	priority := model.PriorityNormal
	for keyword, p := range emailPriorityHints {
		if strings.Contains(subjectUpper, strings.ToUpper(keyword)) {
			priority = p
			break
		}
	}

	// Parse tag from subject
	var tag *model.Tag
	for keyword, t := range emailTagHints {
		if strings.Contains(subjectUpper, strings.ToUpper(keyword)) {
			t := t
			tag = &t
			break
		}
	}

	// Clean title: strip hint markers from subject
	title := req.Subject
	for keyword := range emailPriorityHints {
		title = strings.ReplaceAll(title, keyword, "")
	}
	for keyword := range emailTagHints {
		title = strings.ReplaceAll(title, keyword, "")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = req.Subject
	}

	now := time.Now().UTC()
	srcEmail := model.SourceEmail

	receivedTime := now
	if req.ReceivedAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ReceivedAt); err == nil {
			receivedTime = t
		}
	}

	meta := map[string]any{
		"email_from":        req.From,
		"email_subject":     req.Subject,
		"email_received_at": receivedTime.Format(time.RFC3339),
	}

	task := &model.Task{
		Title:       title,
		Description: req.Body,
		Status:      model.StatusInbox,
		Priority:    priority,
		Source:      &srcEmail,
		Tag:         tag,
		Meta:        meta,
		Checklist:   []model.ChecklistItem{},
		Attachments: []model.Attachment{},
		Activity: []model.ActivityEntry{
			{Action: "created", By: "email-webhook", At: now, Detail: "from: " + req.From},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.repo.Create(ctx, task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task from email"})
		return
	}

	c.JSON(http.StatusCreated, task)
}
