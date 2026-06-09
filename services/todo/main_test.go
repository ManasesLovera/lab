package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetTodos(t *testing.T) {
	// 1. Mock DB
	mockDb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer mockDb.Close()

	// Swap global db
	db = mockDb

	// 2. Set expectation
	rows := sqlmock.NewRows([]string{"id", "title", "description", "owner", "status", "deadline", "created_at"}).
		AddRow("c0a23368-8097-4008-8df0-d1248039ea7e", "Test task", "Task description", "Alex River", "pending", nil, time.Now())
	mock.ExpectQuery("^SELECT id, title, description, owner, status, deadline, created_at FROM todos").
		WillReturnRows(rows)

	// 3. Setup Fiber app
	app := SetupApp()

	// 4. Execute test request
	req := httptest.NewRequest("GET", "/api/todos", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to test app: %v", err)
	}

	// 5. Verify results
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result []Todo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 1 || result[0].Title != "Test task" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestCreateTodoValidation(t *testing.T) {
	// Setup app
	app := SetupApp()

	// Title is required, so passing empty title should fail with 400 Bad Request
	payload := []byte(`{"title":"","description":"No title task"}`)
	req := httptest.NewRequest("POST", "/api/todos", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to test app: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for validation error, got %d", resp.StatusCode)
	}
}

func TestMcpAuthNoKey(t *testing.T) {
	t.Setenv("API_KEY", "mcp-test-key")
	app := SetupApp()

	// Access without API key should fail with 401
	req := httptest.NewRequest("GET", "/mcp/sse", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("failed to test app: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}
