package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Создать встречу"),
			tgbotapi.NewKeyboardButton("Мои встречи"),
		),
	)
}

func MeetingMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Заметка"),
			tgbotapi.NewKeyboardButton("Решение"),
			tgbotapi.NewKeyboardButton("Поручение"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Загрузить аудио"),
			tgbotapi.NewKeyboardButton("Сформировать протокол"),
		),
	)
}

func PlannerButton() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(
				"Создать задачу",
				"https://t.me/napomnimnevajnoe_bot",
			),
		),
	)
}
