package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/aliwatters/ginla/ginla-api/internal/model"
)

// RuleRepository handles MongoDB operations for triage rules.
type RuleRepository struct {
	col         *mongo.Collection
	householdID bson.ObjectID
}

// NewRuleRepository creates a RuleRepository for the given household.
func NewRuleRepository(db *mongo.Database, householdID bson.ObjectID) *RuleRepository {
	return &RuleRepository{
		col:         db.Collection("triage_rules"),
		householdID: householdID,
	}
}

// List returns all rules for the household, sorted by order ascending.
func (r *RuleRepository) List(ctx context.Context) ([]model.Rule, error) {
	filter := bson.D{
		{Key: "household_id", Value: r.householdID},
	}
	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find rules: %w", err)
	}
	defer cursor.Close(ctx)

	var rules []model.Rule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, fmt.Errorf("decode rules: %w", err)
	}
	if rules == nil {
		rules = []model.Rule{}
	}
	return rules, nil
}

// GetByID fetches a single rule by its ObjectID within the household.
func (r *RuleRepository) GetByID(ctx context.Context, id bson.ObjectID) (*model.Rule, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "household_id", Value: r.householdID},
	}

	var rule model.Rule
	if err := r.col.FindOne(ctx, filter).Decode(&rule); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find rule: %w", err)
	}
	return &rule, nil
}

// Create inserts a new rule and returns it with the generated _id.
func (r *RuleRepository) Create(ctx context.Context, rule *model.Rule) error {
	rule.ID = bson.NewObjectID()
	rule.HouseholdID = r.householdID
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now().UTC()
	}

	result, err := r.col.InsertOne(ctx, rule)
	if err != nil {
		return fmt.Errorf("insert rule: %w", err)
	}

	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		rule.ID = oid
	}
	return nil
}

// UpdateFields applies a partial update to the rule.
func (r *RuleRepository) UpdateFields(ctx context.Context, id bson.ObjectID, set bson.D) (*model.Rule, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "household_id", Value: r.householdID},
	}

	updateDoc := bson.D{
		{Key: "$set", Value: set},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated model.Rule
	if err := r.col.FindOneAndUpdate(ctx, filter, updateDoc, opts).Decode(&updated); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("update rule: %w", err)
	}
	return &updated, nil
}

// Delete removes a rule from the collection.
func (r *RuleRepository) Delete(ctx context.Context, id bson.ObjectID) (bool, error) {
	filter := bson.D{
		{Key: "_id", Value: id},
		{Key: "household_id", Value: r.householdID},
	}

	result, err := r.col.DeleteOne(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("delete rule: %w", err)
	}
	return result.DeletedCount > 0, nil
}
