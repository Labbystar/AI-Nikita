package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"meeting-assistant-go/internal/adapter/zoom"
	"meeting-assistant-go/internal/bot"
	"meeting-assistant-go/internal/config"
	"meeting-assistant-go/internal/service"
	"meeting-assistant-go/internal/storage"
)

type App struct {
	cfg config.Config
}

func New(cfg config.Config) *App { return &App{cfg: cfg} }

func (a *App) Run(ctx context.Context) error {
	repo, err := storage.NewSQLiteRepository(a.cfg.DBPath)
	if err != nil {
		return err
	}
	defer repo.Close()

	telegramBot, err := tgbotapi.NewBotAPI(a.cfg.BotToken)
	if err != nil {
		return err
	}
	telegramBot.Debug = false
	log.Printf("authorized on telegram as @%s", telegramBot.Self.UserName)

	var transcriptService service.TranscriptService = &service.StubTranscriptService{}
	if a.cfg.OpenAIAPIKey != "" {
		transcriptService = service.NewOpenAITranscriptService(a.cfg.OpenAIAPIKey, a.cfg.OpenAIBaseURL, a.cfg.OpenAITranscribeModel)
	}

	protocolService := service.ProtocolService(&service.HeuristicProtocolService{})
	if a.cfg.OpenAIAPIKey != "" {
		protocolService = service.NewOpenAIProtocolService(a.cfg.OpenAIAPIKey, a.cfg.OpenAIBaseURL, a.cfg.OpenAIChatModel)
	}

	meetingService := service.NewMeetingService(repo, transcriptService, protocolService)
	handler := bot.NewHandler(telegramBot, meetingService, bot.NewStateStore())

	mux := http.DefaultServeMux
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	if a.cfg.ZoomWebhookEnabled {
		mux.Handle("/webhooks/zoom", zoom.NewWebhookHandler(meetingService))
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", a.cfg.AppPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 2)
	go func() {
		log.Printf("http server on :%d", a.cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	go func() {
		if a.cfg.TelegramUseWebhook {
			errCh <- a.runWebhookMode(ctx, telegramBot, handler)
			return
		}
		errCh <- a.runLongPollingMode(ctx, telegramBot, handler)
	}()

	select {
	case err := <-errCh:
		_ = server.Shutdown(context.Background())
		return err
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return ctx.Err()
	}
}

func (a *App) runLongPollingMode(ctx context.Context, telegramBot *tgbotapi.BotAPI, handler *bot.Handler) error {
	updateCfg := tgbotapi.NewUpdate(0)
	updateCfg.Timeout = 30
	updates := telegramBot.GetUpdatesChan(updateCfg)
	for {
		select {
		case update := <-updates:
			handler.HandleUpdate(ctx, update)
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *App) runWebhookMode(ctx context.Context, telegramBot *tgbotapi.BotAPI, handler *bot.Handler) error {
	if a.cfg.BotPublicURL == "" {
		return fmt.Errorf("BOT_PUBLIC_URL is required for webhook mode")
	}
	webhookURL := fmt.Sprintf("%s/telegram/webhook", a.cfg.BotPublicURL)
	_, err := telegramBot.Request(tgbotapi.NewWebhook(webhookURL))
	if err != nil {
		return err
	}
	updates := telegramBot.ListenForWebhook("/telegram/webhook")
	for {
		select {
		case update := <-updates:
			handler.HandleUpdate(ctx, update)
		case <-ctx.Done():
			_, _ = telegramBot.Request(tgbotapi.DeleteWebhookConfig{})
			return nil
		}
	}
}
