package todoist

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	client *Client
}

func NewMCPServer(client *Client) *MCPServer {
	return &MCPServer{client: client}
}

func (s *MCPServer) RegisterTools(srv *server.MCPServer) {
	srv.AddTool(mcp.NewTool("get_projects",
		mcp.WithDescription("List all Todoist projects."),
	), s.getProjects)

	srv.AddTool(mcp.NewTool("get_labels",
		mcp.WithDescription("List all Todoist labels."),
	), s.getLabels)

	srv.AddTool(mcp.NewTool("get_tasks",
		mcp.WithDescription("Get tasks, optionally filtered by project or Todoist filter query."),
		mcp.WithString("project_id", mcp.Description("Filter by project ID")),
		mcp.WithString("filter", mcp.Description("Todoist filter query (e.g. 'today', 'overdue', '#Work')")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
	), s.getTasks)

	srv.AddTool(mcp.NewTool("create_task",
		mcp.WithDescription("Create a new Todoist task."),
		mcp.WithString("content", mcp.Description("Task title"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Task description / notes")),
		mcp.WithString("project_id", mcp.Description("Project to add task to")),
		mcp.WithString("due_string", mcp.Description("Natural language due date (e.g. 'tomorrow')")),
		mcp.WithString("due_date", mcp.Description("Absolute due date YYYY-MM-DD")),
		mcp.WithNumber("priority", mcp.Description("1 (normal) to 4 (urgent)")),
	), s.createTask)

	srv.AddTool(mcp.NewTool("update_task",
		mcp.WithDescription("Update an existing Todoist task."),
		mcp.WithString("task_id", mcp.Description("Task ID"), mcp.Required()),
		mcp.WithString("content", mcp.Description("New title")),
		mcp.WithString("description", mcp.Description("New description")),
		mcp.WithString("due_string", mcp.Description("New due date (natural language)")),
		mcp.WithString("due_date", mcp.Description("New absolute due date YYYY-MM-DD")),
		mcp.WithNumber("priority", mcp.Description("New priority (1-4)")),
	), s.updateTask)

	srv.AddTool(mcp.NewTool("delete_task",
		mcp.WithDescription("Delete a Todoist task."),
		mcp.WithString("task_id", mcp.Description("Task ID"), mcp.Required()),
	), s.deleteTask)

	srv.AddTool(mcp.NewTool("complete_task",
		mcp.WithDescription("Complete a Todoist task."),
		mcp.WithString("task_id", mcp.Description("Task ID"), mcp.Required()),
	), s.completeTask)
}

func (s *MCPServer) getProjects(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	data, err := s.client.GetProjects(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) getLabels(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	data, err := s.client.GetLabels(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) getTasks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	projectID := strParam(args, "project_id")
	filter := strParam(args, "filter")
	limit := intParam(args, "limit", 20)

	var data []byte
	var err error
	if filter != "" {
		data, err = s.client.GetTasksFiltered(ctx, filter, limit)
	} else {
		data, err = s.client.GetTasks(ctx, projectID, limit)
	}
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) createTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	content := strParam(args, "content")
	if content == "" {
		return mcp.NewToolResultError("content is required"), nil
	}

	task := map[string]interface{}{"content": content}
	for _, key := range []string{"description", "project_id", "due_string", "due_date"} {
		if v := strParam(args, key); v != "" {
			task[key] = v
		}
	}
	if v, ok := args["priority"]; ok && v != nil {
		task["priority"] = v
	}
	if v, ok := args["labels"]; ok && v != nil {
		task["labels"] = v
	}

	data, err := s.client.CreateTask(ctx, task)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) updateTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	taskID := strParam(args, "task_id")
	if taskID == "" {
		return mcp.NewToolResultError("task_id is required"), nil
	}

	update := map[string]interface{}{}
	for _, key := range []string{"content", "description", "due_string", "due_date"} {
		if v := strParam(args, key); v != "" {
			update[key] = v
		}
	}
	if v, ok := args["priority"]; ok && v != nil {
		update["priority"] = v
	}
	if v, ok := args["labels"]; ok && v != nil {
		update["labels"] = v
	}

	data, err := s.client.UpdateTask(ctx, taskID, update)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) deleteTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	taskID := strParam(args, "task_id")
	if taskID == "" {
		return mcp.NewToolResultError("task_id is required"), nil
	}
	if err := s.client.DeleteTask(ctx, taskID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Deleted task %s", taskID)), nil
}

func (s *MCPServer) completeTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	taskID := strParam(args, "task_id")
	if taskID == "" {
		return mcp.NewToolResultError("task_id is required"), nil
	}
	if err := s.client.CompleteTask(ctx, taskID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Completed task %s", taskID)), nil
}

// --- helpers ---

func strParam(args map[string]interface{}, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func intParam(args map[string]interface{}, key string, def int) int {
	v, ok := args[key]
	if !ok || v == nil {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return def
}

func formatJSON(data []byte) string {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return string(data)
	}
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return string(data)
	}
	return string(pretty)
}
