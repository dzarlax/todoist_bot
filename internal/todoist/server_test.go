package todoist

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func callToolRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

func TestMCPServer_CreateTask_MapsLabelsAndParent(t *testing.T) {
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks" {
			t.Errorf("request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"id":"task1"}`))
	})

	srv := NewMCPServer(client)
	result, err := srv.createTask(context.Background(), callToolRequest(map[string]any{
		"content":   "child",
		"parent_id": "parent1",
		"labels":    []any{"health", "todoist-bot"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result)
	}
	if gotBody["content"] != "child" || gotBody["parent_id"] != "parent1" {
		t.Errorf("payload: %v", gotBody)
	}
	labels, ok := gotBody["labels"].([]any)
	if !ok || len(labels) != 2 || labels[0] != "health" || labels[1] != "todoist-bot" {
		t.Errorf("labels: %v", gotBody["labels"])
	}
}

func TestMCPServer_UpdateLabel_RequiresMutableField(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected HTTP request")
	})
	srv := NewMCPServer(client)

	result, err := srv.updateLabel(context.Background(), callToolRequest(map[string]any{
		"label_id": "label1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got %+v", result)
	}
}

func TestMCPServer_UpdateTask_AllowsClearingLabels(t *testing.T) {
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks/task1" {
			t.Errorf("request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"id":"task1"}`))
	})

	srv := NewMCPServer(client)
	result, err := srv.updateTask(context.Background(), callToolRequest(map[string]any{
		"task_id": "task1",
		"labels":  []any{},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result)
	}
	labels, ok := gotBody["labels"].([]any)
	if !ok || len(labels) != 0 {
		t.Errorf("labels: %v", gotBody["labels"])
	}
}

func TestMCPServer_MoveTask_ClearsParentWithinProject(t *testing.T) {
	var gotBody map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks/task1/move" {
			t.Errorf("request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"id":"task1"}`))
	})

	srv := NewMCPServer(client)
	result, err := srv.moveTask(context.Background(), callToolRequest(map[string]any{
		"task_id":      "task1",
		"project_id":   "project1",
		"clear_parent": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %+v", result)
	}
	if _, ok := gotBody["parent_id"]; !ok {
		t.Fatalf("parent_id missing from payload: %v", gotBody)
	}
	if gotBody["parent_id"] != nil {
		t.Errorf("parent_id: got %v, want nil", gotBody["parent_id"])
	}
	if gotBody["project_id"] != "project1" {
		t.Errorf("project_id: got %v, want project1", gotBody["project_id"])
	}
}

func TestMCPServer_MoveTask_RejectsConflictingParentArgs(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected HTTP request")
	})
	srv := NewMCPServer(client)

	result, err := srv.moveTask(context.Background(), callToolRequest(map[string]any{
		"task_id":      "task1",
		"parent_id":    "parent1",
		"clear_parent": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got %+v", result)
	}
}

func TestMCPServer_MoveTask_RejectsClearParentWithoutDestination(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected HTTP request")
	})
	srv := NewMCPServer(client)

	result, err := srv.moveTask(context.Background(), callToolRequest(map[string]any{
		"task_id":      "task1",
		"clear_parent": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("expected tool error, got %+v", result)
	}
}
