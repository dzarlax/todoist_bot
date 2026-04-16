package telegram

import (
	"context"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dzarlax/todoist-bot/internal/config"
	"github.com/dzarlax/todoist-bot/internal/todoist"
)

// mediaGroupFlushDelay is how long we wait for more parts of an album before
// treating the group as complete.
const mediaGroupFlushDelay = 800 * time.Millisecond

type Bot struct {
	api          *tgbotapi.BotAPI
	cfg          *config.Config
	client       *todoist.Client
	projectCache *todoist.ProjectCache
	buffers      *bufferStore
}

func NewBot(cfg *config.Config, client *todoist.Client, cache *todoist.ProjectCache) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, err
	}
	return &Bot{
		api:          api,
		cfg:          cfg,
		client:       client,
		projectCache: cache,
		buffers:      newBufferStore(),
	}, nil
}

// Run blocks until ctx is cancelled, polling Telegram for updates and
// dispatching them.
func (b *Bot) Run(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.api.GetUpdatesChan(u)

	slog.Info("telegram bot started", "user", b.api.Self.UserName)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			b.flushAll(context.Background()) // best-effort flush
			return
		case upd, ok := <-updates:
			if !ok {
				return
			}
			b.handleUpdate(ctx, upd)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, upd tgbotapi.Update) {
	msg := upd.Message
	if msg == nil {
		return
	}
	if msg.EditDate != 0 {
		return // edits are ignored
	}

	// Access control check happens first except for unauthenticated users
	// sending /start (we silently drop unknowns; matches old behaviour).
	username := ""
	if msg.From != nil {
		username = msg.From.UserName
	}
	if !b.cfg.IsAllowed(username) {
		return
	}

	if msg.IsCommand() {
		b.handleCommand(ctx, msg)
		return
	}

	if msg.MediaGroupID != "" {
		b.enqueueMediaGroup(ctx, msg)
		return
	}

	b.handleMessage(ctx, msg)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = tgbotapi.ModeMarkdown
	m.DisableWebPagePreview = true
	if _, err := b.api.Send(m); err != nil {
		slog.Error("telegram send failed", "chat_id", chatID, "error", err)
	}
}

func (b *Bot) sendPlain(chatID int64, text string) {
	if _, err := b.api.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		slog.Error("telegram send failed", "chat_id", chatID, "error", err)
	}
}

// senderName extracts the display name for attribution in the task description.
// Forwarded messages resolve to the original author; direct messages use the
// Telegram sender.
func senderName(msg *tgbotapi.Message) string {
	if msg.ForwardFrom != nil {
		if msg.ForwardFrom.UserName != "" {
			return "@" + msg.ForwardFrom.UserName
		}
		return joinName(msg.ForwardFrom.FirstName, msg.ForwardFrom.LastName)
	}
	if msg.ForwardFromChat != nil && msg.ForwardFromChat.Title != "" {
		return msg.ForwardFromChat.Title
	}
	if msg.From != nil {
		if msg.From.UserName != "" {
			return "@" + msg.From.UserName
		}
		return joinName(msg.From.FirstName, msg.From.LastName)
	}
	return "неизвестный"
}

func joinName(first, last string) string {
	switch {
	case first != "" && last != "":
		return first + " " + last
	case first != "":
		return first
	case last != "":
		return last
	default:
		return ""
	}
}

// flushAll fires any pending timers so buffered tasks aren't lost on shutdown.
// Best-effort — errors are logged.
func (b *Bot) flushAll(ctx context.Context) {
	b.buffers.mu.Lock()
	chatIDs := make([]int64, 0, len(b.buffers.tasks))
	for id := range b.buffers.tasks {
		chatIDs = append(chatIDs, id)
	}
	b.buffers.mu.Unlock()

	for _, id := range chatIDs {
		b.flushTaskBuffer(ctx, id)
	}
}

// helpers for command argument parsing

func commandArgs(msg *tgbotapi.Message) string {
	// tgbotapi's msg.CommandArguments() already strips the command token.
	return strings.TrimSpace(msg.CommandArguments())
}
