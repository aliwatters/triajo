package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aliwatters/ginla/ginla-api/internal/config"
	"github.com/aliwatters/ginla/ginla-api/internal/database"
	"github.com/aliwatters/ginla/ginla-api/internal/handler"
	"github.com/aliwatters/ginla/ginla-api/internal/repository"
)

func main() {
	cfg := config.Load()

	log.Printf("connecting to MongoDB at %s (db: %s)", cfg.MongoURI, cfg.MongoDatabase)
	db, err := database.Connect(context.Background(), cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.Disconnect(ctx); err != nil {
			log.Printf("error disconnecting from MongoDB: %v", err)
		}
	}()

	router := gin.Default()

	taskRepo, err := repository.NewTaskRepository(db.Database)
	if err != nil {
		log.Fatalf("failed to init task repository: %v", err)
	}

	ruleRepo := repository.NewRuleRepository(db.Database, taskRepo.HouseholdID())

	v1 := router.Group("/v1")
	{
		healthHandler := handler.NewHealthHandler(db)
		v1.GET("/health", healthHandler.Check)

		taskHandler := handler.NewTaskHandler(taskRepo)
		v1.POST("/tasks", taskHandler.Create)
		v1.GET("/tasks", taskHandler.List)
		v1.GET("/tasks/:id", taskHandler.Get)
		v1.PATCH("/tasks/:id", taskHandler.Update)
		v1.DELETE("/tasks/:id", taskHandler.Delete)

		emailHandler := handler.NewEmailHandler(taskRepo)
		v1.POST("/tasks/from-email", emailHandler.FromEmail)

		ruleHandler := handler.NewRuleHandler(ruleRepo)
		v1.POST("/rules", ruleHandler.Create)
		v1.GET("/rules", ruleHandler.List)
		v1.GET("/rules/:id", ruleHandler.Get)
		v1.PATCH("/rules/:id", ruleHandler.Update)
		v1.DELETE("/rules/:id", ruleHandler.Delete)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("starting server on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server exited")
}
