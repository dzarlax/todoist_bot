/**
 * Async retry wrapper with exponential backoff
 * @param {Function} fn - Async function to retry
 * @param {Object} options - Retry configuration
 * @returns {Promise} Result of fn
 */
async function retryAsync(fn, options = {}) {
  const maxRetries = options.maxRetries || 3;
  const initialDelay = options.initialDelay || 1000;
  const retryableErrors = options.retryableErrors || [
    'ECONNRESET', 'ETIMEDOUT', 'ENOTFOUND', 'ECONNREFUSED'
  ];
  const retryableStatuses = options.retryableStatuses || [429, 500, 502, 503, 504];

  let lastError;
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      if (attempt === maxRetries) break;

      const shouldRetry = retryableErrors.includes(error.code) ||
                         retryableStatuses.includes(error.response?.status);

      if (!shouldRetry) break;

      const delay = initialDelay * Math.pow(2, attempt);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
  throw lastError;
}

module.exports = { retryAsync };
