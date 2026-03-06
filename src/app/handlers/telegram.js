const { formatTextWithLinks, findProjectNameForUser, parseTaskMeta } = require('../utils/formatter');
const { retryAsync } = require('../utils/retry');
const logger = require('../utils/logger');

async function getMediaLink(bot, msg, maxRetries = 3, retryDelay = 1000) {
    try {
        let fileId;
        let mediaType;
        let fileName = '';

        if (msg.photo) {
            fileId = msg.photo[msg.photo.length - 1].file_id;
            mediaType = 'фото';
        } else if (msg.video) {
            fileId = msg.video.file_id;
            mediaType = 'видео';
            if (msg.video.file_name) fileName = ` (${msg.video.file_name})`;
        } else if (msg.document) {
            fileId = msg.document.file_id;
            mediaType = 'документ';
            if (msg.document.file_name) fileName = ` (${msg.document.file_name})`;
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
            if (msg.animation.file_name) fileName = ` (${msg.animation.file_name})`;
        } else if (msg.sticker) {
            fileId = msg.sticker.file_id;
            mediaType = 'стикер';
            if (msg.sticker.emoji) fileName = ` (${msg.sticker.emoji})`;
        } else if (msg.location) {
            const { latitude, longitude } = msg.location;
            return {
                link: `https://www.google.com/maps?q=${latitude},${longitude}`,
                type: 'локация',
                isDirectLink: true
            };
        } else if (msg.contact) {
            const { first_name, last_name, phone_number } = msg.contact;
            const contactName = [first_name, last_name].filter(Boolean).join(' ');
            return {
                link: `tel:${phone_number}`,
                type: 'контакт',
                description: `${contactName}: ${phone_number}`,
                isDirectLink: true
            };
        } else if (msg.poll) {
            return {
                type: 'опрос',
                description: `Вопрос: ${msg.poll.question}\nВарианты: ${msg.poll.options.map(opt => opt.text).join(', ')}`,
                isDirectLink: true
            };
        } else if (msg.venue) {
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

        if (!fileId) return null;

        const fileLink = await retryAsync(
            async () => await bot.getFileLink(fileId),
            {
                maxRetries: maxRetries,
                initialDelay: retryDelay,
                retryableErrors: ['ECONNRESET', 'ETIMEDOUT', 'ENOTFOUND', 'ECONNREFUSED']
            }
        );
        return { link: fileLink, type: mediaType + fileName };
    } catch (error) {
        logger.error('Ошибка при получении ссылки на медиафайл', {
            error: error.message,
            mediaType: msg.photo ? 'photo' : msg.video ? 'video' : msg.document ? 'document' : 'unknown'
        });
        return null;
    }
}

async function sendTaskToTodoist(chatId, bot, messageBuffer, todoistClient, config) {
    if (!messageBuffer.has(chatId)) return;

    const buffer = messageBuffer.get(chatId);
    if (buffer.messages.length === 0) return;

    const title = buffer.messages[0];
    let description = buffer.messages.slice(1).join('\n');

    const sender = title.split(': ')[0];
    let projectName = findProjectNameForUser(sender, config.projectUsersMapping);

    if (!projectName) {
        logger.warn("Проект для пользователя не найден", { sender, chatId });
        bot.sendMessage(chatId, `Проект для пользователя "${sender}" не найден, задача будет помещена во входящие.`);
        projectName = 'Inbox';
    }

    try {
        const projects = await todoistClient.fetchProjects();
        const projectId = projects[projectName];

        if (!projectId) {
            logger.error("Проект не найден в Todoist", { projectName, chatId });
            bot.sendMessage(chatId, `Проект "${projectName}" не найден.`);
            return;
        }

        const taskData = {
            content: title,
            description: description || '',
            project_id: projectId
        };

        if (config.autoAddDueDate) {
            taskData.due_date = new Date().toISOString().split('T')[0];
        }
        if (buffer.priority && buffer.priority > 1) {
            taskData.priority = buffer.priority;
        }
        if (buffer.labels && buffer.labels.length > 0) {
            taskData.labels = buffer.labels;
        }

        await todoistClient.createTask(taskData);
        const priorityStr = buffer.priority > 1 ? ` [p${buffer.priority}]` : '';
        const labelsStr = buffer.labels?.length ? ` [${buffer.labels.join(', ')}]` : '';
        bot.sendMessage(chatId, `✅ Задача добавлена в "${projectName}"${priorityStr}${labelsStr}${config.autoAddDueDate ? ' (срок на сегодня)' : ''}.`);
    } catch (error) {
        logger.error('Ошибка при добавлении задачи', {
            chatId,
            projectName,
            error: error.message,
            stack: error.stack
        });
        bot.sendMessage(chatId, '❌ Ошибка при добавлении задачи в Todoist.');
    } finally {
        messageBuffer.delete(chatId);
    }
}

async function handleMessage(msg, bot, messageBuffer, todoistClient, config) {
    try {
        const chatId = msg.chat.id;
        if (config.allowedUsernames.length > 0 && !config.allowedUsernames.includes(msg.from?.username)) return;
        if (msg.media_group_id) return;
        if (msg.text && msg.text.startsWith('/')) return;

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

        if (msg.text) {
            messageContent = formatTextWithLinks(msg.text, msg.entities);
        } else {
            mediaInfo = await getMediaLink(bot, msg, config.maxRetries, config.retryDelay);
            if (msg.caption) {
                messageContent = formatTextWithLinks(msg.caption, msg.caption_entities);
                if (mediaInfo) {
                    messageContent += ` [${mediaInfo.type}](${mediaInfo.link})`;
                    mediaInfo = null;
                }
            } else {
                messageContent = mediaInfo ? `[${mediaInfo.type}](${mediaInfo.link})` : '[неизвестный тип медиа]';
                mediaInfo = null;
            }
        }

        if (!messageBuffer.has(chatId)) {
            const { priority, labels, cleanText } = parseTaskMeta(messageContent);
            messageBuffer.set(chatId, {
                messages: [`${senderName}: ${cleanText}`],
                priority,
                labels,
                timer: setTimeout(() => sendTaskToTodoist(chatId, bot, messageBuffer, todoistClient, config), config.timer * 1000)
            });
        } else {
            const buffer = messageBuffer.get(chatId);
            clearTimeout(buffer.timer);
            buffer.messages.push(`${senderName}: ${messageContent}`);
            buffer.timer = setTimeout(() => sendTaskToTodoist(chatId, bot, messageBuffer, todoistClient, config), config.timer * 1000);
        }
    } catch (error) {
        logger.error('Ошибка при обработке сообщения', {
            chatId: msg.chat.id,
            error: error.message,
            stack: error.stack
        });
    }
}

async function handleMediaGroup(mediaGroup, bot, messageBuffer, todoistClient, config) {
    try {
        if (!mediaGroup || mediaGroup.length === 0) return;

        const chatId = mediaGroup[0].chat.id;
        if (config.allowedUsernames.length > 0 && !config.allowedUsernames.includes(mediaGroup[0].from?.username)) return;
        const senderMsg = mediaGroup[0];

        let senderName;
        if (senderMsg.forward_from) {
            senderName = senderMsg.forward_from.username ? `@${senderMsg.forward_from.username}` : `${senderMsg.forward_from.first_name} ${senderMsg.forward_from.last_name || ''}`.trim();
        } else if (senderMsg.forward_from_chat) {
            senderName = senderMsg.forward_from_chat.title;
        } else {
            senderName = senderMsg.from.username ? `@${senderMsg.from.username}` : `${senderMsg.from.first_name} ${senderMsg.from.last_name || ''}`.trim();
        }

        const mediaPromises = mediaGroup.map(msg => getMediaLink(bot, msg, config.maxRetries, config.retryDelay));
        const mediaResults = await Promise.all(mediaPromises);
        const mediaInfos = mediaResults.filter(Boolean);

        const messages = [];
        let groupPriority = 1;
        let groupLabels = [];

        let firstMessageText = '';
        if (senderMsg.caption) {
            let formattedCaption = formatTextWithLinks(senderMsg.caption, senderMsg.caption_entities);
            const { priority, labels, cleanText } = parseTaskMeta(formattedCaption);
            groupPriority = priority;
            groupLabels = labels;
            let captionText = cleanText;
            if (mediaInfos.length > 0) {
                captionText += ` [${mediaInfos[0].type}](${mediaInfos[0].link})`;
            }
            firstMessageText = `${senderName}: ${captionText}`;
        } else {
            const mediaText = mediaInfos.length > 0 ? `[${mediaInfos[0].type}](${mediaInfos[0].link})` : '[медиа группа]';
            firstMessageText = `${senderName}: ${mediaText}`;
        }
        messages.push(firstMessageText);

        for (let i = 1; i < mediaGroup.length; i++) {
            const msg = mediaGroup[i];
            if (msg.caption) {
                let text = formatTextWithLinks(msg.caption, msg.caption_entities);
                if (i < mediaInfos.length) text += ` [${mediaInfos[i].type}](${mediaInfos[i].link})`;
                messages.push(`${senderName}: ${text}`);
            } else if (i < mediaInfos.length) {
                messages.push(`${senderName}: [${mediaInfos[i].type}](${mediaInfos[i].link})`);
            }
        }

        if (!messageBuffer.has(chatId)) {
            messageBuffer.set(chatId, {
                messages: messages,
                priority: groupPriority,
                labels: groupLabels,
                timer: setTimeout(() => sendTaskToTodoist(chatId, bot, messageBuffer, todoistClient, config), config.timer * 1000)
            });
        } else {
            const buffer = messageBuffer.get(chatId);
            clearTimeout(buffer.timer);
            buffer.messages.push(...messages);
            buffer.timer = setTimeout(() => sendTaskToTodoist(chatId, bot, messageBuffer, todoistClient, config), config.timer * 1000);
        }
    } catch (error) {
        logger.error('Ошибка при обработке медиагруппы', {
            chatId: mediaGroup[0]?.chat.id,
            mediaCount: mediaGroup?.length,
            error: error.message,
            stack: error.stack
        });
    }
}

module.exports = {
    handleMessage,
    handleMediaGroup
};
