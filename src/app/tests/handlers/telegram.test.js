const TelegramBot = require('node-telegram-bot-api');
const TodoistClient = require('../../api/todoist');
const { handleMessage, handleMediaGroup } = require('../../handlers/telegram');

jest.mock('node-telegram-bot-api');
jest.mock('../../api/todoist');
jest.mock('../../utils/formatter', () => ({
  formatTextWithLinks: jest.fn((text) => text),
  findProjectNameForUser: jest.fn(() => null)
}));

describe('Telegram Handlers', () => {
  let bot, todoistClient, config, messageBuffer;

  beforeEach(() => {
    jest.clearAllMocks();

    bot = {
      sendMessage: jest.fn(),
      getFileLink: jest.fn(),
      isPolling: jest.fn(() => true)
    };

    todoistClient = {
      fetchProjects: jest.fn().mockResolvedValue({ Inbox: '1', Work: '2' }),
      createTask: jest.fn().mockResolvedValue({ id: '123' })
    };

    config = {
      timer: 5,
      autoAddDueDate: false,
      projectUsersMapping: {},
      maxRetries: 3,
      retryDelay: 1000
    };

    messageBuffer = new Map();
  });

  describe('handleMessage', () => {
    test('should ignore edited messages', async () => {
      const msg = {
        chat: { id: 123 },
        edit_date: 1234567890,
        text: 'test message'
      };

      await handleMessage(msg, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(false);
    });

    test('should ignore commands', async () => {
      const msg = {
        chat: { id: 123 },
        from: { username: 'testuser' },
        text: '/start'
      };

      await handleMessage(msg, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(false);
    });

    test('should handle text message from user', async () => {
      const msg = {
        chat: { id: 123 },
        from: { username: 'testuser', first_name: 'Test' },
        text: 'Test message'
      };

      await handleMessage(msg, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(true);
      expect(messageBuffer.get(123).messages).toContainEqual('@testuser: Test message');
    });

    test('should handle forwarded message', async () => {
      const msg = {
        chat: { id: 123 },
        forward_from: { username: 'originaluser', first_name: 'Original' },
        text: 'Forwarded message'
      };

      await handleMessage(msg, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(true);
      expect(messageBuffer.get(123).messages).toContainEqual('@originaluser: Forwarded message');
    });

    test('should handle forwarded message from channel', async () => {
      const msg = {
        chat: { id: 123 },
        forward_from_chat: { title: 'Test Channel' },
        text: 'Channel message'
      };

      await handleMessage(msg, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(true);
      expect(messageBuffer.get(123).messages).toContainEqual('Test Channel: Channel message');
    });

    test('should buffer multiple messages', async () => {
      const msg1 = {
        chat: { id: 123 },
        from: { username: 'testuser' },
        text: 'First message'
      };

      const msg2 = {
        chat: { id: 123 },
        from: { username: 'testuser' },
        text: 'Second message'
      };

      await handleMessage(msg1, bot, messageBuffer, todoistClient, config);
      await handleMessage(msg2, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.get(123).messages).toHaveLength(2);
    });
  });

  describe('handleMediaGroup', () => {
    test('should ignore empty media group', async () => {
      await handleMediaGroup([], bot, messageBuffer, todoistClient, config);

      expect(bot.sendMessage).not.toHaveBeenCalled();
    });

    test('should handle media group with photos', async () => {
      bot.getFileLink.mockResolvedValue('https://example.com/photo.jpg');

      const mediaGroup = [
        {
          chat: { id: 123 },
          from: { username: 'testuser' },
          photo: [{ file_id: 'file1' }, { file_id: 'file2' }],
          caption: 'Photo caption'
        }
      ];

      await handleMediaGroup(mediaGroup, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(true);
      expect(bot.getFileLink).toHaveBeenCalledWith('file2');
    });

    test('should handle media group with multiple items', async () => {
      bot.getFileLink
        .mockResolvedValueOnce('https://example.com/photo1.jpg')
        .mockResolvedValueOnce('https://example.com/photo2.jpg');

      const mediaGroup = [
        {
          chat: { id: 123 },
          from: { username: 'testuser' },
          photo: [{ file_id: 'file1' }]
        },
        {
          chat: { id: 123 },
          from: { username: 'testuser' },
          photo: [{ file_id: 'file2' }]
        }
      ];

      await handleMediaGroup(mediaGroup, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.get(123).messages).toHaveLength(2);
    });
  });

  describe('Task Creation', () => {
    test('should create task in Inbox when project not found', async () => {
      const msg = {
        chat: { id: 123 },
        from: { username: 'testuser', first_name: 'Test' },
        text: 'Test task'
      };

      await handleMessage(msg, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(true);

      const buffer = messageBuffer.get(123);
      clearTimeout(buffer.timer);

      expect(buffer.messages[0]).toContain('@testuser');
    });
  });

  describe('Error Handling', () => {
    test('should log error when getFileLink fails', async () => {
      bot.getFileLink.mockRejectedValue(new Error('ECONNRESET'));

      const msg = {
        chat: { id: 123 },
        from: { username: 'testuser', first_name: 'Test' },
        photo: [{ file_id: 'file1' }]
      };

      await handleMessage(msg, bot, messageBuffer, todoistClient, config);

      expect(messageBuffer.has(123)).toBe(true);
    });
  });
});
