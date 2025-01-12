package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
)

type MockVideoStorage struct {
	videos map[string]*storage.Video
}

func NewMockStorage() *MockVideoStorage {
	return &MockVideoStorage{
		videos: make(map[string]*storage.Video),
	}
}

func (m *MockVideoStorage) SaveVideo(ctx context.Context, video *storage.Video) error {
	m.videos[video.ID] = video
	return nil
}

func (m *MockVideoStorage) GetVideo(ctx context.Context, id string) (*storage.Video, error) {
	if video, exists := m.videos[id]; exists {
		return video, nil
	}
	return nil, nil
}

func (m *MockVideoStorage) ListVideos(ctx context.Context) ([]*storage.Video, error) {
	videos := make([]*storage.Video, 0, len(m.videos))
	for _, v := range m.videos {
		videos = append(videos, v)
	}
	return videos, nil
}

func (m *MockVideoStorage) UpdateVideoStatus(ctx context.Context, id string, status storage.VideoStatus, errorMsg *string) error {
	if video, exists := m.videos[id]; exists {
		video.Status = status
		video.ErrorMessage = errorMsg
		return nil
	}
	return nil
}

func setupTestEnvironment(t *testing.T) (config.Config, string, func()) {

	tmpDir, err := os.MkdirTemp("", "videoapi-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := config.Config{
		VideoStoragePath: tmpDir,
		MaxVideoSize:     10 * 1024 * 1024,
		MaxDuration:      30,
		MinDuration:      1,
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return cfg, tmpDir, cleanup
}

func createTestVideo(t *testing.T) (*bytes.Buffer, *multipart.Writer) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("video", "test.mp4")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	_, err = fw.Write([]byte("dummy video content"))
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	w.Close()
	return &b, w
}

func TestHandleUpload(t *testing.T) {
	cfg, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockStorage := NewMockStorage()
	handler := NewVideoHandler(cfg, mockStorage)

	tests := []struct {
		name         string
		method       string
		setupReq     func() (*http.Request, error)
		expectedCode int
	}{
		{
			name:   "successful upload",
			method: http.MethodPost,
			setupReq: func() (*http.Request, error) {
				body, w := createTestVideo(t)
				req := httptest.NewRequest(http.MethodPost, "/api/videos", body)
				req.Header.Set("Content-Type", w.FormDataContentType())
				return req, nil
			},
			expectedCode: http.StatusCreated,
		},
		{
			name:   "wrong method",
			method: http.MethodGet,
			setupReq: func() (*http.Request, error) {
				return httptest.NewRequest(http.MethodGet, "/api/videos", nil), nil
			},
			expectedCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := tt.setupReq()
			if err != nil {
				t.Fatalf("Failed to setup request: %v", err)
			}

			rr := httptest.NewRecorder()
			handler.HandleUpload(rr, req)

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedCode)
			}
		})
	}
}

