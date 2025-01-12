package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
)

type ShareHandler struct {
	config  config.Config
	storage storage.VideoStorage
	share   storage.ShareLinkStorage
}

func NewShareHandler(cfg config.Config, store storage.VideoStorage, shareStore storage.ShareLinkStorage) *ShareHandler {
	if store == nil {
		panic("video storage cannot be nil")
	}
	if shareStore == nil {
		panic("share storage cannot be nil")
	}
	return &ShareHandler{
		config:  cfg,
		storage: store,
		share:   shareStore,
	}
}

type CreateShareRequest struct {
	VideoID  string `json:"video_id"`
	Duration int    `json:"duration"`
}

func (h *ShareHandler) HandleShares(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateShare(w, r)
	case http.MethodGet:
		h.handleListShares(w, r)
	default:
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ShareHandler) HandleShareOperations(w http.ResponseWriter, r *http.Request) {
	shareID := strings.TrimPrefix(r.URL.Path, "/api/shares/")
	if shareID == "" {
		SendError(w, http.StatusBadRequest, "share ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetShare(w, r, shareID)
	case http.MethodDelete:
		h.handleDeleteShare(w, r, shareID)
	default:
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ShareHandler) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	var req CreateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	video, err := h.storage.GetVideo(r.Context(), req.VideoID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to get video")
		return
	}
	if video == nil {
		SendError(w, http.StatusNotFound, "video not found")
		return
	}

	if req.Duration <= 0 || req.Duration > 168 {
		SendError(w, http.StatusBadRequest, "invalid duration (must be between 1 and 168 hours)")
		return
	}

	shareID, err := generateID()
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to generate share ID")
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.Duration) * time.Hour)

	shareLink := &storage.ShareLink{
		ID:        shareID,
		VideoID:   req.VideoID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := h.share.SaveShareLink(r.Context(), shareLink); err != nil {
		SendError(w, http.StatusInternalServerError, "failed to save share link")
		return
	}

	SendSuccess(w, http.StatusCreated, shareLink, "share link created successfully")
}

func (h *ShareHandler) handleListShares(w http.ResponseWriter, r *http.Request) {
	videoID := r.URL.Query().Get("video_id")

	var shareLinks []*storage.ShareLink
	var err error

	if videoID != "" {
		shareLinks, err = h.share.GetShareLinksByVideo(r.Context(), videoID)
	} else {
		shareLinks, err = h.share.ListShareLinks(r.Context())
	}

	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to list share links")
		return
	}

	validLinks := make([]*storage.ShareLink, 0)
	now := time.Now()
	for _, link := range shareLinks {
		if link.ExpiresAt.After(now) {
			validLinks = append(validLinks, link)
		}
	}

	SendSuccess(w, http.StatusOK, validLinks, "")
}

func (h *ShareHandler) handleGetShare(w http.ResponseWriter, r *http.Request, shareID string) {
	shareLink, err := h.share.GetShareLink(r.Context(), shareID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to get share link")
		return
	}
	if shareLink == nil {
		SendError(w, http.StatusNotFound, "share link not found")
		return
	}

	if time.Now().After(shareLink.ExpiresAt) {
		SendError(w, http.StatusGone, "share link has expired")
		return
	}

	video, err := h.storage.GetVideo(r.Context(), shareLink.VideoID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to get video")
		return
	}
	if video == nil {
		SendError(w, http.StatusNotFound, "video not found")
		return
	}

	response := struct {
		ShareLink *storage.ShareLink `json:"share_link"`
		Video     *storage.Video     `json:"video"`
	}{
		ShareLink: shareLink,
		Video:     video,
	}

	SendSuccess(w, http.StatusOK, response, "")
}

func (h *ShareHandler) handleDeleteShare(w http.ResponseWriter, r *http.Request, shareID string) {
	if err := h.share.DeleteShareLink(r.Context(), shareID); err != nil {
		SendError(w, http.StatusInternalServerError, "failed to delete share link")
		return
	}

	SendSuccess(w, http.StatusOK, nil, "share link deleted successfully")
}
