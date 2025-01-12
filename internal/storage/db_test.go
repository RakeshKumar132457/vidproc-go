package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDB(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "videoapi-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
	}{
		{
			name:    "valid database path",
			dbPath:  filepath.Join(tmpDir, "test.db"),
			wantErr: false,
		},
		{
			name:    "invalid path",
			dbPath:  "/nonexistent/path/test.db",
			wantErr: true,
		},
		{
			name:    "empty path",
			dbPath:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDB(tt.dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				defer db.Close()

				var tables = []string{"videos", "share_links"}
				for _, table := range tables {
					var name string
					err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
					if err != nil {
						t.Errorf("Table %s was not created", table)
					}
				}

				var indexes = []string{
					"idx_videos_status",
					"idx_share_links_video_id",
					"idx_share_links_expires_at",
				}
				for _, index := range indexes {
					var name string
					err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&name)
					if err != nil {
						t.Errorf("Index %s was not created", index)
					}
				}
			}
		})
	}
}

