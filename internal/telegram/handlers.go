package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dzarlax/todoist-bot/internal/formatter"
)

// ---- commands ----------------------------------------------------------

func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.sendPlain(msg.Chat.ID, "Привет! Я бот для добавления задач в Todoist. Просто перешли мне сообщение или напиши текст, и я создам задачу.")
	case "help":
		help := "Доступные команды:\n/start - Начать работу\n/help - Справка\n/status - Проверить статус"
		if b.cfg.IsAdmin(usernameOf(msg)) {
			help += "\n/list [проект] - Список активных задач"
		}
		b.sendPlain(msg.Chat.ID, help)
	case "status":
		user := "@" + msg.From.UserName
		if msg.From.UserName == "" {
			user = msg.From.FirstName
		}
		role := ""
		if b.cfg.IsAdmin(usernameOf(msg)) {
			role = " (admin)"
		}
		autoDate := "Выкл"
		if b.cfg.AutoAddDueDate {
			autoDate = "Вкл"
		}
		b.sendPlain(msg.Chat.ID, fmt.Sprintf(
			"Бот активен.\nВы вошли как: %s%s\nТаймер склейки: %d сек.\nАвто-дата: %s",
			user, role, b.cfg.Timer, autoDate))
	case "list":
		b.handleListCommand(ctx, msg)
	}
}

func (b *Bot) handleListCommand(ctx context.Context, msg *tgbotapi.Message) {
	if !b.cfg.IsAdmin(usernameOf(msg)) {
		return
	}

	projectFilter := commandArgs(msg)
	var projectID string
	var projectsByID map[string]string

	projects, err := b.projectCache.All(ctx)
	if err != nil {
		slog.Error("fetch projects failed", "error", err)
		b.sendPlain(msg.Chat.ID, "❌ Ошибка при получении задач.")
		return
	}
	projectsByID = make(map[string]string, len(projects))
	for _, p := range projects {
		projectsByID[p.ID] = p.Name
		if projectFilter != "" && strings.EqualFold(p.Name, projectFilter) {
			projectID = p.ID
		}
	}
	if projectFilter != "" && projectID == "" {
		b.sendPlain(msg.Chat.ID, fmt.Sprintf(`Проект "%s" не найден.`, projectFilter))
		return
	}

	tasks, err := b.client.ListTasks(ctx, projectID, 50)
	if err != nil {
		slog.Error("fetch tasks failed", "error", err)
		b.sendPlain(msg.Chat.ID, "❌ Ошибка при получении задач.")
		return
	}
	if len(tasks) == 0 {
		b.sendPlain(msg.Chat.ID, "Нет активных задач.")
		return
	}
	if len(tasks) > 20 {
		tasks = tasks[:20]
	}

	var lines []string
	for _, t := range tasks {
		line := "• " + t.Content
		if t.Priority > 1 {
			line += fmt.Sprintf(" [p%d]", t.Priority)
		}
		if t.Due != nil && t.Due.Date != "" {
			line += " 📅 " + t.Due.Date
		}
		if name := projectsByID[t.ProjectID]; name != "" {
			line += " — " + name
		}
		lines = append(lines, line)
	}
	b.sendPlain(msg.Chat.ID, strings.Join(lines, "\n"))
}

// ---- plain messages ----------------------------------------------------

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	sender := senderName(msg)
	content, err := b.renderMessageContent(msg)
	if err != nil {
		slog.Error("render message failed", "error", err, "chat_id", msg.Chat.ID)
		return
	}
	if content == "" {
		content = "[неизвестный тип медиа]"
	}
	b.appendToBuffer(ctx, msg.Chat.ID, sender, content, true)
}

// renderMessageContent turns a single Telegram message into its markdown body.
func (b *Bot) renderMessageContent(msg *tgbotapi.Message) (string, error) {
	if msg.Text != "" {
		return formatter.FormatTextWithLinks(msg.Text, msg.Entities), nil
	}

	media, err := getMediaLink(b.api, msg)
	if err != nil {
		return "", err
	}

	if msg.Caption != "" {
		body := formatter.FormatTextWithLinks(msg.Caption, msg.CaptionEntities)
		if media != nil {
			body += " " + media.asMarkdown()
		}
		return body, nil
	}
	if media != nil {
		return media.asMarkdown(), nil
	}
	return "", nil
}

// appendToBuffer adds a prepared line to the chat's task buffer, (re)starting
// the debounce timer. If parseFirstMeta is true and the buffer is empty, priority
// and labels are extracted from the content.
func (b *Bot) appendToBuffer(ctx context.Context, chatID int64, sender, content string, parseFirstMeta bool) {
	b.buffers.mu.Lock()
	defer b.buffers.mu.Unlock()

	buf, exists := b.buffers.tasks[chatID]
	if !exists {
		priority := 1
		var labels []string
		cleaned := content
		if parseFirstMeta {
			priority, labels, cleaned = formatter.ParseTaskMeta(content)
		}
		line := fmt.Sprintf("%s: %s", sender, cleaned)
		buf = &taskBuffer{
			messages: []string{line},
			priority: priority,
			labels:   labels,
		}
		b.buffers.tasks[chatID] = buf
	} else {
		buf.messages = append(buf.messages, fmt.Sprintf("%s: %s", sender, content))
		if buf.timer != nil {
			buf.timer.Stop()
		}
	}

	buf.timer = time.AfterFunc(time.Duration(b.cfg.Timer)*time.Second, func() {
		b.flushTaskBuffer(ctx, chatID)
	})
}

// ---- media groups ------------------------------------------------------

func (b *Bot) enqueueMediaGroup(ctx context.Context, msg *tgbotapi.Message) {
	id := msg.MediaGroupID
	b.buffers.mu.Lock()
	defer b.buffers.mu.Unlock()

	g, ok := b.buffers.groups[id]
	if !ok {
		g = &mediaGroupBuffer{}
		b.buffers.groups[id] = g
	}
	g.messages = append(g.messages, msg)
	if g.timer != nil {
		g.timer.Stop()
	}
	g.timer = time.AfterFunc(mediaGroupFlushDelay, func() {
		b.processMediaGroup(ctx, id)
	})
}

func (b *Bot) processMediaGroup(ctx context.Context, groupID string) {
	b.buffers.mu.Lock()
	g, ok := b.buffers.groups[groupID]
	if !ok {
		b.buffers.mu.Unlock()
		return
	}
	delete(b.buffers.groups, groupID)
	b.buffers.mu.Unlock()

	if len(g.messages) == 0 {
		return
	}

	first := g.messages[0].(*tgbotapi.Message)
	chatID := first.Chat.ID
	sender := senderName(first)

	// Resolve media for every part.
	infos := make([]*mediaInfo, 0, len(g.messages))
	for _, raw := range g.messages {
		m := raw.(*tgbotapi.Message)
		info, err := getMediaLink(b.api, m)
		if err != nil {
			slog.Warn("media link failed", "error", err)
			continue
		}
		infos = append(infos, info)
	}

	priority := 1
	var labels []string
	var firstLine string
	if first.Caption != "" {
		caption := formatter.FormatTextWithLinks(first.Caption, first.CaptionEntities)
		var cleaned string
		priority, labels, cleaned = formatter.ParseTaskMeta(caption)
		body := cleaned
		if len(infos) > 0 && infos[0] != nil {
			body += " " + infos[0].asMarkdown()
		}
		firstLine = fmt.Sprintf("%s: %s", sender, body)
	} else {
		body := "[медиа группа]"
		if len(infos) > 0 && infos[0] != nil {
			body = infos[0].asMarkdown()
		}
		firstLine = fmt.Sprintf("%s: %s", sender, body)
	}

	// Additional parts: any caption on subsequent messages + their media.
	var extraLines []string
	for i := 1; i < len(g.messages); i++ {
		m := g.messages[i].(*tgbotapi.Message)
		var info *mediaInfo
		if i < len(infos) {
			info = infos[i]
		}
		if m.Caption != "" {
			body := formatter.FormatTextWithLinks(m.Caption, m.CaptionEntities)
			if info != nil {
				body += " " + info.asMarkdown()
			}
			extraLines = append(extraLines, fmt.Sprintf("%s: %s", sender, body))
		} else if info != nil {
			extraLines = append(extraLines, fmt.Sprintf("%s: %s", sender, info.asMarkdown()))
		}
	}

	b.buffers.mu.Lock()
	defer b.buffers.mu.Unlock()
	buf, exists := b.buffers.tasks[chatID]
	if !exists {
		buf = &taskBuffer{
			messages: append([]string{firstLine}, extraLines...),
			priority: priority,
			labels:   labels,
		}
		b.buffers.tasks[chatID] = buf
	} else {
		buf.messages = append(buf.messages, firstLine)
		buf.messages = append(buf.messages, extraLines...)
		if buf.timer != nil {
			buf.timer.Stop()
		}
	}
	buf.timer = time.AfterFunc(time.Duration(b.cfg.Timer)*time.Second, func() {
		b.flushTaskBuffer(ctx, chatID)
	})
}

// ---- task creation -----------------------------------------------------

func (b *Bot) flushTaskBuffer(ctx context.Context, chatID int64) {
	b.buffers.mu.Lock()
	buf, ok := b.buffers.tasks[chatID]
	if !ok {
		b.buffers.mu.Unlock()
		return
	}
	delete(b.buffers.tasks, chatID)
	b.buffers.mu.Unlock()

	if len(buf.messages) == 0 {
		return
	}

	title := buf.messages[0]
	description := strings.Join(buf.messages[1:], "\n")

	// Extract sender prefix ("@user: …" or "Name: …") to look up project.
	sender := title
	if idx := strings.Index(title, ": "); idx > 0 {
		sender = title[:idx]
	}

	projectName := formatter.FindProjectNameForUser(sender, b.cfg.ProjectUsers)
	notifyFallback := false
	if projectName == "" {
		projectName = "Inbox"
		notifyFallback = true
	}

	projectID, err := b.projectCache.IDByName(ctx, projectName)
	if err != nil {
		slog.Error("project lookup failed", "error", err, "project", projectName)
		b.sendPlain(chatID, "❌ Ошибка при добавлении задачи в Todoist.")
		return
	}
	if projectID == "" {
		slog.Error("project not found in todoist", "project", projectName, "chat_id", chatID)
		b.sendPlain(chatID, fmt.Sprintf(`Проект "%s" не найден.`, projectName))
		return
	}
	if notifyFallback {
		b.sendPlain(chatID, fmt.Sprintf(`Проект для пользователя "%s" не найден, задача будет помещена во входящие.`, sender))
	}

	task := map[string]interface{}{
		"content":     title,
		"description": description,
		"project_id":  projectID,
	}
	if b.cfg.AutoAddDueDate {
		task["due_date"] = time.Now().Format("2006-01-02")
	}
	if buf.priority > 1 {
		task["priority"] = buf.priority
	}
	if len(buf.labels) > 0 {
		task["labels"] = buf.labels
	}

	if _, err := b.client.CreateTask(ctx, task); err != nil {
		slog.Error("create task failed", "error", err, "project", projectName, "chat_id", chatID)
		b.sendPlain(chatID, "❌ Ошибка при добавлении задачи в Todoist.")
		return
	}

	// Success confirmation mirrors original format.
	priorityStr := ""
	if buf.priority > 1 {
		priorityStr = fmt.Sprintf(" [p%d]", buf.priority)
	}
	labelsStr := ""
	if len(buf.labels) > 0 {
		labelsStr = " [" + strings.Join(buf.labels, ", ") + "]"
	}
	dueStr := ""
	if b.cfg.AutoAddDueDate {
		dueStr = " (срок на сегодня)"
	}
	b.sendPlain(chatID, fmt.Sprintf(`✅ Задача добавлена в "%s"%s%s%s.`, projectName, priorityStr, labelsStr, dueStr))
}

// usernameOf returns the sender's username (no '@'), or "".
func usernameOf(msg *tgbotapi.Message) string {
	if msg == nil || msg.From == nil {
		return ""
	}
	return msg.From.UserName
}
