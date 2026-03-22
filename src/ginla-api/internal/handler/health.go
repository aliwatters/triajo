package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliwatters/ginla/ginla-api/internal/database"
)

// HealthHandler holds dependencies for the health endpoint.
type HealthHandler struct {
	db        *database.Client
	startTime time.Time
}

// NewHealthHandler creates a HealthHandler that uses the provided database client.
func NewHealthHandler(db *database.Client) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
	}
}

// healthResponse is the JSON body returned by GET /v1/health.
type healthResponse struct {
	Status  string `json:"status"`
	Mongo   string `json:"mongo"`
	Uptime  string `json:"uptime"`
}

// Check handles GET /v1/health.
// It pings MongoDB and reports overall status.
func (h *HealthHandler) Check(c *gin.Context) {
	mongoStatus := "ok"
	overallStatus := "ok"

	if err := h.db.Ping(context.Background()); err != nil {
		mongoStatus = "error: " + err.Error()
		overallStatus = "degraded"
	}

	uptime := time.Since(h.startTime).Round(time.Second).String()

	statusCode := http.StatusOK
	if overallStatus != "ok" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, healthResponse{
		Status: overallStatus,
		Mongo:  mongoStatus,
		Uptime: uptime,
	})
}
