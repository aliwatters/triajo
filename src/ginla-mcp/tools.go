package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/aliwatters/ginla/ginla-mcp/model"
	"github.com/aliwatters/ginla/ginla-mcp/repository"
)

// toolDef describes one MCP tool for the tools/list response.
type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
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
					"meta":        {Type: "string", Description: "JSON object of additional metadata"},
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
					"id":          {Type: "string", Description: "Task ObjectID (hex string, required)"},
					"status":      {Type: "string", Description: "New status", Enum: []string{"inbox", "pending", "active", "done", "cancelled"}},
					"tag":         {Type: "string", Description: "New tag", Enum: []string{"ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"}},
					"priority":    {Type: "string", Description: "New priority", Enum: []string{"urgent", "high", "normal", "low"}},
					"handler_id":  {Type: "string", Description: "Handler ObjectID (hex string)"},
					"description": {Type: "string", Description: "Updated description"},
					"due":         {Type: "string", Description: "Updated due date (RFC3339)"},
					"checklist":   {Type: "string", Description: "Replacement checklist JSON array"},
					"position":    {Type: "number", Description: "Manual sort position"},
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
		// --- Rule CRUD tools ---
		{
			Name:        "rule_create",
			Description: "Create a new triage rule. Rules match inbox tasks by regex pattern and assign a tag, priority, and optional handler.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"name":       {Type: "string", Description: "Human-readable rule name (required)"},
					"pattern":    {Type: "string", Description: "Regex pattern to match against task title+description (required)"},
					"tag":        {Type: "string", Description: "Tag to assign on match (required)", Enum: []string{"ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"}},
					"priority":   {Type: "string", Description: "Priority to assign on match", Enum: []string{"urgent", "high", "normal", "low"}},
					"handler_id": {Type: "string", Description: "Handler ObjectID to assign on match (hex string)"},
					"order":      {Type: "number", Description: "Sort order for rule evaluation (lower = higher priority, default 100)"},
					"active":     {Type: "string", Description: "Whether the rule is active (true/false, default true)"},
				},
				Required: []string{"name", "pattern", "tag"},
			},
		},
		{
			Name:        "rule_list",
			Description: "List all triage rules for the household.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{},
			},
		},
		{
			Name:        "rule_get",
			Description: "Get a single triage rule by ID.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Rule ObjectID (hex string)"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "rule_update",
			Description: "Update a triage rule.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id":         {Type: "string", Description: "Rule ObjectID (hex string, required)"},
					"name":       {Type: "string", Description: "Updated name"},
					"pattern":    {Type: "string", Description: "Updated regex pattern"},
					"tag":        {Type: "string", Description: "Updated tag", Enum: []string{"ME", "AI", "VA", "FAMILY", "HOUSEKEEPER", "DELEGATE"}},
					"priority":   {Type: "string", Description: "Updated priority", Enum: []string{"urgent", "high", "normal", "low"}},
					"handler_id": {Type: "string", Description: "Updated handler ObjectID (hex string)"},
					"order":      {Type: "number", Description: "Updated sort order"},
					"active":     {Type: "string", Description: "Enable/disable rule (true/false)"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "rule_delete",
			Description: "Delete a triage rule by ID.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Rule ObjectID (hex string)"},
				},
				Required: []string{"id"},
			},
		},
		// --- Agent sync tools ---
		{
			Name:        "agent_sync",
			Description: "Sync tasks from swarm-mcp broadcast bus. Reads recent broadcasts on the 'needs' channel and creates ginla tasks from matching messages.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"since_hours": {Type: "number", Description: "How many hours back to scan broadcasts (default 24)"},
				},
			},
		},
		{
			Name:        "agent_broadcast",
			Description: "Broadcast a ginla task completion event to the swarm-mcp bus.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"task_id": {Type: "string", Description: "Task ObjectID (hex string, required)"},
					"channel": {Type: "string", Description: "Swarm channel to broadcast on (default: fleet)", Enum: []string{"fleet", "api", "needs", "decisions", "deploys", "releases"}},
					"message": {Type: "string", Description: "Optional message to include in broadcast"},
				},
				Required: []string{"task_id"},
			},
		},
		// --- Email to task ---
		{
			Name:        "email_to_task",
			Description: "Convert an email into a ginla task. Parses subject for priority/tag hints like [URGENT], [HIGH], [AI], [VA], [FAMILY].",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"from":        {Type: "string", Description: "Sender email address (required)"},
					"subject":     {Type: "string", Description: "Email subject (required)"},
					"body":        {Type: "string", Description: "Email body text"},
					"received_at": {Type: "string", Description: "When email was received (RFC3339, default: now)"},
				},
				Required: []string{"from", "subject"},
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
	case "rule_create":
		return handleRuleCreate(ctx, rules, args)
	case "rule_list":
		return handleRuleList(ctx, rules)
	case "rule_get":
		return handleRuleGet(ctx, rules, args)
	case "rule_update":
		return handleRuleUpdate(ctx, rules, args)
	case "rule_delete":
		return handleRuleDelete(ctx, rules, args)
	case "agent_sync":
		return handleAgentSync(ctx, tasks, args)
	case "agent_broadcast":
		return handleAgentBroadcast(ctx, tasks, args)
	case "email_to_task":
		return handleEmailToTask(ctx, tasks, args)
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

// --- task tool handlers ---

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

	if metaStr := getString(args, "meta"); metaStr != "" {
		var meta map[string]any
		if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
			return "", fmt.Errorf("invalid meta JSON: %w", err)
		}
		task.Meta = meta
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
		ID    string `json:"id"`
		Title string `json:"title"`
		Tag   string `json:"tag"`
		Rule  string `json:"rule"`
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

// --- rule tool handlers ---

func handleRuleCreate(ctx context.Context, repo *repository.RuleRepository, args map[string]any) (string, error) {
	name := getString(args, "name")
	if name == "" {
		return "", fmt.Errorf("name is required")
	}
	pattern := getString(args, "pattern")
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	tagStr := getString(args, "tag")
	if tagStr == "" {
		return "", fmt.Errorf("tag is required")
	}
	tag := model.Tag(tagStr)
	if !model.ValidTags[tag] {
		return "", fmt.Errorf("invalid tag: %s", tagStr)
	}

	// Validate regex
	if _, err := regexp.Compile(pattern); err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	order := 100
	if o, ok := getFloat(args, "order"); ok {
		order = int(o)
	}

	active := true
	if a := getString(args, "active"); a == "false" {
		active = false
	}

	rule := &model.Rule{
		Name:    name,
		Pattern: pattern,
		Tag:     tag,
		Order:   order,
		Active:  active,
	}

	if pri := getString(args, "priority"); pri != "" {
		p := model.Priority(pri)
		if !model.ValidPriorities[p] {
			return "", fmt.Errorf("invalid priority: %s", pri)
		}
		rule.Priority = &p
	}

	if hid := getString(args, "handler_id"); hid != "" {
		oid, err := parseObjectID(hid)
		if err != nil {
			return "", fmt.Errorf("invalid handler_id: %w", err)
		}
		rule.HandlerID = &oid
	}

	if err := repo.Create(ctx, rule); err != nil {
		return "", fmt.Errorf("create rule: %w", err)
	}

	return toJSON(rule)
}

func handleRuleList(ctx context.Context, repo *repository.RuleRepository) (string, error) {
	rules, err := repo.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list rules: %w", err)
	}
	return toJSON(rules)
}

func handleRuleGet(ctx context.Context, repo *repository.RuleRepository, args map[string]any) (string, error) {
	id := getString(args, "id")
	oid, err := parseObjectID(id)
	if err != nil {
		return "", fmt.Errorf("invalid id: %w", err)
	}

	rule, err := repo.GetByID(ctx, oid)
	if err != nil {
		return "", fmt.Errorf("get rule: %w", err)
	}
	if rule == nil {
		return "", fmt.Errorf("rule not found: %s", id)
	}

	return toJSON(rule)
}

func handleRuleUpdate(ctx context.Context, repo *repository.RuleRepository, args map[string]any) (string, error) {
	id := getString(args, "id")
	oid, err := parseObjectID(id)
	if err != nil {
		return "", fmt.Errorf("invalid id: %w", err)
	}

	set := bson.D{}

	if name := getString(args, "name"); name != "" {
		set = append(set, bson.E{Key: "name", Value: name})
	}
	if pattern := getString(args, "pattern"); pattern != "" {
		if _, err := regexp.Compile(pattern); err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}
		set = append(set, bson.E{Key: "pattern", Value: pattern})
	}
	if tagStr := getString(args, "tag"); tagStr != "" {
		tag := model.Tag(tagStr)
		if !model.ValidTags[tag] {
			return "", fmt.Errorf("invalid tag: %s", tagStr)
		}
		set = append(set, bson.E{Key: "tag", Value: tag})
	}
	if pri := getString(args, "priority"); pri != "" {
		p := model.Priority(pri)
		if !model.ValidPriorities[p] {
			return "", fmt.Errorf("invalid priority: %s", pri)
		}
		set = append(set, bson.E{Key: "priority", Value: p})
	}
	if hid := getString(args, "handler_id"); hid != "" {
		hoid, err := parseObjectID(hid)
		if err != nil {
			return "", fmt.Errorf("invalid handler_id: %w", err)
		}
		set = append(set, bson.E{Key: "handler_id", Value: hoid})
	}
	if o, ok := getFloat(args, "order"); ok {
		set = append(set, bson.E{Key: "order", Value: int(o)})
	}
	if a := getString(args, "active"); a != "" {
		set = append(set, bson.E{Key: "active", Value: a == "true"})
	}

	if len(set) == 0 {
		return "", fmt.Errorf("no fields to update")
	}

	updated, err := repo.UpdateFields(ctx, oid, set)
	if err != nil {
		return "", fmt.Errorf("update rule: %w", err)
	}
	if updated == nil {
		return "", fmt.Errorf("rule not found: %s", id)
	}

	return toJSON(updated)
}

func handleRuleDelete(ctx context.Context, repo *repository.RuleRepository, args map[string]any) (string, error) {
	id := getString(args, "id")
	oid, err := parseObjectID(id)
	if err != nil {
		return "", fmt.Errorf("invalid id: %w", err)
	}

	deleted, err := repo.Delete(ctx, oid)
	if err != nil {
		return "", fmt.Errorf("delete rule: %w", err)
	}
	if !deleted {
		return "", fmt.Errorf("rule not found: %s", id)
	}

	result := map[string]any{"deleted": true, "id": id}
	return toJSON(result)
}

// --- agent sync tool handlers ---

// swarmMessage represents a message from swarm-mcp's JSONL store.
type swarmMessage struct {
	ID        string         `json:"id"`
	Channel   string         `json:"channel"`
	Message   string         `json:"message"`
	AgentID   string         `json:"agent_id"`
	Timestamp string         `json:"timestamp"`
	Metadata  map[string]any `json:"metadata"`
}

func handleAgentSync(ctx context.Context, repo *repository.TaskRepository, args map[string]any) (string, error) {
	sinceHours := 24.0
	if h, ok := getFloat(args, "since_hours"); ok && h > 0 {
		sinceHours = h
	}
	since := time.Now().UTC().Add(-time.Duration(sinceHours) * time.Hour)

	swarmPath := os.Getenv("HOME")
	if swarmPath == "" {
		swarmPath = "/root"
	}
	messagesFile := swarmPath + "/.local/share/swarm-mcp/messages.jsonl"

	f, err := os.Open(messagesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return toJSON(map[string]any{"created": 0, "messages_file": messagesFile, "note": "swarm messages file not found"})
		}
		return "", fmt.Errorf("open swarm messages: %w", err)
	}
	defer f.Close()

	var created []map[string]any
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var msg swarmMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// Only process 'needs' channel messages
		if msg.Channel != "needs" {
			continue
		}

		// Parse timestamp and filter by since
		if msg.Timestamp != "" {
			ts, err := time.Parse(time.RFC3339, msg.Timestamp)
			if err == nil && ts.Before(since) {
				continue
			}
		}

		// Create task from message
		now := time.Now().UTC()
		srcAgent := model.SourceAgent
		tagAI := model.TagAI
		title := msg.Message
		if len(title) > 100 {
			title = title[:100] + "..."
		}
		if title == "" {
			title = "Agent request from " + msg.AgentID
		}

		meta := map[string]any{
			"swarm_message_id": msg.ID,
			"swarm_agent_id":   msg.AgentID,
			"swarm_channel":    msg.Channel,
			"swarm_timestamp":  msg.Timestamp,
		}
		if msg.Metadata != nil {
			for k, v := range msg.Metadata {
				meta["swarm_"+k] = v
			}
		}

		task := &model.Task{
			Title:       title,
			Description: msg.Message,
			Status:      model.StatusInbox,
			Priority:    model.PriorityNormal,
			Source:      &srcAgent,
			Tag:         &tagAI,
			Meta:        meta,
			Checklist:   []model.ChecklistItem{},
			Attachments: []model.Attachment{},
			CreatedAt:   now,
			UpdatedAt:   now,
			Activity: []model.ActivityEntry{
				{Action: "created", By: "agent-sync", At: now, Detail: "imported from swarm-mcp needs channel"},
			},
		}

		if err := repo.Create(ctx, task); err != nil {
			return "", fmt.Errorf("create task from swarm message: %w", err)
		}

		created = append(created, map[string]any{
			"task_id":    task.ID.Hex(),
			"task_title": task.Title,
			"message_id": msg.ID,
			"agent_id":   msg.AgentID,
		})
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read swarm messages: %w", err)
	}

	if created == nil {
		created = []map[string]any{}
	}

	return toJSON(map[string]any{
		"created": len(created),
		"tasks":   created,
	})
}

func handleAgentBroadcast(ctx context.Context, repo *repository.TaskRepository, args map[string]any) (string, error) {
	taskID := getString(args, "task_id")
	oid, err := parseObjectID(taskID)
	if err != nil {
		return "", fmt.Errorf("invalid task_id: %w", err)
	}

	task, err := repo.GetByID(ctx, oid)
	if err != nil {
		return "", fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return "", fmt.Errorf("task not found: %s", taskID)
	}

	channel := getString(args, "channel")
	if channel == "" {
		channel = "fleet"
	}

	customMsg := getString(args, "message")
	message := customMsg
	if message == "" {
		message = fmt.Sprintf("ginla task %s [%s]: %s", string(task.Status), task.ID.Hex(), task.Title)
	}

	swarmMsg := map[string]any{
		"id":        fmt.Sprintf("ginla-%s-%d", task.ID.Hex(), time.Now().UnixNano()),
		"channel":   channel,
		"message":   message,
		"agent_id":  "ginla-mcp",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"metadata": map[string]any{
			"task_id":     task.ID.Hex(),
			"task_title":  task.Title,
			"task_status": string(task.Status),
			"source":      "ginla",
		},
	}

	line, err := json.Marshal(swarmMsg)
	if err != nil {
		return "", fmt.Errorf("marshal broadcast: %w", err)
	}

	swarmPath := os.Getenv("HOME")
	if swarmPath == "" {
		swarmPath = "/root"
	}
	messagesFile := swarmPath + "/.local/share/swarm-mcp/messages.jsonl"

	// Ensure directory exists
	dir := swarmPath + "/.local/share/swarm-mcp"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create swarm dir: %w", err)
	}

	f, err := os.OpenFile(messagesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("open swarm messages file: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n", line); err != nil {
		return "", fmt.Errorf("write broadcast: %w", err)
	}

	return toJSON(map[string]any{
		"broadcast": true,
		"channel":   channel,
		"message":   message,
		"task_id":   taskID,
	})
}

// --- email to task handler ---

// priorityHints maps subject keywords to priorities.
var priorityHints = map[string]model.Priority{
	"[URGENT]":   model.PriorityUrgent,
	"[HIGH]":     model.PriorityHigh,
	"[NORMAL]":   model.PriorityNormal,
	"[LOW]":      model.PriorityLow,
	"URGENT:":    model.PriorityUrgent,
	"HIGH:":      model.PriorityHigh,
	"[CRITICAL]": model.PriorityUrgent,
}

// tagHints maps subject keywords to tags.
var tagHints = map[string]model.Tag{
	"[AI]":          model.TagAI,
	"[VA]":          model.TagVA,
	"[FAMILY]":      model.TagFamily,
	"[HOUSEKEEPER]": model.TagHousekeeper,
	"[DELEGATE]":    model.TagDelegate,
	"[ME]":          model.TagMe,
}

func handleEmailToTask(ctx context.Context, repo *repository.TaskRepository, args map[string]any) (string, error) {
	from := getString(args, "from")
	if from == "" {
		return "", fmt.Errorf("from is required")
	}
	subject := getString(args, "subject")
	if subject == "" {
		return "", fmt.Errorf("subject is required")
	}
	body := getString(args, "body")
	receivedAt := getString(args, "received_at")

	// Parse priority from subject
	priority := model.PriorityNormal
	for keyword, p := range priorityHints {
		if strings.Contains(strings.ToUpper(subject), strings.ToUpper(keyword)) {
			priority = p
			break
		}
	}

	// Parse tag from subject
	var tag *model.Tag
	subjectUpper := strings.ToUpper(subject)
	for keyword, t := range tagHints {
		if strings.Contains(subjectUpper, strings.ToUpper(keyword)) {
			t := t
			tag = &t
			break
		}
	}

	// Clean up subject for use as title (remove hint markers)
	title := subject
	for keyword := range priorityHints {
		title = strings.ReplaceAll(title, keyword, "")
	}
	for keyword := range tagHints {
		title = strings.ReplaceAll(title, keyword, "")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = subject // fallback to original if all hints
	}

	now := time.Now().UTC()
	srcEmail := model.SourceEmail

	receivedTime := now
	if receivedAt != "" {
		if t, err := time.Parse(time.RFC3339, receivedAt); err == nil {
			receivedTime = t
		}
	}

	meta := map[string]any{
		"email_from":        from,
		"email_subject":     subject,
		"email_received_at": receivedTime.Format(time.RFC3339),
	}

	task := &model.Task{
		Title:       title,
		Description: body,
		Status:      model.StatusInbox,
		Priority:    priority,
		Source:      &srcEmail,
		Tag:         tag,
		Meta:        meta,
		Checklist:   []model.ChecklistItem{},
		Attachments: []model.Attachment{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Activity: []model.ActivityEntry{
			{Action: "created", By: "email-to-task", At: now, Detail: fmt.Sprintf("from: %s", from)},
		},
	}

	if err := repo.Create(ctx, task); err != nil {
		return "", fmt.Errorf("create task from email: %w", err)
	}

	return toJSON(task)
}
