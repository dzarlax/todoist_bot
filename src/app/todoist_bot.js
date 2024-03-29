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
    // Проверка, нужно ли добавлять дату выполнения
    const addDueDate = process.env.AUTO_ADD_DUE_DATE === 'true';

    if (!messageBuffer.has(chatId)) return;

    const buffer = messageBuffer.get(chatId);
    if (buffer.messages.length === 0) return;

    const title = buffer.messages[0];
    let description = buffer.messages.slice(1).join('\n');
    const sender = title.split(': ')[0];
    let projectName = findProjectNameForUser(sender);

    if (!projectName) {
        console.error("Проект для пользователя не найден:", sender);
        bot.sendMessage(chatId, `Проект для пользователя "${sender}" не найден, задача будет помещена во входящие.`);
        projectName = 'Inbox';
    }

    let projects;
    try {
        projects = await fetchProjects();
    } catch (error) {
        console.error("Ошибка при получении списка проектов:", error);
        bot.sendMessage(chatId, 'Произошла ошибка при попытке получить список проектов.');
        return;
    }

    const projectId = projects[projectName];
    if (!projectId) {
        console.error("Проект не найден:", projectName);
        bot.sendMessage(chatId, `Проект "${projectName}" не найден.`);
        return;
    }

    // Формирование тела запроса с учётом опционального добавления due_date
    const taskData = {
        content: title + (description ? `\n\nОписание:\n${description}` : ''),
        project_id: projectId
    };

    if (addDueDate) {
        // Получение сегодняшней даты в формате YYYY-MM-DD
        const today = new Date().toISOString().split('T')[0];
        taskData.due_date = today;
    }

    axios.post('https://api.todoist.com/rest/v2/tasks', taskData, {
        headers: {
            'Authorization': `Bearer ${todoistToken}`
        }
    })
    .then(response => {
        bot.sendMessage(chatId, `Задача успешно добавлена в проект "${projectName}"${addDueDate ? ' с датой выполнения на сегодня' : ''}!`);
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
