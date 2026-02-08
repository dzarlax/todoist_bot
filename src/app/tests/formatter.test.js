const { formatTextWithLinks, findProjectNameForUser } = require('../utils/formatter');

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
