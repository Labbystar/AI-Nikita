package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"meeting-assistant-go/internal/domain"
)

//go:embed schema.sql
var schemaSQL string

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(path string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &SQLiteRepository{db: db}, nil
}

func (r *SQLiteRepository) Close() error { return r.db.Close() }

func (r *SQLiteRepository) CreateMeeting(ctx context.Context, in domain.CreateMeetingInput) (*domain.Meeting, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO meetings (id, title, source_type, source_url, source_meeting_id, status, created_by_telegram_id, started_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, in.Title, string(in.SourceType), in.SourceURL, in.SourceMeetingID, string(domain.MeetingInProgress), in.CreatedByTelegramID, now, now, now)
	if err != nil {
		return nil, err
	}
	return r.GetMeetingByID(ctx, id)
}

func (r *SQLiteRepository) UpdateMeetingProtocol(ctx context.Context, meetingID string, summary, protocol string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE meetings
		SET summary_text = ?, protocol_text = ?, updated_at = ?
		WHERE id = ?
	`, nullable(summary), nullable(protocol), time.Now().UTC(), meetingID)
	return err
}

func (r *SQLiteRepository) UpdateMeetingTranscript(ctx context.Context, meetingID string, transcript string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE meetings
		SET transcript = ?, updated_at = ?
		WHERE id = ?
	`, nullable(transcript), time.Now().UTC(), meetingID)
	return err
}

func (r *SQLiteRepository) FinishMeeting(ctx context.Context, meetingID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE meetings
		SET status = ?, ended_at = ?, updated_at = ?
		WHERE id = ?
	`, string(domain.MeetingFinished), now, now, meetingID)
	return err
}

func (r *SQLiteRepository) GetMeetingByID(ctx context.Context, meetingID string) (*domain.Meeting, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, source_type, source_url, source_meeting_id, status, transcript, protocol_text, summary_text,
		       started_at, ended_at, created_by_telegram_id, created_at, updated_at
		FROM meetings WHERE id = ?
	`, meetingID)
	return scanMeeting(row)
}

func (r *SQLiteRepository) GetMeetingBySourceMeetingID(ctx context.Context, sourceMeetingID string) (*domain.Meeting, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, source_type, source_url, source_meeting_id, status, transcript, protocol_text, summary_text,
		       started_at, ended_at, created_by_telegram_id, created_at, updated_at
		FROM meetings WHERE source_meeting_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, sourceMeetingID)
	return scanMeeting(row)
}

func (r *SQLiteRepository) ListMeetingsByUser(ctx context.Context, telegramUserID int64, limit int) ([]domain.Meeting, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, source_type, source_url, source_meeting_id, status, transcript, protocol_text, summary_text,
		       started_at, ended_at, created_by_telegram_id, created_at, updated_at
		FROM meetings
		WHERE created_by_telegram_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, telegramUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []domain.Meeting
	for rows.Next() {
		meeting, err := scanMeetingRows(rows)
		if err != nil {
			return nil, err
		}
		meetings = append(meetings, *meeting)
	}
	return meetings, rows.Err()
}

func (r *SQLiteRepository) AddParticipant(ctx context.Context, meetingID, displayName string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO meeting_participants (id, meeting_id, display_name)
		VALUES (?, ?, ?)
	`, uuid.NewString(), meetingID, strings.TrimSpace(displayName))
	return err
}

func (r *SQLiteRepository) ListParticipants(ctx context.Context, meetingID string) ([]domain.MeetingParticipant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, meeting_id, display_name, telegram_user_id, role, created_at
		FROM meeting_participants
		WHERE meeting_id = ?
		ORDER BY created_at ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MeetingParticipant
	for rows.Next() {
		var p domain.MeetingParticipant
		var telegramUserID sql.NullInt64
		var role sql.NullString
		if err := rows.Scan(&p.ID, &p.MeetingID, &p.DisplayName, &telegramUserID, &role, &p.CreatedAt); err != nil {
			return nil, err
		}
		if telegramUserID.Valid {
			v := telegramUserID.Int64
			p.TelegramUserID = &v
		}
		if role.Valid {
			v := role.String
			p.Role = &v
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) AddArtifact(ctx context.Context, in domain.CreateArtifactInput) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO meeting_artifacts (id, meeting_id, artifact_type, file_id, file_url, text_content, mime_type)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, uuid.NewString(), in.MeetingID, string(in.ArtifactType), in.FileID, in.FileURL, in.TextContent, in.MimeType)
	return err
}

func (r *SQLiteRepository) ListArtifacts(ctx context.Context, meetingID string) ([]domain.MeetingArtifact, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, meeting_id, artifact_type, file_id, file_url, text_content, mime_type, created_at
		FROM meeting_artifacts
		WHERE meeting_id = ?
		ORDER BY created_at ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MeetingArtifact
	for rows.Next() {
		var a domain.MeetingArtifact
		var artifactType string
		var fileID, fileURL, textContent, mimeType sql.NullString
		if err := rows.Scan(&a.ID, &a.MeetingID, &artifactType, &fileID, &fileURL, &textContent, &mimeType, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.ArtifactType = domain.ArtifactType(artifactType)
		if fileID.Valid {
			v := fileID.String
			a.FileID = &v
		}
		if fileURL.Valid {
			v := fileURL.String
			a.FileURL = &v
		}
		if textContent.Valid {
			v := textContent.String
			a.TextContent = &v
		}
		if mimeType.Valid {
			v := mimeType.String
			a.MimeType = &v
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) AddItem(ctx context.Context, in domain.CreateItemInput) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO meeting_items (id, meeting_id, item_type, content, assigned_to, deadline, status, confidence)
		VALUES (?, ?, ?, ?, ?, ?, 'open', ?)
	`, uuid.NewString(), in.MeetingID, string(in.ItemType), in.Content, in.AssignedTo, in.Deadline, in.Confidence)
	return err
}

func (r *SQLiteRepository) ListItems(ctx context.Context, meetingID string) ([]domain.MeetingItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, meeting_id, item_type, content, assigned_to, deadline, status, confidence, created_at
		FROM meeting_items
		WHERE meeting_id = ?
		ORDER BY created_at ASC
	`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MeetingItem
	for rows.Next() {
		var i domain.MeetingItem
		var itemType string
		var assignedTo, deadline sql.NullString
		if err := rows.Scan(&i.ID, &i.MeetingID, &itemType, &i.Content, &assignedTo, &deadline, &i.Status, &i.Confidence, &i.CreatedAt); err != nil {
			return nil, err
		}
		i.ItemType = domain.ItemType(itemType)
		if assignedTo.Valid {
			v := assignedTo.String
			i.AssignedTo = &v
		}
		if deadline.Valid {
			v := deadline.String
			i.Deadline = &v
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CreateTask(ctx context.Context, task domain.Task) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (id, meeting_id, title, description, assigned_to, deadline, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.MeetingID, task.Title, task.Description, task.AssignedTo, task.Deadline, task.Status, task.CreatedAt)
	return err
}

func scanMeeting(row *sql.Row) (*domain.Meeting, error) {
	var m domain.Meeting
	var sourceType, status string
	var sourceURL, sourceMeetingID, transcript, protocolText, summaryText sql.NullString
	var startedAt, endedAt sql.NullTime
	if err := row.Scan(
		&m.ID, &m.Title, &sourceType, &sourceURL, &sourceMeetingID, &status, &transcript, &protocolText, &summaryText,
		&startedAt, &endedAt, &m.CreatedByTelegramID, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return nil, err
	}
	m.SourceType = domain.MeetingSource(sourceType)
	m.Status = domain.MeetingStatus(status)
	if sourceURL.Valid {
		v := sourceURL.String
		m.SourceURL = &v
	}
	if sourceMeetingID.Valid {
		v := sourceMeetingID.String
		m.SourceMeetingID = &v
	}
	if transcript.Valid {
		v := transcript.String
		m.Transcript = &v
	}
	if protocolText.Valid {
		v := protocolText.String
		m.ProtocolText = &v
	}
	if summaryText.Valid {
		v := summaryText.String
		m.SummaryText = &v
	}
	if startedAt.Valid {
		v := startedAt.Time
		m.StartedAt = &v
	}
	if endedAt.Valid {
		v := endedAt.Time
		m.EndedAt = &v
	}
	return &m, nil
}

func scanMeetingRows(rows *sql.Rows) (*domain.Meeting, error) {
	var m domain.Meeting
	var sourceType, status string
	var sourceURL, sourceMeetingID, transcript, protocolText, summaryText sql.NullString
	var startedAt, endedAt sql.NullTime
	if err := rows.Scan(
		&m.ID, &m.Title, &sourceType, &sourceURL, &sourceMeetingID, &status, &transcript, &protocolText, &summaryText,
		&startedAt, &endedAt, &m.CreatedByTelegramID, &m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return nil, err
	}
	m.SourceType = domain.MeetingSource(sourceType)
	m.Status = domain.MeetingStatus(status)
	if sourceURL.Valid {
		v := sourceURL.String
		m.SourceURL = &v
	}
	if sourceMeetingID.Valid {
		v := sourceMeetingID.String
		m.SourceMeetingID = &v
	}
	if transcript.Valid {
		v := transcript.String
		m.Transcript = &v
	}
	if protocolText.Valid {
		v := protocolText.String
		m.ProtocolText = &v
	}
	if summaryText.Valid {
		v := summaryText.String
		m.SummaryText = &v
	}
	if startedAt.Valid {
		v := startedAt.Time
		m.StartedAt = &v
	}
	if endedAt.Valid {
		v := endedAt.Time
		m.EndedAt = &v
	}
	return &m, nil
}

func nullable(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
