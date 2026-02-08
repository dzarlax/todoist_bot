const { config, validateConfig } = require('../../config');

describe('Configuration', () => {
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    jest.resetModules();
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  test('should load basic config from environment', () => {
    process.env.TELEGRAM_TOKEN = 'test-telegram-token';
    process.env.TODOIST_TOKEN = 'test-todoist-token';
    process.env.TIMER = '10';
    process.env.AUTO_ADD_DUE_DATE = 'true';

    const { config: testConfig } = require('../../config');

    expect(testConfig.telegramToken).toBe('test-telegram-token');
    expect(testConfig.todoistToken).toBe('test-todoist-token');
    expect(testConfig.timer).toBe(10);
    expect(testConfig.autoAddDueDate).toBe(true);
  });

  test('should parse PROJECT_USERS_* environment variables', () => {
    process.env.TELEGRAM_TOKEN = 'test';
    process.env.TODOIST_TOKEN = 'test';
    process.env.PROJECT_USERS_WORK = '@user1,@user2';
    process.env.PROJECT_USERS_PERSONAL = '@user3';

    const { config: testConfig } = require('../../config');

    expect(testConfig.projectUsersMapping).toEqual({
      WORK: ['@user1', '@user2'],
      PERSONAL: ['@user3']
    });
  });

  test('should handle default values', () => {
    process.env.TELEGRAM_TOKEN = 'test';
    process.env.TODOIST_TOKEN = 'test';

    const { config: testConfig } = require('../../config');

    expect(testConfig.timer).toBe(5);
    expect(testConfig.autoAddDueDate).toBe(false);
    expect(testConfig.maxRetries).toBe(3);
    expect(testConfig.retryDelay).toBe(1000);
  });

  test('should parse retry configuration', () => {
    process.env.TELEGRAM_TOKEN = 'test';
    process.env.TODOIST_TOKEN = 'test';
    process.env.MAX_RETRIES = '5';
    process.env.RETRY_DELAY = '2000';

    const { config: testConfig } = require('../../config');

    expect(testConfig.maxRetries).toBe(5);
    expect(testConfig.retryDelay).toBe(2000);
  });

  test('should exit on missing TELEGRAM_TOKEN', () => {
    delete process.env.TELEGRAM_TOKEN;
    process.env.TODOIST_TOKEN = 'test-token';

    const { validateConfig: testValidateConfig } = require('../../config');

    const exitSpy = jest.spyOn(process, 'exit').mockImplementation(() => {
      throw new Error('process.exit called');
    });

    expect(() => testValidateConfig()).toThrow('process.exit called');
    expect(exitSpy).toHaveBeenCalledWith(1);

    exitSpy.mockRestore();
  });

  test('should exit on missing TODOIST_TOKEN', () => {
    process.env.TELEGRAM_TOKEN = 'test-token';
    delete process.env.TODOIST_TOKEN;

    const { validateConfig: testValidateConfig } = require('../../config');

    const exitSpy = jest.spyOn(process, 'exit').mockImplementation(() => {
      throw new Error('process.exit called');
    });

    expect(() => testValidateConfig()).toThrow('process.exit called');
    expect(exitSpy).toHaveBeenCalledWith(1);

    exitSpy.mockRestore();
  });

  test('should handle project names with special characters', () => {
    process.env.TELEGRAM_TOKEN = 'test';
    process.env.TODOIST_TOKEN = 'test';
    process.env['PROJECT_USERS_Project With Spaces'] = '@user1,@user2';

    const { config: testConfig } = require('../../config');

    expect(testConfig.projectUsersMapping['Project With Spaces']).toEqual(['@user1', '@user2']);
  });

  test('should trim whitespace from usernames', () => {
    process.env.TELEGRAM_TOKEN = 'test';
    process.env.TODOIST_TOKEN = 'test';
    process.env.PROJECT_USERS_WORK = ' @user1 , @user2 , @user3 ';

    const { config: testConfig } = require('../../config');

    expect(testConfig.projectUsersMapping.WORK).toEqual(['@user1', '@user2', '@user3']);
  });
});
