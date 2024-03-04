const TelegramBot = require('node-telegram-bot-api');
const axios = require('axios');

// Токен, который вы получили от BotFather
const token = process.env.TELEGRAM_TOKEN;

// API токен Todoist
const todoistToken = process.env.TODOIST_TOKEN;

// Создаем экземпляр бота
const bot = new TelegramBot(token, { polling: true });

const messageBuffer = new Map(); // Буфер для хранения сообщений

function sendTaskToTodoist(chatId) {
  const buffer = messageBuffer.get(chatId);
  if (!buffer || buffer.messages.length === 0) return;

  const title = buffer.messages[0]; // Первое сообщение уже содержит ник отправителя
  let description = buffer.messages.slice(1).join('\n'); // Остальные сообщения тоже содержат ники

  // Отправка задачи в Todoist без изменений в 'title' и 'description'
  axios.post('https://api.todoist.com/rest/v2/tasks', {
      content: title + (description ? `\n\nОписание:\n${description}` : ''),
  }, {
      headers: {
          'Authorization': `Bearer ${todoistToken}`
      }
  })
  .then(response => {
      bot.sendMessage(chatId, 'Задача успешно добавлена!');
  })
  .catch(error => {
      console.error(error);
      bot.sendMessage(chatId, 'Произошла ошибка при добавлении задачи.');
  });

  // Очистка буфера после отправки
  messageBuffer.delete(chatId);
}


function handleMessage(msg) {
  const chatId = msg.chat.id;
  const text = msg.text;

  // Определение ника отправителя
  const senderName = msg.from.username ? `@${msg.from.username}` : `${msg.from.first_name} ${msg.from.last_name || ''}`.trim();
  const textWithSender = `${senderName}: ${text}`; // Добавляем ник отправителя к каждому сообщению

  if (!messageBuffer.has(chatId)) {
      messageBuffer.set(chatId, {
          messages: [textWithSender], // Сохраняем сообщение уже с ником отправителя
          timer: setTimeout(() => sendTaskToTodoist(chatId), 5000) // Таймер на 5 секунд
      });
  } else {
      const buffer = messageBuffer.get(chatId);
      clearTimeout(buffer.timer);
      buffer.messages.push(textWithSender); // Добавляем в буфер сообщение с ником
      buffer.timer = setTimeout(() => sendTaskToTodoist(chatId), 5000);
  }
}


bot.on('message', (msg) => {
    // Проверяем, что сообщение является текстом и не начинается на '/'
    if (msg.text && !msg.text.startsWith('/')) {
        handleMessage(msg);
    }
});