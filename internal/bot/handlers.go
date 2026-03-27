package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"meeting-assistant-go/internal/domain"
	"meeting-assistant-go/internal/service"
)

type Handler struct {
	bot      *tgbotapi.BotAPI
	meetings *service.MeetingService
	state    *StateStore
}

func NewHandler(bot *tgbotapi.BotAPI, meetings *service.MeetingService, state *StateStore) *Handler {
	return &Handler{bot: bot, meetings: meetings, state: state}
}

func (h *Handler) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}
	if update.Message == nil {
		return
	}

	msg := update.Message
	chatID := msg.Chat.ID
	state := h.state.Get(chatID)

	if msg.IsCommand() {
		h.handleCommand(ctx, msg)
		return
	}

	if msg.Audio != nil || msg.Voice != nil || msg.Document != nil {
		h.handleAudioUpload(ctx, msg, state)
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	switch state.PendingAction {
	case "awaiting_title":
		h.createMeetingWithTitle(ctx, msg, state)
		return
	case "awaiting_participant":
		h.addParticipant(ctx, msg, state)
		return
	case "awaiting_note":
		h.addTypedItem(ctx, chatID, state, domain.ItemNote, text)
		state.PendingAction = ""
		h.replyWithMeetingMenu(chatID, "Заметка сохранена.")
		return
	case "awaiting_decision":
		h.addTypedItem(ctx, chatID, state, domain.ItemDecision, text)
		state.PendingAction = ""
		h.replyWithMeetingMenu(chatID, "Решение сохранено.")
		return
	case "awaiting_action":
		h.addActionSimple(ctx, chatID, state, text)
		state.PendingAction = ""
		h.replyWithMeetingMenu(chatID, "Поручение сохранено.")
		return
	}

	switch text {
	case "Создать встречу":
		state.PendingAction = "awaiting_source"
		m := tgbotapi.NewMessage(chatID, "Выберите источник встречи:")
		m.ReplyMarkup = sourceKeyboard()
		h.send(m)
		return
	case "Мои встречи":
		h.showMeetingsList(ctx, chatID, msg.From.ID)
		return
	case "Участник":
		if state.DraftMeetingID == "" {
			h.reply(chatID, "Нет активной встречи. Сначала создайте её.")
			return
		}
		state.PendingAction = "awaiting_participant"
		h.reply(chatID, "Отправьте имя участника одним сообщением.")
		return
	case "Заметка":
		if state.DraftMeetingID == "" {
			h.reply(chatID, "Нет активной встречи. Сначала создайте её.")
			return
		}
		state.PendingAction = "awaiting_note"
		h.reply(chatID, "Отправьте текст заметки одним сообщением.")
		return
	case "Решение":
		if state.DraftMeetingID == "" {
			h.reply(chatID, "Нет активной встречи. Сначала создайте её.")
			return
		}
		state.PendingAction = "awaiting_decision"
		h.reply(chatID, "Отправьте текст решения одним сообщением.")
		return
	case "Поручение":
		if state.DraftMeetingID == "" {
			h.reply(chatID, "Нет активной встречи. Сначала создайте её.")
			return
		}
		state.PendingAction = "awaiting_action"
		h.reply(chatID, "Отправьте поручение одним сообщением. Можно в формате: ФИО | срок | текст")
		return
	case "Загрузить аудио":
		if state.DraftMeetingID == "" {
			h.reply(chatID, "Нет активной встречи. Сначала создайте её.")
			return
		}
		h.reply(chatID, "Отправьте audio, voice или документ с аудио.")
		return
	case "Сформировать протокол":
		h.finishMeeting(ctx, chatID, state)
		return
	}

	if state.DraftMeetingID != "" {
		h.replyWithMeetingMenu(chatID, "Используйте кнопки меню встречи.")
		return
	}

	h.replyWithMainMenu(chatID, "Выберите действие:")
}

func (h *Handler) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	switch msg.Command() {
	case "start":
		h.state.Reset(chatID)
		h.replyWithMainMenu(chatID, "Meeting Assistant. Создайте встречу, выберите источник и после встречи получите протокол.")
	case "cancel":
		h.state.Reset(chatID)
		h.replyWithMainMenu(chatID, "Текущий сценарий сброшен.")
	default:
		h.reply(chatID, "Используйте кнопки меню.")
	}
}

func (h *Handler) handleCallback(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	chatID := cq.Message.Chat.ID
	state := h.state.Get(chatID)
	_ = h.answerCallback(cq.ID)

	switch {
	case cq.Data == "source:zoom" || cq.Data == "source:telemost" || cq.Data == "source:offline" || cq.Data == "source:upload":
		state.DraftSource = strings.TrimPrefix(cq.Data, "source:")
		state.PendingAction = "awaiting_title"
		h.reply(chatID, "Введите название встречи одним сообщением.")
	case strings.HasPrefix(cq.Data, "meeting:open:"):
		meetingID := strings.TrimPrefix(cq.Data, "meeting:open:")
		h.openMeeting(ctx, chatID, meetingID)
	default:
		h.reply(chatID, "Неизвестное действие.")
	}
}

func (h *Handler) createMeetingWithTitle(ctx context.Context, msg *tgbotapi.Message, state *SessionState) {
	source := domain.MeetingSource(state.DraftSource)
	meeting, err := h.meetings.CreateMeeting(ctx, domain.CreateMeetingInput{
		Title:               strings.TrimSpace(msg.Text),
		SourceType:          source,
		CreatedByTelegramID: msg.From.ID,
	})
	if err != nil {
		h.reply(msg.Chat.ID, "Не удалось создать встречу: "+err.Error())
		return
	}

	state.DraftMeetingID = meeting.ID
	state.PendingAction = ""
	state.AwaitingUpload = true

	text := fmt.Sprintf(
		"Встреча создана.\n\nНазвание: %s\nID: %s\nИсточник: %s\n\nЧто можно сделать дальше:\n• добавить участника\n• добавить заметку\n• добавить решение\n• добавить поручение\n• загрузить аудио\n• сформировать протокол",
		meeting.Title,
		meeting.ID,
		meeting.SourceType,
	)

	m := tgbotapi.NewMessage(msg.Chat.ID, text)
	m.ReplyMarkup = MeetingMenu()
	h.send(m)
}

func (h *Handler) addParticipant(ctx context.Context, msg *tgbotapi.Message, state *SessionState) {
	if state.DraftMeetingID == "" {
		h.reply(msg.Chat.ID, "Нет активной встречи.")
		return
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		h.reply(msg.Chat.ID, "Имя участника пустое.")
		return
	}
	if err := h.meetings.AddParticipant(ctx, state.DraftMeetingID, text); err != nil {
		h.reply(msg.Chat.ID, "Не удалось добавить участника: "+err.Error())
		return
	}
	state.PendingAction = ""
	h.replyWithMeetingMenu(msg.Chat.ID, "Участник добавлен.")
}

func (h *Handler) addTypedItem(ctx context.Context, chatID int64, state *SessionState, itemType domain.ItemType, content string) {
	if state.DraftMeetingID == "" {
		h.reply(chatID, "Нет активной встречи. Сначала создайте её.")
		return
	}
	if strings.TrimSpace(content) == "" {
		h.reply(chatID, "Текст пустой.")
		return
	}
	if err := h.meetings.AddItem(ctx, domain.CreateItemInput{
		MeetingID:  state.DraftMeetingID,
		ItemType:   itemType,
		Content:    content,
		Confidence: 1,
	}); err != nil {
		h.reply(chatID, "Не удалось сохранить элемент: "+err.Error())
	}
}

func (h *Handler) addActionSimple(ctx context.Context, chatID int64, state *SessionState, text string) {
	if state.DraftMeetingID == "" {
		h.reply(chatID, "Нет активной встречи. Сначала создайте её.")
		return
	}

	var assignedTo *string
	var deadline *string
	content := strings.TrimSpace(text)
	parts := strings.SplitN(text, "|", 3)
	if len(parts) == 3 {
		a := strings.TrimSpace(parts[0])
		d := strings.TrimSpace(parts[1])
		c := strings.TrimSpace(parts[2])
		if a != "" {
			assignedTo = &a
		}
		if d != "" {
			deadline = &d
		}
		if c != "" {
			content = c
		}
	}

	if strings.TrimSpace(content) == "" {
		h.reply(chatID, "Текст поручения пустой.")
		return
	}

	if err := h.meetings.AddItem(ctx, domain.CreateItemInput{
		MeetingID:  state.DraftMeetingID,
		ItemType:   domain.ItemAction,
		Content:    content,
		AssignedTo: assignedTo,
		Deadline:   deadline,
		Confidence: 1,
	}); err != nil {
		h.reply(chatID, "Не удалось сохранить поручение: "+err.Error())
	}
}

func (h *Handler) handleAudioUpload(ctx context.Context, msg *tgbotapi.Message, state *SessionState) {
	if state.DraftMeetingID == "" {
		h.reply(msg.Chat.ID, "Нет активной встречи.")
		return
	}

	var fileID, mimeType string
	switch {
	case msg.Audio != nil:
		fileID = msg.Audio.FileID
		mimeType = msg.Audio.MimeType
	case msg.Voice != nil:
		fileID = msg.Voice.FileID
		mimeType = "audio/ogg"
	case msg.Document != nil:
		fileID = msg.Document.FileID
		mimeType = msg.Document.MimeType
	default:
		h.reply(msg.Chat.ID, "Поддерживается audio, voice или document с аудио.")
		return
	}

	url, err := h.bot.GetFileDirectURL(fileID)
	if err != nil {
		h.reply(msg.Chat.ID, "Не удалось получить файл из Telegram: "+err.Error())
		return
	}
	if err := h.meetings.AddArtifact(ctx, domain.CreateArtifactInput{
		MeetingID:    state.DraftMeetingID,
		ArtifactType: domain.ArtifactAudio,
		FileID:       &fileID,
		FileURL:      &url,
		MimeType:     &mimeType,
	}); err != nil {
		h.reply(msg.Chat.ID, "Не удалось сохранить артефакт: "+err.Error())
		return
	}

	h.reply(msg.Chat.ID, "Файл получен. Пытаюсь сделать расшифровку...")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	transcript, err := h.meetings.AddTranscriptFromFileURL(ctx, state.DraftMeetingID, url, mimeType)
	if err != nil {
		h.reply(msg.Chat.ID, "Файл сохранён, но расшифровка не удалась: "+err.Error())
		return
	}

	preview := transcript
	if len(preview) > 700 {
		preview = preview[:700] + "\n..."
	}
	h.replyWithMeetingMenu(msg.Chat.ID, "Расшифровка готова:\n\n"+preview)
}

func (h *Handler) finishMeeting(ctx context.Context, chatID int64, state *SessionState) {
	if state.DraftMeetingID == "" {
		h.reply(chatID, "Нет активной встречи.")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	protocol, err := h.meetings.FinalizeMeeting(ctx, state.DraftMeetingID)
	if err != nil {
		h.reply(chatID, "Не удалось сформировать протокол: "+err.Error())
		return
	}

	h.state.Reset(chatID)
	if len(protocol) > 3500 {
		protocol = protocol[:3500] + "\n..."
	}
	msg := tgbotapi.NewMessage(chatID, "Протокол сформирован:\n\n"+protocol)
	msg.ReplyMarkup = PlannerButton()
	h.send(msg)
	h.replyWithMainMenu(chatID, "Можно создать новую встречу или открыть список ваших встреч.")
}

func (h *Handler) showMeetingsList(ctx context.Context, chatID int64, telegramUserID int64) {
	meetings, err := h.meetings.ListMeetings(ctx, telegramUserID, 10)
	if err != nil {
		h.reply(chatID, "Не удалось загрузить встречи: "+err.Error())
		return
	}
	if len(meetings) == 0 {
		h.reply(chatID, "У вас пока нет встреч.")
		return
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(meetings))
	for _, meeting := range meetings {
		title := meeting.Title
		if len(title) > 28 {
			title = title[:28] + "…"
		}
		label := fmt.Sprintf("%s | %s | %s", meeting.CreatedAt.Format("02.01"), meeting.Status, title)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "meeting:open:"+meeting.ID),
		))
	}

	msg := tgbotapi.NewMessage(chatID, "Мои встречи:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
	h.send(msg)
}

func (h *Handler) openMeeting(ctx context.Context, chatID int64, meetingID string) {
	meeting, err := h.meetings.GetMeeting(ctx, meetingID)
	if err != nil {
		if IsNotFound(err) {
			h.reply(chatID, "Встреча не найдена.")
			return
		}
		h.reply(chatID, "Не удалось открыть встречу: "+err.Error())
		return
	}

	transcriptStatus := "нет"
	if meeting.Transcript != nil && strings.TrimSpace(*meeting.Transcript) != "" {
		transcriptStatus = "есть"
	}
	protocolStatus := "нет"
	if meeting.ProtocolText != nil && strings.TrimSpace(*meeting.ProtocolText) != "" {
		protocolStatus = "есть"
	}

	text := fmt.Sprintf(
		"Карточка встречи\n\nНазвание: %s\nID: %s\nИсточник: %s\nСтатус: %s\nСоздана: %s\nTranscript: %s\nПротокол: %s",
		meeting.Title,
		meeting.ID,
		meeting.SourceType,
		meeting.Status,
		meeting.CreatedAt.Format("02.01.2006 15:04"),
		transcriptStatus,
		protocolStatus,
	)

	if meeting.Status != domain.MeetingFinished {
		st := h.state.Get(chatID)
		st.DraftMeetingID = meeting.ID
		msg := tgbotapi.NewMessage(chatID, text+"\n\nЭта встреча установлена как активная.")
		msg.ReplyMarkup = MeetingMenu()
		h.send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = MainMenu()
	h.send(msg)
}

func (h *Handler) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	h.send(msg)
}

func (h *Handler) replyWithMainMenu(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = MainMenu()
	h.send(msg)
}

func (h *Handler) replyWithMeetingMenu(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = MeetingMenu()
	h.send(msg)
}

func (h *Handler) send(msg tgbotapi.Chattable) {
	if _, err := h.bot.Send(msg); err != nil {
		log.Println("send telegram message error:", err)
	}
}

func (h *Handler) answerCallback(id string) error {
	_, err := h.bot.Request(tgbotapi.NewCallback(id, ""))
	return err
}

func IsNotFound(err error) bool {
	return err == sql.ErrNoRows
}
