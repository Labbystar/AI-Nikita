package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"meeting-assistant-go/internal/domain"
)

type MeetingService struct {
	repo      Repository
	stt       TranscriptService
	protocols ProtocolService
}

func NewMeetingService(repo Repository, stt TranscriptService, protocols ProtocolService) *MeetingService {
	return &MeetingService{repo: repo, stt: stt, protocols: protocols}
}

func (s *MeetingService) CreateMeeting(ctx context.Context, in domain.CreateMeetingInput) (*domain.Meeting, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	return s.repo.CreateMeeting(ctx, in)
}

func (s *MeetingService) AddParticipant(ctx context.Context, meetingID, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("participant name is empty")
	}
	return s.repo.AddParticipant(ctx, meetingID, name)
}

func (s *MeetingService) AddItem(ctx context.Context, in domain.CreateItemInput) error {
	if strings.TrimSpace(in.Content) == "" {
		return fmt.Errorf("item content is empty")
	}
	if in.Confidence == 0 {
		in.Confidence = 1
	}
	return s.repo.AddItem(ctx, in)
}

func (s *MeetingService) AddArtifact(ctx context.Context, in domain.CreateArtifactInput) error {
	return s.repo.AddArtifact(ctx, in)
}

func (s *MeetingService) AddTranscriptFromFileURL(ctx context.Context, meetingID, fileURL, mimeType string) (string, error) {
	transcript, err := s.stt.TranscribeFromURL(ctx, fileURL, mimeType)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateMeetingTranscript(ctx, meetingID, transcript); err != nil {
		return "", err
	}
	if err := s.repo.AddArtifact(ctx, domain.CreateArtifactInput{
		MeetingID:    meetingID,
		ArtifactType: domain.ArtifactTranscript,
		TextContent:  &transcript,
		MimeType:     &mimeType,
	}); err != nil {
		return "", err
	}
	return transcript, nil
}

func (s *MeetingService) FinalizeMeeting(ctx context.Context, meetingID string) (string, error) {
	meeting, err := s.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return "", err
	}
	participants, err := s.repo.ListParticipants(ctx, meetingID)
	if err != nil {
		return "", err
	}
	items, err := s.repo.ListItems(ctx, meetingID)
	if err != nil {
		return "", err
	}

	summary, protocol, extractedActions, err := s.protocols.BuildProtocol(ctx, meeting, participants, items)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateMeetingProtocol(ctx, meetingID, summary, protocol); err != nil {
		return "", err
	}

	for _, item := range extractedActions {
		task := domain.Task{
			ID:         uuid.NewString(),
			MeetingID:  &meetingID,
			Title:      item.Content,
			AssignedTo: item.AssignedTo,
			Deadline:   item.Deadline,
			Status:     "open",
			CreatedAt:  time.Now().UTC(),
		}
		_ = s.repo.CreateTask(ctx, task)
	}

	if err := s.repo.FinishMeeting(ctx, meetingID); err != nil {
		return "", err
	}
	return protocol, nil
}

func (s *MeetingService) GetMeeting(ctx context.Context, meetingID string) (*domain.Meeting, error) {
	return s.repo.GetMeetingByID(ctx, meetingID)
}

func (s *MeetingService) ListMeetings(ctx context.Context, telegramUserID int64, limit int) ([]domain.Meeting, error) {
	return s.repo.ListMeetingsByUser(ctx, telegramUserID, limit)
}
