package todoist

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("token")
	c.baseURL = srv.URL
	c.httpClient = srv.Client()
	return c, srv
}

func TestClient_ListProjects_UnwrapsResults(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("missing bearer token: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/projects" {
			t.Errorf("path: %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"1","name":"Inbox"},{"id":"2","name":"Work"}]}`))
	})
	projects, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 2 || projects[0].Name != "Inbox" || projects[1].Name != "Work" {
		t.Errorf("unexpected projects: %v", projects)
	}
}

func TestClient_ListTasks_IncludesProjectFilter(t *testing.T) {
	var gotQuery string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	if _, err := c.ListTasks(context.Background(), "42", 5); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "project_id=42") || !strings.Contains(gotQuery, "limit=5") {
		t.Errorf("query: %q", gotQuery)
	}
}

func TestClient_GetTasksWithOptions_IncludesStructuralFilters(t *testing.T) {
	var gotQuery string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			t.Errorf("path: %q", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	_, err := c.GetTasksWithOptions(context.Background(), TaskListOptions{
		ProjectID: "project 1",
		SectionID: "section 1",
		ParentID:  "parent 1",
		Label:     "health",
		Limit:     7,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"project_id=project+1",
		"section_id=section+1",
		"parent_id=parent+1",
		"label=health",
		"limit=7",
	} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}

func TestClient_GetTask_UsesTaskIDPath(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tasks/task1" {
			t.Errorf("request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"task1","project_id":"project1"}`))
	})
	if _, err := c.GetTask(context.Background(), "task1"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_CreateTask_SendsJSON(t *testing.T) {
	var gotBody string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type: %q", r.Header.Get("Content-Type"))
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"id":"123","content":"hi"}`))
	})
	payload := map[string]interface{}{"content": "hi", "priority": 2}
	if _, err := c.CreateTask(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(gotBody), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["content"] != "hi" || parsed["priority"].(float64) != 2 {
		t.Errorf("payload: %v", parsed)
	}
}

func TestClient_CreateLabel_SendsJSON(t *testing.T) {
	var gotBody string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/labels" {
			t.Errorf("request: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"id":"label1","name":"health"}`))
	})
	payload := map[string]interface{}{"name": "health", "color": "berry_red", "is_favorite": true}
	if _, err := c.CreateLabel(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(gotBody), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["name"] != "health" || parsed["color"] != "berry_red" || parsed["is_favorite"] != true {
		t.Errorf("payload: %v", parsed)
	}
}

func TestClient_UpdateAndDeleteLabel_UseLabelIDPath(t *testing.T) {
	var requests []string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"id":"label1","name":"health"}`))
		}
	})
	if _, err := c.UpdateLabel(context.Background(), "label1", map[string]interface{}{"name": "health"}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteLabel(context.Background(), "label1"); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /labels/label1", "DELETE /labels/label1"}
	if strings.Join(requests, ",") != strings.Join(want, ",") {
		t.Errorf("requests: got %v, want %v", requests, want)
	}
}

func TestClient_MoveTask_UsesMoveEndpoint(t *testing.T) {
	var gotBody string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks/task1/move" {
			t.Errorf("request: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"id":"task1"}`))
	})
	if _, err := c.MoveTask(context.Background(), "task1", map[string]interface{}{"parent_id": "parent1"}); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(gotBody), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["parent_id"] != "parent1" {
		t.Errorf("payload: %v", parsed)
	}
}

func TestClient_HTTPError_ReturnsHTTPError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limit"))
	})
	_, err := c.GetProjects(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsRetryable(err) {
		t.Error("429 should be retryable")
	}
}

func TestUnwrapResults_PlainArray(t *testing.T) {
	in := []byte(`[{"id":"1","name":"Inbox"}]`)
	out, err := unwrapResults(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Errorf("expected passthrough, got %s", out)
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&httpError{StatusCode: 429}, true},
		{&httpError{StatusCode: 502}, true},
		{&httpError{StatusCode: 400}, false},
		{&httpError{StatusCode: 404}, false},
		{nil, false},
	}
	for _, c := range cases {
		if got := IsRetryable(c.err); got != c.want {
			t.Errorf("%v → %v, want %v", c.err, got, c.want)
		}
	}
}
