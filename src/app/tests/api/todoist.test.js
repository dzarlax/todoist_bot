const axios = require('axios');
const TodoistClient = require('../../api/todoist');

jest.mock('axios');

describe('TodoistClient', () => {
  let todoistClient;

  beforeEach(() => {
    jest.clearAllMocks();
    todoistClient = new TodoistClient('test-token', 3, 1000);
  });

  test('should initialize with correct config', () => {
    expect(todoistClient.token).toBe('test-token');
    expect(todoistClient.baseUrl).toBe('https://api.todoist.com/api/v1');
    expect(todoistClient.maxRetries).toBe(3);
    expect(todoistClient.retryDelay).toBe(1000);
    expect(todoistClient.cacheTTL).toBe(1000 * 60 * 10);
  });

  test('should return correct headers', () => {
    const headers = todoistClient.headers;
    expect(headers).toEqual({
      'Authorization': 'Bearer test-token',
      'Content-Type': 'application/json'
    });
  });

  test('should fetch and cache projects', async () => {
    const mockProjects = [
      { id: '1', name: 'Work' },
      { id: '2', name: 'Personal' }
    ];
    axios.get.mockResolvedValue({ data: { results: mockProjects } });

    const result = await todoistClient.fetchProjects();

    expect(result).toEqual({ Work: '1', Personal: '2' });
    expect(axios.get).toHaveBeenCalledTimes(1);
    expect(axios.get).toHaveBeenCalledWith(
      'https://api.todoist.com/api/v1/projects',
      { headers: todoistClient.headers }
    );
  });

  test('should use cache on repeated calls within TTL', async () => {
    const mockProjects = [{ id: '1', name: 'Work' }];
    axios.get.mockResolvedValue({ data: { results: mockProjects } });

    await todoistClient.fetchProjects();
    const result = await todoistClient.fetchProjects();

    expect(result).toEqual({ Work: '1' });
    expect(axios.get).toHaveBeenCalledTimes(1);
  });

  test('should fetch new projects after cache expires', async () => {
    jest.useFakeTimers();

    const mockProjects1 = [{ id: '1', name: 'Work' }];
    const mockProjects2 = [{ id: '2', name: 'Personal' }];
    axios.get
      .mockResolvedValueOnce({ data: { results: mockProjects1 } })
      .mockResolvedValueOnce({ data: { results: mockProjects2 } });

    const result1 = await todoistClient.fetchProjects();
    expect(result1).toEqual({ Work: '1' });

    jest.advanceTimersByTime(11 * 60 * 1000);

    const result2 = await todoistClient.fetchProjects();
    expect(result2).toEqual({ Personal: '2' });

    expect(axios.get).toHaveBeenCalledTimes(2);

    jest.useRealTimers();
  });

  test('should return cached projects on API error', async () => {
    const mockProjects = [{ id: '1', name: 'Work' }];
    axios.get
      .mockResolvedValueOnce({ data: { results: mockProjects } })
      .mockRejectedValueOnce(new Error('Network error'));

    const result1 = await todoistClient.fetchProjects();
    expect(result1).toEqual({ Work: '1' });

    const result2 = await todoistClient.fetchProjects();
    expect(result2).toEqual({ Work: '1' });
  });

  test('should return empty object on initial API error', async () => {
    axios.get.mockRejectedValue(new Error('Network error'));

    const result = await todoistClient.fetchProjects();

    expect(result).toEqual({});
  });

  test('should create task successfully', async () => {
    const mockTask = {
      id: '123',
      content: 'Test task',
      project_id: '1'
    };
    axios.post.mockResolvedValue({ data: mockTask });

    const taskData = {
      content: 'Test task',
      description: 'Test description',
      project_id: '1'
    };

    const result = await todoistClient.createTask(taskData);

    expect(result).toEqual(mockTask);
    expect(axios.post).toHaveBeenCalledWith(
      'https://api.todoist.com/api/v1/tasks',
      taskData,
      { headers: todoistClient.headers }
    );
  });

  test('should handle error when creating task', async () => {
    const error = new Error('API Error');
    error.response = {
      status: 400,
      data: { error: 'Invalid request' }
    };
    axios.post.mockRejectedValue(error);

    const taskData = {
      content: 'Test task',
      project_id: '1'
    };

    await expect(todoistClient.createTask(taskData)).rejects.toThrow('API Error');
  });

  test('should use custom retry configuration', () => {
    const client = new TodoistClient('token', 5, 2000);
    expect(client.maxRetries).toBe(5);
    expect(client.retryDelay).toBe(2000);
  });

  test('should use default retry configuration', () => {
    const client = new TodoistClient('token');
    expect(client.maxRetries).toBe(3);
    expect(client.retryDelay).toBe(1000);
  });
});
