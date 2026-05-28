# Todoist Telegram Bot + MCP Server

## Table of Contents
- [Description](#description)
- [Key Features](#key-features)
- [Why Go?](#why-go)
- [Configuration](#configuration)
- [Commands](#commands)
- [Message Syntax](#message-syntax)
- [Media Support](#media-support)
- [MCP Server](#mcp-server)
- [Technologies](#technologies)
- [Getting Started](#getting-started)
- [Development](#development)
- [Migrating from the Node.js version](#migrating-from-the-nodejs-version)
- [Описание](#описание)
- [Основные функции](#основные-функции)
- [Почему Go?](#почему-go)
- [Конфигурация](#конфигурация)
- [Команды](#команды)
- [Синтаксис сообщений](#синтаксис-сообщений)
- [Поддержка медиа](#поддержка-медиа)
- [MCP-сервер](#mcp-сервер)
- [Технологии](#технологии-ru)
- [Начало работы](#начало-работы)
- [Разработка](#разработка)
- [Миграция с Node.js-версии](#миграция-с-nodejs-версии)

---

## Description

Go service that does two things in one process, one container, one binary:

1. **Telegram → Todoist bot.** Forward messages or write text to the bot — it creates tasks in Todoist in the right project automatically. Messages arriving within the timer window merge into a single task.
2. **Todoist MCP server.** Exposes full CRUD over HTTP at `/todoist` so other AI agents (e.g. a personal assistant) can read and write your Todoist through standard MCP tools.

Originally a Node.js bot; rewritten in Go (2026-04) with the MCP server merged in from the `personal_memory` project — so everything Todoist-related now lives in one place.

## Key Features

- **Automatic task creation**: Text and forwarded messages become Todoist tasks. Messages within the timer window are merged into one task (first line = title, rest = description).
- **Priority via `!p1`–`!p4`**: Add `!p2` to your message — the task gets priority 2. Marker is stripped from the final title.
- **Labels via `#tag`**: Write `#work` or `#urgent` — the task gets those labels. Multiple tags supported. Does not match `#` inside URLs.
- **Media support**: Photos, videos, documents, audio, voice, animations, stickers, locations, contacts, polls, venues — all converted to Markdown links embedded in the task.
- **Album / media-group support**: Telegram albums are aggregated and attached as multi-line task content.
- **URL & hyperlink detection**: Plain URLs, `www.` prefixes, and Telegram `text_link` entities become clickable Markdown links in Todoist.
- **Project mapping by username**: Each Telegram user routes to a Todoist project via `PROJECT_USERS_*` env vars. Unknown senders fall back to Inbox with a notification.
- **Auto-whitelist**: Users listed in `PROJECT_USERS_*` are automatically whitelisted — no duplicate config.
- **Bot-admin role**: One user is admin via `BOT_ADMIN`. Only they can run `/list`.
- **Optional due date**: `AUTO_ADD_DUE_DATE=true` → every new task gets today's date.
- **Embedded MCP server**: `/todoist` endpoint with 11 tools, `X-API-Key` protected. Call it from `personal-assistant`, Claude Desktop, or any MCP client.
- **Docker-ready**: Single static binary (~11 MB), alpine image, GitHub Actions builds & pushes to GHCR.

## Why Go?

This repo once held a Node.js bot; the MCP server came from `personal_memory`, which itself was Python before being rewritten. When it came time to unify them, Go was the obvious target:

- **Stack consistency.** Every other Go service in my Personal stack (`personal_memory`, `personal_assistant`, `book`, `calendar-mcp`) already uses the same libraries (`chi`, `mcp-go`, `slog`). Staying on Go means one toolchain, one set of idioms, and code can be copied between projects almost verbatim — which is literally what happened with `internal/todoist`.
- **`mcp-go` was already battle-tested.** The Go port of the MCP SDK was running in `personal_memory` for months. Rewriting the Python MCP server into Go gave `mcp-go` another real-world consumer without rewriting the wire protocol layer.
- **One static binary, no runtime.** `CGO_ENABLED=0 go build` → an ~11 MB binary that runs on `FROM alpine` with nothing but `ca-certificates`. The old Node image dragged in 200 MB of npm deps; a Python equivalent would pull in a similar amount of wheels.
- **Concurrency is free.** Telegram long-polling + HTTP server + debounce timers run as three goroutines with a single `sync.Mutex` around the message buffer. In Python this would mean `asyncio` + `aiohttp` + careful locking; in Node it was callback soup.
- **Strong typing catches bot-API mismatches at compile time.** `tgbotapi.Message.Photo[]` vs `.Video` vs `.Document` — the compiler forces you to handle every field rather than hoping `msg.photo?.[0]?.file_id` doesn't panic at runtime.

## Configuration

Parsed in `internal/config/config.go`. All values come from environment variables (`.env` is auto-loaded via `godotenv`).

| Variable | Required | Default | Description |
|---|---|---|---|
| `TELEGRAM_TOKEN` | yes | — | Bot token from @BotFather |
| `TODOIST_TOKEN` | yes | — | Todoist API token (Settings → Integrations → Developer) |
| `API_KEY` | recommended | — | Required header value (`X-API-Key`) for MCP `/todoist` calls. Empty disables auth — dev only. |
| `PORT` | — | `3000` | HTTP port for `/health` and `/todoist` |
| `BOT_ADMIN` | — | — | Admin Telegram username (with or without `@`). Gates `/list`. |
| `TIMER` | — | `5` | Seconds to buffer messages before creating a task |
| `AUTO_ADD_DUE_DATE` | — | `false` | `true` → every task gets today's date as due |
| `PROJECT_USERS_<NAME>` | — | — | Map project `<NAME>` to users, e.g. `PROJECT_USERS_Work="@alice,@bob"` |
| `ALLOWED_USERNAMES` | — | — | Extra whitelisted usernames, comma-separated, `@` optional |
| `MAX_RETRIES` | — | `3` | Retries for transient Todoist API failures (429, 5xx) |
| `RETRY_DELAY` | — | `1000` | Initial retry backoff in ms (doubles each attempt) |

**Access control notes:**
- `@username` entries in `PROJECT_USERS_*` auto-add to the whitelist.
- Full-name entries (without `@`) are used for routing but not whitelisting.
- Empty whitelist = allow all users (legacy behaviour).

## Commands

| Command | Who | Description |
|---|---|---|
| `/start` | Everyone | Welcome message |
| `/help` | Everyone | List commands (admin sees `/list`) |
| `/status` | Everyone | Bot status + your role + current timer/auto-date settings |
| `/list [project]` | Admin only | Show up to 20 active tasks, optionally filtered by project name |

## Message Syntax

```
Buy groceries !p2 #personal #home
```
- `!p2` → priority 2 (1 = normal, 4 = urgent). Defaults to 1 if absent.
- `#personal #home` → labels. Multiple supported.

Both markers are stripped from the final task title.

Examples of what the bot does with common inputs:

| You send | Task title in Todoist |
|---|---|
| `Call Mom !p3 #family` | `@you: Call Mom` (priority 3, label `family`) |
| Photo with caption `Kitchen remodel !p2` | `@you: Kitchen remodel [фото](https://…)` |
| Forwarded message from `@alice` | `@alice: <original text>` |
| `Read https://example.com` | `@you: Read [https://example.com](https://example.com)` |

## Media Support

All 10+ Telegram attachment types are rendered as clickable Markdown:

| Type | Rendered as |
|---|---|
| Photo | `[фото](https://t.me/file/…)` |
| Video | `[видео (file.mp4)](https://t.me/file/…)` |
| Document | `[документ (report.pdf)](https://…)` |
| Audio | `[аудио (Title - Artist)](https://…)` |
| Voice message | `[голосовое сообщение](https://…)` |
| Animation / GIF | `[анимация](https://…)` |
| Sticker | `[стикер (😀)](https://…)` |
| Location | `[локация](https://www.google.com/maps?q=…)` |
| Contact | `[контакт](tel:+123…) — Name: phone` |
| Poll | Inline question + options |
| Venue | `[место](https://…) — Name, address` |

## MCP Server

The `/todoist` endpoint uses `mcp-go`'s Streamable HTTP transport and is protected by `X-API-Key`.

**11 tools exposed:**

| Tool | Purpose |
|---|---|
| `get_projects` | List all Todoist projects |
| `get_labels` | List all labels |
| `create_label` | Create a personal label (`name` required; optional `color`, `order`, `is_favorite`) |
| `update_label` | Update a personal label (`label_id` required) |
| `delete_label` | Delete a personal label and remove it from tasks |
| `get_tasks` | List tasks (optional `project_id`, `section_id`, `parent_id`, `label`, `filter`, `limit`) |
| `create_task` | Create a task (`content` required; optional `description`, `project_id`, `section_id`, `parent_id`, `due_string`/`due_date`, `priority`, `labels`) |
| `update_task` | Update an existing task (`task_id` required; optional `content`, `description`, `due_string`/`due_date`, `priority`, `labels`) |
| `move_task` | Move a task to another project, section, or parent (`task_id` required) |
| `delete_task` | Delete a task |
| `complete_task` | Mark a task complete |

**Example `mcp.json` entry** for Claude Desktop or `personal-assistant`:

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

`GET /health` returns `{"status":"ok"}` and is unauthenticated — useful for Docker healthchecks and Traefik probes.

## Technologies

Go 1.24 · [`go-telegram-bot-api/v5`](https://github.com/go-telegram-bot-api/telegram-bot-api) · [`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go) · [`go-chi/chi/v5`](https://github.com/go-chi/chi) · [`joho/godotenv`](https://github.com/joho/godotenv) · Todoist REST API v1 · Docker · GitHub Actions → GHCR.

## Getting Started

### Prerequisites
- [Telegram](https://telegram.org/) account and bot token from @BotFather
- [Todoist](https://todoist.com/) account and API token (Settings → Integrations → Developer)
- [Docker](https://www.docker.com/) installed

### Installation

```bash
git clone https://github.com/dzarlax/todoist_bot.git
cd todoist_bot
cp docker-compose_example.yml docker-compose.yml
```

Edit `docker-compose.yml` and fill in your values:

```yaml
environment:
  TELEGRAM_TOKEN: your_botfather_token
  TODOIST_TOKEN: your_todoist_api_token
  API_KEY: pick-a-strong-key
  BOT_ADMIN: "@yourusername"
  PROJECT_USERS_Work: "@alice,@bob"
  PROJECT_USERS_Family: "@wife"
  TIMER: 5
  AUTO_ADD_DUE_DATE: "true"
```

### Launch

```bash
docker compose up -d
docker compose logs -f todoist-bot
```

### Stop

```bash
docker compose down
```

## Development

```bash
go run ./cmd/todoist-bot   # requires .env with TELEGRAM_TOKEN, TODOIST_TOKEN
go test ./...              # unit tests
go vet ./...
go build ./...
```

CI runs `go vet` + `go test` on every push/PR and builds/pushes `ghcr.io/dzarlax/todoist-bot:latest` + `:${sha}` on pushes to `main` (`.github/workflows/docker.yml`).

## Migrating from the Node.js version

If you're currently running the old Node bot (`dzarlax/todoist_bot` on Docker Hub), the upgrade is drop-in. Todoist data is untouched — the bot is stateless, it only talks to Todoist's cloud API.

**Changes you need to know:**
- **Image**: `dzarlax/todoist_bot` (Docker Hub) → `ghcr.io/dzarlax/todoist-bot` (GHCR). Note the **dash** in the new name.
- **Container name**: `todoist` → `todoist-bot` (by convention — you can still set whatever you want).
- **Healthcheck URL**: unchanged — `http://localhost:3000/health`.
- **Tokens**: `TELEGRAM_TOKEN` / `TODOIST_TOKEN` — unchanged, reuse them as-is.
- **Project/user config**: `PROJECT_USERS_*`, `BOT_ADMIN`, `ALLOWED_USERNAMES`, `TIMER`, `AUTO_ADD_DUE_DATE` — **all unchanged**.
- **New**: `API_KEY` — only needed if you plan to call the MCP `/todoist` endpoint from another service. Leave empty/unset for pure bot-mode.
- **Removed (no longer needed)**: `LOG_LEVEL`, `NODE_ENV`, `MAX_RETRIES`/`RETRY_DELAY` (now hardcoded sensible defaults — still overridable).

**Step-by-step:**

1. Stash your old token values. If they're in `docker-compose.yml`, pull them out now — moving them into `.env` is recommended for the new setup.

   ```bash
   grep -E 'TELEGRAM_TOKEN|TODOIST_TOKEN|BOT_ADMIN|PROJECT_USERS|ALLOWED_USERNAMES' docker-compose.yml
   ```

2. Stop and remove the old container. Don't worry — no volumes to lose.

   ```bash
   docker compose down
   ```

3. Pull the latest `docker-compose_example.yml` from this repo (or update your existing `docker-compose.yml` by hand):

   ```bash
   wget https://raw.githubusercontent.com/dzarlax/todoist_bot/main/docker-compose_example.yml -O docker-compose.yml
   ```

4. Create `.env` alongside `docker-compose.yml` with your tokens:

   ```env
   TELEGRAM_TOKEN=<existing value>
   TODOIST_TOKEN=<existing value>
   API_KEY=<new, pick a strong random string>
   ```

5. Edit `docker-compose.yml` to set `BOT_ADMIN`, `PROJECT_USERS_*`, etc. — values carry over 1:1.

6. Start up:

   ```bash
   docker compose pull
   docker compose up -d
   docker compose logs -f todoist-bot
   ```

   You should see `INFO starting todoist-bot` followed by `INFO telegram bot started user=<your_bot_username>` within a second. If the logs show `telegram init failed error="Not Found"`, double-check the Telegram token was copied intact (including the `:` and the secret part).

7. **Optional — wire up the MCP endpoint.** Point any MCP client at `http://<host>:3000/todoist` with `X-API-Key: <your API_KEY>`. If you were previously hitting a Todoist MCP hosted in another service, switch its URL and key here.

**Rollback**, if needed: `docker compose down` and put back the old `image: dzarlax/todoist_bot` line + restart. No data migration to undo.

---

# Todoist Telegram Bot + MCP-сервер

## Описание

Go-сервис, который делает две вещи в одном процессе, одном контейнере, одном бинарнике:

1. **Telegram → Todoist бот.** Пересылаете сообщение или пишете текст — бот создаёт задачу в нужном проекте Todoist. Сообщения в пределах таймера склеиваются в одну задачу.
2. **Todoist MCP-сервер.** Полный CRUD по HTTP на `/todoist`, чтобы другие AI-агенты (например, `personal-assistant`) могли читать и писать ваш Todoist через стандартные MCP-инструменты.

Изначально был Node.js-бот; переписан на Go (2026-04) с интеграцией MCP-сервера из проекта `personal_memory` — теперь всё, что касается Todoist, в одном месте.

## Основные функции

- **Автоматическое создание задач**: Текстовые и пересланные сообщения становятся задачами Todoist. Сообщения в пределах таймера объединяются в одну задачу (первая строка — заголовок, остальные — описание).
- **Приоритет через `!p1`–`!p4`**: Добавьте `!p2` в сообщение — задача получит приоритет 2. Маркер удаляется из итогового заголовка.
- **Ярлыки через `#tag`**: Пишете `#work` или `#срочно` — задача получит эти метки. Несколько поддерживается. Не ловит `#` внутри URL.
- **Поддержка медиа**: Фото, видео, документы, аудио, голосовые, анимации, стикеры, локации, контакты, опросы, места — всё конвертируется в Markdown-ссылки в задаче.
- **Альбомы / медиа-группы**: Альбомы Telegram агрегируются и вкладываются многострочным описанием.
- **Распознавание URL и гиперссылок**: Обычные URL, `www.` префиксы и Telegram `text_link` становятся кликабельными Markdown-ссылками в Todoist.
- **Маппинг проектов по пользователю**: Каждый Telegram-пользователь роутится в Todoist-проект через `PROJECT_USERS_*`. Неизвестные отправители — в Inbox с уведомлением.
- **Автоматический вайтлист**: Пользователи из `PROJECT_USERS_*` автоматически добавляются в whitelist — без дубля конфига.
- **Роль админа**: Один пользователь — админ через `BOT_ADMIN`. Только он может вызывать `/list`.
- **Опциональная дата выполнения**: `AUTO_ADD_DUE_DATE=true` → каждая новая задача получает срок на сегодня.
- **Встроенный MCP-сервер**: Эндпоинт `/todoist` с 11 инструментами, защищён `X-API-Key`. Подключается из `personal-assistant`, Claude Desktop или любого MCP-клиента.
- **Docker-ready**: Один статический бинарник (~11 МБ), alpine-образ, GitHub Actions пушат в GHCR.

## Почему Go?

Раньше тут жил Node.js-бот; MCP-сервер пришёл из `personal_memory`, который сам был на Python до переписывания. Когда встала задача объединить всё это в одном месте, Go был очевидным выбором:

- **Консистентность стека.** Все остальные Go-сервисы в моём Personal-стеке (`personal_memory`, `personal_assistant`, `book`, `calendar-mcp`) уже используют одни и те же библиотеки (`chi`, `mcp-go`, `slog`). Остаться на Go — значит иметь один тулчейн, одни идиомы и возможность буквально копировать код между проектами — что и случилось с `internal/todoist`.
- **`mcp-go` уже обкатан.** Go-порт MCP SDK работал в `personal_memory` несколько месяцев. Переписывая Python-версию MCP-сервера на Go, я получил ещё одного боевого потребителя `mcp-go` без переписывания уровня протокола.
- **Один статический бинарник, без runtime.** `CGO_ENABLED=0 go build` → ~11 МБ бинарник на `FROM alpine` и `ca-certificates`. Старый Node-образ тянул 200 МБ npm-зависимостей; Python-эквивалент притащил бы похожий объём wheels.
- **Конкурентность бесплатная.** Telegram long-polling + HTTP-сервер + debounce-таймеры — три горутины с одним `sync.Mutex` вокруг буфера сообщений. На Python — `asyncio` + `aiohttp` + аккуратная блокировка; на Node — callback-суп.
- **Строгая типизация ловит несовпадения с bot API на компиляции.** `tgbotapi.Message.Photo[]` против `.Video` против `.Document` — компилятор заставляет обработать каждое поле, а не надеяться, что `msg.photo?.[0]?.file_id` не упадёт в рантайме.

## Конфигурация

Парсится в `internal/config/config.go`. Все значения — из переменных окружения (`.env` автозагружается через `godotenv`).

| Переменная | Обяз. | По умолчанию | Описание |
|---|---|---|---|
| `TELEGRAM_TOKEN` | да | — | Токен бота от @BotFather |
| `TODOIST_TOKEN` | да | — | API-токен Todoist (Настройки → Интеграции → Разработчик) |
| `API_KEY` | рекоменд. | — | Значение заголовка `X-API-Key` для MCP-вызовов `/todoist`. Пустой отключает auth — только для dev. |
| `PORT` | — | `3000` | HTTP-порт для `/health` и `/todoist` |
| `BOT_ADMIN` | — | — | Админ Telegram username (с `@` или без). Открывает `/list`. |
| `TIMER` | — | `5` | Секунды буферизации сообщений перед созданием задачи |
| `AUTO_ADD_DUE_DATE` | — | `false` | `true` → каждая задача получает срок на сегодня |
| `PROJECT_USERS_<NAME>` | — | — | Маппинг проекта `<NAME>` на пользователей, например `PROJECT_USERS_Work="@alice,@bob"` |
| `ALLOWED_USERNAMES` | — | — | Дополнительные разрешённые пользователи, через запятую, `@` опционально |
| `MAX_RETRIES` | — | `3` | Повторы при временных ошибках Todoist API (429, 5xx) |
| `RETRY_DELAY` | — | `1000` | Начальный retry backoff в мс (удваивается с каждой попыткой) |

**Про контроль доступа:**
- Записи `@username` в `PROJECT_USERS_*` автоматически добавляются в whitelist.
- Записи с полным именем (без `@`) используются для роутинга, но не для whitelist.
- Пустой whitelist = разрешить всем (legacy поведение).

## Команды

| Команда | Кому | Описание |
|---|---|---|
| `/start` | Всем | Приветственное сообщение |
| `/help` | Всем | Список команд (админ видит `/list`) |
| `/status` | Всем | Статус бота + ваша роль + текущий таймер и авто-дата |
| `/list [проект]` | Только админ | До 20 активных задач, опционально фильтр по проекту |

## Синтаксис сообщений

```
Купить продукты !p2 #личное #дом
```
- `!p2` → приоритет 2 (1 = обычный, 4 = срочный). По умолчанию 1 если нет маркера.
- `#личное #дом` → ярлыки. Несколько поддерживается.

Оба маркера удаляются из итогового заголовка задачи.

Примеры того, что бот делает с типичными вводами:

| Вы шлёте | Заголовок задачи в Todoist |
|---|---|
| `Позвонить маме !p3 #семья` | `@вы: Позвонить маме` (приоритет 3, ярлык `семья`) |
| Фото с подписью `Ремонт кухни !p2` | `@вы: Ремонт кухни [фото](https://…)` |
| Пересланное от `@alice` | `@alice: <оригинальный текст>` |
| `Почитать https://example.com` | `@вы: Почитать [https://example.com](https://example.com)` |

## Поддержка медиа

Все 10+ типов вложений Telegram рендерятся как кликабельный Markdown:

| Тип | Рендерится как |
|---|---|
| Фото | `[фото](https://t.me/file/…)` |
| Видео | `[видео (file.mp4)](https://t.me/file/…)` |
| Документ | `[документ (report.pdf)](https://…)` |
| Аудио | `[аудио (Название - Исполнитель)](https://…)` |
| Голосовое | `[голосовое сообщение](https://…)` |
| Анимация / GIF | `[анимация](https://…)` |
| Стикер | `[стикер (😀)](https://…)` |
| Локация | `[локация](https://www.google.com/maps?q=…)` |
| Контакт | `[контакт](tel:+123…) — Имя: телефон` |
| Опрос | Вопрос + варианты встроенно |
| Место | `[место](https://…) — Название, адрес` |

## MCP-сервер

Эндпоинт `/todoist` использует Streamable HTTP транспорт из `mcp-go` и защищён `X-API-Key`.

**Экспонируются 11 инструментов:**

| Инструмент | Назначение |
|---|---|
| `get_projects` | Список всех проектов Todoist |
| `get_labels` | Список всех меток |
| `create_label` | Создать персональную метку (`name` обязателен; опционально `color`, `order`, `is_favorite`) |
| `update_label` | Обновить персональную метку (`label_id` обязателен) |
| `delete_label` | Удалить персональную метку и снять её с задач |
| `get_tasks` | Список задач (опционально `project_id`, `section_id`, `parent_id`, `label`, `filter`, `limit`) |
| `create_task` | Создать задачу (`content` обязателен; опционально `description`, `project_id`, `section_id`, `parent_id`, `due_string`/`due_date`, `priority`, `labels`) |
| `update_task` | Обновить существующую задачу (`task_id` обязателен; опционально `content`, `description`, `due_string`/`due_date`, `priority`, `labels`) |
| `move_task` | Переместить задачу в другой проект, секцию или под родителя (`task_id` обязателен) |
| `delete_task` | Удалить задачу |
| `complete_task` | Завершить задачу |

**Пример записи в `mcp.json`** для Claude Desktop или `personal-assistant`:

```json
{
  "mcpServers": {
    "todoist": {
      "url": "http://todoist-bot:3000/todoist",
      "headers": { "X-API-Key": "ваш-api-ключ" }
    }
  }
}
```

`GET /health` возвращает `{"status":"ok"}` и не требует аутентификации — удобно для Docker healthcheck'ов и Traefik-проб.

<h2 id="технологии-ru">Технологии</h2>

Go 1.24 · [`go-telegram-bot-api/v5`](https://github.com/go-telegram-bot-api/telegram-bot-api) · [`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go) · [`go-chi/chi/v5`](https://github.com/go-chi/chi) · [`joho/godotenv`](https://github.com/joho/godotenv) · Todoist REST API v1 · Docker · GitHub Actions → GHCR.

## Начало работы

### Предварительные требования
- Аккаунт [Telegram](https://telegram.org/) и токен бота от @BotFather
- Аккаунт [Todoist](https://todoist.com/) и API-токен (Настройки → Интеграции → Разработчик)
- Установленный [Docker](https://www.docker.com/)

### Установка

```bash
git clone https://github.com/dzarlax/todoist_bot.git
cd todoist_bot
cp docker-compose_example.yml docker-compose.yml
```

Откройте `docker-compose.yml` и заполните переменные:

```yaml
environment:
  TELEGRAM_TOKEN: ваш_токен_от_botfather
  TODOIST_TOKEN: ваш_todoist_api_токен
  API_KEY: выберите-сильный-ключ
  BOT_ADMIN: "@ваш_username"
  PROJECT_USERS_Work: "@alice,@bob"
  PROJECT_USERS_Family: "@wife"
  TIMER: 5
  AUTO_ADD_DUE_DATE: "true"
```

### Запуск

```bash
docker compose up -d
docker compose logs -f todoist-bot
```

### Остановка

```bash
docker compose down
```

## Разработка

```bash
go run ./cmd/todoist-bot   # нужен .env с TELEGRAM_TOKEN, TODOIST_TOKEN
go test ./...              # юнит-тесты
go vet ./...
go build ./...
```

CI запускает `go vet` + `go test` на каждый push/PR и собирает/пушит `ghcr.io/dzarlax/todoist-bot:latest` + `:${sha}` при пушах в `main` (`.github/workflows/docker.yml`).

## Миграция с Node.js-версии

Если сейчас у вас запущен старый Node-бот (`dzarlax/todoist_bot` на Docker Hub), обновление — drop-in. Данные Todoist не трогаются — бот stateless, он просто ходит в облачный API Todoist.

**Что изменилось:**
- **Образ**: `dzarlax/todoist_bot` (Docker Hub) → `ghcr.io/dzarlax/todoist-bot` (GHCR). Обратите внимание на **дефис** в новом имени.
- **Имя контейнера**: `todoist` → `todoist-bot` (по соглашению — можно ставить любое).
- **Healthcheck URL**: не изменился — `http://localhost:3000/health`.
- **Токены**: `TELEGRAM_TOKEN` / `TODOIST_TOKEN` — без изменений, переиспользуйте as-is.
- **Конфиг проектов/пользователей**: `PROJECT_USERS_*`, `BOT_ADMIN`, `ALLOWED_USERNAMES`, `TIMER`, `AUTO_ADD_DUE_DATE` — **без изменений**.
- **Новое**: `API_KEY` — нужен только если будете вызывать MCP `/todoist` из другого сервиса. Для чисто бот-режима можно не задавать.
- **Убрано (больше не нужно)**: `LOG_LEVEL`, `NODE_ENV`, `MAX_RETRIES`/`RETRY_DELAY` (теперь sensible defaults зашиты в код, но всё ещё переопределяемые).

**Пошагово:**

1. Запишите куда-нибудь значения старых токенов. Если они прямо в `docker-compose.yml` — вытащите их сейчас, в новом setup'е их рекомендуется держать в `.env`.

   ```bash
   grep -E 'TELEGRAM_TOKEN|TODOIST_TOKEN|BOT_ADMIN|PROJECT_USERS|ALLOWED_USERNAMES' docker-compose.yml
   ```

2. Остановите и удалите старый контейнер. Волноваться не о чем — volumes для потери нет.

   ```bash
   docker compose down
   ```

3. Подтяните свежий `docker-compose_example.yml` из этого репо (или обновите свой вручную):

   ```bash
   wget https://raw.githubusercontent.com/dzarlax/todoist_bot/main/docker-compose_example.yml -O docker-compose.yml
   ```

4. Создайте `.env` рядом с `docker-compose.yml` с токенами:

   ```env
   TELEGRAM_TOKEN=<старое значение>
   TODOIST_TOKEN=<старое значение>
   API_KEY=<новое, сгенерируйте сильную случайную строку>
   ```

5. Отредактируйте `docker-compose.yml`: проставьте `BOT_ADMIN`, `PROJECT_USERS_*` и т.д. — значения переносятся 1:1.

6. Запускайте:

   ```bash
   docker compose pull
   docker compose up -d
   docker compose logs -f todoist-bot
   ```

   В логах в течение секунды должны появиться `INFO starting todoist-bot` и `INFO telegram bot started user=<имя_вашего_бота>`. Если видите `telegram init failed error="Not Found"` — проверьте, что Telegram-токен скопирован целиком (включая `:` и секретную часть).

7. **Опционально — подключите MCP-эндпоинт.** Направьте любой MCP-клиент на `http://<host>:3000/todoist` с `X-API-Key: <ваш API_KEY>`. Если раньше ходили в Todoist MCP, хостящийся в другом сервисе, — смените URL и ключ здесь.

**Откатиться**, если понадобится: `docker compose down` и вернуть старую строку `image: dzarlax/todoist_bot` + перезапуск. Никакой миграции данных откатывать не нужно.
