package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliwatters/ginla/ginla-api/internal/model"
	"github.com/aliwatters/ginla/ginla-api/internal/repository"
)

// SyncHandler holds dependencies for calendar sync and import endpoints.
type SyncHandler struct {
	repo *repository.TaskRepository
}

// NewSyncHandler creates a SyncHandler.
func NewSyncHandler(repo *repository.TaskRepository) *SyncHandler {
	return &SyncHandler{repo: repo}
}

// ---- request types ---------------------------------------------------------

// calendarSyncRequest is the optional JSON body for POST /v1/sync/calendar.
type calendarSyncRequest struct {
	Since *time.Time `json:"since"` // optional; default 24h ago
}

// calendarWebhookRequest is the JSON body for POST /v1/tasks/from-calendar.
type calendarWebhookRequest struct {
	Title       string `json:"title"        binding:"required"`
	Description string `json:"description"`
	Start       string `json:"start"        binding:"required"` // RFC3339
	End         string `json:"end"`
	EventID     string `json:"event_id"`
	CalendarID  string `json:"calendar_id"`
}

// todoistImportRequest is the JSON body for POST /v1/import/todoist.
type todoistImportRequest struct {
	APIToken string `json:"api_token" binding:"required"`
}

// anylistImportRequest is the JSON body for POST /v1/import/anylist.
type anylistImportRequest struct {
	Items []anylistItem `json:"items" binding:"required"`
}

type anylistItem struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Checked  bool   `json:"checked"`
}

// appleRemindersImportRequest is the JSON body for POST /v1/import/apple_reminders.
type appleRemindersImportRequest struct {
	Reminders []appleReminder `json:"reminders" binding:"required"`
}

type appleReminder struct {
	Title     string `json:"title"`
	Notes     string `json:"notes"`
	DueDate   string `json:"dueDate"` // RFC3339
	Completed bool   `json:"completed"`
	List      string `json:"list"`
}

// ---- calendar handlers -----------------------------------------------------

// CalendarSync handles POST /v1/sync/calendar.
// Returns tasks with due dates updated since the given time, ready for the
// caller to pass to gsuite-mcp calendar tools.
func (h *SyncHandler) CalendarSync(c *gin.Context) {
	var req calendarSyncRequest
	// body is optional — ignore bind errors
	_ = c.ShouldBindJSON(&req)

	since := time.Now().UTC().Add(-24 * time.Hour)
	if req.Since != nil {
		since = *req.Since
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tasks, err := h.repo.ListTasksWithDueSince(ctx, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks for sync"})
		return
	}

	type calendarEvent struct {
		TaskID          string `json:"task_id"`
		Title           string `json:"title"`
		Description     string `json:"description"`
		DueDate         string `json:"due_date"`
		CalendarEventID string `json:"calendar_event_id,omitempty"`
		Action          string `json:"action"`
	}

	events := make([]calendarEvent, 0, len(tasks))
	for _, t := range tasks {
		if t.Due == nil {
			continue
		}
		ev := calendarEvent{
			TaskID:      t.ID.Hex(),
			Title:       t.Title,
			Description: t.Description,
			DueDate:     t.Due.Format(time.RFC3339),
			Action:      "create",
		}
		if t.Meta != nil {
			if eid, ok := t.Meta["calendar_event_id"].(string); ok && eid != "" {
				ev.CalendarEventID = eid
				ev.Action = "update"
			}
		}
		events = append(events, ev)
	}

	c.JSON(http.StatusOK, gin.H{
		"events":       events,
		"count":        len(events),
		"synced_since": since.Format(time.RFC3339),
	})
}

// FromCalendar handles POST /v1/tasks/from-calendar.
func (h *SyncHandler) FromCalendar(c *gin.Context) {
	var req calendarWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time (use RFC3339)"})
		return
	}

	now := time.Now().UTC()
	src := model.SourceCalendar

	meta := map[string]any{
		"calendar_event_start": req.Start,
	}
	if req.End != "" {
		meta["calendar_event_end"] = req.End
	}
	if req.EventID != "" {
		meta["calendar_event_id"] = req.EventID
	}
	if req.CalendarID != "" {
		meta["calendar_id"] = req.CalendarID
	}

	task := &model.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      model.StatusInbox,
		Priority:    model.PriorityNormal,
		Source:      &src,
		Due:         &startTime,
		Meta:        meta,
		Checklist:   []model.ChecklistItem{},
		Attachments: []model.Attachment{},
		Activity: []model.ActivityEntry{
			{Action: "created", By: "calendar-webhook", At: now, Detail: "imported from Google Calendar"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.repo.Create(ctx, task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task from calendar"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// ---- import handlers -------------------------------------------------------

// todoistTask represents a task from Todoist REST API v2.
type todoistTask struct {
	ID          string   `json:"id"`
	Content     string   `json:"content"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"` // 1=normal, 2=high, 3=very high, 4=urgent
	Labels      []string `json:"labels"`
	Due         *struct {
		Date     string `json:"date"`
		Datetime string `json:"datetime"`
	} `json:"due"`
}

func mapTodoistPriority(p int) model.Priority {
	switch p {
	case 4:
		return model.PriorityUrgent
	case 3:
		return model.PriorityHigh
	case 2:
		return model.PriorityNormal
	default:
		return model.PriorityLow
	}
}

// Import handles POST /v1/import/:source.
func (h *SyncHandler) Import(c *gin.Context) {
	source := c.Param("source")
	switch source {
	case "todoist":
		h.importTodoist(c)
	case "anylist":
		h.importAnyList(c)
	case "apple_reminders":
		h.importAppleReminders(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown source: %s", source)})
	}
}

func (h *SyncHandler) importTodoist(c *gin.Context) {
	var req todoistImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch tasks from Todoist API
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "GET", "https://api.todoist.com/rest/v2/tasks", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request"})
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIToken)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach Todoist API"})
		return
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("todoist API error %d", resp.StatusCode)})
		return
	}

	var items []todoistTask
	if err := json.Unmarshal(bodyBytes, &items); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to parse todoist response"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC()
	src := model.Source("todoist")
	imported := 0

	for _, tt := range items {
		task := &model.Task{
			Title:       tt.Content,
			Description: tt.Description,
			Status:      model.StatusInbox,
			Priority:    mapTodoistPriority(tt.Priority),
			Source:      &src,
			Meta:        map[string]any{"todoist_id": tt.ID},
			Checklist:   []model.ChecklistItem{},
			Attachments: []model.Attachment{},
			CreatedAt:   now,
			UpdatedAt:   now,
			Activity: []model.ActivityEntry{
				{Action: "created", By: "import-todoist", At: now, Detail: fmt.Sprintf("todoist_id: %s", tt.ID)},
			},
		}

		if len(tt.Labels) > 0 {
			t := model.Tag(strings.ToUpper(tt.Labels[0]))
			if model.ValidTags[t] {
				task.Tag = &t
			}
		}

		if tt.Due != nil {
			dueStr := tt.Due.Datetime
			if dueStr == "" {
				dueStr = tt.Due.Date + "T00:00:00Z"
			}
			if t, err := time.Parse(time.RFC3339, dueStr); err == nil {
				task.Due = &t
			}
		}

		if err := h.repo.Create(ctx, task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
			return
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{"imported": imported, "source": "todoist"})
}

func (h *SyncHandler) importAnyList(c *gin.Context) {
	var req anylistImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC()
	src := model.Source("anylist")
	tag := model.TagVA
	imported := 0

	for _, item := range req.Items {
		if item.Checked || item.Name == "" {
			continue
		}

		task := &model.Task{
			Title:       item.Name,
			Status:      model.StatusInbox,
			Priority:    model.PriorityNormal,
			Source:      &src,
			Tag:         &tag,
			Meta:        map[string]any{"anylist_category": item.Category},
			Checklist:   []model.ChecklistItem{},
			Attachments: []model.Attachment{},
			CreatedAt:   now,
			UpdatedAt:   now,
			Activity: []model.ActivityEntry{
				{Action: "created", By: "import-anylist", At: now, Detail: fmt.Sprintf("category: %s", item.Category)},
			},
		}

		if err := h.repo.Create(ctx, task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
			return
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{"imported": imported, "source": "anylist"})
}

func (h *SyncHandler) importAppleReminders(c *gin.Context) {
	var req appleRemindersImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC()
	src := model.Source("apple_reminders")
	imported := 0

	for _, r := range req.Reminders {
		if r.Completed || r.Title == "" {
			continue
		}

		var tag *model.Tag
		t := model.Tag(strings.ToUpper(r.List))
		if model.ValidTags[t] {
			tag = &t
		}

		task := &model.Task{
			Title:       r.Title,
			Description: r.Notes,
			Status:      model.StatusInbox,
			Priority:    model.PriorityNormal,
			Source:      &src,
			Tag:         tag,
			Meta:        map[string]any{"reminder_list": r.List},
			Checklist:   []model.ChecklistItem{},
			Attachments: []model.Attachment{},
			CreatedAt:   now,
			UpdatedAt:   now,
			Activity: []model.ActivityEntry{
				{Action: "created", By: "import-apple-reminders", At: now, Detail: fmt.Sprintf("list: %s", r.List)},
			},
		}

		if r.DueDate != "" {
			if t, err := time.Parse(time.RFC3339, r.DueDate); err == nil {
				task.Due = &t
			}
		}

		if err := h.repo.Create(ctx, task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
			return
		}
		imported++
	}

	c.JSON(http.StatusOK, gin.H{"imported": imported, "source": "apple_reminders"})
}
