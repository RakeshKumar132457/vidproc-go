package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
	"vidproc-go/internal/video"
)

type VideoHandler struct {
	config    config.Config
	storage   storage.VideoStorage
	processor video.Processor
}

func NewVideoHandler(cfg config.Config, store storage.VideoStorage) *VideoHandler {
	return &VideoHandler{
		config:    cfg,
		storage:   store,
		processor: video.NewFFmpegProcessor(),
	}
}

func (h *VideoHandler) HandleVideos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleUpload(w, r)
	case http.MethodGet:
		h.handleList(w, r)
	default:
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *VideoHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.config.MaxVideoSize); err != nil {
		SendError(w, http.StatusBadRequest, "file too large")
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		SendError(w, http.StatusBadRequest, "failed to get video file")
		return
	}
	defer file.Close()

	if !isValidVideoType(header.Filename) {
		SendError(w, http.StatusBadRequest, "invalid video format")
		return
	}

	id, err := generateID()
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to generate video ID")
		return
	}

	filename := fmt.Sprintf("%s_%s", id, filepath.Base(header.Filename))
	filepath := filepath.Join(h.config.VideoStoragePath, filename)

	size, err := h.saveUploadedFile(file, filepath)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to save video")
		return
	}

	info, err := h.processor.GetVideoInfo(r.Context(), filepath)
	if err != nil {
		os.Remove(filepath)
		SendError(w, http.StatusBadRequest, "invalid video file")
		return
	}

	if info.Duration < float64(h.config.MinDuration) || info.Duration > float64(h.config.MaxDuration) {
		os.Remove(filepath)
		SendError(w, http.StatusBadRequest, fmt.Sprintf("video duration must be between %d and %d seconds",
			h.config.MinDuration, h.config.MaxDuration))
		return
	}

	video := &storage.Video{
		ID:       id,
		Filename: filename,
		Size:     size,
		Duration: int(info.Duration),
		Status:   storage.StatusCompleted,
	}

	if err := h.storage.SaveVideo(r.Context(), video); err != nil {
		os.Remove(filepath)
		SendError(w, http.StatusInternalServerError, "failed to save video metadata")
		return
	}

	SendSuccess(w, http.StatusCreated, video, "video uploaded successfully")
}

func (h *VideoHandler) handleList(w http.ResponseWriter, r *http.Request) {
	videos, err := h.storage.ListVideos(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to list videos")
		return
	}

	SendSuccess(w, http.StatusOK, videos, "")
}

type TrimRequest struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

func (h *VideoHandler) HandleTrim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	videoID := strings.TrimPrefix(r.URL.Path, "/api/videos/trim/")
	if videoID == "" {
		SendError(w, http.StatusBadRequest, "video ID required")
		return
	}

	video, err := h.storage.GetVideo(r.Context(), videoID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to get video")
		return
	}
	if video == nil {
		SendError(w, http.StatusNotFound, "video not found")
		return
	}

	var req TrimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Start < 0 || req.End > float64(video.Duration) || req.Start >= req.End {
		SendError(w, http.StatusBadRequest, "invalid trim parameters")
		return
	}

	trimmedID, _ := generateID()
	originalExt := filepath.Ext(video.Filename)
	trimmedFilename := fmt.Sprintf("%s_trimmed%s", trimmedID, originalExt)
	trimmedPath := filepath.Join(h.config.VideoStoragePath, trimmedFilename)

	originalPath := filepath.Join(h.config.VideoStoragePath, video.Filename)
	if err := h.processor.Trim(r.Context(), originalPath, trimmedPath, req.Start, req.End); err != nil {
		SendError(w, http.StatusInternalServerError, "failed to trim video")
		return
	}

	info, err := h.processor.GetVideoInfo(r.Context(), trimmedPath)
	if err != nil {
		os.Remove(trimmedPath)
		SendError(w, http.StatusInternalServerError, "failed to process trimmed video")
		return
	}

	trimmedVideo := &storage.Video{
		ID:       trimmedID,
		Filename: trimmedFilename,
		Size:     info.Size,
		Duration: int(info.Duration),
		Status:   storage.StatusCompleted,
	}

	if err := h.storage.SaveVideo(r.Context(), trimmedVideo); err != nil {
		os.Remove(trimmedPath)
		SendError(w, http.StatusInternalServerError, "failed to save trimmed video metadata")
		return
	}

	SendSuccess(w, http.StatusOK, trimmedVideo, "video trimmed successfully")
}

type MergeRequest struct {
	VideoIDs []string `json:"video_ids"`
}

func (h *VideoHandler) HandleMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req MergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.VideoIDs) < 2 {
		SendError(w, http.StatusBadRequest, "at least two videos required for merging")
		return
	}

	var videoPaths []string
	var totalDuration int

	for _, id := range req.VideoIDs {
		video, err := h.storage.GetVideo(r.Context(), id)
		if err != nil {
			SendError(w, http.StatusInternalServerError, "failed to get video")
			return
		}
		if video == nil {
			SendError(w, http.StatusNotFound, fmt.Sprintf("video %s not found", id))
			return
		}
		videoPaths = append(videoPaths, filepath.Join(h.config.VideoStoragePath, video.Filename))
		totalDuration += video.Duration
	}

	mergedID, _ := generateID()
	mergedFilename := fmt.Sprintf("%s_merged.mp4", mergedID)
	mergedPath := filepath.Join(h.config.VideoStoragePath, mergedFilename)

	if err := h.processor.Merge(r.Context(), videoPaths, mergedPath); err != nil {
		SendError(w, http.StatusInternalServerError, "failed to merge videos")
		return
	}

	info, err := h.processor.GetVideoInfo(r.Context(), mergedPath)
	if err != nil {
		os.Remove(mergedPath)
		SendError(w, http.StatusInternalServerError, "failed to process merged video")
		return
	}

	mergedVideo := &storage.Video{
		ID:       mergedID,
		Filename: mergedFilename,
		Size:     info.Size,
		Duration: int(info.Duration),
		Status:   storage.StatusCompleted,
	}

	if err := h.storage.SaveVideo(r.Context(), mergedVideo); err != nil {
		os.Remove(mergedPath)
		SendError(w, http.StatusInternalServerError, "failed to save merged video metadata")
		return
	}

	SendSuccess(w, http.StatusOK, mergedVideo, "videos merged successfully")
}

func (h *VideoHandler) saveUploadedFile(file multipart.File, filepath string) (int64, error) {
	dst, err := os.Create(filepath)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		return 0, fmt.Errorf("failed to save file: %w", err)
	}

	return size, nil
}

func isValidVideoType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validTypes := map[string]bool{
		".mp4":  true,
		".mov":  true,
		".avi":  true,
		".mkv":  true,
		".webm": true,
	}
	return validTypes[ext]
}

func generateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
