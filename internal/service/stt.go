package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

type OpenAITranscriptService struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOpenAITranscriptService(apiKey, baseURL, model string) *OpenAITranscriptService {
	return &OpenAITranscriptService{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: &http.Client{},
	}
}

func (s *OpenAITranscriptService) TranscribeFromURL(ctx context.Context, fileURL, mimeType string) (string, error) {
	if strings.TrimSpace(fileURL) == "" {
		return "", fmt.Errorf("empty file url")
	}
	if strings.TrimSpace(s.apiKey) == "" {
		return "", fmt.Errorf("OPENAI_API_KEY is not set")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("download source file: status=%d body=%s", resp.StatusCode, string(body))
	}
	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("model", s.model)
	part, err := writer.CreateFormFile("file", guessFileName(fileURL, mimeType))
	if err != nil {
		return "", err
	}
	if _, err := part.Write(fileBytes); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	apiReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	apiReq.Header.Set("Content-Type", writer.FormDataContentType())

	apiResp, err := s.httpClient.Do(apiReq)
	if err != nil {
		return "", err
	}
	defer apiResp.Body.Close()
	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return "", err
	}
	if apiResp.StatusCode >= 300 {
		return "", fmt.Errorf("transcription API error: status=%d body=%s", apiResp.StatusCode, string(body))
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse transcription response: %w", err)
	}
	if strings.TrimSpace(parsed.Text) == "" {
		return "", fmt.Errorf("empty transcript returned by API")
	}
	return parsed.Text, nil
}

type StubTranscriptService struct{}

func (s *StubTranscriptService) TranscribeFromURL(ctx context.Context, fileURL, mimeType string) (string, error) {
	return "", fmt.Errorf("transcription service is not configured")
}

func guessFileName(fileURL, mimeType string) string {
	name := filepath.Base(fileURL)
	if name == "." || name == "/" || name == "" {
		switch {
		case strings.Contains(mimeType, "mpeg"):
			return "meeting.mp3"
		case strings.Contains(mimeType, "ogg"):
			return "meeting.ogg"
		case strings.Contains(mimeType, "mp4"):
			return "meeting.m4a"
		default:
			return "meeting.audio"
		}
	}
	return name
}
