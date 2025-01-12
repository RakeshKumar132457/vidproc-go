package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupShareTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS share_links (
            id TEXT PRIMARY KEY,
            video_id TEXT NOT NULL,
            expires_at TIMESTAMP NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	if err != nil {
		t.Fatalf("Failed to create share_links table: %v", err)
	}

	return db, func() {
		db.Close()
	}
}

func TestShareLinkStorage(t *testing.T) {
	db, cleanup := setupShareTestDB(t)
	defer cleanup()

	storage := NewShareLinkStorage(db)
	ctx := context.Background()

	testLink := &ShareLink{
		ID:        "test-share-id",
		VideoID:   "test-video-id",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	t.Run("SaveShareLink", func(t *testing.T) {
		err := storage.SaveShareLink(ctx, testLink)
		if err != nil {
			t.Errorf("SaveShareLink failed: %v", err)
		}
	})

	t.Run("GetShareLink", func(t *testing.T) {
		link, err := storage.GetShareLink(ctx, testLink.ID)
		if err != nil {
			t.Errorf("GetShareLink failed: %v", err)
		}
		if link == nil {
			t.Error("GetShareLink returned nil link")
			return
		}
		if link.ID != testLink.ID {
			t.Errorf("GetShareLink returned wrong link: got %v, want %v", link.ID, testLink.ID)
		}
	})

	t.Run("GetShareLinksByVideo", func(t *testing.T) {
		links, err := storage.GetShareLinksByVideo(ctx, testLink.VideoID)
		if err != nil {
			t.Errorf("GetShareLinksByVideo failed: %v", err)
		}
		if len(links) == 0 {
			t.Error("GetShareLinksByVideo returned empty list")
		}
	})

	t.Run("ListShareLinks", func(t *testing.T) {
		links, err := storage.ListShareLinks(ctx)
		if err != nil {
			t.Errorf("ListShareLinks failed: %v", err)
		}
		if len(links) == 0 {
			t.Error("ListShareLinks returned empty list")
		}
	})

	t.Run("DeleteShareLink", func(t *testing.T) {
		err := storage.DeleteShareLink(ctx, testLink.ID)
		if err != nil {
			t.Errorf("DeleteShareLink failed: %v", err)
		}

		link, err := storage.GetShareLink(ctx, testLink.ID)
		if err != nil {
			t.Errorf("GetShareLink after delete failed: %v", err)
		}
		if link != nil {
			t.Error("Share link was not deleted")
		}
	})
}
