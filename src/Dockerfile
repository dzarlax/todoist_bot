# Используйте официальный образ Node.js как базовый
FROM node:14

# Установите рабочую директорию в контейнере
WORKDIR /usr/src/app

# Копируйте файлы package.json и package-lock.json
COPY package*.json ./

# Установите зависимости проекта
RUN npm install

# Копируйте исходный код проекта в рабочую директорию
COPY . .

# Задайте команду для запуска бота
CMD [ "node", "todoist_bot.js" ]
