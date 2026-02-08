# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Todoist Telegram Bot is a Node.js service that integrates Telegram messages with Todoist tasks. It runs in a Docker container and polls the Telegram Bot API for messages, then creates tasks in Todoist via their REST API.

### Environment Variables Configuration

The bot uses environment variables for configuration, parsed in `src/app/config/index.js`:

- `TELEGRAM_TOKEN`: Telegram bot token from BotFather (required)
- `TODOIST_TOKEN`: Todoist API token (required)
- `TIMER`: Seconds to wait before concatenating multiple messages into a single task (default: 5)
- `AUTO_ADD_DUE_DATE`: Set to `"true"` to automatically set task due date to today
- `PROJECT_USERS_*`: Mapping of projects to users. Format: `PROJECT_USERS_PROJECTNAME: "@user1,@user2"`
  - The project name (PROJECTNAME) must match the Todoist project name exactly
  - For projects with spaces or special characters, quote the value in docker-compose.yml
  - Users can be specified as `@username` or full names

### Architecture

The bot has been refactored into a modular structure:

- **Entry Point**: `src/app/todoist_bot.js` - Initializes the bot and attaches handlers.
- **Config**: `src/app/config/index.js` - Environment variable parsing and validation.
- **API Client**: `src/app/api/todoist.js` - `TodoistClient` class with built-in project caching.
- **Handlers**: `src/app/handlers/telegram.js` - Logic for processing messages and media groups.
- **Utils**: `src/app/utils/formatter.js` - Markdown formatting and user-to-project mapping logic.

### Message Processing Flow

1. **Message Reception**: Bot polls Telegram API using `node-telegram-bot-api` with polling enabled
2. **Sender Identification**: Extracts sender name from forwarded messages or original sender (@username or full name)
3. **Content Formatting**:
   - Text messages: Processes URLs and hyperlinks into Markdown format
   - Media messages: Converts media files to Markdown links `[type](url)` embedded in content
   - Captions are processed with same URL formatting as text
4. **Message Buffering**: Messages stored in Map with timer. First message becomes task title, rest become description
5. **Task Creation**: On timer expiry, creates task in Todoist with mapped project

### Key Logic Details

**Message Buffering** (`src/app/handlers/telegram.js`):
- Messages are buffered per chat ID
- Timer resets on each new message within `TIMER` seconds
- First message in buffer becomes task content (title)
- Subsequent messages joined with newlines as task description

**Project Caching** (`src/app/api/todoist.js`):
- Caches Todoist project list for 10 minutes to reduce API calls
- Cache key: `projects_cache`
- Reduces latency and avoids rate limits

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
- **Commands** (messages starting with `/`) are handled separately: `/start`, `/help`, `/status`
- **Media groups** (albums) are processed together via `bot.on('mediagroup', ...)`
- **Project fallback**: If no project found for user, defaults to "Inbox" and notifies user
- **Error handling**: All async functions use try-catch; bot continues running after individual message errors
