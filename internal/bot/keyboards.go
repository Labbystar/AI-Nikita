package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func mainMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создать встречу", "meeting:new"),
			tgbotapi.NewInlineKeyboardButtonData("Мои встречи", "meeting:list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Завершить активную", "meeting:finish"),
		),
	)
}

func sourceKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Zoom", "source:zoom"),
			tgbotapi.NewInlineKeyboardButtonData("Телемост", "source:telemost"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Оффлайн", "source:offline"),
			tgbotapi.NewInlineKeyboardButtonData("Загрузка записи", "source:upload"),
		),
	)
}
