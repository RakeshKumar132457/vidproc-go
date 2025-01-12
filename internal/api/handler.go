package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
)

type VideoHandler struct {
	config  config.Config
	storage storage.VideoStorage
}

func NewVideoHandler(cfg config.Config, store storage.VideoStorage) *VideoHandler {
	return &VideoHandler{
		config:  cfg,
		storage: store,
	}
}

func generateID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (h *VideoHandler) saveUploadedFile(file multipart.File, filename string) (int64, error) {

	dst, err := os.Create(filepath.Join(h.config.VideoStoragePath, filename))
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

func (h *VideoHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

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

	id, err := generateID()
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to generate video ID")
		return
	}

	filename := fmt.Sprintf("%s_%s", id, filepath.Base(header.Filename))

	size, err := h.saveUploadedFile(file, filename)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to save video")
		return
	}

	video := &storage.Video{
		ID:       id,
		Filename: filename,
		Size:     size,
		Status:   storage.StatusPending,
	}

	if err := h.storage.SaveVideo(r.Context(), video); err != nil {

		os.Remove(filepath.Join(h.config.VideoStoragePath, filename))
		SendError(w, http.StatusInternalServerError, "failed to save video metadata")
		return
	}

	SendSuccess(w, http.StatusCreated, video, "video uploaded successfully")
}

func (h *VideoHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	videos, err := h.storage.ListVideos(r.Context())
	if err != nil {
		SendError(w, http.StatusInternalServerError, "failed to list videos")
		return
	}

	SendSuccess(w, http.StatusOK, videos, "")
}

