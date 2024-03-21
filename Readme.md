# Todoist Telegram Bot

## Table of Contents
- [Description (EN)](#description)
- [Key Features (EN)](#key-features)
- [Technologies (EN)](#technologies)
- [Getting Started (EN)](#getting-started)
  - [Prerequisites (EN)](#prerequisites)
  - [Installation and Launch (EN)](#installation-and-launch)
    - [Cloning the Repository (EN)](#cloning-the-repository)
    - [Setting up docker-compose (EN)](#setting-up-docker-compose)
    - [Launching the Bot (EN)](#launching-the-bot)
    - [Checking the Bot's Operation (EN)](#checking-the-bots-operation)
    - [Stopping the Bot (EN)](#stopping-the-bot)
- [Описание (RU)](#описание)
- [Основные функции (RU)](#основные-функции)
- [Технологии (RU)](#технологии)
- [Начало работы (RU)](#начало-работы)
  - [Предварительные требования (RU)](#предварительные-требования)
  - [Установка и запуск (RU)](#установка-и-запуск)
    - [Клонирование репозитория (RU)](#клонирование-репозитория)
    - [Настройка docker-compose (RU)](#настройка-docker-compose)
    - [Запуск бота (RU)](#запуск-бота)
    - [Проверка работы бота (RU)](#проверка-работы-бота)
    - [Остановка бота (RU)](#остановка-бота)

## Description

The Todoist Telegram Bot is an automated assistant designed to integrate your Todoist account with Telegram. It allows you to quickly and conveniently add tasks to your Todoist directly from Telegram using simple text messages or by forwarding messages from other users. The bot automatically identifies the project by the user's name and adds tasks to the corresponding section of your Todoist. There is also an option to automatically add today's date as the due date for tasks.

## Key Features

- **Automatic task addition**: The bot accepts text messages and forwards them as new tasks to your Todoist. This can be either direct text messages to the bot or messages forwarded from other users.
- **Message forwarding**: You can forward messages from other users


## Description

The Todoist Telegram Bot is an automated assistant designed to integrate your Todoist account with Telegram. It allows you to quickly and conveniently add tasks to your Todoist directly from Telegram using simple text messages or by forwarding messages from other users. The bot automatically identifies the project by the user's name and adds tasks to the corresponding section of your Todoist. There is also an option to automatically add today's date as the due date for tasks.

## Key Features

- **Automatic task addition**: The bot accepts text messages and forwards them as new tasks to your Todoist. This can be either direct text messages to the bot or messages forwarded from other users.
- **Message forwarding**: You can forward messages from other users or from other chats to the bot, and it will automatically add these messages as tasks to your Todoist.
- **Project identification by user name**: Based on the preliminary setup, the bot associates Telegram users with projects in Todoist, automatically placing tasks in the appropriate projects.
- **Optional due date addition**: You can configure the bot so that each added task automatically receives a due date — the current date.
- **Easy setup and launch via Docker**: Thanks to the use of Docker and Docker Compose, launching and maintaining the bot does not require complex settings.

## Technologies

The bot is developed in Node.js using the `node-telegram-bot-api` library for interacting with the Telegram API and `axios` for requests to the Todoist API. All of this is wrapped in a Docker container for convenience of deployment and launching.

## Getting Started

### Prerequisites

- An account on [Telegram](https://telegram.org/) for creating a bot
- An account on [Todoist](https://todoist.com/) to obtain an API token
- [Docker](https://www.docker.com/) installed (for running in a container)

### Installation and Launch

#### Cloning the Repository

Open a terminal and execute the following command to clone the repository:

```bash
git clone your_repository_link
```
Navigate to the project directory:

```bash
cd todoist_bot
```

#### Setting up docker-compose
Open the file docker-compose_example.yml and fill in the following variables:

```makefile
PROJECT_USERS_PROJECTNAME1: "@user1,@user2" # Replace PROJECTNAME1 with the name of the project in Todoist (if the project is in Russian, then it should be taken in quotes), replace @user1,@user2 with the users whose messages you want to add to this project
TELEGRAM_TOKEN: your_botfather_token
TODOIST_TOKEN: your_todoist_api_token
TIMER: time_for_timer_in_seconds
ADD_DUE_DATE: 'true' or 'false' # Depending on whether you want to automatically set the task's due date to today
```
Save the file as docker-compose.yml.

#### Launching the Bot
Launch the bot using Docker Compose:
```bash
docker-compose up -d
```
After executing this command, the bot should start working.

Checking the Bot's Operation
Send a message to your bot on Telegram to check its operation.

#### Stopping the Bot
To stop the bot, execute the following command:

```bash
docker-compose down
```


# Todoist Telegram Bot

## Описание

Todoist Telegram Bot — это автоматизированный помощник, предназначенный для интеграции вашего аккаунта Todoist с Telegram. С его помощью вы можете быстро и удобно добавлять задачи в ваш Todoist прямо из Telegram, используя простые текстовые сообщения или пересылая сообщения от других пользователей. Бот автоматически определяет проект по имени пользователя и добавляет задачи в соответствующий раздел вашего Todoist. Также есть возможность настройки автоматического добавления даты выполнения задач на сегодняшний день.

## Основные функции

- **Автоматическое добавление задач**: Бот принимает текстовые сообщения и добавляет их как новые задачи в ваш Todoist. Это может быть как написание текста напрямую боту, так и пересылка сообщений от других пользователей.
- **Пересылка сообщений**: Вы можете пересылать сообщения от других пользователей или из других чатов боту, и он автоматически добавит эти сообщения как задачи в ваш Todoist.
- **Определение проекта по имени пользователя**: На основе предварительной настройки бот ассоциирует пользователей Telegram с проектами в Todoist, автоматически размещая задачи в нужных проектах.
- **Опциональное добавление даты выполнения**: Вы можете настроить бота так, чтобы каждая добавленная задача автоматически получала срок выполнения — текущую дату.
- **Легкая настройка и запуск через Docker**: Благодаря использованию Docker и Docker Compose, запуск и обслуживание бота не требует сложных настроек.


## Технологии

Бот разработан на Node.js с использованием библиотеки `node-telegram-bot-api` для взаимодействия с Telegram API и `axios` для запросов к Todoist API. Всё это обёрнуто в контейнер Docker для удобства развёртывания и запуска.

## Начало работы

### Предварительные требования

- Учетная запись на [Telegram](https://telegram.org/) для создания бота
- Учетная запись на [Todoist](https://todoist.com/) для получения API токена
- Установленный [Docker](https://www.docker.com/) (для запуска в контейнере)

### Установка и запуск

#### Клонирование репозитория

Откройте терминал и выполните следующую команду для клонирования репозитория:

```bash
git clone ссылка_на_ваш_репозиторий
```
Перейдите в директорию проекта:

```bash
cd todoist_bot
```
#### Настройка docker-compose

Откройте файл docker-compose_example.yml и заполните следующие переменные
```makefile
PROJECT_USERS_PROJECTNAME1: "@user1,@user2" ( PROJECTNAME1 нужно заменить на название проекта в Todoist (если проект на русском, то его нужно взять в кавычки), @user1,@user2 нужно заменить на пользователей сообщения от которых вы хотите добавлять в этот проект)
TELEGRAM_TOKEN: ваш_токен_от_botfather
TODOIST_TOKEN: ваш_todoist_api_токен
TIMER: время_для_таймера_в_секундах
ADD_DUE_DATE: 'true' или 'false' (в зависимости от того, хотите ли вы автоматически устанавливать срок выполнения задач на сегодня)
```
Сохраните файл как docker-compose.yml

#### Запуск бота

Запустите бота с помощью Docker Compose:

```bash
docker-compose up -d
```
После выполнения этой команды бот должен начать работу.

Проверка работы бота

Отправьте сообщение вашему боту в Telegram, чтобы проверить его работу.

#### Остановка бота
Чтобы остановить бота, выполните следующую команду:

```bash
docker-compose down
```

