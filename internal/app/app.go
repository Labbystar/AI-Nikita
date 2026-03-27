package app

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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

	// STT
	var transcriptService service.TranscriptService = &service.StubTranscriptService{}
	if a.cfg.OpenAIAPIKey != "" {
		transcriptService = service.NewOpenAITranscriptService(
			a.cfg.OpenAIAPIKey,
			a.cfg.OpenAIBaseURL,
			a.cfg.OpenAITranscribeModel,
		)
	}

	// Protocol
	protocolService := service.ProtocolService(&service.HeuristicProtocolService{})
	if a.cfg.OpenAIAPIKey != "" {
		protocolService = service.NewOpenAIProtocolService(
			a.cfg.OpenAIAPIKey,
			a.cfg.OpenAIBaseURL,
			a.cfg.OpenAIChatModel,
		)
	}

	meetingService := service.NewMeetingService(repo, transcriptService, protocolService)
	handler := bot.NewHandler(telegramBot, meetingService, bot.NewStateStore())

	// 🚀 Только polling (без webhook)
	updateCfg := tgbotapi.NewUpdate(0)
	updateCfg.Timeout = 30

	updates := telegramBot.GetUpdatesChan(updateCfg)

	log.Println("bot started in polling mode")

	for {
		select {
		case update := <-updates:
			handler.HandleUpdate(ctx, update)

		case <-ctx.Done():
			log.Println("shutting down bot")
			return nil
		}
	}
}
