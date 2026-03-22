package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/aliwatters/ginla/ginla-mcp/model"
	"github.com/aliwatters/ginla/ginla-mcp/repository"
)

// toolDef describes one MCP tool for the tools/list response.
type toolDef struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// toolList returns all available tool definitions.
func toolList() []toolDef {
	return []toolDef{
		{
			Name:        "task_create",
			Description: "Create a new task. Auto-sets status=inbox, created_at, updated_at, and an activity entry.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"title":       {Type: "string", Description: "Task title (required)"},
					"description": {Type: "string", Description: "Detailed description"},
					"tag":         {Type: "string", Description: "Handler tag", Enum: []string{"ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"}},
					"priority":    {Type: "string", Description: "Task priority", Enum: []string{"urgent", "high", "normal", "low"}},
					"due":         {Type: "string", Description: "Due date (RFC3339)"},
					"source":      {Type: "string", Description: "Task source", Enum: []string{"manual", "agent", "email", "calendar", "voice", "screenshot"}},
					"checklist":   {Type: "string", Description: "JSON array of checklist items: [{\"text\":\"...\",\"done\":false}]"},
					"parent_id":   {Type: "string", Description: "Parent task ObjectID (hex string)"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "task_list",
			Description: "List tasks with optional filters.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"status":     {Type: "string", Description: "Filter by status", Enum: []string{"inbox", "pending", "active", "done", "cancelled"}},
					"tag":        {Type: "string", Description: "Filter by tag", Enum: []string{"ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"}},
					"priority":   {Type: "string", Description: "Filter by priority", Enum: []string{"urgent", "high", "normal", "low"}},
					"handler_id": {Type: "string", Description: "Filter by handler ObjectID (hex string)"},
					"due_before": {Type: "string", Description: "Filter tasks due before this date (RFC3339)"},
					"due_after":  {Type: "string", Description: "Filter tasks due after this date (RFC3339)"},
					"parent_id":  {Type: "string", Description: "Filter by parent task ObjectID (hex string)"},
					"limit":      {Type: "number", Description: "Max results (default 20)"},
					"offset":     {Type: "number", Description: "Pagination offset"},
				},
			},
		},
		{
			Name:        "task_get",
			Description: "Get a single task by ID.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Task ObjectID (hex string)"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "task_update",
			Description: "Update a task. Appends an activity entry. Setting status=done sets done_at.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id":         {Type: "string", Description: "Task ObjectID (hex string, required)"},
					"status":     {Type: "string", Description: "New status", Enum: []string{"inbox", "pending", "active", "done", "cancelled"}},
					"tag":        {Type: "string", Description: "New tag", Enum: []string{"ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"}},
					"priority":   {Type: "string", Description: "New priority", Enum: []string{"urgent", "high", "normal", "low"}},
					"handler_id": {Type: "string", Description: "Handler ObjectID (hex string)"},
					"description": {Type: "string", Description: "Updated description"},
					"due":        {Type: "string", Description: "Updated due date (RFC3339)"},
					"checklist":  {Type: "string", Description: "Replacement checklist JSON array"},
					"position":   {Type: "number", Description: "Manual sort position"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "task_triage",
			Description: "Run auto-triage on all inbox tasks. Tests each rule's regex pattern against title+description. First match sets tag (and optionally handler_id, priority) and moves status to pending.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{},
			},
		},
		{
			Name:        "inbox_count",
			Description: "Get a quick count of inbox tasks.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{},
			},
		},
	}
}

// callTool dispatches a tools/call request to the appropriate handler.
func callTool(ctx context.Context, tasks *repository.TaskRepository, rules *repository.RuleRepository, name string, args map[string]any) (string, error) {
	switch name {
	case "task_create":
		return handleTaskCreate(ctx, tasks, args)
	case "task_list":
		return handleTaskList(ctx, tasks, args)
	case "task_get":
		return handleTaskGet(ctx, tasks, args)
	case "task_update":
		return handleTaskUpdate(ctx, tasks, args)
	case "task_triage":
		return handleTaskTriage(ctx, tasks, rules)
	case "inbox_count":
		return handleInboxCount(ctx, tasks)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// --- helpers ---

func toJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getString(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(args map[string]any, key string) (float64, bool) {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return n, true
		case json.Number:
			f, err := n.Float64()
			if err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func parseObjectID(s string) (bson.ObjectID, error) {
	if s == "" {
		return bson.ObjectID{}, fmt.Errorf("empty id")
	}
	return bson.ObjectIDFromHex(s)
}

// --- tool handlers ---

func handleTaskCreate(ctx context.Context, repo *repository.TaskRepository, args map[string]any) (string, error) {
	title := getString(args, "title")
	if title == "" {
		return "", fmt.Errorf("title is required")
	}

	now := time.Now().UTC()
	task := &model.Task{
		Title:       title,
		Description: getString(args, "description"),
		Status:      model.StatusInbox,
		Priority:    model.PriorityNormal,
		Checklist:   []model.ChecklistItem{},
		Attachments: []model.Attachment{},
		Meta:        map[string]any{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Activity: []model.ActivityEntry{
			{Action: "created", By: "agent", At: now},
		},
	}

	if tag := getString(args, "tag"); tag != "" {
		t := model.Tag(tag)
		if !model.ValidTags[t] {
			return "", fmt.Errorf("invalid tag: %s", tag)
		}
		task.Tag = &t
	}

	if pri := getString(args, "priority"); pri != "" {
		p := model.Priority(pri)
		if !model.ValidPriorities[p] {
			return "", fmt.Errorf("invalid priority: %s", pri)
		}
		task.Priority = p
	}

	if src := getString(args, "source"); src != "" {
		s := model.Source(src)
		task.Source = &s
	}

	if due := getString(args, "due"); due != "" {
		t, err := time.Parse(time.RFC3339, due)
		if err != nil {
			return "", fmt.Errorf("invalid due date: %w", err)
		}
		task.Due = &t
	}

	if pid := getString(args, "parent_id"); pid != "" {
		oid, err := parseObjectID(pid)
		if err != nil {
			return "", fmt.Errorf("invalid parent_id: %w", err)
		}
		task.ParentID = &oid
	}

	if cl := getString(args, "checklist"); cl != "" {
		var items []model.ChecklistItem
		if err := json.Unmarshal([]byte(cl), &items); err != nil {
			return "", fmt.Errorf("invalid checklist JSON: %w", err)
		}
		task.Checklist = items
	}

	if err := repo.Create(ctx, task); err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	return toJSON(task)
}

func handleTaskList(ctx context.Context, repo *repository.TaskRepository, args map[string]any) (string, error) {
	f := repository.TaskFilter{}

	if s := getString(args, "status"); s != "" {
		st := model.Status(s)
		f.Status = &st
	}
	if t := getString(args, "tag"); t != "" {
		tg := model.Tag(t)
		f.Tag = &tg
	}
	if p := getString(args, "priority"); p != "" {
		pr := model.Priority(p)
		f.Priority = &pr
	}
	if h := getString(args, "handler_id"); h != "" {
		oid, err := parseObjectID(h)
		if err != nil {
			return "", fmt.Errorf("invalid handler_id: %w", err)
		}
		f.HandlerID = &oid
	}
	if pid := getString(args, "parent_id"); pid != "" {
		oid, err := parseObjectID(pid)
		if err != nil {
			return "", fmt.Errorf("invalid parent_id: %w", err)
		}
		f.ParentID = &oid
	}
	if db := getString(args, "due_before"); db != "" {
		t, err := time.Parse(time.RFC3339, db)
		if err != nil {
			return "", fmt.Errorf("invalid due_before: %w", err)
		}
		f.DueBefore = &t
	}
	if da := getString(args, "due_after"); da != "" {
		t, err := time.Parse(time.RFC3339, da)
		if err != nil {
			return "", fmt.Errorf("invalid due_after: %w", err)
		}
		f.DueAfter = &t
	}

	limit := int64(20)
	if l, ok := getFloat(args, "limit"); ok {
		limit = int64(l)
	}
	f.Limit = limit

	if off, ok := getFloat(args, "offset"); ok {
		f.Offset = int64(off)
	}

	tasks, err := repo.List(ctx, f)
	if err != nil {
		return "", fmt.Errorf("list tasks: %w", err)
	}

	return toJSON(tasks)
}

func handleTaskGet(ctx context.Context, repo *repository.TaskRepository, args map[string]any) (string, error) {
	id := getString(args, "id")
	oid, err := parseObjectID(id)
	if err != nil {
		return "", fmt.Errorf("invalid id: %w", err)
	}

	task, err := repo.GetByID(ctx, oid)
	if err != nil {
		return "", fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return "", fmt.Errorf("task not found: %s", id)
	}

	return toJSON(task)
}

func handleTaskUpdate(ctx context.Context, repo *repository.TaskRepository, args map[string]any) (string, error) {
	id := getString(args, "id")
	oid, err := parseObjectID(id)
	if err != nil {
		return "", fmt.Errorf("invalid id: %w", err)
	}

	now := time.Now().UTC()
	set := bson.D{}
	detail := "updated"

	if s := getString(args, "status"); s != "" {
		st := model.Status(s)
		if !model.ValidStatuses[st] {
			return "", fmt.Errorf("invalid status: %s", s)
		}
		set = append(set, bson.E{Key: "status", Value: st})
		detail = fmt.Sprintf("status → %s", s)
		if st == model.StatusDone {
			set = append(set, bson.E{Key: "done_at", Value: now})
		}
	}
	if t := getString(args, "tag"); t != "" {
		tg := model.Tag(t)
		if !model.ValidTags[tg] {
			return "", fmt.Errorf("invalid tag: %s", t)
		}
		set = append(set, bson.E{Key: "tag", Value: tg})
	}
	if p := getString(args, "priority"); p != "" {
		pr := model.Priority(p)
		if !model.ValidPriorities[pr] {
			return "", fmt.Errorf("invalid priority: %s", p)
		}
		set = append(set, bson.E{Key: "priority", Value: pr})
	}
	if h := getString(args, "handler_id"); h != "" {
		hoid, err := parseObjectID(h)
		if err != nil {
			return "", fmt.Errorf("invalid handler_id: %w", err)
		}
		set = append(set, bson.E{Key: "handler_id", Value: hoid})
	}
	if desc := getString(args, "description"); desc != "" {
		set = append(set, bson.E{Key: "description", Value: desc})
	}
	if due := getString(args, "due"); due != "" {
		t, err := time.Parse(time.RFC3339, due)
		if err != nil {
			return "", fmt.Errorf("invalid due date: %w", err)
		}
		set = append(set, bson.E{Key: "due", Value: t})
	}
	if cl := getString(args, "checklist"); cl != "" {
		var items []model.ChecklistItem
		if err := json.Unmarshal([]byte(cl), &items); err != nil {
			return "", fmt.Errorf("invalid checklist JSON: %w", err)
		}
		set = append(set, bson.E{Key: "checklist", Value: items})
	}
	if pos, ok := getFloat(args, "position"); ok {
		set = append(set, bson.E{Key: "position", Value: pos})
	}

	if len(set) == 0 {
		return "", fmt.Errorf("no fields to update")
	}

	entry := model.ActivityEntry{
		Action: "updated",
		By:     "agent",
		At:     now,
		Detail: detail,
	}

	updated, err := repo.UpdateFields(ctx, oid, set, entry)
	if err != nil {
		return "", fmt.Errorf("update task: %w", err)
	}
	if updated == nil {
		return "", fmt.Errorf("task not found: %s", id)
	}

	return toJSON(updated)
}

func handleTaskTriage(ctx context.Context, tasks *repository.TaskRepository, rules *repository.RuleRepository) (string, error) {
	// Fetch all inbox tasks
	statusInbox := model.StatusInbox
	inboxTasks, err := tasks.List(ctx, repository.TaskFilter{
		Status: &statusInbox,
		Limit:  1000,
	})
	if err != nil {
		return "", fmt.Errorf("list inbox tasks: %w", err)
	}

	// Fetch active rules sorted by order
	activeRules, err := rules.ListActive(ctx)
	if err != nil {
		return "", fmt.Errorf("list rules: %w", err)
	}

	type triageResult struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Tag     string `json:"tag"`
		Rule    string `json:"rule"`
	}

	var triaged []triageResult
	now := time.Now().UTC()

	for _, task := range inboxTasks {
		combined := task.Title + " " + task.Description

		for _, rule := range activeRules {
			re, err := regexp.Compile("(?i)" + rule.Pattern)
			if err != nil {
				// skip invalid regex
				continue
			}
			if !re.MatchString(combined) {
				continue
			}

			// Build update set
			set := bson.D{
				{Key: "tag", Value: rule.Tag},
				{Key: "status", Value: model.StatusPending},
			}
			if rule.HandlerID != nil {
				set = append(set, bson.E{Key: "handler_id", Value: *rule.HandlerID})
			}
			if rule.Priority != nil {
				set = append(set, bson.E{Key: "priority", Value: *rule.Priority})
			}

			entry := model.ActivityEntry{
				Action: "tagged",
				By:     "auto-triage",
				At:     now,
				Detail: fmt.Sprintf("matched rule: %s", rule.Name),
			}

			_, err = tasks.UpdateFields(ctx, task.ID, set, entry)
			if err != nil {
				return "", fmt.Errorf("update task %s: %w", task.ID.Hex(), err)
			}

			triaged = append(triaged, triageResult{
				ID:    task.ID.Hex(),
				Title: task.Title,
				Tag:   string(rule.Tag),
				Rule:  rule.Name,
			})
			break // first matching rule wins
		}
	}

	result := map[string]any{
		"count":   len(triaged),
		"triaged": triaged,
	}
	return toJSON(result)
}

func handleInboxCount(ctx context.Context, repo *repository.TaskRepository) (string, error) {
	count, err := repo.CountInbox(ctx)
	if err != nil {
		return "", fmt.Errorf("count inbox: %w", err)
	}

	result := map[string]int64{"count": count}
	return toJSON(result)
}
