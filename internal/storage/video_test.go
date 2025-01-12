package storage

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS videos (
            id TEXT PRIMARY KEY,
            filename TEXT NOT NULL,
            size INTEGER NOT NULL,
            duration INTEGER NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            status TEXT NOT NULL,
            error_message TEXT
        );

        CREATE TABLE IF NOT EXISTS share_links (
            id TEXT PRIMARY KEY,
            video_id TEXT NOT NULL,
            expires_at TIMESTAMP NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY (video_id) REFERENCES videos(id)
        );
    `)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db, func() {
		db.Close()
	}
}

func TestVideoStorage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	storage := NewVideoStorage(db).(*SQLiteVideoStorage)
	ctx := context.Background()

	t.Run("SaveAndGetVideo", func(t *testing.T) {
		video := &Video{
			ID:       "test-id",
			Filename: "test.mp4",
			Size:     1000,
			Duration: 60,
			Status:   StatusCompleted,
		}

		err := storage.SaveVideo(ctx, video)
		if err != nil {
			t.Fatalf("SaveVideo failed: %v", err)
		}

		retrieved, err := storage.GetVideo(ctx, video.ID)
		if err != nil {
			t.Fatalf("GetVideo failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("GetVideo returned nil for existing video")
		}

		if retrieved.ID != video.ID ||
			retrieved.Filename != video.Filename ||
			retrieved.Size != video.Size ||
			retrieved.Duration != video.Duration ||
			retrieved.Status != video.Status {
			t.Errorf("Retrieved video doesn't match saved video")
		}
	})

	t.Run("ListVideos", func(t *testing.T) {
		videos := []*Video{
			{
				ID:       "test-id-1",
				Filename: "test1.mp4",
				Size:     1000,
				Duration: 60,
				Status:   StatusCompleted,
			},
			{
				ID:       "test-id-2",
				Filename: "test2.mp4",
				Size:     2000,
				Duration: 120,
				Status:   StatusCompleted,
			},
		}

		for _, v := range videos {
			err := storage.SaveVideo(ctx, v)
			if err != nil {
				t.Fatalf("Failed to save test video: %v", err)
			}
		}

		listed, err := storage.ListVideos(ctx)
		if err != nil {
			t.Fatalf("ListVideos failed: %v", err)
		}

		if len(listed) < len(videos) {
			t.Errorf("Expected at least %d videos, got %d", len(videos), len(listed))
		}
	})

	t.Run("UpdateVideoStatus", func(t *testing.T) {
		videoID := "test-status-update"
		video := &Video{
			ID:       videoID,
			Filename: "test.mp4",
			Size:     1000,
			Duration: 60,
			Status:   StatusPending,
		}

		err := storage.SaveVideo(ctx, video)
		if err != nil {
			t.Fatalf("Failed to save test video: %v", err)
		}

		errorMsg := "test error"
		err = storage.UpdateVideoStatus(ctx, videoID, StatusFailed, &errorMsg)
		if err != nil {
			t.Fatalf("UpdateVideoStatus failed: %v", err)
		}

		updated, err := storage.GetVideo(ctx, videoID)
		if err != nil {
			t.Fatalf("Failed to get updated video: %v", err)
		}
		if updated == nil {
			t.Fatal("GetVideo returned nil after status update")
		}

		if updated.Status != StatusFailed {
			t.Errorf("Expected status %v, got %v", StatusFailed, updated.Status)
		}
		if updated.ErrorMessage == nil || *updated.ErrorMessage != errorMsg {
			t.Errorf("Expected error message %v, got %v", errorMsg, updated.ErrorMessage)
		}
	})

	t.Run("GetNonExistentVideo", func(t *testing.T) {
		video, err := storage.GetVideo(ctx, "non-existent-id")
		if err != nil {
			t.Errorf("Expected nil error for non-existent video, got: %v", err)
		}
		if video != nil {
			t.Errorf("Expected nil video for non-existent ID, got: %v", video)
		}
	})
}
