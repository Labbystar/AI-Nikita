package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"meeting-assistant-go/internal/domain"
	"meeting-assistant-go/internal/service"
)

type WebhookHandler struct {
	meetings *service.MeetingService
}

func NewWebhookHandler(meetings *service.MeetingService) *WebhookHandler {
	return &WebhookHandler{meetings: meetings}
}

type webhookEnvelope struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

type recordingPayload struct {
	Object struct {
		ID    any    `json:"id"`
		Topic string `json:"topic"`
	} `json:"object"`
}

// MVP-обработчик. Для production нужно добавить полную Zoom signature validation и event challenge flow.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var env webhookEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	if strings.Contains(strings.ToLower(env.Event), "endpoint.url_validation") {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plainToken":"implement-signature-validation"}`))
		return
	}

	switch env.Event {
	case "recording.completed", "meeting.ended", "meeting.summary_completed", "meeting.transcript_completed":
		if err := h.handleRecordingLikeEvent(ctx, env.Payload); err != nil {
			log.Println("zoom webhook processing error:", err)
			http.Error(w, "processing error", http.StatusInternalServerError)
			return
		}
	default:
		log.Println("zoom webhook ignored event:", env.Event)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WebhookHandler) handleRecordingLikeEvent(ctx context.Context, payload json.RawMessage) error {
	var p recordingPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	meetingID := fmt.Sprint(p.Object.ID)
	if meetingID == "" {
		return fmt.Errorf("empty zoom meeting id")
	}
	_, err := h.meetings.CreateMeeting(ctx, domain.CreateMeetingInput{
		Title:               fallbackTitle(p.Object.Topic, meetingID),
		SourceType:          domain.SourceZoom,
		SourceMeetingID:     &meetingID,
		CreatedByTelegramID: 0,
	})
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "unique") {
		// если встреча уже существует или есть другая ошибка, пока просто логируем.
		log.Println("zoom create/update note:", err)
	}
	return nil
}

func fallbackTitle(topic, meetingID string) string {
	if strings.TrimSpace(topic) != "" {
		return topic
	}
	return "Zoom meeting " + meetingID
}
