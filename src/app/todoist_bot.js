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

// Функция для форматирования текста с учетом ссылок
function formatTextWithLinks(text, entities) {
  if (!text) return text;
  
  // Если нет сущностей, обрабатываем только обычные URL
  if (!entities || !Array.isArray(entities) || entities.length === 0) {
    // Регулярное выражение для поиска URL
    const urlRegex = /(https?:\/\/|www\.)[^\s]+(\.[^\s]+)+/gi;
    
    // Находим все URL в тексте
    const urls = text.match(urlRegex) || [];
    
    // Если нет URL, возвращаем исходный текст
    if (urls.length === 0) return text;
    
    let formattedText = text;
    
    // Заменяем каждый URL на его Markdown-версию
    urls.forEach(url => {
      // Если URL начинается с www., добавляем https://
      const fullUrl = url.startsWith('www.') ? `https://${url}` : url;
      
      // Заменяем URL на его Markdown-версию
      formattedText = formattedText.replace(url, `[${url}](${fullUrl})`);
    });
    
    return formattedText;
  }
  
  // Сортируем сущности в обратном порядке, чтобы начать с конца текста
  // Это позволяет избежать проблем с изменением индексов при вставке
  const sortedEntities = [...entities].sort((a, b) => b.offset - a.offset);
  
  let formattedText = text;
  
  // Обрабатываем каждую сущность
  sortedEntities.forEach(entity => {
    if (entity.type === 'text_link' && entity.url) {
      // Извлекаем текст ссылки
      const linkText = text.substring(entity.offset, entity.offset + entity.length);
      
      // Форматируем ссылку в Markdown для Todoist
      const markdownLink = `[${linkText}](${entity.url})`;
      
      // Заменяем текст на форматированную ссылку
      formattedText = 
        formattedText.substring(0, entity.offset) + 
        markdownLink + 
        formattedText.substring(entity.offset + entity.length);
    }
  });
  
  // После обработки форматированных ссылок, обрабатываем обычные URL
  // Регулярное выражение для поиска URL
  const urlRegex = /(https?:\/\/|www\.)[^\s]+(\.[^\s]+)+/gi;
  
  // Находим все URL в тексте
  const urls = formattedText.match(urlRegex) || [];
  
  // Заменяем каждый URL на его Markdown-версию
  urls.forEach(url => {
    // Проверяем, не является ли URL уже частью Markdown-ссылки
    const markdownLinkRegex = /\[([^\]]+)\]\(([^)]+)\)/g;
    let isInMarkdownLink = false;
    
    let match;
    while ((match = markdownLinkRegex.exec(formattedText)) !== null) {
      if (match[2] === url || match[0].includes(url)) {
        isInMarkdownLink = true;
        break;
      }
    }
    
    // Если URL не является частью Markdown-ссылки, форматируем его
    if (!isInMarkdownLink) {
      // Если URL начинается с www., добавляем https://
      const fullUrl = url.startsWith('www.') ? `https://${url}` : url;
      
      // Заменяем URL на его Markdown-версию
      formattedText = formattedText.replace(url, `[${url}](${fullUrl})`);
    }
  });
  
  return formattedText;
}

// Функция для получения ссылки на медиафайл
async function getMediaLink(msg) {
  try {
    let fileId;
    let mediaType;
    let fileName = '';

    if (msg.photo) {
      // Для фото берем последний элемент массива (наибольшее разрешение)
      fileId = msg.photo[msg.photo.length - 1].file_id;
      mediaType = 'фото';
    } else if (msg.video) {
      fileId = msg.video.file_id;
      mediaType = 'видео';
      if (msg.video.file_name) {
        fileName = ` (${msg.video.file_name})`;
      }
    } else if (msg.document) {
      fileId = msg.document.file_id;
      mediaType = 'документ';
      if (msg.document.file_name) {
        fileName = ` (${msg.document.file_name})`;
      }
    } else if (msg.audio) {
      fileId = msg.audio.file_id;
      mediaType = 'аудио';
      if (msg.audio.title || msg.audio.performer) {
        fileName = ` (${[msg.audio.title, msg.audio.performer].filter(Boolean).join(' - ')})`;
      } else if (msg.audio.file_name) {
        fileName = ` (${msg.audio.file_name})`;
      }
    } else if (msg.voice) {
      fileId = msg.voice.file_id;
      mediaType = 'голосовое сообщение';
    } else if (msg.animation) {
      fileId = msg.animation.file_id;
      mediaType = 'анимация';
      if (msg.animation.file_name) {
        fileName = ` (${msg.animation.file_name})`;
      }
    } else if (msg.sticker) {
      fileId = msg.sticker.file_id;
      mediaType = 'стикер';
      if (msg.sticker.emoji) {
        fileName = ` (${msg.sticker.emoji})`;
      }
    } else if (msg.location) {
      // Для локаций создаем ссылку на Google Maps
      const { latitude, longitude } = msg.location;
      return {
        link: `https://www.google.com/maps?q=${latitude},${longitude}`,
        type: 'локация',
        isDirectLink: true
      };
    } else if (msg.contact) {
      // Для контактов создаем текстовое представление
      const { first_name, last_name, phone_number } = msg.contact;
      const contactName = [first_name, last_name].filter(Boolean).join(' ');
      return {
        link: `tel:${phone_number}`,
        type: 'контакт',
        description: `${contactName}: ${phone_number}`,
        isDirectLink: true
      };
    } else if (msg.poll) {
      // Для опросов создаем текстовое представление
      return {
        type: 'опрос',
        description: `Вопрос: ${msg.poll.question}\nВарианты: ${msg.poll.options.map(opt => opt.text).join(', ')}`,
        isDirectLink: true
      };
    } else if (msg.venue) {
      // Для мест (venue) создаем ссылку и описание
      const { latitude, longitude } = msg.venue.location;
      const { title, address } = msg.venue;
      return {
        link: `https://www.google.com/maps?q=${latitude},${longitude}`,
        type: 'место',
        description: `${title}, ${address}`,
        isDirectLink: true
      };
    } else {
      return null;
    }

    // Если это не прямая ссылка (локация, контакт, опрос), получаем ссылку на файл
    if (!fileId) return null;
    
    const fileLink = await bot.getFileLink(fileId);
    return { link: fileLink, type: mediaType + fileName };
  } catch (error) {
    console.error('Ошибка при получении ссылки на медиафайл:', error);
    return null;
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
    
    // Медиа файлы теперь включены непосредственно в сообщения, поэтому не нужно добавлять их отдельно
    
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
        content: title,
        description: description ? description : '',
        project_id: projectId
    };

    if (addDueDate) {
        // Получение сегодняшней даты в формате YYYY-MM-DD
        const today = new Date().toISOString().split('T')[0];
        taskData.due_date = today;
    }

    try {
        await axios.post('https://api.todoist.com/rest/v2/tasks', taskData, {
            headers: {
                'Authorization': `Bearer ${todoistToken}`
            }
        });
        bot.sendMessage(chatId, `Задача успешно добавлена в проект "${projectName}"${addDueDate ? ' с датой выполнения на сегодня' : ''}!`);
    } catch (error) {
        console.error(error);
        bot.sendMessage(chatId, 'Произошла ошибка при добавлении задачи.');
    } finally {
        messageBuffer.delete(chatId);
    }
}

// Обработка входящих сообщений
async function handleMessage(msg) {
  try {
    const chatId = msg.chat.id;
    
    // Пропускаем сообщения, которые являются частью медиагруппы
    // Они будут обработаны отдельным обработчиком media_group
    if (msg.media_group_id) return;
    
    let senderName;
    if (msg.forward_from) {
        senderName = msg.forward_from.username ? `@${msg.forward_from.username}` : `${msg.forward_from.first_name} ${msg.forward_from.last_name || ''}`.trim();
    } else if (msg.forward_from_chat) {
        senderName = msg.forward_from_chat.title;
    } else {
        senderName = msg.from.username ? `@${msg.from.username}` : `${msg.from.first_name} ${msg.from.last_name || ''}`.trim();
    }

    let messageContent;
    let mediaInfo = null;

    // Проверяем, содержит ли сообщение текст
    if (msg.text) {
      // Форматируем текст с учетом ссылок
      messageContent = formatTextWithLinks(msg.text, msg.entities);
    } else {
      // Если нет текста, проверяем наличие медиа и получаем ссылку
      mediaInfo = await getMediaLink(msg);
      
      // Если есть подпись к медиа, используем её как текст и форматируем ссылки
      if (msg.caption) {
        messageContent = formatTextWithLinks(msg.caption, msg.caption_entities);
        
        // Добавляем ссылку на медиа прямо в текст сообщения
        if (mediaInfo) {
          messageContent += ` [${mediaInfo.type}](${mediaInfo.link})`;
          mediaInfo = null; // Обнуляем mediaInfo, так как мы уже добавили ссылку в текст
        }
      } else {
        // Если нет подписи, используем тип медиа как текст с ссылкой
        messageContent = mediaInfo ? `[${mediaInfo.type}](${mediaInfo.link})` : '[неизвестный тип медиа]';
        mediaInfo = null; // Обнуляем mediaInfo, так как мы уже добавили ссылку в текст
      }
    }

    const messageWithSender = `${senderName}: ${messageContent}`;

    if (!messageBuffer.has(chatId)) {
        messageBuffer.set(chatId, {
            messages: [messageWithSender],
            media: mediaInfo ? [mediaInfo] : [],
            timer: setTimeout(() => sendTaskToTodoist(chatId), timer*1000)
        });
    } else {
        const buffer = messageBuffer.get(chatId);
        clearTimeout(buffer.timer);
        buffer.messages.push(messageWithSender);
        
        if (mediaInfo) {
          if (!buffer.media) {
            buffer.media = [];
          }
          buffer.media.push(mediaInfo);
        }
        
        buffer.timer = setTimeout(() => sendTaskToTodoist(chatId), timer*1000);
    }
  } catch (error) {
    console.error('Ошибка при обработке сообщения:', error);
    // Не прерываем работу бота при ошибке в одном сообщении
  }
}

// Обработчик для медиагрупп (альбомов)
bot.on('mediagroup', async (mediaGroup) => {
  try {
    if (!mediaGroup || mediaGroup.length === 0) return;
    
    const chatId = mediaGroup[0].chat.id;
    const senderMsg = mediaGroup[0];
    
    let senderName;
    if (senderMsg.forward_from) {
        senderName = senderMsg.forward_from.username ? `@${senderMsg.forward_from.username}` : `${senderMsg.forward_from.first_name} ${senderMsg.forward_from.last_name || ''}`.trim();
    } else if (senderMsg.forward_from_chat) {
        senderName = senderMsg.forward_from_chat.title;
    } else {
        senderName = senderMsg.from.username ? `@${senderMsg.from.username}` : `${senderMsg.from.first_name} ${senderMsg.from.last_name || ''}`.trim();
    }
    
    // Получаем ссылки на все медиа из группы
    const mediaPromises = mediaGroup.map(msg => getMediaLink(msg));
    const mediaResults = await Promise.all(mediaPromises);
    const mediaInfos = mediaResults.filter(Boolean);
    
    // Создаем сообщения с медиа и их подписями
    const messages = [];
    
    // Обрабатываем первое сообщение
    let firstMessage = '';
    if (senderMsg.caption) {
      let formattedCaption = formatTextWithLinks(senderMsg.caption, senderMsg.caption_entities);
      // Добавляем ссылку на медиа прямо в текст
      if (mediaInfos.length > 0) {
        formattedCaption += ` [${mediaInfos[0].type}](${mediaInfos[0].link})`;
      }
      firstMessage = `${senderName}: ${formattedCaption}`;
    } else {
      // Если нет подписи, используем тип медиа как текст
      const mediaText = mediaInfos.length > 0 ? `[${mediaInfos[0].type}](${mediaInfos[0].link})` : '[медиа группа]';
      firstMessage = `${senderName}: ${mediaText}`;
    }
    messages.push(firstMessage);
    
    // Обрабатываем остальные сообщения в группе
    for (let i = 1; i < mediaGroup.length; i++) {
      const msg = mediaGroup[i];
      if (msg.caption) {
        let formattedCaption = formatTextWithLinks(msg.caption, msg.caption_entities);
        // Добавляем ссылку на медиа прямо в текст
        if (i < mediaInfos.length) {
          formattedCaption += ` [${mediaInfos[i].type}](${mediaInfos[i].link})`;
        }
        messages.push(`${senderName}: ${formattedCaption}`);
      } else if (i < mediaInfos.length) {
        // Если нет подписи, создаем сообщение только с медиа
        messages.push(`${senderName}: [${mediaInfos[i].type}](${mediaInfos[i].link})`);
      }
    }
    
    if (!messageBuffer.has(chatId)) {
        messageBuffer.set(chatId, {
            messages: messages,
            media: [], // Медиа уже включены в сообщения
            timer: setTimeout(() => sendTaskToTodoist(chatId), timer*1000)
        });
    } else {
        const buffer = messageBuffer.get(chatId);
        clearTimeout(buffer.timer);
        buffer.messages.push(...messages);
        buffer.timer = setTimeout(() => sendTaskToTodoist(chatId), timer*1000);
    }
  } catch (error) {
    console.error('Ошибка при обработке медиагруппы:', error);
  }
});

// Обработчик для отредактированных сообщений
bot.on('edited_message', (msg) => {
  // Игнорируем редактирование сообщений, так как они уже могли быть добавлены в задачу
  console.log('Сообщение отредактировано, но изменения не будут отражены в задаче:', msg.text || msg.caption || '[медиа]');
});

// Обновляем обработчик сообщений, чтобы он обрабатывал все типы сообщений
bot.on('message', (msg) => {
    // Пропускаем команды
    if (msg.text && msg.text.startsWith('/')) return;
    
    // Обрабатываем все остальные сообщения
    handleMessage(msg);
});

// Обработка ошибок, чтобы бот не падал
bot.on('polling_error', (error) => {
  console.error('Ошибка в работе бота:', error);
});
