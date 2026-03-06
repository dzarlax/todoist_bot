# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Todoist Telegram Bot is a Node.js service that integrates Telegram messages with Todoist tasks. It runs in a Docker container and polls the Telegram Bot API for messages, then creates tasks in Todoist via their REST API.

### Environment Variables Configuration

The bot uses environment variables for configuration, parsed in `src/app/config/index.js`:

- `TELEGRAM_TOKEN`: Telegram bot token from BotFather (required)
- `TODOIST_TOKEN`: Todoist API token (required)
- `BOT_ADMIN`: Telegram username of the bot administrator (e.g. `@dzarlax`). Admin has access to management commands (`/list`).
- `TIMER`: Seconds to wait before concatenating multiple messages into a single task (default: 5)
- `AUTO_ADD_DUE_DATE`: Set to `"true"` to automatically set task due date to today
- `PROJECT_USERS_*`: Mapping of projects to users. Format: `PROJECT_USERS_PROJECTNAME: "@user1,@user2"`
  - The project name (PROJECTNAME) must match the Todoist project name exactly
  - For projects with spaces or special characters, quote the value in docker-compose.yml
  - Users specified as `@username` are automatically whitelisted
  - Full names (without `@`) are supported for project mapping but not for whitelisting
- `ALLOWED_USERNAMES`: Extra whitelisted usernames not in any project mapping (comma-separated, `@` optional)

### Architecture

The bot has been refactored into a modular structure:

- **Entry Point**: `src/app/todoist_bot.js` - Initializes the bot, attaches handlers, defines commands including admin-only `/list`.
- **Config**: `src/app/config/index.js` - Environment variable parsing, validation, and auto-whitelist derivation.
- **API Client**: `src/app/api/todoist.js` - `TodoistClient` class with project caching and `getTasks()`.
- **Handlers**: `src/app/handlers/telegram.js` - Message/media processing with whitelist check, priority and label parsing.
- **Utils**: `src/app/utils/formatter.js` - Markdown formatting, user-to-project mapping, and `parseTaskMeta`.

### Message Processing Flow

1. **Message Reception**: Bot polls Telegram API using `node-telegram-bot-api` with polling enabled
2. **Sender Identification**: Extracts sender name from forwarded messages or original sender (@username or full name)
3. **Content Formatting**:
   - Text messages: Processes URLs and hyperlinks into Markdown format
   - Media messages: Converts media files to Markdown links `[type](url)` embedded in content
   - Captions are processed with same URL formatting as text
4. **Task Meta Parsing**: Priority (`!p1`–`!p4`) and labels (`#tag`) extracted from first message; markers stripped from title
5. **Message Buffering**: Messages stored in Map with timer, priority, and labels. First message becomes task title, rest become description
6. **Task Creation**: On timer expiry, creates task in Todoist with mapped project, priority, and labels

### Key Logic Details

**Access Control** (`src/app/handlers/telegram.js`, `src/app/todoist_bot.js`):
- Whitelist auto-derived from `@username` entries in `PROJECT_USERS_*` + optional `ALLOWED_USERNAMES`
- Empty whitelist = allow all users
- `BOT_ADMIN` username gets access to management commands (`/list`, future: `/done`, `/del`)
- `isAdmin(msg)` helper defined in `todoist_bot.js`

**Message Buffering** (`src/app/handlers/telegram.js`):
- Messages are buffered per chat ID with `{ messages, priority, labels, timer }`
- Timer resets on each new message within `TIMER` seconds
- First message in buffer becomes task content (title); priority/labels parsed from it
- Subsequent messages joined with newlines as task description

**Task Meta Parsing** (`src/app/utils/formatter.js:parseTaskMeta`):
- `!p1`–`!p4` → sets task priority (default: 1)
- `#word` → adds label (multiple supported); does not match `#` inside URLs
- Returns `{ priority, labels, cleanText }` with markers stripped

**Todoist API** (`src/app/api/todoist.js`):
- Base URL: `https://api.todoist.com/api/v1` (REST API v1)
- Response format: `{ results: [...] }` — always unwrap before use
- Project list cached for 10 minutes to reduce API calls
- `getTasks(projectId?)` — fetches active tasks, optionally filtered by project

**Media Handling** (`src/app/handlers/telegram.js:getMediaLink`):
- Supports 10+ media types: photos, videos, documents, audio, voice, animations, stickers, locations, contacts, polls, venues
- Each converted to Markdown link format: `[type](url)` embedded directly in message
- Special handling for locations (Google Maps link), contacts (tel: link), polls (text description)

**URL Formatting** (`src/app/utils/formatter.js:formatTextWithLinks`):
- First processes Telegram's `text_link` entities (formatted hyperlinks with custom text)
- Then detects plain URLs and converts to Markdown format
- Preserves existing Markdown links to avoid double-formatting

### Language Conventions

- **Russian language** used in: code comments, console logs, bot messages to users
- Variable names and function names in English
- Error messages to users are in Russian

## Development Commands

### NPM Scripts (Run inside `src/`)
```bash
npm start      # Start the bot (requires .env file with tokens)
npm run dev    # Start with watch mode for development
npm test       # Run Jest unit tests
npm run lint   # Run ESLint
```

For local development, create a `.env` file in `src/` with the required environment variables.

### Docker
```bash
# Copy and configure docker-compose
cp docker-compose_example.yml docker-compose.yml
# Edit docker-compose.yml with your tokens and settings

# Build and start
docker-compose up -d --build

# View logs
docker-compose logs -f todoist_bot

# Stop the bot
docker-compose down

# Rebuild after code changes
docker-compose up -d --build
```

## Key Implementation Details

- **Edited messages** are ignored (logged but not processed)
- **Commands** (messages starting with `/`) are handled separately: `/start`, `/help`, `/status`, `/list` (admin only)
- **Media groups** (albums) are processed together via `bot.on('mediagroup', ...)`
- **Project fallback**: If no project found for user, defaults to "Inbox" and notifies user
- **Error handling**: All async functions use try-catch; bot continues running after individual message errors
