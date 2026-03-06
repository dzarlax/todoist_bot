const TelegramBot = require('node-telegram-bot-api');
const { config, validateConfig } = require('./config');
const TodoistClient = require('./api/todoist');
const { handleMessage, handleMediaGroup } = require('./handlers/telegram');
const { createHealthCheckServer } = require('./server');
const logger = require('./utils/logger');

// Валидация конфигурации при запуске
validateConfig();

// Инициализация бота
const bot = new TelegramBot(config.telegramToken, { polling: true });

// Инициализация API клиента Todoist
const todoistClient = new TodoistClient(config.todoistToken, config.maxRetries, config.retryDelay);

// Буфер сообщений (chatId -> { messages: [], timer: null })
const messageBuffer = new Map();

logger.info('Todoist Bot запущен...', {
  projects: Object.keys(config.projectUsersMapping).join(', ') || 'нет пользовательских проектов',
  timer: config.timer,
  autoAddDueDate: config.autoAddDueDate,
  maxRetries: config.maxRetries,
  retryDelay: config.retryDelay
});

// Запуск health check сервера
const healthServer = createHealthCheckServer(todoistClient, bot);

const isAdmin = (msg) => config.botAdmin && msg.from?.username === config.botAdmin;

// Обработка команд
bot.onText(/\/start/, (msg) => {
  bot.sendMessage(msg.chat.id, 'Привет! Я бот для добавления задач в Todoist. Просто перешли мне сообщение или напиши текст, и я создам задачу.');
});

bot.onText(/\/help/, (msg) => {
  const adminCommands = isAdmin(msg) ? '\n/list [проект] - Список активных задач' : '';
  bot.sendMessage(msg.chat.id, `Доступные команды:\n/start - Начать работу\n/help - Справка\n/status - Проверить статус${adminCommands}`);
});

bot.onText(/\/status/, (msg) => {
  const user = msg.from.username ? `@${msg.from.username}` : `${msg.from.first_name}`;
  const role = isAdmin(msg) ? ' (admin)' : '';
  bot.sendMessage(msg.chat.id, `Бот активен.\nВы вошли как: ${user}${role}\nТаймер склейки: ${config.timer} сек.\nАвто-дата: ${config.autoAddDueDate ? 'Вкл' : 'Выкл'}`);
});

bot.onText(/\/list(.*)/, async (msg, match) => {
  if (!isAdmin(msg)) return;

  const projectFilter = match[1].trim();
  try {
    const projects = await todoistClient.fetchProjects();
    let projectId = null;

    if (projectFilter) {
      projectId = projects[projectFilter];
      if (!projectId) {
        bot.sendMessage(msg.chat.id, `Проект "${projectFilter}" не найден.`);
        return;
      }
    }

    const tasks = await todoistClient.getTasks(projectId);
    if (!tasks.length) {
      bot.sendMessage(msg.chat.id, 'Нет активных задач.');
      return;
    }

    const projectNames = Object.fromEntries(Object.entries(projects).map(([k, v]) => [v, k]));
    const lines = tasks.slice(0, 20).map(t => {
      const project = projectNames[t.project_id] || '';
      const due = t.due ? ` 📅 ${t.due.string}` : '';
      const prio = t.priority > 1 ? ` [p${t.priority}]` : '';
      return `• ${t.content}${prio}${due}${project ? ` — ${project}` : ''}`;
    });

    bot.sendMessage(msg.chat.id, lines.join('\n'), { parse_mode: 'Markdown' });
  } catch (error) {
    logger.error('Ошибка при получении задач', { error: error.message });
    bot.sendMessage(msg.chat.id, '❌ Ошибка при получении задач.');
  }
});

// Основной обработчик сообщений
bot.on('message', (msg) => {
  handleMessage(msg, bot, messageBuffer, todoistClient, config);
});

// Обработчик медиагрупп
bot.on('mediagroup', (mediaGroup) => {
  handleMediaGroup(mediaGroup, bot, messageBuffer, todoistClient, config);
});

// Логирование ошибок
bot.on('polling_error', (error) => {
  logger.error('Ошибка Polling', { error: error.message });
});

// Корректное завершение работы
const gracefulShutdown = (signal) => {
  logger.info(`Получен сигнал ${signal}, остановка бота...`);
  bot.stopPolling();
  healthServer.close(() => {
    logger.info('Health check server остановлен');
    process.exit(0);
  });

  // Принудительное завершение через 10 секунд
  setTimeout(() => {
    logger.error('Принудительное завершение работы');
    process.exit(1);
  }, 10000);
};

process.on('SIGINT', () => gracefulShutdown('SIGINT'));
process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
