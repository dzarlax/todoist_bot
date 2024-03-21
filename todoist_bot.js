const TelegramBot = require('node-telegram-bot-api');
const axios = require('axios');

// Токен, который вы получили от BotFather
const token = process.env.TELEGRAM_TOKEN;

// API токен Todoist
const todoistToken = process.env.TODOIST_TOKEN;

// Таймер для склеивания сообщений
const timer = process.env.TIMER;

// Создаем экземпляр бота
const bot = new TelegramBot(token, { polling: true });

const messageBuffer = new Map(); // Буфер для хранения сообщений

// Преобразование переменных окружения в объект проект-список пользователей
const projectToUsersMapping = {};
Object.keys(process.env).forEach(key => {
  if (key.startsWith('PROJECT_USERS_')) {
    const projectName = key.replace('PROJECT_USERS_', '');
    const users = process.env[key].split(',').map(user => user.trim());
    projectToUsersMapping[projectName] = users;
  }
});

console.log(projectToUsersMapping)

// Функция для определения проекта по имени пользователя
function findProjectNameForUser(username) {
  for (const [projectName, users] of Object.entries(projectToUsersMapping)) {
    if (users.includes(username)) {
      return projectName;
    }
  }
  return null; // Возвращаем null, если проект для пользователя не найден
}


// Функция для получения списка проектов и их ID из Todoist
async function fetchProjects() {
    try {
        const response = await axios.get('https://api.todoist.com/rest/v2/projects', {
            headers: {
                'Authorization': `Bearer ${todoistToken}`
            }
        });
        const projects = {};
        response.data.forEach(project => {
            projects[project.name] = project.id;
        });
        console.log(projects)
        return projects;
    } catch (error) {
        console.error("Ошибка при получении проектов:", error);
        return {};
    }
}

// Функция для отправки задачи в Todoist
async function sendTaskToTodoist(chatId) {
    if (!messageBuffer.has(chatId)) return;

    const buffer = messageBuffer.get(chatId);
    if (buffer.messages.length === 0) return;

    const title = buffer.messages[0];
    let description = buffer.messages.slice(1).join('\n');
    const sender = title.split(': ')[0];
  const projectName = findProjectNameForUser(sender); // Используйте функцию для определения имени проекта

  if (!projectName) {
    console.error("Проект для пользователя не найден:", sender);
    bot.sendMessage(chatId, `Проект для пользователя "${sender}" не найден, задача будет помещена во входящие.`);
    projectName = 'Inbox'
  }

    const projects = await fetchProjects(); // Получаем список проектов
    const projectId = projects[projectName]; // Получаем ID проекта по имени
    console.log(projects)
    console.log(projectId)
    if (!projectId) {
        console.error("Проект не найден:", projectName);
        bot.sendMessage(chatId, `Проект "${projectName}" не найден.`);
        return;
    }

    axios.post('https://api.todoist.com/rest/v2/tasks', {
        content: title + (description ? `\n\nОписание:\n${description}` : ''),
        project_id: projectId,
    }, {
        headers: {
            'Authorization': `Bearer ${todoistToken}`
        }
    })
    .then(response => {
        bot.sendMessage(chatId, `Задача успешно добавлена в проект "${projectName}"!`);
    })
    .catch(error => {
        console.error(error);
        bot.sendMessage(chatId, 'Произошла ошибка при добавлении задачи.');
    });

    messageBuffer.delete(chatId);
}

// Обработка входящих сообщений
function handleMessage(msg) {
  const chatId = msg.chat.id;
  const text = msg.text;

  let senderName;
  if (msg.forward_from) {
      senderName = msg.forward_from.username ? `@${msg.forward_from.username}` : `${msg.forward_from.first_name} ${msg.forward_from.last_name || ''}`.trim();
  } else if (msg.forward_from_chat) {
      senderName = msg.forward_from_chat.title;
  } else {
      senderName = msg.from.username ? `@${msg.from.username}` : `${msg.from.first_name} ${msg.from.last_name || ''}`.trim();
  }

  const messageWithSender = `${senderName}: ${text}`;

  if (!messageBuffer.has(chatId)) {
      messageBuffer.set(chatId, {
          messages: [messageWithSender],
          timer: setTimeout(() => sendTaskToTodoist(chatId), timer*1000) // Таймер на 5 секунд
      });
  } else {
      const buffer = messageBuffer.get(chatId);
      clearTimeout(buffer.timer);
      buffer.messages.push(messageWithSender);
      buffer.timer = setTimeout(() => sendTaskToTodoist(chatId), timer*1000);
  }
}

bot.on('message', (msg) => {
    if (msg.text && !msg.text.startsWith('/')) {
        handleMessage(msg);
    }
});
