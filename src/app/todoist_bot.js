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

// Обработка команд
bot.onText(/\/start/, (msg) => {
  bot.sendMessage(msg.chat.id, 'Привет! Я бот для добавления задач в Todoist. Просто перешли мне сообщение или напиши текст, и я создам задачу.');
});

bot.onText(/\/help/, (msg) => {
  bot.sendMessage(msg.chat.id, 'Доступные команды:\n/start - Начать работу\n/help - Справка\n/status - Проверить статус');
});

bot.onText(/\/status/, (msg) => {
  const user = msg.from.username ? `@${msg.from.username}` : `${msg.from.first_name}`;
  bot.sendMessage(msg.chat.id, `Бот активен.\nВы вошли как: ${user}\nТаймер склейки: ${config.timer} сек.\nАвто-дата: ${config.autoAddDueDate ? 'Вкл' : 'Выкл'}`);
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
