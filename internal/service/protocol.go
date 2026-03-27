package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"meeting-assistant-go/internal/domain"
	"meeting-assistant-go/internal/util"
)

type HeuristicProtocolService struct{}

func (s *HeuristicProtocolService) BuildProtocol(ctx context.Context, meeting *domain.Meeting, participants []domain.MeetingParticipant, items []domain.MeetingItem) (string, string, []domain.MeetingItem, error) {
	var participantNames []string
	for _, p := range participants {
		participantNames = append(participantNames, p.DisplayName)
	}
	var notes, decisions, actions, risks []string
	var actionItems []domain.MeetingItem
	for _, item := range items {
		switch item.ItemType {
		case domain.ItemNote:
			notes = append(notes, item.Content)
		case domain.ItemDecision:
			decisions = append(decisions, item.Content)
		case domain.ItemAction:
			line := item.Content
			if item.AssignedTo != nil || item.Deadline != nil {
				parts := []string{item.Content}
				if item.AssignedTo != nil {
					parts = append(parts, "ответственный: "+*item.AssignedTo)
				}
				if item.Deadline != nil {
					parts = append(parts, "срок: "+*item.Deadline)
				}
				line = strings.Join(parts, " | ")
			}
			actions = append(actions, line)
			actionItems = append(actionItems, item)
		case domain.ItemRisk:
			risks = append(risks, item.Content)
		}
	}

	summary := fmt.Sprintf("Встреча \"%s\". Участники: %s. Зафиксировано заметок: %d, решений: %d, поручений: %d.",
		meeting.Title, strings.Join(participantNames, ", "), len(notes), len(decisions), len(actions))

	transcriptBlock := "—"
	if meeting.Transcript != nil && strings.TrimSpace(*meeting.Transcript) != "" {
		text := *meeting.Transcript
		if len(text) > 2000 {
			text = text[:2000] + "\n..."
		}
		transcriptBlock = text
	}

	protocol := fmt.Sprintf(`Протокол встречи

Тема: %s
Источник: %s
Участники:
%s

Краткое резюме:
%s

Принятые решения:
%s

Поручения:
%s

Заметки:
%s

Риски / спорные места:
%s

Фрагмент transcript:
%s`,
		meeting.Title,
		meeting.SourceType,
		util.JoinBullets(participantNames),
		summary,
		util.JoinBullets(decisions),
		util.JoinBullets(actions),
		util.JoinBullets(notes),
		util.JoinBullets(risks),
		transcriptBlock,
	)

	return summary, protocol, actionItems, nil
}

type OpenAIProtocolService struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	fallback   *HeuristicProtocolService
}

func NewOpenAIProtocolService(apiKey, baseURL, model string) *OpenAIProtocolService {
	return &OpenAIProtocolService{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: &http.Client{},
		fallback:   &HeuristicProtocolService{},
	}
}

func (s *OpenAIProtocolService) BuildProtocol(ctx context.Context, meeting *domain.Meeting, participants []domain.MeetingParticipant, items []domain.MeetingItem) (string, string, []domain.MeetingItem, error) {
	if strings.TrimSpace(s.apiKey) == "" {
		return s.fallback.BuildProtocol(ctx, meeting, participants, items)
	}

	systemPrompt := "Ты помощник руководителя и секретарь встреч. Верни управленческий протокол на русском языке. Строго разделяй вывод на блоки: Краткое резюме, Принятые решения, Поручения, Неопределенности и спорные места. Не выдумывай факты. Если срок или ответственный не определены, так и пиши."
	userPrompt := buildProtocolPrompt(meeting, participants, items)

	payload := map[string]any{
		"model": s.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.2,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return s.fallback.BuildProtocol(ctx, meeting, participants, items)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil || resp == nil {
		return s.fallback.BuildProtocol(ctx, meeting, participants, items)
	}
	defer resp.Body.Close()
	respBody, _ := ioReadAllLimit(resp.Body, 1<<20)
	if resp.StatusCode >= 300 {
		return s.fallback.BuildProtocol(ctx, meeting, participants, items)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil || len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return s.fallback.BuildProtocol(ctx, meeting, participants, items)
	}

	summary, _, actionItems, _ := s.fallback.BuildProtocol(ctx, meeting, participants, items)
	return summary, parsed.Choices[0].Message.Content, actionItems, nil
}

func buildProtocolPrompt(meeting *domain.Meeting, participants []domain.MeetingParticipant, items []domain.MeetingItem) string {
	var pNames []string
	for _, p := range participants {
		pNames = append(pNames, p.DisplayName)
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("Тема: %s", meeting.Title))
	lines = append(lines, fmt.Sprintf("Источник: %s", meeting.SourceType))
	lines = append(lines, fmt.Sprintf("Участники: %s", strings.Join(pNames, ", ")))
	if meeting.Transcript != nil && strings.TrimSpace(*meeting.Transcript) != "" {
		lines = append(lines, "Transcript:")
		lines = append(lines, *meeting.Transcript)
	}
	lines = append(lines, "Структурированные элементы:")
	for _, item := range items {
		line := fmt.Sprintf("- %s: %s", item.ItemType, item.Content)
		if item.AssignedTo != nil {
			line += "; assigned_to=" + *item.AssignedTo
		}
		if item.Deadline != nil {
			line += "; deadline=" + *item.Deadline
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ioReadAllLimit(r io.Reader, n int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, n))
}
