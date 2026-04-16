package todoist

import (
	"context"
	"strings"
	"sync"
	"time"
)

type ProjectCache struct {
	client *Client
	ttl    time.Duration

	mu         sync.Mutex
	projects   []Project
	byNameLC   map[string]string
	fetchedAt  time.Time
}

func NewProjectCache(client *Client, ttl time.Duration) *ProjectCache {
	return &ProjectCache{client: client, ttl: ttl}
}

func (c *ProjectCache) refresh(ctx context.Context) error {
	projects, err := c.client.ListProjects(ctx)
	if err != nil {
		return err
	}
	byName := make(map[string]string, len(projects))
	for _, p := range projects {
		byName[strings.ToLower(p.Name)] = p.ID
	}
	c.projects = projects
	c.byNameLC = byName
	c.fetchedAt = time.Now()
	return nil
}

// ensure guarantees cache is fresh (≤ ttl old). Caller must hold c.mu.
func (c *ProjectCache) ensure(ctx context.Context) error {
	if c.byNameLC != nil && time.Since(c.fetchedAt) < c.ttl {
		return nil
	}
	return c.refresh(ctx)
}

// IDByName returns the project id for the given name (case-insensitive), or "" if not found.
// Falls back to a forced refresh once if not found, in case the project was just created.
func (c *ProjectCache) IDByName(ctx context.Context, name string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensure(ctx); err != nil {
		return "", err
	}
	if id, ok := c.byNameLC[strings.ToLower(name)]; ok {
		return id, nil
	}
	// miss: force refresh once
	if err := c.refresh(ctx); err != nil {
		return "", err
	}
	return c.byNameLC[strings.ToLower(name)], nil
}

func (c *ProjectCache) All(ctx context.Context) ([]Project, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensure(ctx); err != nil {
		return nil, err
	}
	out := make([]Project, len(c.projects))
	copy(out, c.projects)
	return out, nil
}

func (c *ProjectCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byNameLC = nil
}
