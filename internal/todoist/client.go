package todoist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.todoist.com/api/v1"

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type httpError struct {
	StatusCode int
	Body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("todoist api: status %d: %s", e.StatusCode, e.Body)
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &httpError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	return respBody, nil
}

// unwrapResults handles the `{results: [...]}` wrapper returned by Todoist API v1.
// If the payload is already a bare JSON array (legacy v2 behaviour), it's returned as-is.
func unwrapResults(data []byte) (json.RawMessage, error) {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return data, nil
	}
	var wrapper struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Results) == 0 {
		return data, nil
	}
	return wrapper.Results, nil
}

// --- raw calls used by MCP ---

func (c *Client) GetProjects(ctx context.Context) ([]byte, error) {
	return c.do(ctx, http.MethodGet, "/projects", nil)
}

func (c *Client) GetLabels(ctx context.Context) ([]byte, error) {
	return c.do(ctx, http.MethodGet, "/labels", nil)
}

func (c *Client) GetTasks(ctx context.Context, projectID string, limit int) ([]byte, error) {
	path := fmt.Sprintf("/tasks?limit=%d", limit)
	if projectID != "" {
		path += "&project_id=" + url.QueryEscape(projectID)
	}
	return c.do(ctx, http.MethodGet, path, nil)
}

func (c *Client) GetTasksFiltered(ctx context.Context, filter string, limit int) ([]byte, error) {
	path := fmt.Sprintf("/tasks/filter?query=%s&limit=%d", url.QueryEscape(filter), limit)
	return c.do(ctx, http.MethodGet, path, nil)
}

func (c *Client) CreateTask(ctx context.Context, task map[string]interface{}) ([]byte, error) {
	return c.do(ctx, http.MethodPost, "/tasks", task)
}

func (c *Client) UpdateTask(ctx context.Context, taskID string, update map[string]interface{}) ([]byte, error) {
	return c.do(ctx, http.MethodPost, "/tasks/"+taskID, update)
}

func (c *Client) DeleteTask(ctx context.Context, taskID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/tasks/"+taskID, nil)
	return err
}

func (c *Client) CompleteTask(ctx context.Context, taskID string) error {
	_, err := c.do(ctx, http.MethodPost, "/tasks/"+taskID+"/close", nil)
	return err
}

// --- typed helpers used by the Telegram side ---

type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Task struct {
	ID          string   `json:"id"`
	Content     string   `json:"content"`
	Description string   `json:"description"`
	ProjectID   string   `json:"project_id"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels"`
	Due         *struct {
		Date string `json:"date"`
	} `json:"due"`
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	data, err := c.GetProjects(ctx)
	if err != nil {
		return nil, err
	}
	results, err := unwrapResults(data)
	if err != nil {
		return nil, err
	}
	var projects []Project
	if err := json.Unmarshal(results, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (c *Client) ListTasks(ctx context.Context, projectID string, limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 50
	}
	data, err := c.GetTasks(ctx, projectID, limit)
	if err != nil {
		return nil, err
	}
	results, err := unwrapResults(data)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	if err := json.Unmarshal(results, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// IsRetryable reports whether err is worth retrying (network errors, 429, 5xx).
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var he *httpError
	if errors.As(err, &he) {
		return he.StatusCode == http.StatusTooManyRequests || he.StatusCode >= 500
	}
	return true // network/timeout errors
}
