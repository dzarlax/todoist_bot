package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_RequiresTokens(t *testing.T) {
	clearEnv(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected error when tokens missing")
	}
}

func TestLoad_AutoWhitelistFromProjectUsers(t *testing.T) {
	clearEnv(t)
	t.Setenv("TELEGRAM_TOKEN", "t")
	t.Setenv("TODOIST_TOKEN", "k")
	t.Setenv("PROJECT_USERS_Family", "@alice,@bob")
	t.Setenv("PROJECT_USERS_Work", "@alice,Carl Stein")
	t.Setenv("ALLOWED_USERNAMES", "@dan, eve")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	wantAllowed := []string{"alice", "bob", "dan", "eve"}
	for _, u := range wantAllowed {
		if !cfg.IsAllowed(u) {
			t.Errorf("expected %q to be allowed", u)
		}
	}
	if cfg.IsAllowed("mallory") {
		t.Error("mallory should not be allowed")
	}
	// Bare full-name entry does NOT get whitelisted by itself.
	if cfg.IsAllowed("Carl Stein") {
		t.Error("full-name entry should not be auto-whitelisted")
	}

	if got := cfg.ProjectUsers["Family"]; len(got) != 2 {
		t.Errorf("Family users: got %v", got)
	}
}

func TestLoad_EmptyWhitelistAllowsAll(t *testing.T) {
	clearEnv(t)
	t.Setenv("TELEGRAM_TOKEN", "t")
	t.Setenv("TODOIST_TOKEN", "k")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsAllowed("anyone") {
		t.Error("empty whitelist should allow all users")
	}
	if !cfg.IsAllowed("") {
		t.Error("empty whitelist should allow empty username too")
	}
}

func TestLoad_BotAdminStripsAt(t *testing.T) {
	clearEnv(t)
	t.Setenv("TELEGRAM_TOKEN", "t")
	t.Setenv("TODOIST_TOKEN", "k")
	t.Setenv("BOT_ADMIN", "@dzarlax")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BotAdmin != "dzarlax" {
		t.Errorf("bot admin: got %q, want %q", cfg.BotAdmin, "dzarlax")
	}
	if !cfg.IsAdmin("dzarlax") {
		t.Error("IsAdmin should accept bare username")
	}
	if cfg.IsAdmin("someone") {
		t.Error("IsAdmin should reject non-admin")
	}
}

func TestLoad_TimerAndDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("TELEGRAM_TOKEN", "t")
	t.Setenv("TODOIST_TOKEN", "k")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timer != 5 {
		t.Errorf("default timer: got %d, want 5", cfg.Timer)
	}
	if cfg.AutoAddDueDate {
		t.Error("AutoAddDueDate should default false")
	}
	if cfg.Port != "3000" {
		t.Errorf("default port: got %q, want 3000", cfg.Port)
	}
}

// clearEnv removes config-related vars so each test starts from a clean slate.
// It uses os.Unsetenv (true unset) with manual cleanup to restore originals.
func clearEnv(t *testing.T) {
	t.Helper()
	prefixes := []string{"TELEGRAM_TOKEN", "TODOIST_TOKEN", "API_KEY", "PORT",
		"TIMER", "AUTO_ADD_DUE_DATE", "BOT_ADMIN", "ALLOWED_USERNAMES",
		"MAX_RETRIES", "RETRY_DELAY"}
	toRestore := map[string]string{}
	for _, env := range os.Environ() {
		eq := strings.IndexByte(env, '=')
		if eq < 0 {
			continue
		}
		key := env[:eq]
		if strings.HasPrefix(key, "PROJECT_USERS_") {
			toRestore[key] = env[eq+1:]
			_ = os.Unsetenv(key)
			continue
		}
		for _, p := range prefixes {
			if key == p {
				toRestore[key] = env[eq+1:]
				_ = os.Unsetenv(key)
				break
			}
		}
	}
	t.Cleanup(func() {
		for k, v := range toRestore {
			_ = os.Setenv(k, v)
		}
	})
}
