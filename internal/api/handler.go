package api

import (
	"net/http"
)

type VideoHandler struct {
}

func NewVideoHandler() *VideoHandler {
	return &VideoHandler{}
}

func (h *VideoHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	SendError(w, http.StatusNotImplemented, "upload not implemented yet")
}

func (h *VideoHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	SendError(w, http.StatusNotImplemented, "list not implemented yet")
}

func (h *VideoHandler) HandleTrim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	SendError(w, http.StatusNotImplemented, "trim not implemented yet")
}

func (h *VideoHandler) HandleMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	SendError(w, http.StatusNotImplemented, "merge not implemented yet")
}

func (h *VideoHandler) HandleShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	SendError(w, http.StatusNotImplemented, "share not implemented yet")
}
