# Todoist Telegram Bot + MCP Server

Go service that (a) relays Telegram messages into Todoist tasks and (b) exposes a Todoist MCP server so other AI agents can read/write your Todoist over HTTP. One binary, one process, one Docker image.

## Features

- **Telegram → Todoist:** text, forwards, media (photos/videos/docs/audio/voice/stickers/locations/contacts/polls/venues), albums.
- **Task syntax in messages:** `!p1`..`!p4` for priority, `#tag` for labels. Markers are stripped from the title.
- **Message buffering:** messages within `TIMER` seconds merge into a single task (first line = title, rest = description).
- **Project routing:** Telegram user → Todoist project via `PROJECT_USERS_*` env vars. Fallback to Inbox.
- **URL rendering:** bare URLs and Telegram `text_link` entities become Markdown links in Todoist.
- **Admin-only management:** `/list [project]` via `BOT_ADMIN`.
- **Whitelist:** auto-derived from `@username` entries in project mappings; `ALLOWED_USERNAMES` for extras.
- **MCP server at `/todoist`:** 7 tools (get_projects, get_labels, get_tasks, create_task, update_task, delete_task, complete_task). Protected by `X-API-Key`.

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `TELEGRAM_TOKEN` | yes | — | Bot token from @BotFather |
| `TODOIST_TOKEN` | yes | — | Todoist API token (Settings → Integrations → Developer) |
| `API_KEY` | recommended | — | Required header value for MCP `/todoist` calls. Empty = no auth. |
| `PORT` | — | `3000` | HTTP port |
| `BOT_ADMIN` | — | — | Admin Telegram username (e.g. `@dzarlax`) |
| `TIMER` | — | `5` | Seconds to buffer messages before creating a task |
| `AUTO_ADD_DUE_DATE` | — | `false` | `true` → every task gets today's date as due |
| `PROJECT_USERS_<NAME>` | — | — | `PROJECT_USERS_Work: "@alice,@bob"` |
| `ALLOWED_USERNAMES` | — | — | Extra whitelisted usernames, comma-separated |

## Commands

| Command | Who | Description |
|---|---|---|
| `/start` | Everyone | Welcome message |
| `/help` | Everyone | List commands |
| `/status` | Everyone | Show bot status and your role |
| `/list [project]` | Admin only | List up to 20 active tasks, optionally filtered by project name |

### Task syntax

```
Buy groceries !p2 #personal #home
```
`!p2` sets priority 2, `#personal #home` adds labels. Both markers are stripped from the final task title.

## MCP server usage

The MCP endpoint is at `POST http://<host>:<port>/todoist` using the Streamable HTTP transport from `mark3labs/mcp-go`. Provide `X-API-Key: <API_KEY>`.

Example `mcp.json` entry (Claude Desktop / personal-assistant):

```json
{
  "mcpServers": {
    "todoist": {
      "url": "http://todoist-bot:3000/todoist",
      "headers": { "X-API-Key": "your-api-key" }
    }
  }
}
```

## Running

### Docker

```bash
cp docker-compose_example.yml docker-compose.yml
# edit tokens + API_KEY + PROJECT_USERS_*
docker compose up -d
docker compose logs -f todoist-bot
```

### Locally

```bash
# put tokens in .env (godotenv.Load() is called at startup)
go run ./cmd/todoist-bot
```

## Development

```bash
go test ./...
go vet ./...
go build ./...
```

Stack: Go 1.24 · `go-telegram-bot-api/v5` · `mark3labs/mcp-go` · `go-chi/chi/v5` · Todoist REST API v1.
