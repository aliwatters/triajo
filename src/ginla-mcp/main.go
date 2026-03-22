package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/aliwatters/ginla/ginla-mcp/repository"
)

const (
	serverName    = "ginla-mcp"
	serverVersion = "0.1.0"
	mcpVersion    = "2024-11-05"
)

// jsonRPCRequest is an incoming JSON-RPC 2.0 message.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonRPCResponse is an outgoing JSON-RPC 2.0 message.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is a standard JSON-RPC error object.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("[ginla-mcp] ")

	mongoURI := getEnv("MONGO_URI", "mongodb://saturn.local:27017")
	mongoDatabase := getEnv("MONGO_DATABASE", "ginla")

	log.Printf("connecting to MongoDB: %s / %s", mongoURI, mongoDatabase)

	ctx := context.Background()

	db, client, err := connectMongo(ctx, mongoURI, mongoDatabase)
	if err != nil {
		log.Fatalf("mongo connection failed: %v", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("mongo disconnect error: %v", err)
		}
	}()

	taskRepo, err := repository.NewTaskRepository(db)
	if err != nil {
		log.Fatalf("task repository init failed: %v", err)
	}

	ruleRepo := repository.NewRuleRepository(db, taskRepo.HouseholdID())

	log.Printf("ready — reading stdin")

	encoder := json.NewEncoder(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	// increase buffer for large messages
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("parse error: %v", err)
			writeError(encoder, nil, -32700, "parse error")
			continue
		}

		// Notifications have no id and no response is expected
		if req.ID == nil {
			log.Printf("notification: %s", req.Method)
			continue
		}

		resp := handleRequest(ctx, taskRepo, ruleRepo, &req)
		if err := encoder.Encode(resp); err != nil {
			log.Printf("encode error: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("stdin error: %v", err)
	}
	log.Printf("stdin closed, exiting")
}

func handleRequest(ctx context.Context, tasks *repository.TaskRepository, rules *repository.RuleRepository, req *jsonRPCRequest) jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": mcpVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": serverName, "version": serverVersion},
			},
		}

	case "tools/list":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": toolList()},
		}

	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, fmt.Sprintf("invalid params: %v", err))
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}

		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		text, err := callTool(callCtx, tasks, rules, params.Name, params.Arguments)
		if err != nil {
			log.Printf("tool error [%s]: %v", params.Name, err)
			return errorResponse(req.ID, -32603, err.Error())
		}

		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"content": []map[string]string{
					{"type": "text", "text": text},
				},
			},
		}

	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func errorResponse(id *json.RawMessage, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	}
}

func writeError(enc *json.Encoder, id *json.RawMessage, code int, message string) {
	_ = enc.Encode(errorResponse(id, code, message))
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func connectMongo(ctx context.Context, uri, dbName string) (*mongo.Database, *mongo.Client, error) {
	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, fmt.Errorf("mongo ping: %w", err)
	}

	return client.Database(dbName), client, nil
}
