package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"vidproc-go/internal/storage"
)

func TestHandleCreateShare(t *testing.T) {
	cfg, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockStorage := NewMockStorage()
	handler := NewShareHandler(cfg, mockStorage)

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
		shareReq   CreateShareRequest
		wantStatus int
		wantErrMsg string
	}{
		{
			name: "valid share request",
			shareReq: CreateShareRequest{
				VideoID:  "test-video",
				Duration: 24,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "invalid video ID",
			shareReq: CreateShareRequest{
				VideoID:  "nonexistent",
				Duration: 24,
			},
			wantStatus: http.StatusNotFound,
			wantErrMsg: "video not found",
		},
		{
			name: "invalid duration",
			shareReq: CreateShareRequest{
				VideoID:  "test-video",
				Duration: 0,
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid duration (must be between 1 and 168 hours)",
		},
		{
			name: "duration too long",
			shareReq: CreateShareRequest{
				VideoID:  "test-video",
				Duration: 169,
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid duration (must be between 1 and 168 hours)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.shareReq)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/shares", bytes.NewBuffer(reqBody))
			rr := httptest.NewRecorder()

			handler.HandleShares(rr, req)

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

func TestHandleGetShare(t *testing.T) {
	cfg, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockStorage := NewMockStorage()
	handler := NewShareHandler(cfg, mockStorage)

	testVideo := &storage.Video{
		ID:       "test-video",
		Filename: "test.mp4",
		Size:     1000,
		Duration: 10,
		Status:   storage.StatusCompleted,
	}
	mockStorage.SaveVideo(context.Background(), testVideo)

	validShare := &storage.ShareLink{
		ID:        "valid-share",
		VideoID:   "test-video",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	mockStorage.SaveShareLink(context.Background(), validShare)

	expiredShare := &storage.ShareLink{
		ID:        "expired-share",
		VideoID:   "test-video",
		ExpiresAt: time.Now().Add(-24 * time.Hour),
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	mockStorage.SaveShareLink(context.Background(), expiredShare)

	tests := []struct {
		name       string
		shareID    string
		wantStatus int
		wantErrMsg string
	}{
		{
			name:       "valid share",
			shareID:    "valid-share",
			wantStatus: http.StatusOK,
		},
		{
			name:       "expired share",
			shareID:    "expired-share",
			wantStatus: http.StatusGone,
			wantErrMsg: "share link has expired",
		},
		{
			name:       "nonexistent share",
			shareID:    "nonexistent",
			wantStatus: http.StatusNotFound,
			wantErrMsg: "share link not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/shares/"+tt.shareID, nil)
			rr := httptest.NewRecorder()

			handler.HandleShareOperations(rr, req)

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

func TestHandleListShares(t *testing.T) {
	cfg, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockStorage := NewMockStorage()
	handler := NewShareHandler(cfg, mockStorage)

	testVideos := map[string]*storage.Video{
		"video1": {
			ID:       "video1",
			Filename: "test1.mp4",
			Status:   storage.StatusCompleted,
		},
		"video2": {
			ID:       "video2",
			Filename: "test2.mp4",
			Status:   storage.StatusCompleted,
		},
	}

	for _, video := range testVideos {
		mockStorage.SaveVideo(context.Background(), video)
	}

	shareLinks := []*storage.ShareLink{
		{
			ID:        "share1",
			VideoID:   "video1",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		},
		{
			ID:        "share2",
			VideoID:   "video1",
			ExpiresAt: time.Now().Add(-24 * time.Hour),
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
		{
			ID:        "share3",
			VideoID:   "video2",
			ExpiresAt: time.Now().Add(48 * time.Hour),
			CreatedAt: time.Now(),
		},
	}

	for _, link := range shareLinks {
		mockStorage.SaveShareLink(context.Background(), link)
	}

	tests := []struct {
		name       string
		videoID    string
		wantStatus int
		wantCount  int
		wantErrMsg string
	}{
		{
			name:       "list all valid shares",
			videoID:    "",
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "list shares for video1",
			videoID:    "video1",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "list shares for video2",
			videoID:    "video2",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "list shares for nonexistent video",
			videoID:    "nonexistent",
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/shares"
			if tt.videoID != "" {
				url += "?video_id=" + tt.videoID
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			handler.HandleShares(rr, req)

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
			} else {
				var response struct {
					Status  string               `json:"status"`
					Data    []*storage.ShareLink `json:"data"`
					Message string               `json:"message"`
				}
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if len(response.Data) != tt.wantCount {
					t.Errorf("Handler returned wrong number of shares: got %v want %v",
						len(response.Data), tt.wantCount)
				}
			}
		})
	}
}

func TestHandleDeleteShare(t *testing.T) {
	cfg, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	mockStorage := NewMockStorage()
	handler := NewShareHandler(cfg, mockStorage)

	testVideo := &storage.Video{
		ID:       "test-video",
		Filename: "test.mp4",
		Status:   storage.StatusCompleted,
	}
	mockStorage.SaveVideo(context.Background(), testVideo)

	shareLink := &storage.ShareLink{
		ID:        "test-share",
		VideoID:   "test-video",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	mockStorage.SaveShareLink(context.Background(), shareLink)

	tests := []struct {
		name       string
		shareID    string
		wantStatus int
		wantErrMsg string
	}{
		{
			name:       "delete existing share",
			shareID:    "test-share",
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete nonexistent share",
			shareID:    "nonexistent",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/shares/"+tt.shareID, nil)
			rr := httptest.NewRecorder()

			handler.HandleShareOperations(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v",
					status, tt.wantStatus)
			}

			if tt.shareID == "test-share" {
				share, err := mockStorage.GetShareLink(context.Background(), tt.shareID)
				if err != nil {
					t.Fatalf("Failed to check share deletion: %v", err)
				}
				if share != nil {
					t.Error("Share was not deleted")
				}
			}
		})
	}
}

func (m *MockVideoStorage) SaveShareLink(ctx context.Context, link *storage.ShareLink) error {
	m.shareLinks[link.ID] = link
	return nil
}

func (m *MockVideoStorage) GetShareLink(ctx context.Context, id string) (*storage.ShareLink, error) {
	link, exists := m.shareLinks[id]
	if !exists {
		return nil, nil
	}
	return link, nil
}

func (m *MockVideoStorage) GetShareLinksByVideo(ctx context.Context, videoID string) ([]*storage.ShareLink, error) {
	var links []*storage.ShareLink
	for _, link := range m.shareLinks {
		if link.VideoID == videoID {
			links = append(links, link)
		}
	}
	return links, nil
}

func (m *MockVideoStorage) ListShareLinks(ctx context.Context) ([]*storage.ShareLink, error) {
	var links []*storage.ShareLink
	for _, link := range m.shareLinks {
		links = append(links, link)
	}
	return links, nil
}

func (m *MockVideoStorage) DeleteShareLink(ctx context.Context, id string) error {
	delete(m.shareLinks, id)
	return nil
}

type MockVideoStorage struct {
	videos     map[string]*storage.Video
	shareLinks map[string]*storage.ShareLink
}

func NewMockStorage() *MockVideoStorage {
	return &MockVideoStorage{
		videos:     make(map[string]*storage.Video),
		shareLinks: make(map[string]*storage.ShareLink),
	}
}
