package e2e

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"vidproc-go/internal/api"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
)

func setupTestEnvironment(t *testing.T) (*httptest.Server, config.Config, func()) {

	tmpDir, err := os.MkdirTemp("", "vidproc-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	videoDir := filepath.Join(tmpDir, "videos")
	dbDir := filepath.Join(tmpDir, "db")

	err = os.MkdirAll(videoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create video directory: %v", err)
	}

	err = os.MkdirAll(dbDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}

	cfg := config.Config{
		Port:             "8080",
		Environment:      "test",
		APIToken:         "test-token",
		DBPath:           filepath.Join(dbDir, "test.db"),
		VideoStoragePath: videoDir,
		MaxVideoSize:     10 * 1024 * 1024,
		MaxDuration:      300,
		MinDuration:      1,
	}

	db, err := sql.Open("sqlite3", filepath.Join(dbDir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
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

	router := api.NewRouter(db, cfg)
	server := httptest.NewServer(router.SetupRoutes())

	cleanup := func() {
		server.Close()
		os.RemoveAll(tmpDir)
	}

	return server, cfg, cleanup
}

func createTestVideo(t *testing.T) (*bytes.Buffer, *multipart.Writer) {

	tmpFile := filepath.Join(os.TempDir(), "test.mp4")
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "testsrc=duration=5:size=1280x720:rate=30",
		"-c:v", "libx264",
		"-y",
		tmpFile,
	)

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create test video: %v", err)
	}
	defer os.Remove(tmpFile)

	videoData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read test video: %v", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("video", "test.mp4")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := part.Write(videoData); err != nil {
		t.Fatalf("Failed to write video data: %v", err)
	}

	writer.Close()
	return &buf, writer
}

func checkDependencies(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found, skipping video processing tests")
	}
}

func TestE2EVideoProcessing(t *testing.T) {
	checkDependencies(t)
	server, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("UploadVideo", func(t *testing.T) {
		body, writer := createTestVideo(t)

		req, err := http.NewRequest(http.MethodPost, server.URL+"/api/videos", body)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status %d, got %d\nResponse body: %s", http.StatusCreated, resp.StatusCode, string(body))
		}

		var videoResp struct {
			Status  string         `json:"status"`
			Data    *storage.Video `json:"data"`
			Message string         `json:"message"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		videoID := videoResp.Data.ID

		t.Run("TrimVideo", func(t *testing.T) {
			trimReq := api.TrimRequest{
				Start: 0,
				End:   5,
			}

			reqBody, _ := json.Marshal(trimReq)
			req, _ := http.NewRequest(http.MethodPost,
				fmt.Sprintf("%s/api/videos/trim/%s", server.URL, videoID),
				bytes.NewBuffer(reqBody))

			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Trim request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}
		})

		t.Run("ShareVideo", func(t *testing.T) {
			shareReq := api.CreateShareRequest{
				VideoID:  videoID,
				Duration: 24,
			}

			reqBody, _ := json.Marshal(shareReq)
			req, _ := http.NewRequest(http.MethodPost,
				server.URL+"/api/shares",
				bytes.NewBuffer(reqBody))

			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Share request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
			}

			var shareResp struct {
				Status  string             `json:"status"`
				Data    *storage.ShareLink `json:"data"`
				Message string             `json:"message"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&shareResp); err != nil {
				t.Fatalf("Failed to decode share response: %v", err)
			}

			shareID := shareResp.Data.ID
			req, _ = http.NewRequest(http.MethodGet,
				fmt.Sprintf("%s/api/shares/%s", server.URL, shareID),
				nil)

			req.Header.Set("Authorization", "Bearer test-token")

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Share access request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}
		})
	})
}

func TestE2EErrorCases(t *testing.T) {
	server, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("InvalidAuth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/videos", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
	})

	t.Run("InvalidVideoFormat", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, _ := writer.CreateFormFile("video", "test.txt")
		part.Write([]byte("not a video"))
		writer.Close()

		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/videos", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer test-token")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})
}
