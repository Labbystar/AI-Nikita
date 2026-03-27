package domain

import "time"

type MeetingSource string

const (
	SourceZoom     MeetingSource = "zoom"
	SourceTelemost MeetingSource = "telemost"
	SourceOffline  MeetingSource = "offline"
	SourceUpload   MeetingSource = "upload"
)

type MeetingStatus string

const (
	MeetingDraft      MeetingStatus = "draft"
	MeetingInProgress MeetingStatus = "in_progress"
	MeetingFinished   MeetingStatus = "finished"
)

type ItemType string

const (
	ItemNote     ItemType = "note"
	ItemDecision ItemType = "decision"
	ItemAction   ItemType = "action"
	ItemRisk     ItemType = "risk"
)

type ArtifactType string

const (
	ArtifactAudio      ArtifactType = "audio"
	ArtifactVideo      ArtifactType = "video"
	ArtifactTranscript ArtifactType = "transcript"
	ArtifactSummary    ArtifactType = "summary"
)

type Meeting struct {
	ID                  string
	Title               string
	SourceType          MeetingSource
	SourceURL           *string
	SourceMeetingID     *string
	Status              MeetingStatus
	Transcript          *string
	ProtocolText        *string
	SummaryText         *string
	StartedAt           *time.Time
	EndedAt             *time.Time
	CreatedByTelegramID int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type MeetingParticipant struct {
	ID             string
	MeetingID      string
	DisplayName    string
	TelegramUserID *int64
	Role           *string
	CreatedAt      time.Time
}

type MeetingArtifact struct {
	ID           string
	MeetingID    string
	ArtifactType ArtifactType
	FileID       *string
	FileURL      *string
	TextContent  *string
	MimeType     *string
	CreatedAt    time.Time
}

type MeetingItem struct {
	ID         string
	MeetingID  string
	ItemType   ItemType
	Content    string
	AssignedTo *string
	Deadline   *string
	Status     string
	Confidence float64
	CreatedAt  time.Time
}

type Task struct {
	ID          string
	MeetingID   *string
	Title       string
	Description *string
	AssignedTo  *string
	Deadline    *string
	Status      string
	CreatedAt   time.Time
}

type CreateMeetingInput struct {
	Title               string
	SourceType          MeetingSource
	SourceURL           *string
	SourceMeetingID     *string
	CreatedByTelegramID int64
}

type CreateItemInput struct {
	MeetingID  string
	ItemType   ItemType
	Content    string
	AssignedTo *string
	Deadline   *string
	Confidence float64
}

type CreateArtifactInput struct {
	MeetingID    string
	ArtifactType ArtifactType
	FileID       *string
	FileURL      *string
	TextContent  *string
	MimeType     *string
}
