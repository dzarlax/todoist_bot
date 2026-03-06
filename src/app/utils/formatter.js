/**
 * Функция для определения проекта по имени пользователя
 */
function findProjectNameForUser(username, projectToUsersMapping) {
    for (const [projectName, users] of Object.entries(projectToUsersMapping)) {
        if (users.includes(username)) {
            return projectName;
        }
    }
    return null;
}

/**
 * Функция для форматирования текста с учетом ссылок
 */
function formatTextWithLinks(text, entities) {
    if (!text) return text;

    let processedText = text;

    // 1. Сначала обрабатываем сущности (text_link), если они есть
    if (entities && Array.isArray(entities) && entities.length > 0) {
        const sortedEntities = [...entities].sort((a, b) => b.offset - a.offset);
        sortedEntities.forEach(entity => {
            if (entity.type === 'text_link' && entity.url) {
                const linkText = processedText.substring(entity.offset, entity.offset + entity.length);
                const markdownLink = `[${linkText}](${entity.url})`;
                processedText =
                    processedText.substring(0, entity.offset) +
                    markdownLink +
                    processedText.substring(entity.offset + entity.length);
            }
        });
    }

    // 2. Теперь ищем обычные URL, но пропускаем те, что уже являются частью Markdown-ссылок
    // Регулярное выражение: либо находим существующую markdown-ссылку [..](..), либо URL
    // Группа 1: существующая ссылка
    // Группа 2: обычный URL
    const combinedRegex = /(\[[^\]]+\]\([^)]+\))|((?:https?:\/\/|www\.)[^\s]+(?:\.[^\s]+)+)/gi;

    return processedText.replace(combinedRegex, (match, g1, g2) => {
        // Если сработала группа 1 (уже ссылка), возвращаем как есть
        if (g1) return g1;

        // Если сработала группа 2 (чистый URL), форматируем его
        if (g2) {
            const fullUrl = g2.startsWith('www.') ? `https://${g2}` : g2;
            return `[${g2}](${fullUrl})`;
        }

        return match;
    });
}

/**
 * Извлекает приоритет (!p1-!p4) и ярлыки (#tag) из текста задачи.
 * Возвращает очищенный текст без маркеров.
 */
function parseTaskMeta(text) {
    if (!text) return { priority: 1, labels: [], cleanText: text };

    let priority = 1;
    const priorityMatch = text.match(/(?:^|\s)!p([1-4])(?=\s|$)/i);
    if (priorityMatch) {
        priority = parseInt(priorityMatch[1], 10);
    }

    const labels = [];
    const labelRegex = /(?:^|\s)#(\w+)/g;
    let match;
    while ((match = labelRegex.exec(text)) !== null) {
        labels.push(match[1]);
    }

    const cleanText = text
        .replace(/(^|\s)!p[1-4](?=\s|$)/gi, '$1')
        .replace(/(^|\s)#\w+/g, '$1')
        .replace(/\s+/g, ' ')
        .trim();

    return { priority, labels, cleanText };
}

module.exports = {
    findProjectNameForUser,
    formatTextWithLinks,
    parseTaskMeta
};
