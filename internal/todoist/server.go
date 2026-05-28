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

	srv.AddTool(mcp.NewTool("create_label",
		mcp.WithDescription("Create a Todoist personal label."),
		mcp.WithString("name", mcp.Description("Label name"), mcp.Required()),
		mcp.WithString("color", mcp.Description("Todoist label color name or ID")),
		mcp.WithNumber("order", mcp.Description("Label order in the label list")),
		mcp.WithBoolean("is_favorite", mcp.Description("Whether the label is marked as favorite")),
	), s.createLabel)

	srv.AddTool(mcp.NewTool("update_label",
		mcp.WithDescription("Update a Todoist personal label."),
		mcp.WithString("label_id", mcp.Description("Label ID"), mcp.Required()),
		mcp.WithString("name", mcp.Description("New label name")),
		mcp.WithString("color", mcp.Description("New Todoist label color name or ID")),
		mcp.WithNumber("order", mcp.Description("New label order")),
		mcp.WithBoolean("is_favorite", mcp.Description("Whether the label is marked as favorite")),
	), s.updateLabel)

	srv.AddTool(mcp.NewTool("delete_label",
		mcp.WithDescription("Delete a Todoist personal label and remove it from tasks."),
		mcp.WithString("label_id", mcp.Description("Label ID"), mcp.Required()),
	), s.deleteLabel)

	srv.AddTool(mcp.NewTool("get_tasks",
		mcp.WithDescription("Get tasks, optionally filtered by project or Todoist filter query."),
		mcp.WithString("project_id", mcp.Description("Filter by project ID")),
		mcp.WithString("section_id", mcp.Description("Filter by section ID")),
		mcp.WithString("parent_id", mcp.Description("Filter by parent task ID to list subtasks")),
		mcp.WithString("label", mcp.Description("Filter by label name")),
		mcp.WithString("filter", mcp.Description("Todoist filter query (e.g. 'today', 'overdue', '#Work')")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
	), s.getTasks)

	srv.AddTool(mcp.NewTool("create_task",
		mcp.WithDescription("Create a new Todoist task."),
		mcp.WithString("content", mcp.Description("Task title"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Task description / notes")),
		mcp.WithString("project_id", mcp.Description("Project to add task to")),
		mcp.WithString("section_id", mcp.Description("Section to add task to")),
		mcp.WithString("parent_id", mcp.Description("Parent task ID for creating a subtask")),
		mcp.WithString("due_string", mcp.Description("Natural language due date (e.g. 'tomorrow')")),
		mcp.WithString("due_date", mcp.Description("Absolute due date YYYY-MM-DD")),
		mcp.WithNumber("priority", mcp.Description("1 (normal) to 4 (urgent)")),
		mcp.WithArray("labels", mcp.Description("Todoist label names"), mcp.WithStringItems()),
	), s.createTask)

	srv.AddTool(mcp.NewTool("update_task",
		mcp.WithDescription("Update an existing Todoist task."),
		mcp.WithString("task_id", mcp.Description("Task ID"), mcp.Required()),
		mcp.WithString("content", mcp.Description("New title")),
		mcp.WithString("description", mcp.Description("New description")),
		mcp.WithString("due_string", mcp.Description("New due date (natural language)")),
		mcp.WithString("due_date", mcp.Description("New absolute due date YYYY-MM-DD")),
		mcp.WithNumber("priority", mcp.Description("New priority (1-4)")),
		mcp.WithArray("labels", mcp.Description("Todoist label names"), mcp.WithStringItems()),
	), s.updateTask)

	srv.AddTool(mcp.NewTool("move_task",
		mcp.WithDescription("Move a Todoist task to another project, section, or parent task."),
		mcp.WithString("task_id", mcp.Description("Task ID"), mcp.Required()),
		mcp.WithString("project_id", mcp.Description("Destination project ID")),
		mcp.WithString("section_id", mcp.Description("Destination section ID")),
		mcp.WithString("parent_id", mcp.Description("Destination parent task ID")),
		mcp.WithBoolean("clear_section", mcp.Description("Move the task out of its current section")),
		mcp.WithBoolean("clear_parent", mcp.Description("Move the task out from under its parent task")),
	), s.moveTask)

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

func (s *MCPServer) createLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	name := strParam(args, "name")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	label := map[string]interface{}{"name": name}
	addStringFields(label, args, "color")
	addRawFields(label, args, "order", "is_favorite")

	data, err := s.client.CreateLabel(ctx, label)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) updateLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	labelID := strParam(args, "label_id")
	if labelID == "" {
		return mcp.NewToolResultError("label_id is required"), nil
	}

	update := map[string]interface{}{}
	addStringFields(update, args, "name", "color")
	addRawFields(update, args, "order", "is_favorite")
	if len(update) == 0 {
		return mcp.NewToolResultError("at least one label field is required"), nil
	}

	data, err := s.client.UpdateLabel(ctx, labelID, update)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) deleteLabel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	labelID := strParam(args, "label_id")
	if labelID == "" {
		return mcp.NewToolResultError("label_id is required"), nil
	}
	if err := s.client.DeleteLabel(ctx, labelID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Deleted label %s", labelID)), nil
}

func (s *MCPServer) getTasks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	filter := strParam(args, "filter")
	limit := intParam(args, "limit", 20)

	var data []byte
	var err error
	if filter != "" {
		data, err = s.client.GetTasksFiltered(ctx, filter, limit)
	} else {
		data, err = s.client.GetTasksWithOptions(ctx, TaskListOptions{
			ProjectID: strParam(args, "project_id"),
			SectionID: strParam(args, "section_id"),
			ParentID:  strParam(args, "parent_id"),
			Label:     strParam(args, "label"),
			Limit:     limit,
		})
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
	addStringFields(task, args, "description", "project_id", "section_id", "parent_id", "due_string", "due_date")
	if v, ok := args["priority"]; ok && v != nil {
		task["priority"] = v
	}
	if labels, ok := stringSliceParam(args, "labels"); ok {
		task["labels"] = labels
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
	addStringFields(update, args, "content", "description", "due_string", "due_date")
	if v, ok := args["priority"]; ok && v != nil {
		update["priority"] = v
	}
	if labels, ok := stringSliceParam(args, "labels"); ok {
		update["labels"] = labels
	}

	data, err := s.client.UpdateTask(ctx, taskID, update)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(formatJSON(data)), nil
}

func (s *MCPServer) moveTask(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	taskID := strParam(args, "task_id")
	if taskID == "" {
		return mcp.NewToolResultError("task_id is required"), nil
	}
	if strParam(args, "parent_id") != "" && boolParam(args, "clear_parent") {
		return mcp.NewToolResultError("parent_id and clear_parent cannot both be set"), nil
	}
	if strParam(args, "section_id") != "" && boolParam(args, "clear_section") {
		return mcp.NewToolResultError("section_id and clear_section cannot both be set"), nil
	}

	move := map[string]interface{}{}
	addStringFields(move, args, "project_id", "section_id", "parent_id")
	if boolParam(args, "clear_parent") {
		move["parent_id"] = nil
	}
	if boolParam(args, "clear_section") {
		move["section_id"] = nil
	}
	if len(move) == 0 {
		return mcp.NewToolResultError("at least one destination field is required"), nil
	}

	data, err := s.client.MoveTask(ctx, taskID, move)
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

func stringSliceParam(args map[string]interface{}, key string) ([]string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return nil, false
	}
	switch labels := v.(type) {
	case []string:
		return labels, true
	case []interface{}:
		out := make([]string, 0, len(labels))
		for _, label := range labels {
			if s, ok := label.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out, true
	}
	return nil, false
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

func boolParam(args map[string]interface{}, key string) bool {
	v, ok := args[key]
	if !ok || v == nil {
		return false
	}
	b, _ := v.(bool)
	return b
}

func addStringFields(dst map[string]interface{}, args map[string]interface{}, keys ...string) {
	for _, key := range keys {
		if v := strParam(args, key); v != "" {
			dst[key] = v
		}
	}
}

func addRawFields(dst map[string]interface{}, args map[string]interface{}, keys ...string) {
	for _, key := range keys {
		if v, ok := args[key]; ok && v != nil {
			dst[key] = v
		}
	}
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
