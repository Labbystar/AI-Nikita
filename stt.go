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

	switch {
	case msg.IsCommand():
		h.handleCommand(ctx, msg)
	case state.PendingAction == "awaiting_title":
		h.createMeetingWithTitle(ctx, msg, state)
	case state.PendingAction == "awaiting_participant":
		h.addParticipant(ctx, msg, state)
	case state.AwaitingUpload && (msg.Audio != nil || msg.Voice != nil || msg.Document != nil):
		h.handleAudioUpload(ctx, msg, state)
	default:
		h.reply(chatID, "Не понял сообщение. Используйте меню или команды /note, /decision, /action.")
	}
}

func (h *Handler) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	state := h.state.Get(chatID)
	cmd := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())

	switch cmd {
	case "start":
		m := tgbotapi.NewMessage(chatID, "Meeting Assistant. Создайте встречу, выберите источник и после встречи получите протокол.")
		m.ReplyMarkup = mainMenuKeyboard()
		h.send(m)
	case "new_meeting":
		state.PendingAction = "awaiting_source"
		m := tgbotapi.NewMessage(chatID, "Выберите источник встречи:")
		m.ReplyMarkup = sourceKeyboard()
		h.send(m)
	case "my_meetings":
		meetings, err := h.meetings.ListMeetings(ctx, msg.From.ID, 10)
		if err != nil {
			h.reply(chatID, "Не удалось загрузить встречи: "+err.Error())
			return
		}
		if len(meetings) == 0 {
			h.reply(chatID, "У вас пока нет встреч.")
			return
		}
		var lines []string
		for _, meeting := range meetings {
			lines = append(lines, fmt.Sprintf("• %s [%s] (%s) id=%s", meeting.Title, meeting.SourceType, meeting.Status, meeting.ID))
		}
		h.reply(chatID, strings.Join(lines, "\n"))
	case "note":
		h.addTypedItem(ctx, chatID, state, domain.ItemNote, args)
	case "decision":
		h.addTypedItem(ctx, chatID, state, domain.ItemDecision, args)
	case "action":
		h.addActionItem(ctx, chatID, state, args)
	case "finish":
		h.finishMeeting(ctx, chatID, state)
	case "cancel":
		h.state.Reset(chatID)
		h.reply(chatID, "Текущий сценарий сброшен.")
	default:
		h.reply(chatID, "Неизвестная команда.")
	}
}

func (h *Handler) handleCallback(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	chatID := cq.Message.Chat.ID
	state := h.state.Get(chatID)
	_ = h.answerCallback(cq.ID)

	switch cq.Data {
	case "meeting:new":
		state.PendingAction = "awaiting_source"
		m := tgbotapi.NewMessage(chatID, "Выберите источник встречи:")
		m.ReplyMarkup = sourceKeyboard()
		h.send(m)
	case "meeting:list":
		meetings, err := h.meetings.ListMeetings(ctx, cq.From.ID, 10)
		if err != nil {
			h.reply(chatID, "Не удалось загрузить встречи: "+err.Error())
			return
		}
		if len(meetings) == 0 {
			h.reply(chatID, "У вас пока нет встреч.")
			return
		}
		var lines []string
		for _, meeting := range meetings {
			lines = append(lines, fmt.Sprintf("• %s [%s] (%s) id=%s", meeting.Title, meeting.SourceType, meeting.Status, meeting.ID))
		}
		h.reply(chatID, strings.Join(lines, "\n"))
	case "meeting:finish":
		h.finishMeeting(ctx, chatID, state)
	case "source:zoom", "source:telemost", "source:offline", "source:upload":
		state.DraftSource = strings.TrimPrefix(cq.Data, "source:")
		state.PendingAction = "awaiting_title"
		h.reply(chatID, "Введите название встречи одним сообщением.")
	default:
		h.reply(chatID, "Неизвестное действие.")
	}
}

func (h *Handler) createMeetingWithTitle(ctx context.Context, msg *tgbotapi.Message, state *SessionState) {
	source := domain.MeetingSource(state.DraftSource)
	meeting, err := h.meetings.CreateMeeting(ctx, domain.CreateMeetingInput{
		Title:               msg.Text,
		SourceType:          source,
		CreatedByTelegramID: msg.From.ID,
	})
	if err != nil {
		h.reply(msg.Chat.ID, "Не удалось создать встречу: "+err.Error())
		return
	}
	state.DraftMeetingID = meeting.ID
	state.PendingAction = "awaiting_participant"
	state.AwaitingUpload = source == domain.SourceOffline || source == domain.SourceUpload || source == domain.SourceTelemost

	text := fmt.Sprintf("Встреча создана.\nID: %s\nИсточник: %s\n\nТеперь можно:\n- прислать имя участника одним сообщением\n- написать /note\n- написать /decision\n- написать /action ФИО | срок | текст\n- отправить аудио/voice после встречи\n- написать /finish для формирования протокола", meeting.ID, meeting.SourceType)
	if source == domain.SourceZoom {
		text += "\n\nДля Zoom можно позже связать встречу по webhook/source_meeting_id."
	}
	h.reply(msg.Chat.ID, text)
}

func (h *Handler) addParticipant(ctx context.Context, msg *tgbotapi.Message, state *SessionState) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		h.reply(msg.Chat.ID, "Имя участника пустое.")
		return
	}
	if strings.HasPrefix(text, "/") {
		h.handleCommand(ctx, msg)
		return
	}
	if err := h.meetings.AddParticipant(ctx, state.DraftMeetingID, text); err != nil {
		h.reply(msg.Chat.ID, "Не удалось добавить участника: "+err.Error())
		return
	}
	h.reply(msg.Chat.ID, "Участник добавлен. Можно отправить еще одно имя участника, либо перейти к /note, /decision, /action, /finish.")
}

func (h *Handler) addTypedItem(ctx context.Context, chatID int64, state *SessionState, itemType domain.ItemType, content string) {
	if state.DraftMeetingID == "" {
		h.reply(chatID, "Нет активной встречи. Сначала создайте её через /new_meeting.")
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
		return
	}
	h.reply(chatID, fmt.Sprintf("Сохранено как %s.", itemType))
}

func (h *Handler) addActionItem(ctx context.Context, chatID int64, state *SessionState, args string) {
	if state.DraftMeetingID == "" {
		h.reply(chatID, "Нет активной встречи. Сначала создайте её через /new_meeting.")
		return
	}
	parts := strings.SplitN(args, "|", 3)
	if len(parts) < 3 {
		h.reply(chatID, "Формат: /action ФИО | срок | текст")
		return
	}
	assignedTo := strings.TrimSpace(parts[0])
	deadline := strings.TrimSpace(parts[1])
	content := strings.TrimSpace(parts[2])
	if content == "" {
		h.reply(chatID, "Текст поручения пустой.")
		return
	}
	if err := h.meetings.AddItem(ctx, domain.CreateItemInput{
		MeetingID:  state.DraftMeetingID,
		ItemType:   domain.ItemAction,
		Content:    content,
		AssignedTo: &assignedTo,
		Deadline:   &deadline,
		Confidence: 1,
	}); err != nil {
		h.reply(chatID, "Не удалось сохранить поручение: "+err.Error())
		return
	}
	h.reply(chatID, "Поручение сохранено.")
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

	statusMsg := tgbotapi.NewMessage(msg.Chat.ID, "Файл получен. Пытаюсь сделать расшифровку...")
	h.send(statusMsg)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	transcript, err := h.meetings.AddTranscriptFromFileURL(ctx, state.DraftMeetingID, url, mimeType)
	if err != nil {
		h.reply(msg.Chat.ID, "Файл сохранен, но расшифровка не удалась: "+err.Error())
		return
	}
	preview := transcript
	if len(preview) > 700 {
		preview = preview[:700] + "\n..."
	}
	h.reply(msg.Chat.ID, "Расшифровка готова:\n\n"+preview+"\n\nТеперь можно писать /finish")
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
	h.reply(chatID, "Протокол сформирован:\n\n"+protocol)
}

func (h *Handler) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
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
