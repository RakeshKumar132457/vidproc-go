package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
)

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
			handler.handleUpload(rr, req)

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedCode)
			}
		})
	}
}

func TestHandleTrim(t *testing.T) {
	cfg, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockStorage := NewMockStorage()
	handler := NewVideoHandler(cfg, mockStorage)

	testVideo := &storage.Video{
		ID:       "test-video",
		Filename: "test.mp4",
		Size:     1000,
		Duration: 10,
		Status:   storage.StatusCompleted,
	}
	mockStorage.SaveVideo(context.Background(), testVideo)

	tests := []struct {
		name       string
		videoID    string
		trimReq    TrimRequest
		wantStatus int
		wantErrMsg string
	}{
		{
			name:    "valid trim request",
			videoID: "test-video",
			trimReq: TrimRequest{
				Start: 0,
				End:   5,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid video ID",
			videoID: "nonexistent",
			trimReq: TrimRequest{
				Start: 0,
				End:   5,
			},
			wantStatus: http.StatusNotFound,
			wantErrMsg: "video not found",
		},
		{
			name:    "invalid trim parameters",
			videoID: "test-video",
			trimReq: TrimRequest{
				Start: 6,
				End:   5,
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid trim parameters",
		},
		{
			name:    "trim end beyond duration",
			videoID: "test-video",
			trimReq: TrimRequest{
				Start: 0,
				End:   15,
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid trim parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.trimReq)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/videos/trim/"+tt.videoID, bytes.NewBuffer(reqBody))
			rr := httptest.NewRecorder()

			handler.HandleTrim(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}

			if tt.wantErrMsg != "" {
				var response Response
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Error != tt.wantErrMsg {
					t.Errorf("Handler returned wrong error message: got %v want %v",
						response.Error, tt.wantErrMsg)
				}
			}
		})
	}
}

func TestHandleMerge(t *testing.T) {
	cfg, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockStorage := NewMockStorage()
	handler := NewVideoHandler(cfg, mockStorage)

	testVideos := []*storage.Video{
		{
			ID:       "video1",
			Filename: "test1.mp4",
			Size:     1000,
			Duration: 10,
			Status:   storage.StatusCompleted,
		},
		{
			ID:       "video2",
			Filename: "test2.mp4",
			Size:     1000,
			Duration: 10,
			Status:   storage.StatusCompleted,
		},
	}

	for _, video := range testVideos {
		mockStorage.SaveVideo(context.Background(), video)
	}

	tests := []struct {
		name       string
		mergeReq   MergeRequest
		wantStatus int
		wantErrMsg string
	}{
		{
			name: "valid merge request",
			mergeReq: MergeRequest{
				VideoIDs: []string{"video1", "video2"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "insufficient videos",
			mergeReq: MergeRequest{
				VideoIDs: []string{"video1"},
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "at least two videos required for merging",
		},
		{
			name: "nonexistent video",
			mergeReq: MergeRequest{
				VideoIDs: []string{"video1", "nonexistent"},
			},
			wantStatus: http.StatusNotFound,
			wantErrMsg: "video nonexistent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.mergeReq)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/videos/merge", bytes.NewBuffer(reqBody))
			rr := httptest.NewRecorder()

			handler.HandleMerge(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}

			if tt.wantErrMsg != "" {
				var response Response
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Error != tt.wantErrMsg {
					t.Errorf("Handler returned wrong error message: got %v want %v",
						response.Error, tt.wantErrMsg)
				}
			}
		})
	}
}
