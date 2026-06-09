package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Todo represents a task in the database
type Todo struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Owner       string     `json:"owner"`
	Deadline    *time.Time `json:"deadline"`
	CreatedAt   time.Time  `json:"created_at"`
}

// JSON-RPC 2.0 structures for MCP
type JsonRpcRequest struct {
	JsonRpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type JsonRpcResponse struct {
	JsonRpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JsonRpcErr `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type JsonRpcErr struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Tool definition
type McpTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type McpListToolsResult struct {
	Tools []McpTool `json:"tools"`
}

type McpCallToolResult struct {
	Content []McpTextContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

type McpTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Global active connections map for SSE
var (
	connections   = make(map[string]chan string)
	connectionsMu sync.RWMutex
	db            *sql.DB
)

func main() {
	// 1. Initialize environment and DB
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://todo_user:todo_pass_2026_x@postgres:5432/todo_db?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Wait for DB to be ready
	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		log.Println("Waiting for database connection...")
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database not reachable: %v", err)
	}

	// Run migration
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			title VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			owner VARCHAR(255) DEFAULT '',
			deadline TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		ALTER TABLE todos ADD COLUMN IF NOT EXISTS owner VARCHAR(255) DEFAULT '';
	`)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	app := SetupApp()

	log.Printf("Starting Todo server on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}

// SetupApp configures the Fiber application routes and middleware
func SetupApp() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Static route for UI
	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendFile("./templates/index.html")
	})

	// 3. REST API Routes
	api := app.Group("/api")
	
	api.Get("/todos", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database not initialized"})
		}
		rows, err := db.Query("SELECT id, title, description, owner, status, deadline, created_at FROM todos ORDER BY created_at DESC")
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		todos := []Todo{}
		for rows.Next() {
			var t Todo
			err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Owner, &t.Status, &t.Deadline, &t.CreatedAt)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
			todos = append(todos, t)
		}
		return c.JSON(todos)
	})

	api.Post("/todos", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database not initialized"})
		}
		var input struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Status      string `json:"status"`
			Owner       string `json:"owner"`
			Deadline    string `json:"deadline"`
		}
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if input.Title == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Title is required"})
		}

		var deadlineVal sql.NullTime
		if input.Deadline != "" {
			t, err := time.Parse(time.RFC3339, input.Deadline)
			if err == nil {
				deadlineVal.Time = t
				deadlineVal.Valid = true
			} else {
				// Try simple date
				t, err = time.Parse("2006-01-02", input.Deadline)
				if err == nil {
					deadlineVal.Time = t
					deadlineVal.Valid = true
				}
			}
		}

		statusVal := input.Status
		if statusVal == "" {
			statusVal = "scheduled"
		}

		var t Todo
		err := db.QueryRow(
			"INSERT INTO todos (title, description, owner, status, deadline) VALUES ($1, $2, $3, $4, $5) RETURNING id, title, description, owner, status, deadline, created_at",
			input.Title, input.Description, input.Owner, statusVal, deadlineVal,
		).Scan(&t.ID, &t.Title, &t.Description, &t.Owner, &t.Status, &t.Deadline, &t.CreatedAt)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(201).JSON(t)
	})

	api.Put("/todos/:id", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database not initialized"})
		}
		id := c.Params("id")
		var input struct {
			Title       *string `json:"title"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
			Owner       *string `json:"owner"`
			Deadline    *string `json:"deadline"`
		}
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		if input.Title != nil && *input.Title == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Title cannot be empty"})
		}

		var deadlineVal sql.NullTime
		hasDeadlineUpdate := false
		if input.Deadline != nil {
			hasDeadlineUpdate = true
			if *input.Deadline != "" {
				t, err := time.Parse(time.RFC3339, *input.Deadline)
				if err == nil {
					deadlineVal.Time = t
					deadlineVal.Valid = true
				}
			}
		}

		_, err := db.Exec(
			"UPDATE todos SET title = COALESCE($1, title), description = COALESCE($2, description), status = COALESCE($3, status), owner = COALESCE($4, owner), deadline = CASE WHEN $5::boolean THEN $6 ELSE deadline END WHERE id = $7",
			input.Title, input.Description, input.Status, input.Owner, hasDeadlineUpdate, deadlineVal, id,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		var t Todo
		err = db.QueryRow("SELECT id, title, description, owner, status, deadline, created_at FROM todos WHERE id = $1", id).
			Scan(&t.ID, &t.Title, &t.Description, &t.Owner, &t.Status, &t.Deadline, &t.CreatedAt)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Todo not found"})
		}

		return c.JSON(t)
	})

	api.Delete("/todos/:id", func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database not initialized"})
		}
		id := c.Params("id")
		_, err := db.Exec("DELETE FROM todos WHERE id = $1", id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "deleted"})
	})

	// 4. MCP Server Endpoints (SSE-based)
	// Middleware to authenticate MCP requests via token
	mcpAuth := func(c *fiber.Ctx) error {
		expectedToken := os.Getenv("API_KEY")
		if expectedToken == "" {
			return c.Next() // If not set, allow
		}

		authHeader := c.Get("Authorization")
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			// Fallback to query param
			token = c.Query("token")
		}

		if token != expectedToken {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized API Key"})
		}
		return c.Next()
	}

	app.Get("/mcp/sse", mcpAuth, func(c *fiber.Ctx) error {
		connID := uuid.New().String()
		msgChan := make(chan string, 100)

		connectionsMu.Lock()
		connections[connID] = msgChan
		connectionsMu.Unlock()

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("Transfer-Encoding", "chunked")

		// Send endpoint info to client
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			// Send connection URL
			fmt.Fprintf(w, "event: endpoint\ndata: /mcp/messages?connection_id=%s\n\n", connID)
			w.Flush()

			// Keep alive ticker
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			defer func() {
				connectionsMu.Lock()
				delete(connections, connID)
				connectionsMu.Unlock()
				close(msgChan)
			}()

			for {
				select {
				case msg, ok := <-msgChan:
					if !ok {
						return
					}
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
					w.Flush()
				case <-ticker.C:
					// SSE ping
					fmt.Fprintf(w, ": ping\n\n")
					w.Flush()
				case <-c.Context().Done():
					return
				}
			}
		})

		return nil
	})

	app.Post("/mcp/messages", mcpAuth, func(c *fiber.Ctx) error {
		connID := c.Query("connection_id")
		if connID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "connection_id query parameter required"})
		}

		connectionsMu.RLock()
		msgChan, ok := connections[connID]
		connectionsMu.RUnlock()

		if !ok {
			return c.Status(404).JSON(fiber.Map{"error": "Connection not found or expired"})
		}

		var req JsonRpcRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON-RPC payload"})
		}

		// Handle request asynchronously or in a worker and stream back via SSE
		go handleMcpRequest(req, msgChan)

		return c.SendStatus(202) // Accepted
	})

	return app
}

// Process JSON-RPC request and push output back to the SSE stream
func handleMcpRequest(req JsonRpcRequest, out chan string) {
	var response JsonRpcResponse
	response.JsonRpc = "2.0"
	response.ID = req.ID

	switch req.Method {
	case "initialize":
		// Respond with server details
		response.Result = fiber.Map{
			"protocolVersion": "2024-11-05",
			"capabilities": fiber.Map{
				"tools": fiber.Map{},
			},
			"serverInfo": fiber.Map{
				"name":    "todo-mcp-server",
				"version": "1.0.0",
			},
		}

	case "tools/list":
		// List the exposed tools
		tools := []McpTool{
			{
				Name:        "list_todos",
				Description: "List all active or completed todo items. Filters: status ('pending' or 'completed').",
				InputSchema: fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"status": fiber.Map{
							"type":        "string",
							"description": "Optional filter: 'pending' or 'completed'",
						},
					},
				},
			},
			{
				Name:        "create_todo",
				Description: "Create a new todo item.",
				InputSchema: fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"title": fiber.Map{
							"type":        "string",
							"description": "The title of the todo item (required)",
						},
						"description": fiber.Map{
							"type":        "string",
							"description": "The optional description",
						},
						"owner": fiber.Map{
							"type":        "string",
							"description": "Optional owner name/team",
						},
						"deadline": fiber.Map{
							"type":        "string",
							"description": "Optional deadline date or datetime (RFC3339 format)",
						},
					},
					"required": []string{"title"},
				},
			},
			{
				Name:        "update_todo",
				Description: "Update an existing todo item's status, title, description, or deadline.",
				InputSchema: fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"id": fiber.Map{
							"type":        "string",
							"description": "The UUID of the todo to update",
						},
						"title": fiber.Map{
							"type":        "string",
							"description": "Updated title",
						},
						"description": fiber.Map{
							"type":        "string",
							"description": "Updated description",
						},
						"owner": fiber.Map{
							"type":        "string",
							"description": "Updated owner name/team",
						},
						"status": fiber.Map{
							"type":        "string",
							"description": "Updated status: 'pending' or 'completed'",
						},
						"deadline": fiber.Map{
							"type":        "string",
							"description": "Updated deadline date (RFC3339 format)",
						},
					},
					"required": []string{"id"},
				},
			},
			{
				Name:        "delete_todo",
				Description: "Delete a todo item by ID.",
				InputSchema: fiber.Map{
					"type": "object",
					"properties": fiber.Map{
						"id": fiber.Map{
							"type":        "string",
							"description": "The UUID of the todo to delete",
						},
					},
					"required": []string{"id"},
				},
			},
		}

		response.Result = McpListToolsResult{Tools: tools}

	case "tools/call":
		// Handle tool execution
		var callParams struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			response.Error = &JsonRpcErr{Code: -32602, Message: "Invalid call parameters"}
			break
		}

		result, err := executeTool(callParams.Name, callParams.Arguments)
		if err != nil {
			response.Result = McpCallToolResult{
				Content: []McpTextContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}
		} else {
			response.Result = result
		}

	default:
		// Method not found
		response.Error = &JsonRpcErr{Code: -32601, Message: fmt.Sprintf("Method %s not found", req.Method)}
	}

	payload, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	// Send back to client stream
	out <- string(payload)
}

func executeTool(name string, arguments []byte) (McpCallToolResult, error) {
	var res McpCallToolResult
	res.Content = []McpTextContent{}

	switch name {
	case "list_todos":
		var args struct {
			Status string `json:"status"`
		}
		json.Unmarshal(arguments, &args)

		query := "SELECT id, title, description, owner, status, deadline, created_at FROM todos"
		var rows *sql.Rows
		var err error
		if args.Status != "" {
			query += " WHERE status = $1 ORDER BY created_at DESC"
			rows, err = db.Query(query, args.Status)
		} else {
			query += " ORDER BY created_at DESC"
			rows, err = db.Query(query)
		}

		if err != nil {
			return res, err
		}
		defer rows.Close()

		todos := []Todo{}
		for rows.Next() {
			var t Todo
			err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Owner, &t.Status, &t.Deadline, &t.CreatedAt)
			if err == nil {
				todos = append(todos, t)
			}
		}

		data, _ := json.MarshalIndent(todos, "", "  ")
		res.Content = append(res.Content, McpTextContent{
			Type: "text",
			Text: fmt.Sprintf("Found %d todos:\n%s", len(todos), string(data)),
		})

	case "create_todo":
		var args struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Owner       string `json:"owner"`
			Deadline    string `json:"deadline"`
		}
		json.Unmarshal(arguments, &args)
		if args.Title == "" {
			return res, fmt.Errorf("title is required")
		}

		var deadlineVal sql.NullTime
		if args.Deadline != "" {
			t, err := time.Parse(time.RFC3339, args.Deadline)
			if err == nil {
				deadlineVal.Time = t
				deadlineVal.Valid = true
			} else {
				t, err = time.Parse("2006-01-02", args.Deadline)
				if err == nil {
					deadlineVal.Time = t
					deadlineVal.Valid = true
				}
			}
		}

		var t Todo
		err := db.QueryRow(
			"INSERT INTO todos (title, description, owner, status, deadline) VALUES ($1, $2, $3, 'scheduled', $4) RETURNING id, title, description, owner, status, deadline, created_at",
			args.Title, args.Description, args.Owner, deadlineVal,
		).Scan(&t.ID, &t.Title, &t.Description, &t.Owner, &t.Status, &t.Deadline, &t.CreatedAt)

		if err != nil {
			return res, err
		}

		res.Content = append(res.Content, McpTextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully created todo item with ID: %s", t.ID),
		})

	case "update_todo":
		var args struct {
			ID          string  `json:"id"`
			Title       *string `json:"title"`
			Description *string `json:"description"`
			Status      *string `json:"status"`
			Owner       *string `json:"owner"`
			Deadline    *string `json:"deadline"`
		}
		json.Unmarshal(arguments, &args)
		if args.ID == "" {
			return res, fmt.Errorf("id is required")
		}

		var deadlineVal sql.NullTime
		hasDeadlineUpdate := false
		if args.Deadline != nil {
			hasDeadlineUpdate = true
			if *args.Deadline != "" {
				t, err := time.Parse(time.RFC3339, *args.Deadline)
				if err == nil {
					deadlineVal.Time = t
					deadlineVal.Valid = true
				}
			}
		}

		_, err := db.Exec(
			"UPDATE todos SET title = COALESCE($1, title), description = COALESCE($2, description), status = COALESCE($3, status), owner = COALESCE($4, owner), deadline = CASE WHEN $5::boolean THEN $6 ELSE deadline END WHERE id = $7",
			args.Title, args.Description, args.Status, args.Owner, hasDeadlineUpdate, deadlineVal, args.ID,
		)
		if err != nil {
			return res, err
		}

		res.Content = append(res.Content, McpTextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully updated todo item with ID: %s", args.ID),
		})

	case "delete_todo":
		var args struct {
			ID string `json:"id"`
		}
		json.Unmarshal(arguments, &args)
		if args.ID == "" {
			return res, fmt.Errorf("id is required")
		}

		_, err := db.Exec("DELETE FROM todos WHERE id = $1", args.ID)
		if err != nil {
			return res, err
		}

		res.Content = append(res.Content, McpTextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted todo item with ID: %s", args.ID),
		})

	default:
		return res, fmt.Errorf("unknown tool name: %s", name)
	}

	return res, nil
}
