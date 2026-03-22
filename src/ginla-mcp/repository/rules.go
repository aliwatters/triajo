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

// RuleRepository handles MongoDB operations for triage rules.
type RuleRepository struct {
	col         *mongo.Collection
	householdID bson.ObjectID
}

// NewRuleRepository creates a RuleRepository using the given database.
// It looks up the household from the tasks repository's household ID.
func NewRuleRepository(db *mongo.Database, householdID bson.ObjectID) *RuleRepository {
	return &RuleRepository{
		col:         db.Collection("rules"),
		householdID: householdID,
	}
}

// ListActive returns all active rules for the household, sorted by order ascending.
func (r *RuleRepository) ListActive(ctx context.Context) ([]model.Rule, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.D{
		{Key: "household_id", Value: r.householdID},
		{Key: "active", Value: true},
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
