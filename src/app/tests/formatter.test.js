const { formatTextWithLinks, findProjectNameForUser, parseTaskMeta } = require('../utils/formatter');

describe('Formatter Utility', () => {
    test('formatTextWithLinks should format plain URLs', () => {
        const text = 'Check this site: https://google.com and www.example.com';
        const formatted = formatTextWithLinks(text, []);
        expect(formatted).toContain('[https://google.com](https://google.com)');
        expect(formatted).toContain('[www.example.com](https://www.example.com)');
    });

    test('formatTextWithLinks should preserve existing markdown links', () => {
        const text = 'Link [Google](https://google.com) and plain https://yahoo.com';
        const formatted = formatTextWithLinks(text, []);
        expect(formatted).toBe('Link [Google](https://google.com) and plain [https://yahoo.com](https://yahoo.com)');
    });

    test('parseTaskMeta should extract priority', () => {
        const { priority, labels, cleanText } = parseTaskMeta('Купить молоко !p2');
        expect(priority).toBe(2);
        expect(labels).toEqual([]);
        expect(cleanText).toBe('Купить молоко');
    });

    test('parseTaskMeta should extract labels', () => {
        const { priority, labels, cleanText } = parseTaskMeta('Встреча #work #urgent');
        expect(priority).toBe(1);
        expect(labels).toEqual(['work', 'urgent']);
        expect(cleanText).toBe('Встреча');
    });

    test('parseTaskMeta should extract both priority and labels', () => {
        const { priority, labels, cleanText } = parseTaskMeta('Задача !p3 #personal #home');
        expect(priority).toBe(3);
        expect(labels).toEqual(['personal', 'home']);
        expect(cleanText).toBe('Задача');
    });

    test('parseTaskMeta should return defaults when no markers', () => {
        const { priority, labels, cleanText } = parseTaskMeta('Обычная задача');
        expect(priority).toBe(1);
        expect(labels).toEqual([]);
        expect(cleanText).toBe('Обычная задача');
    });

    test('parseTaskMeta should not match hashtags inside URLs', () => {
        const { labels, cleanText } = parseTaskMeta('[link](https://example.com/path#section)');
        expect(labels).toEqual([]);
        expect(cleanText).toBe('[link](https://example.com/path#section)');
    });

    test('findProjectNameForUser should return correct project name', () => {
        const mapping = {
            'Work': ['@boss', 'Boss Name'],
            'Personal': ['@wife']
        };
        expect(findProjectNameForUser('@boss', mapping)).toBe('Work');
        expect(findProjectNameForUser('Boss Name', mapping)).toBe('Work');
        expect(findProjectNameForUser('@wife', mapping)).toBe('Personal');
        expect(findProjectNameForUser('@unknown', mapping)).toBeNull();
    });
});
