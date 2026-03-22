package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/aliwatters/ginla/ginla-mcp/model"
)

// TaskRepository handles MongoDB operations for tasks.
type TaskRepository struct {
	col         *mongo.Collection
	householdID bson.ObjectID
}

// NewTaskRepository creates a TaskRepository. It resolves the single seeded
// household_id from the database so callers do not need to supply it.
func NewTaskRepository(db *mongo.Database) (*TaskRepository, error) {
	households := db.Collection("households")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := households.FindOne(ctx, bson.D{}).Decode(&result); err != nil {
		return nil, fmt.Errorf("resolve household: %w", err)
	}

	return &TaskRepository{
		col:         db.Collection("tasks"),
		householdID: result.ID,
	}, nil
}

// HouseholdID returns the default household ObjectID.
func (r *TaskRepository) HouseholdID() bson.ObjectID {
	return r.householdID
}

// TaskFilter holds optional query parameters for listing tasks.
type TaskFilter struct {
	Status    *model.Status
	Tag       *model.Tag
	HandlerID *bson.ObjectID
	Priority  *model.Priority
	DueBefore *time.Time
	DueAfter  *time.Time
	ParentID  *bson.ObjectID
	Sort      string // "created_at" | "due" | "position"
	Limit     int64
	Offset    int64
}

// buildFilter constructs a bson.D filter scoped to the household.
func (r *TaskRepository) buildFilter(f TaskFilter) bson.D {
	filter := bson.D{{Key: "household_id", Value: r.householdID}}

	if f.Status != nil {
		filter = append(filter, bson.E{Key: "status", Value: *f.Status})
	}
	if f.Tag != nil {
		filter = append(filter, bson.E{Key: "tag", Value: *f.Tag})
	}
	if f.HandlerID != nil {
		filter = append(filter, bson.E{Key: "handler_id", Value: *f.HandlerID})
	}
	if f.Priority != nil {
		filter = append(filter, bson.E{Key: "priority", Value: *f.Priority})
	}
	if f.ParentID != nil {
		filter = append(filter, bson.E{Key: "parent_id", Value: *f.ParentID})
	}

	// due date range
	if f.DueBefore != nil || f.DueAfter != nil {
		due := bson.D{}
		if f.DueAfter != nil {
			due = append(due, bson.E{Key: "$gte", Value: *f.DueAfter})
		}
		if f.DueBefore != nil {
			due = append(due, bson.E{Key: "$lte", Value: *f.DueBefore})
		}
		filter = append(filter, bson.E{Key: "due", Value: due})
	}

	return filter
}

// List returns tasks matching the filter with pagination.
func (r *TaskRepository) List(ctx context.Context, f TaskFilter) ([]model.Task, error) {
	filter := r.buildFilter(f)

	// sort: default created_at desc; due and position asc
	sortField := "created_at"
	sortDir := -1
	switch f.Sort {
	case "due", "position":
		sortField = f.Sort
		sortDir = 1
	}
	sortDoc := bson.D{{Key: sortField, Value: sortDir}}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetSort(sortDoc).
		SetLimit(limit).
		SetSkip(f.Offset)

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []model.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("decode tasks: %w", err)
	}
	if tasks == nil {
		tasks = []model.Task{}
	}
	return tasks, nil
}

// GetByID fetches a single task by its ObjectID within the household.
func (r *TaskRepository) GetByID(ctx context.Context, id bson.ObjectID) (*model.Task, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "household_id", Value: r.householdID},
	}

	var task model.Task
	if err := r.col.FindOne(ctx, filter).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find task: %w", err)
	}
	return &task, nil
}

// Create inserts a new task and returns it with the generated _id.
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	task.ID = bson.NewObjectID()
	task.HouseholdID = r.householdID

	result, err := r.col.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		task.ID = oid
	}
	return nil
}

// UpdateFields applies a partial update to the task, appending an activity entry.
func (r *TaskRepository) UpdateFields(ctx context.Context, id bson.ObjectID, set bson.D, activityEntry model.ActivityEntry) (*model.Task, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "household_id", Value: r.householdID},
	}

	// prepend updated_at to the set fields
	setFields := bson.D{{Key: "updated_at", Value: activityEntry.At}}
	setFields = append(setFields, set...)

	updateDoc := bson.D{
		{Key: "$set", Value: setFields},
		{Key: "$push", Value: bson.D{{Key: "activity", Value: activityEntry}}},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated model.Task
	if err := r.col.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&updated); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("update task: %w", err)
	}
	return &updated, nil
}

// CountInbox returns the number of inbox tasks for the household.
func (r *TaskRepository) CountInbox(ctx context.Context) (int64, error) {
	status := model.StatusInbox
	filter := r.buildFilter(TaskFilter{Status: &status})
	count, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("count inbox: %w", err)
	}
	return count, nil
}
