require('dotenv').config();
const logger = require('../utils/logger');

const config = {
    telegramToken: process.env.TELEGRAM_TOKEN,
    todoistToken: process.env.TODOIST_TOKEN,
    timer: parseInt(process.env.TIMER || '5', 10),
    autoAddDueDate: process.env.AUTO_ADD_DUE_DATE === 'true',
    projectUsersMapping: {},
    maxRetries: parseInt(process.env.MAX_RETRIES || '3', 10),
    retryDelay: parseInt(process.env.RETRY_DELAY || '1000', 10)
};

// Преобразование переменных окружения в объект проект-список пользователей
Object.keys(process.env).forEach(key => {
    if (key.startsWith('PROJECT_USERS_')) {
        const projectName = key.replace('PROJECT_USERS_', '');
        const users = process.env[key].split(',').map(user => user.trim());
        config.projectUsersMapping[projectName] = users;
    }
});

/**
 * Валидация конфигурации
 */
function validateConfig() {
    const missing = [];
    if (!config.telegramToken) missing.push('TELEGRAM_TOKEN');
    if (!config.todoistToken) missing.push('TODOIST_TOKEN');

    if (missing.length > 0) {
        logger.error(`Ошибка: Отсутствуют обязательные переменные окружения: ${missing.join(', ')}`);
        process.exit(1);
    }
}

module.exports = {
    config,
    validateConfig
};
