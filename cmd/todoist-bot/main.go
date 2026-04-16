package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"

	"github.com/dzarlax/todoist-bot/internal/config"
	"github.com/dzarlax/todoist-bot/internal/middleware"
	"github.com/dzarlax/todoist-bot/internal/telegram"
	"github.com/dzarlax/todoist-bot/internal/todoist"
)

func main() {
	_ = godotenv.Load() // optional; ignore if absent

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	slog.Info("starting todoist-bot",
		"port", cfg.Port,
		"timer_sec", cfg.Timer,
		"auto_due", cfg.AutoAddDueDate,
		"projects", len(cfg.ProjectUsers),
		"whitelist", len(cfg.AllowedUsernames),
	)

	todoistClient := todoist.NewClient(cfg.TodoistToken)
	projectCache := todoist.NewProjectCache(todoistClient, 10*time.Minute)

	bot, err := telegram.NewBot(cfg, todoistClient, projectCache)
	if err != nil {
		slog.Error("telegram init failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- HTTP / MCP server ------------------------------------------------
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mcpSrv := server.NewMCPServer("todoist-bot", "1.0.0",
		server.WithToolCapabilities(true),
	)
	todoist.NewMCPServer(todoistClient).RegisterTools(mcpSrv)
	mcpHTTP := server.NewStreamableHTTPServer(mcpSrv)

	r.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(cfg.APIKey))
		r.Handle("/todoist", mcpHTTP)
		r.Handle("/todoist/", mcpHTTP)
	})

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// --- run bot + server -------------------------------------------------
	go bot.Run(ctx)

	go func() {
		slog.Info("http server listening", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "error", err)
	}
}
