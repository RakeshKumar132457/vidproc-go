package storage

import (
	"context"
	"database/sql"
	"time"
)

type VideoStatus string

const (
	StatusPending    VideoStatus = "pending"
	StatusProcessing VideoStatus = "processing"
	StatusCompleted  VideoStatus = "completed"
	StatusFailed     VideoStatus = "failed"
)

type Video struct {
	ID           string      `json:"id"`
	Filename     string      `json:"filename"`
	Size         int64       `json:"size"`
	Duration     int         `json:"duration"`
	CreatedAt    time.Time   `json:"created_at"`
	Status       VideoStatus `json:"status"`
	ErrorMessage *string     `json:"error_message,omitempty"`
}

type VideoStorage interface {
	SaveVideo(ctx context.Context, video *Video) error
	GetVideo(ctx context.Context, id string) (*Video, error)
	ListVideos(ctx context.Context) ([]*Video, error)
	UpdateVideoStatus(ctx context.Context, id string, status VideoStatus, errorMsg *string) error
}

type SQLiteVideoStorage struct {
	db *sql.DB
}

func NewVideoStorage(db *sql.DB) VideoStorage {
	return &SQLiteVideoStorage{db: db}
}

func (s *SQLiteVideoStorage) SaveVideo(ctx context.Context, video *Video) error {
	query := `
        INSERT INTO videos (id, filename, size, duration, status, error_message)
        VALUES (?, ?, ?, ?, ?, ?)
    `
	_, err := s.db.ExecContext(ctx, query,
		video.ID,
		video.Filename,
		video.Size,
		video.Duration,
		video.Status,
		video.ErrorMessage,
	)
	return err
}

func (s *SQLiteVideoStorage) GetVideo(ctx context.Context, id string) (*Video, error) {
	query := `
        SELECT id, filename, size, duration, created_at, status, error_message
        FROM videos
        WHERE id = ?
    `
	var video Video
	var errorMsg sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&video.ID,
		&video.Filename,
		&video.Size,
		&video.Duration,
		&video.CreatedAt,
		&video.Status,
		&errorMsg,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if errorMsg.Valid {
		video.ErrorMessage = &errorMsg.String
	}
	return &video, nil
}

func (s *SQLiteVideoStorage) ListVideos(ctx context.Context) ([]*Video, error) {
	query := `
        SELECT id, filename, size, duration, created_at, status, error_message
        FROM videos
        ORDER BY created_at DESC
    `
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*Video
	for rows.Next() {
		var video Video
		var errorMsg sql.NullString
		err := rows.Scan(
			&video.ID,
			&video.Filename,
			&video.Size,
			&video.Duration,
			&video.CreatedAt,
			&video.Status,
			&errorMsg,
		)
		if err != nil {
			return nil, err
		}
		if errorMsg.Valid {
			video.ErrorMessage = &errorMsg.String
		}
		videos = append(videos, &video)
	}
	return videos, rows.Err()
}

func (s *SQLiteVideoStorage) UpdateVideoStatus(ctx context.Context, id string, status VideoStatus, errorMsg *string) error {
	query := `
        UPDATE videos
        SET status = ?, error_message = ?
        WHERE id = ?
    `
	_, err := s.db.ExecContext(ctx, query, status, errorMsg, id)
	return err
}
