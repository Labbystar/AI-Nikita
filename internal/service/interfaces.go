package service

import (
	"context"

	"meeting-assistant-go/internal/domain"
)

type Repository interface {
	CreateMeeting(ctx context.Context, in domain.CreateMeetingInput) (*domain.Meeting, error)
	UpdateMeetingProtocol(ctx context.Context, meetingID string, summary, protocol string) error
	UpdateMeetingTranscript(ctx context.Context, meetingID string, transcript string) error
	FinishMeeting(ctx context.Context, meetingID string) error
	GetMeetingByID(ctx context.Context, meetingID string) (*domain.Meeting, error)
	GetMeetingBySourceMeetingID(ctx context.Context, sourceMeetingID string) (*domain.Meeting, error)
	ListMeetingsByUser(ctx context.Context, telegramUserID int64, limit int) ([]domain.Meeting, error)
	AddParticipant(ctx context.Context, meetingID, displayName string) error
	ListParticipants(ctx context.Context, meetingID string) ([]domain.MeetingParticipant, error)
	AddArtifact(ctx context.Context, in domain.CreateArtifactInput) error
	ListArtifacts(ctx context.Context, meetingID string) ([]domain.MeetingArtifact, error)
	AddItem(ctx context.Context, in domain.CreateItemInput) error
	ListItems(ctx context.Context, meetingID string) ([]domain.MeetingItem, error)
	CreateTask(ctx context.Context, task domain.Task) error
}

type TranscriptService interface {
	TranscribeFromURL(ctx context.Context, fileURL, mimeType string) (string, error)
}

type ProtocolService interface {
	BuildProtocol(ctx context.Context, meeting *domain.Meeting, participants []domain.MeetingParticipant, items []domain.MeetingItem) (summary string, protocol string, extractedActions []domain.MeetingItem, err error)
}
