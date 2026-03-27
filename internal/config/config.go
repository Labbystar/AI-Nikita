package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	BotToken              string
	BotPublicURL          string
	AppPort               int
	DBPath                string
	TelegramUseWebhook    bool
	OpenAIAPIKey          string
	OpenAIBaseURL         string
	OpenAITranscribeModel string
	OpenAIChatModel       string
	ZoomWebhookSecret     string
	ZoomWebhookEnabled    bool
}

func Load() (Config, error) {
	cfg := Config{
		BotToken:              os.Getenv("BOT_TOKEN"),
		BotPublicURL:          getenv("BOT_PUBLIC_URL", ""),
		AppPort:               getenvInt("APP_PORT", 8080),
		DBPath:                getenv("DB_PATH", "meeting_assistant.db"),
		TelegramUseWebhook:    getenvBool("TELEGRAM_USE_WEBHOOK", false),
		OpenAIAPIKey:          getenv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:         getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAITranscribeModel: getenv("OPENAI_TRANSCRIBE_MODEL", "whisper-1"),
		OpenAIChatModel:       getenv("OPENAI_CHAT_MODEL", "gpt-4.1-mini"),
		ZoomWebhookSecret:     getenv("ZOOM_WEBHOOK_SECRET", ""),
		ZoomWebhookEnabled:    getenvBool("ZOOM_WEBHOOK_ENABLED", false),
	}
	if cfg.BotToken == "" {
		return Config{}, fmt.Errorf("BOT_TOKEN is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
