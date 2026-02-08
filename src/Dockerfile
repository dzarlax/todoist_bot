# Используйте официальный образ Node.js облегченной версии
FROM node:20-alpine

# Установите рабочую директорию
WORKDIR /usr/src/app

# Создаем непривилегированного пользователя
RUN addgroup -S nodegroup && adduser -S nodeuser -G nodegroup

# Копируйте файлы package.json и package-lock.json
COPY src/package*.json ./

# Установите только production зависимости
RUN npm install --omit=dev

# Создаем директорию для логов
RUN mkdir -p logs && chown -R nodeuser:nodegroup logs

# Копируем исходный код проекта
COPY --chown=nodeuser:nodegroup src/ .

# Переключаемся на непривилегированного пользователя
USER nodeuser

# Expose port for health checks
EXPOSE 3000

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:3000/health', (r) => {process.exit(r.statusCode === 200 ? 0 : 1)})"

# Команда для запуска
CMD [ "npm", "start" ]
