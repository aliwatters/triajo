package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Tag represents who handles the task.
type Tag string

const (
	TagMe          Tag = "ME"
	TagAI          Tag = "AI"
	TagVA          Tag = "VA"
	TagFamily      Tag = "FAMILY"
	TagHousekeeper Tag = "HOUSEKEEPER"
	TagDelegate    Tag = "DELEGATE"
)

// ValidTags is the set of allowed tag values.
var ValidTags = map[Tag]bool{
	TagMe:          true,
	TagAI:          true,
	TagVA:          true,
	TagFamily:      true,
	TagHousekeeper: true,
	TagDelegate:    true,
}

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusInbox     Status = "inbox"
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusDone      Status = "done"
	StatusCancelled Status = "cancelled"
)

// ValidStatuses is the set of allowed status values.
var ValidStatuses = map[Status]bool{
	StatusInbox:     true,
	StatusPending:   true,
	StatusActive:    true,
	StatusDone:      true,
	StatusCancelled: true,
}

// Priority represents how urgent a task is.
type Priority string

const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// ValidPriorities is the set of allowed priority values.
var ValidPriorities = map[Priority]bool{
	PriorityUrgent: true,
	PriorityHigh:   true,
	PriorityNormal: true,
	PriorityLow:    true,
}

// Source represents how the task was created.
type Source string

const (
	SourceManual     Source = "manual"
	SourceAgent      Source = "agent"
	SourceEmail      Source = "email"
	SourceCalendar   Source = "calendar"
	SourceVoice      Source = "voice"
	SourceScreenshot Source = "screenshot"
)

// ChecklistItem is a single inline checkbox.
type ChecklistItem struct {
	Text string `bson:"text" json:"text"`
	Done bool   `bson:"done" json:"done"`
}

// Recurrence holds the RRULE and next spawn time.
type Recurrence struct {
	RRule  string    `bson:"rrule"   json:"rrule"`
	NextAt time.Time `bson:"next_at" json:"next_at"`
}

// Attachment is a file linked to the task.
type Attachment struct {
	URL  string `bson:"url"  json:"url"`
	Name string `bson:"name" json:"name"`
	Type string `bson:"type" json:"type"`
}

// ActivityEntry is one record in the append-only audit trail.
type ActivityEntry struct {
	Action string    `bson:"action"           json:"action"`
	By     string    `bson:"by"               json:"by"`
	At     time.Time `bson:"at"               json:"at"`
	Detail string    `bson:"detail,omitempty" json:"detail,omitempty"`
}

// Task is the core domain object.
type Task struct {
	ID          bson.ObjectID   `bson:"_id,omitempty"       json:"id"`
	HouseholdID bson.ObjectID   `bson:"household_id"        json:"household_id"`
	Title       string          `bson:"title"               json:"title"`
	Description string          `bson:"description"         json:"description"`
	Checklist   []ChecklistItem `bson:"checklist"           json:"checklist"`
	Tag         *Tag            `bson:"tag"                 json:"tag"`
	HandlerID   *bson.ObjectID  `bson:"handler_id"          json:"handler_id"`
	Status      Status          `bson:"status"              json:"status"`
	Priority    Priority        `bson:"priority"            json:"priority"`
	Position    *float64        `bson:"position"            json:"position"`
	Due         *time.Time      `bson:"due"                 json:"due"`
	Source      *Source         `bson:"source"              json:"source"`
	Meta        map[string]any  `bson:"meta"                json:"meta"`
	ParentID    *bson.ObjectID  `bson:"parent_id"           json:"parent_id"`
	Recurrence  *Recurrence     `bson:"recurrence"          json:"recurrence"`
	Attachments []Attachment    `bson:"attachments"         json:"attachments"`
	Activity    []ActivityEntry `bson:"activity"            json:"activity"`
	CreatedAt   time.Time       `bson:"created_at"          json:"created_at"`
	UpdatedAt   time.Time       `bson:"updated_at"          json:"updated_at"`
	DoneAt      *time.Time      `bson:"done_at"             json:"done_at"`
}
