# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Todoist Telegram Bot is a Go service that does two things in one process:

1. **Telegram → Todoist bot.** Long-polls Telegram, buffers incoming messages per chat, and creates a Todoist task when the debounce timer fires.
2. **Todoist MCP server.** Exposes Todoist CRUD as MCP tools over HTTP at `/todoist` (Streamable HTTP transport, X-API-Key auth) so other services (e.g. `personal-assistant`) can call Todoist through this bot.

Originally a Node.js bot, rewritten in Go (2026-04) with the MCP server merged in from the `personal_memory` project. Docker image: `ghcr.io/dzarlax/todoist-bot`.

## Environment Variables

Parsed in `internal/config/config.go`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `TELEGRAM_TOKEN` | yes | — | Telegram bot token from @BotFather |
| `TODOIST_TOKEN` | yes | — | Todoist API token |
| `API_KEY` | — | — | X-API-Key value required for MCP `/todoist` calls. Empty = no auth (dev only). |
| `PORT` | — | `3000` | HTTP port (`/health`, `/todoist`) |
| `BOT_ADMIN` | — | — | Admin Telegram username (with or without `@`). Gates `/list`. |
| `TIMER` | — | `5` | Seconds to buffer messages before creating a task |
| `AUTO_ADD_DUE_DATE` | — | `false` | `true` → every task gets today's date as due |
| `PROJECT_USERS_<NAME>` | — | — | Map project `<NAME>` to users, e.g. `PROJECT_USERS_Work: "@alice,@bob"` |
| `ALLOWED_USERNAMES` | — | — | Extra whitelisted usernames (comma-separated, `@` optional) |

**Access control:** `@username` entries in `PROJECT_USERS_*` are auto-added to the whitelist. `ALLOWED_USERNAMES` adds extras. Empty whitelist = allow all (legacy behaviour).

## Architecture

```
cmd/todoist-bot/main.go     # composition root: config → clients → bot + HTTP server
internal/
  config/                   # env parsing, whitelist derivation, IsAllowed/IsAdmin
  todoist/
    client.go               # HTTP client for Todoist REST API v1 (handles {results:[...]} wrapper)
    server.go               # mcp-go MCPServer wiring 7 tools
    cache.go                # 10-min TTL project name→id cache (bot side only)
  telegram/
    bot.go                  # polling loop, Bot type, graceful shutdown
    handlers.go             # command routing, message & media-group task pipelines
    buffer.go               # per-chat taskBuffer + per-groupID mediaGroupBuffer
    media.go                # Telegram media → `[type](url)` markdown rendering
  formatter/                # ParseTaskMeta, FormatTextWithLinks, FindProjectNameForUser
  middleware/               # APIKeyAuth for /todoist
```

**Process model:** one binary, one process. `cmd/todoist-bot/main.go` starts the Telegram polling loop and an HTTP server (chi router: `/health` unauthenticated, `/todoist` with X-API-Key auth) in separate goroutines. SIGINT/SIGTERM triggers graceful shutdown of both.

## Message Processing Flow

1. **Reception:** `tgbotapi.BotAPI.GetUpdatesChan` long-polls. `EditDate != 0` messages are dropped.
2. **Access check:** `cfg.IsAllowed(username)`.
3. **Commands vs data:** `msg.IsCommand()` → `handleCommand`; `MediaGroupID != ""` → `enqueueMediaGroup`; otherwise `handleMessage`.
4. **Sender resolution:** `ForwardFrom` > `ForwardFromChat.Title` > `From.UserName`/full name.
5. **Content rendering:** Text messages go through `formatter.FormatTextWithLinks`. Media → `getMediaLink` returning `[type](url)` markdown. Captions combine both.
6. **Buffering:** First line sets priority/labels via `ParseTaskMeta`; subsequent lines appended. `time.AfterFunc` debounce timer fires `flushTaskBuffer`.
7. **Task creation:** Project resolved via `ProjectCache.IDByName`. Falls back to "Inbox" with a user notification if no mapping. Posts a Russian confirmation on success.

## MCP Server

Served at `POST /todoist` (and `/todoist/`) using `mcp-go`'s `StreamableHTTPServer`. Protected by `APIKeyAuth(cfg.APIKey)`. Seven tools:

| Tool | Purpose |
|---|---|
| `get_projects` | List all Todoist projects |
| `get_labels` | List all labels |
| `get_tasks` | List tasks (optional `project_id`, `filter`, `limit`) |
| `create_task` | Create a task (`content` required; optional `description`, `project_id`, `due_string`/`due_date`, `priority`, `labels`) |
| `update_task` | Update a task (`task_id` required) |
| `delete_task` | Delete a task |
| `complete_task` | Mark task complete |

History: these tools were first built in `github.com/Dzarlax-AI/personal-memory/internal/todoist`. The `/todoist` route still exists there as a transitional duplicate — consumers will be migrated to this bot's endpoint separately.

## Language Conventions

- Russian for bot messages to users and comments that document user-facing behaviour.
- English for code, identifiers, and logs.

## Development

### Commands (root of repo)

```bash
go run ./cmd/todoist-bot   # requires .env in repo root or exported env
go test ./...              # all unit tests
go vet ./...
go build ./...
```

For local dev put `TELEGRAM_TOKEN`, `TODOIST_TOKEN`, `BOT_ADMIN`, `PROJECT_USERS_*`, `API_KEY` into a `.env` file. `godotenv.Load()` runs at startup.

### Docker

```bash
cp docker-compose_example.yml docker-compose.yml
# edit tokens
docker compose up -d --build
docker compose logs -f todoist-bot
```

### CI

`.github/workflows/docker.yml` runs `go vet` + `go test` on every push and PR, and pushes `ghcr.io/dzarlax/todoist-bot:latest` + `:${sha}` on pushes to `main`.

## Key Implementation Details

- **Todoist API v1** response format is `{"results":[...]}`. `unwrapResults` handles both that and plain-array responses.
- **RE2 has no lookaround.** `ParseTaskMeta` uses `\b` plus `(^|\s)` with `$1` replacement to preserve boundary whitespace.
- **Media groups** are not a built-in concept in `go-telegram-bot-api/v5` — `enqueueMediaGroup` collects parts by `MediaGroupID` and flushes after `mediaGroupFlushDelay` (800ms).
- **Shutdown** flushes pending task buffers best-effort. The MCP server uses `http.Server.Shutdown` with a 10s timeout.
