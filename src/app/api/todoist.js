const axios = require('axios');
const { retryAsync } = require('../utils/retry');
const logger = require('../utils/logger');

class TodoistClient {
    constructor(token, maxRetries = 3, retryDelay = 1000) {
        this.token = token;
        this.baseUrl = 'https://api.todoist.com/api/v1';
        this.projectsCache = null;
        this.lastCacheUpdate = 0;
        this.cacheTTL = 1000 * 60 * 10; // 10 минут
        this.maxRetries = maxRetries;
        this.retryDelay = retryDelay;
    }

    get headers() {
        return {
            'Authorization': `Bearer ${this.token}`,
            'Content-Type': 'application/json'
        };
    }

    /**
     * Получение списка проектов с кэшированием
     */
    async fetchProjects() {
        const now = Date.now();
        if (this.projectsCache && (now - this.lastCacheUpdate < this.cacheTTL)) {
            return this.projectsCache;
        }

        try {
            const response = await retryAsync(
                async () => axios.get(`${this.baseUrl}/projects`, { headers: this.headers }),
                {
                    maxRetries: this.maxRetries,
                    initialDelay: this.retryDelay
                }
            );
            const data = response.data;
            const list = (data && data.results) ? data.results : data;
            const projects = {};
            list.forEach(project => {
                projects[project.name] = project.id;
            });

            this.projectsCache = projects;
            this.lastCacheUpdate = now;
            logger.info('Список проектов Todoist обновлен', { projectCount: Object.keys(projects).length });
            return projects;
        } catch (error) {
            logger.error("Ошибка при получении проектов", {
                error: error.message,
                status: error.response?.status
            });
            return this.projectsCache || {};
        }
    }

    /**
     * Получение активных задач
     */
    async getTasks(projectId = null) {
        const params = projectId ? { project_id: projectId } : {};
        const response = await axios.get(`${this.baseUrl}/tasks`, { headers: this.headers, params });
        const data = response.data;
        return (data && data.results) ? data.results : data;
    }

    /**
     * Создание задачи
     */
    async createTask(taskData) {
        try {
            const response = await retryAsync(
                async () => axios.post(`${this.baseUrl}/tasks`, taskData, { headers: this.headers }),
                {
                    maxRetries: this.maxRetries,
                    initialDelay: this.retryDelay
                }
            );
            logger.info('Задача создана в Todoist', {
                taskId: response.data.id,
                projectId: taskData.project_id,
                content: taskData.content
            });
            return response.data;
        } catch (error) {
            logger.error("Ошибка при добавлении задачи в Todoist", {
                error: error.message,
                status: error.response?.status,
                responseData: error.response?.data,
                taskData: { content: taskData.content, projectId: taskData.project_id }
            });
            throw error;
        }
    }
}

module.exports = TodoistClient;
