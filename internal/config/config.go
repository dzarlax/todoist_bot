package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TelegramToken  string
	TodoistToken   string
	APIKey         string
	Port           string
	Timer          int
	AutoAddDueDate bool
	MaxRetries     int
	RetryDelayMs   int

	BotAdmin string // without leading '@'

	// ProjectUsers: projectName → list of user identifiers ("@username" or "First Last")
	ProjectUsers map[string][]string

	// AllowedUsernames: usernames (without '@') that are permitted to talk to the bot.
	// Empty set means "allow everyone" (legacy behaviour).
	AllowedUsernames map[string]struct{}
}

// Load reads configuration from environment variables. Call after godotenv.Load() if needed.
// Returns (cfg, error). On missing required vars the error lists them.
func Load() (*Config, error) {
	cfg := &Config{
		TelegramToken:    os.Getenv("TELEGRAM_TOKEN"),
		TodoistToken:     os.Getenv("TODOIST_TOKEN"),
		APIKey:           os.Getenv("API_KEY"),
		Port:             envDefault("PORT", "3000"),
		Timer:            envInt("TIMER", 5),
		AutoAddDueDate:   os.Getenv("AUTO_ADD_DUE_DATE") == "true",
		MaxRetries:       envInt("MAX_RETRIES", 3),
		RetryDelayMs:     envInt("RETRY_DELAY", 1000),
		BotAdmin:         strings.TrimPrefix(strings.TrimSpace(os.Getenv("BOT_ADMIN")), "@"),
		ProjectUsers:     map[string][]string{},
		AllowedUsernames: map[string]struct{}{},
	}

	// Parse PROJECT_USERS_<NAME>=user1,user2,...
	for _, kv := range os.Environ() {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		key, val := kv[:eq], kv[eq+1:]
		if !strings.HasPrefix(key, "PROJECT_USERS_") {
			continue
		}
		projectName := strings.TrimPrefix(key, "PROJECT_USERS_")
		var users []string
		for _, u := range strings.Split(val, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				users = append(users, u)
			}
		}
		cfg.ProjectUsers[projectName] = users
	}

	// Auto-whitelist: @username entries from PROJECT_USERS_*
	for _, users := range cfg.ProjectUsers {
		for _, u := range users {
			if strings.HasPrefix(u, "@") {
				cfg.AllowedUsernames[strings.TrimPrefix(u, "@")] = struct{}{}
			}
		}
	}
	// Extras from ALLOWED_USERNAMES
	if extra := os.Getenv("ALLOWED_USERNAMES"); extra != "" {
		for _, u := range strings.Split(extra, ",") {
			u = strings.TrimPrefix(strings.TrimSpace(u), "@")
			if u != "" {
				cfg.AllowedUsernames[u] = struct{}{}
			}
		}
	}

	var missing []string
	if cfg.TelegramToken == "" {
		missing = append(missing, "TELEGRAM_TOKEN")
	}
	if cfg.TodoistToken == "" {
		missing = append(missing, "TODOIST_TOKEN")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// IsAllowed reports whether the given Telegram username may use the bot.
// Empty username is never allowed when a whitelist is set.
func (c *Config) IsAllowed(username string) bool {
	if len(c.AllowedUsernames) == 0 {
		return true
	}
	if username == "" {
		return false
	}
	_, ok := c.AllowedUsernames[username]
	return ok
}

// IsAdmin reports whether the given Telegram username is the bot admin.
func (c *Config) IsAdmin(username string) bool {
	return c.BotAdmin != "" && username != "" && c.BotAdmin == username
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
