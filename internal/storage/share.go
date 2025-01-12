package storage

import (
	"context"
	"database/sql"
	"time"
)

type ShareLink struct {
	ID        string    `json:"id"`
	VideoID   string    `json:"video_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type SQLiteShareLinkStorage struct {
	db *sql.DB
}

func NewShareLinkStorage(db *sql.DB) ShareLinkStorage {
	return &SQLiteShareLinkStorage{db: db}
}

type ShareLinkStorage interface {
	SaveShareLink(ctx context.Context, link *ShareLink) error
	GetShareLink(ctx context.Context, id string) (*ShareLink, error)
	GetShareLinksByVideo(ctx context.Context, videoID string) ([]*ShareLink, error)
	ListShareLinks(ctx context.Context) ([]*ShareLink, error)
	DeleteShareLink(ctx context.Context, id string) error
}

func (s *SQLiteShareLinkStorage) SaveShareLink(ctx context.Context, link *ShareLink) error {
	query := `
        INSERT INTO share_links (id, video_id, expires_at, created_at)
        VALUES (?, ?, ?, ?)
    `
	_, err := s.db.ExecContext(ctx, query,
		link.ID,
		link.VideoID,
		link.ExpiresAt,
		link.CreatedAt,
	)
	return err
}

func (s *SQLiteShareLinkStorage) GetShareLink(ctx context.Context, id string) (*ShareLink, error) {
	query := `
        SELECT id, video_id, expires_at, created_at
        FROM share_links
        WHERE id = ?
    `
	var link ShareLink
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&link.ID,
		&link.VideoID,
		&link.ExpiresAt,
		&link.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (s *SQLiteShareLinkStorage) GetShareLinksByVideo(ctx context.Context, videoID string) ([]*ShareLink, error) {
	query := `
        SELECT id, video_id, expires_at, created_at
        FROM share_links
        WHERE video_id = ?
        ORDER BY created_at DESC
    `
	rows, err := s.db.QueryContext(ctx, query, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*ShareLink
	for rows.Next() {
		var link ShareLink
		err := rows.Scan(
			&link.ID,
			&link.VideoID,
			&link.ExpiresAt,
			&link.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		links = append(links, &link)
	}
	return links, rows.Err()
}

func (s *SQLiteShareLinkStorage) ListShareLinks(ctx context.Context) ([]*ShareLink, error) {
	query := `
        SELECT id, video_id, expires_at, created_at
        FROM share_links
        ORDER BY created_at DESC
    `
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*ShareLink
	for rows.Next() {
		var link ShareLink
		err := rows.Scan(
			&link.ID,
			&link.VideoID,
			&link.ExpiresAt,
			&link.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		links = append(links, &link)
	}
	return links, rows.Err()
}

func (s *SQLiteShareLinkStorage) DeleteShareLink(ctx context.Context, id string) error {
	query := `DELETE FROM share_links WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
