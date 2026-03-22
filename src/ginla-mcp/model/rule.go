package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Rule is a pattern-matching rule for auto-triage.
type Rule struct {
	ID          bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	HouseholdID bson.ObjectID  `bson:"household_id"  json:"household_id"`
	Name        string         `bson:"name"          json:"name"`
	Pattern     string         `bson:"pattern"       json:"pattern"`
	Tag         Tag            `bson:"tag"           json:"tag"`
	HandlerID   *bson.ObjectID `bson:"handler_id"    json:"handler_id"`
	Priority    *Priority      `bson:"priority"      json:"priority"`
	Order       int            `bson:"order"         json:"order"`
	Active      bool           `bson:"active"        json:"active"`
	CreatedAt   time.Time      `bson:"created_at"    json:"created_at"`
}
