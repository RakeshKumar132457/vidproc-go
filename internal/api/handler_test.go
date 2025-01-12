package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleUpload(t *testing.T) {
	handler := NewVideoHandler()

	tests := []struct {
		name          string
		method        string
		expectedCode  int
		expectedError string
	}{
		{
			name:          "wrong method",
			method:        http.MethodGet,
			expectedCode:  http.StatusMethodNotAllowed,
			expectedError: "method not allowed",
		},
		{
			name:          "not implemented",
			method:        http.MethodPost,
			expectedCode:  http.StatusNotImplemented,
			expectedError: "upload not implemented yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/videos", nil)
			rr := httptest.NewRecorder()

			handler.HandleUpload(rr, req)

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedCode)
			}

			var response Response
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}

			if response.Error != tt.expectedError {
				t.Errorf("handler returned unexpected error: got %v want %v",
					response.Error, tt.expectedError)
			}
		})
	}
}

func TestHandleList(t *testing.T) {
	handler := NewVideoHandler()

	tests := []struct {
		name          string
		method        string
		expectedCode  int
		expectedError string
	}{
		{
			name:          "wrong method",
			method:        http.MethodPost,
			expectedCode:  http.StatusMethodNotAllowed,
			expectedError: "method not allowed",
		},
		{
			name:          "not implemented",
			method:        http.MethodGet,
			expectedCode:  http.StatusNotImplemented,
			expectedError: "list not implemented yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/videos", nil)
			rr := httptest.NewRecorder()

			handler.HandleList(rr, req)

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedCode)
			}

			var response Response
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}

			if response.Error != tt.expectedError {
				t.Errorf("handler returned unexpected error: got %v want %v",
					response.Error, tt.expectedError)
			}
		})
	}
}
