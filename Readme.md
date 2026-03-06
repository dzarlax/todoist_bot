# Todoist Telegram Bot

## Table of Contents
- [Description](#description)
- [Key Features](#key-features)
- [Configuration](#configuration)
- [Commands](#commands)
- [Technologies](#technologies)
- [Getting Started](#getting-started)
- [Описание](#описание)
- [Основные функции](#основные-функции)
- [Конфигурация](#конфигурация)
- [Команды](#команды)
- [Технологии](#технологии)
- [Начало работы](#начало-работы)

---

## Description

The Todoist Telegram Bot is an automated assistant that integrates your Todoist account with Telegram. Forward messages or write text to the bot — it will create tasks in the right Todoist project automatically. Messages sent within a timer interval are merged into a single task.

## Key Features

- **Automatic task creation**: Text messages and forwards are added as Todoist tasks. Messages within the timer window are merged into one task.
- **Priority via `!p1`–`!p4`**: Add `!p2` to your message to set the task priority. The marker is removed from the task text.
- **Labels via `#tag`**: Add `#work` or `#urgent` to tag the task. Multiple tags are supported.
- **Media support**: Photos, videos, documents, audio, voice messages, stickers, locations, and more — all converted to Markdown links in the task.
- **URL & hyperlink detection**: Plain URLs and Telegram formatted hyperlinks are preserved as clickable Markdown links in Todoist.
- **Project mapping by username**: Each Telegram user is mapped to a Todoist project via `PROJECT_USERS_*` env vars.
- **Auto whitelist**: Users defined in `PROJECT_USERS_*` are automatically whitelisted. No extra configuration needed.
- **Bot admin role**: One user can be designated as admin via `BOT_ADMIN` — only they can run management commands like `/list`.
- **Optional due date**: Set `AUTO_ADD_DUE_DATE=true` to automatically assign today's date to every new task.
- **Docker-ready**: Simple setup and deployment via Docker Compose.

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `TELEGRAM_TOKEN` | yes | — | Telegram bot token from @BotFather |
| `TODOIST_TOKEN` | yes | — | Todoist API token from Settings → Integrations |
| `BOT_ADMIN` | — | — | Admin Telegram username (e.g. `@dzarlax`). Admin-only commands available. |
| `TIMER` | — | `5` | Seconds to wait before flushing buffered messages into one task |
| `AUTO_ADD_DUE_DATE` | — | `false` | Set `true` to automatically set task due date to today |
| `PROJECT_USERS_<NAME>` | — | — | Map project `<NAME>` to users, e.g. `PROJECT_USERS_Work: "@alice,@bob"` |
| `ALLOWED_USERNAMES` | — | — | Extra allowed usernames not in any project mapping (comma-separated) |
| `MAX_RETRIES` | — | `3` | Number of retries for failed API calls |
| `RETRY_DELAY` | — | `1000` | Initial retry delay in milliseconds |
| `LOG_LEVEL` | — | `info` | Logging level: `error`, `warn`, `info`, `debug` |
| `PORT` | — | `3000` | Health check server port |
| `NODE_ENV` | — | `development` | Set to `production` in Docker |

## Commands

| Command | Who | Description |
|---|---|---|
| `/start` | Everyone | Welcome message |
| `/help` | Everyone | List available commands |
| `/status` | Everyone | Show bot status and your role |
| `/list [project]` | Admin only | Show active tasks. Optionally filter by project name. |

### Task syntax

```
Buy groceries !p2 #personal #home
```
- `!p2` — sets priority 2 (1=normal, 4=urgent)
- `#personal #home` — assigns labels

Both markers are stripped from the final task title.

## Technologies

Node.js · `node-telegram-bot-api` · `axios` · Todoist REST API v1 · Docker

## Getting Started

### Prerequisites

- [Telegram](https://telegram.org/) account and a bot token from @BotFather
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
  PROJECT_USERS_Work: "@alice,@bob"
  PROJECT_USERS_Family: "@wife"
  TELEGRAM_TOKEN: your_botfather_token
  TODOIST_TOKEN: your_todoist_api_token
  BOT_ADMIN: "@yourusername"
  TIMER: 5
  AUTO_ADD_DUE_DATE: "true"
```

### Launch

```bash
docker-compose up -d
docker-compose logs -f todoist_bot
```

### Stop

```bash
docker-compose down
```

---

# Todoist Telegram Bot

## Описание

Todoist Telegram Bot — автоматизированный помощник для интеграции Todoist с Telegram. Перешлите боту сообщение или напишите текст — он создаст задачу в нужном проекте Todoist. Сообщения в пределах таймера объединяются в одну задачу.

## Основные функции

- **Автоматическое создание задач**: Текстовые сообщения и пересылки добавляются как задачи Todoist. Сообщения в пределах таймера склеиваются в одну задачу.
- **Приоритет через `!p1`–`!p4`**: Добавьте `!p2` в сообщение — задача получит приоритет 2. Маркер удаляется из текста.
- **Ярлыки через `#tag`**: Пишите `#work` или `#срочно` — задача получит соответствующие ярлыки. Поддерживается несколько ярлыков.
- **Поддержка медиафайлов**: Фото, видео, документы, аудио, голосовые, стикеры, локации — всё конвертируется в Markdown-ссылки в задаче.
- **Распознавание URL и гиперссылок**: Обычные и форматированные ссылки из Telegram сохраняются как кликабельные ссылки в Todoist.
- **Маппинг проектов по пользователям**: Каждый пользователь Telegram привязывается к проекту Todoist через переменные `PROJECT_USERS_*`.
- **Автоматический вайтлист**: Пользователи из `PROJECT_USERS_*` вайтлистятся автоматически.
- **Роль администратора**: Один пользователь назначается админом через `BOT_ADMIN` — только он имеет доступ к командам управления задачами.
- **Автоматическая дата выполнения**: `AUTO_ADD_DUE_DATE=true` — каждая новая задача получает срок на сегодня.
- **Docker**: Простая настройка и запуск через Docker Compose.

## Конфигурация

| Переменная | Обязательная | По умолчанию | Описание |
|---|---|---|---|
| `TELEGRAM_TOKEN` | да | — | Токен бота от @BotFather |
| `TODOIST_TOKEN` | да | — | API токен Todoist из Настройки → Интеграции |
| `BOT_ADMIN` | — | — | Telegram username администратора (например, `@dzarlax`) |
| `TIMER` | — | `5` | Секунды ожидания перед созданием задачи из буфера |
| `AUTO_ADD_DUE_DATE` | — | `false` | `true` — автоматически ставить срок на сегодня |
| `PROJECT_USERS_<NAME>` | — | — | Маппинг проекта `<NAME>` на пользователей, например `PROJECT_USERS_Work: "@alice,@bob"` |
| `ALLOWED_USERNAMES` | — | — | Дополнительные разрешённые пользователи вне проектов |
| `MAX_RETRIES` | — | `3` | Количество повторных попыток при ошибках API |
| `RETRY_DELAY` | — | `1000` | Начальная задержка повтора в миллисекундах |
| `LOG_LEVEL` | — | `info` | Уровень логирования: `error`, `warn`, `info`, `debug` |
| `PORT` | — | `3000` | Порт health check сервера |
| `NODE_ENV` | — | `development` | Установите `production` в Docker |

## Команды

| Команда | Кто | Описание |
|---|---|---|
| `/start` | Все | Приветственное сообщение |
| `/help` | Все | Список доступных команд |
| `/status` | Все | Статус бота и ваша роль |
| `/list [проект]` | Только админ | Список активных задач. Можно фильтровать по проекту. |

### Синтаксис задач

```
Купить продукты !p2 #личное #дом
```
- `!p2` — приоритет 2 (1=обычный, 4=срочный)
- `#личное #дом` — ярлыки

Оба маркера удаляются из итогового названия задачи.

## Технологии

Node.js · `node-telegram-bot-api` · `axios` · Todoist REST API v1 · Docker

## Начало работы

### Предварительные требования

- Аккаунт [Telegram](https://telegram.org/) и токен бота от @BotFather
- Аккаунт [Todoist](https://todoist.com/) и API токен (Настройки → Интеграции → Разработчик)
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
  PROJECT_USERS_Work: "@alice,@bob"
  PROJECT_USERS_Family: "@masik904"
  TELEGRAM_TOKEN: ваш_токен_от_botfather
  TODOIST_TOKEN: ваш_todoist_api_токен
  BOT_ADMIN: "@ваш_username"
  TIMER: 5
  AUTO_ADD_DUE_DATE: "true"
```

### Запуск

```bash
docker-compose up -d
docker-compose logs -f todoist_bot
```

### Остановка

```bash
docker-compose down
```
