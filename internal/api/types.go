package api

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func SendJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func SendError(w http.ResponseWriter, status int, message string) {
	SendJSON(w, status, Response{
		Status: "error",
		Error:  message,
	})
}

func SendSuccess(w http.ResponseWriter, status int, data interface{}, message string) {
	SendJSON(w, status, Response{
		Status:  "success",
		Data:    data,
		Message: message,
	})
}
