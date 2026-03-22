package handler

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/aliwatters/ginla/ginla-api/internal/model"
	"github.com/aliwatters/ginla/ginla-api/internal/repository"
)

// RuleHandler holds dependencies for the rules endpoints.
type RuleHandler struct {
	repo *repository.RuleRepository
}

// NewRuleHandler creates a RuleHandler.
func NewRuleHandler(repo *repository.RuleRepository) *RuleHandler {
	return &RuleHandler{repo: repo}
}

// ---- request/response types ------------------------------------------------

// createRuleRequest is the JSON body for POST /v1/rules.
type createRuleRequest struct {
	Name      string          `json:"name"       binding:"required"`
	Pattern   string          `json:"pattern"    binding:"required"`
	Tag       model.Tag       `json:"tag"        binding:"required"`
	HandlerID *bson.ObjectID  `json:"handler_id"`
	Priority  *model.Priority `json:"priority"`
	Order     *int            `json:"order"`
	Active    *bool           `json:"active"`
}

// updateRuleRequest is the JSON body for PATCH /v1/rules/:id.
type updateRuleRequest struct {
	Name      *string         `json:"name"`
	Pattern   *string         `json:"pattern"`
	Tag       *model.Tag      `json:"tag"`
	HandlerID *bson.ObjectID  `json:"handler_id"`
	Priority  *model.Priority `json:"priority"`
	Order     *int            `json:"order"`
	Active    *bool           `json:"active"`
}

// ---- handlers --------------------------------------------------------------

// Create handles POST /v1/rules.
func (h *RuleHandler) Create(c *gin.Context) {
	var req createRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !model.ValidTags[req.Tag] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag"})
		return
	}
	if req.Priority != nil && !model.ValidPriorities[*req.Priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority"})
		return
	}
	if _, err := regexp.Compile(req.Pattern); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid regex pattern: " + err.Error()})
		return
	}

	order := 100
	if req.Order != nil {
		order = *req.Order
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}

	rule := &model.Rule{
		Name:      req.Name,
		Pattern:   req.Pattern,
		Tag:       req.Tag,
		HandlerID: req.HandlerID,
		Priority:  req.Priority,
		Order:     order,
		Active:    active,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.repo.Create(ctx, rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// List handles GET /v1/rules.
func (h *RuleHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	rules, err := h.repo.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rules"})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// Get handles GET /v1/rules/:id.
func (h *RuleHandler) Get(c *gin.Context) {
	oid, ok := parseObjectID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	rule, err := h.repo.GetByID(ctx, oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get rule"})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// Update handles PATCH /v1/rules/:id.
func (h *RuleHandler) Update(c *gin.Context) {
	oid, ok := parseObjectID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	var req updateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Tag != nil && !model.ValidTags[*req.Tag] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag"})
		return
	}
	if req.Priority != nil && !model.ValidPriorities[*req.Priority] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priority"})
		return
	}
	if req.Pattern != nil {
		if _, err := regexp.Compile(*req.Pattern); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid regex pattern: " + err.Error()})
			return
		}
	}

	set := bson.D{}
	if req.Name != nil {
		set = append(set, bson.E{Key: "name", Value: *req.Name})
	}
	if req.Pattern != nil {
		set = append(set, bson.E{Key: "pattern", Value: *req.Pattern})
	}
	if req.Tag != nil {
		set = append(set, bson.E{Key: "tag", Value: *req.Tag})
	}
	if req.HandlerID != nil {
		set = append(set, bson.E{Key: "handler_id", Value: *req.HandlerID})
	}
	if req.Priority != nil {
		set = append(set, bson.E{Key: "priority", Value: *req.Priority})
	}
	if req.Order != nil {
		set = append(set, bson.E{Key: "order", Value: *req.Order})
	}
	if req.Active != nil {
		set = append(set, bson.E{Key: "active", Value: *req.Active})
	}

	if len(set) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	updated, err := h.repo.UpdateFields(ctx, oid, set)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rule"})
		return
	}
	if updated == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// Delete handles DELETE /v1/rules/:id.
func (h *RuleHandler) Delete(c *gin.Context) {
	oid, ok := parseObjectID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rule id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	deleted, err := h.repo.Delete(ctx, oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.Status(http.StatusNoContent)
}
