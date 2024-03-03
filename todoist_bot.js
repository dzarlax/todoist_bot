const TelegramBot = require('node-telegram-bot-api');
const axios = require('axios');

// Получение токенов из переменных окружения
const token = process.env.TELEGRAM_TOKEN;
const todoistToken = process.env.TODOIST_TOKEN;

// Создаем экземпляр бота
const bot = new TelegramBot(token, { polling: true });



bot.on('message', (msg) => {
  const chatId = msg.chat.id;
  let text = msg.text;

  // Проверяем, является ли сообщение пересланным
  if (msg.forward_from || msg.forward_from_chat) {
    let senderName = '';

    // Определяем, есть ли информация о пользователе, который переслал сообщение
    if (msg.forward_from) {
      // Используем имя пользователя (без @), имя или фамилию
      senderName = msg.forward_from.username ? `${msg.forward_from.username}` : `${msg.forward_from.first_name} ${msg.forward_from.last_name ? msg.forward_from.last_name : ''}`.trim();
    } else if (msg.forward_from_chat) {
      // Используем имя группы (без @) или его название
      senderName = msg.forward_from_chat.username ? `${msg.forward_from_chat.username}` : msg.forward_from_chat.title;
    }

    // Добавляем имя отправителя в начало текста задачи
    text = `${senderName}: ${text}`;
  }

  // Проверяем, что сообщение является текстом и не начинается на '/'
  if (text && !text.startsWith('/')) {
    // Добавление задачи в Todoist
    axios.post('https://api.todoist.com/rest/v2/tasks', {
      content: text,
    }, {
      headers: {
        'Authorization': `Bearer ${todoistToken}`
      }
    })
    .then(response => {
      bot.sendMessage(chatId, 'Задача успешно добавлена!');
    })
    .catch(error => {
      bot.sendMessage(chatId, 'Произошла ошибка при добавлении задачи.');
      console.error(error);
    });
  }
});

