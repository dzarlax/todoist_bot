const express = require('express');
const logger = require('./utils/logger');

const PORT = process.env.PORT || 3000;

function createHealthCheckServer(todoistClient, bot) {
  const app = express();

  app.get('/health', (req, res) => {
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
  });

  app.get('/ready', async (req, res) => {
    const checks = {
      telegram: bot.isPolling?.() || true,
      todoist: false,
      config: true
    };

    try {
      await todoistClient.fetchProjects();
      checks.todoist = true;
    } catch (error) {
      logger.error('Readiness check failed for Todoist', { error: error.message });
    }

    const allHealthy = Object.values(checks).every(v => v);
    res.status(allHealthy ? 200 : 503).json({
      status: allHealthy ? 'ready' : 'not_ready',
      checks,
      timestamp: new Date().toISOString()
    });
  });

  const server = app.listen(PORT, () => {
    logger.info(`Health check server started on port ${PORT}`);
  });

  return server;
}

module.exports = { createHealthCheckServer };
