package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	validToken := "test-token"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SendSuccess(w, http.StatusOK, nil, "success")
	})

	tests := []struct {
		name          string
		token         string
		expectedCode  int
		expectedError string
	}{
		{
			name:         "valid token",
			token:        validToken,
			expectedCode: http.StatusOK,
		},
		{
			name:          "missing token",
			token:         "",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "missing authorization header",
		},
		{
			name:          "invalid token",
			token:         "wrong-token",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "invalid token",
		},
		{
			name:          "invalid format",
			token:         "invalid-format",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "invalid authorization header format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.token != "" {
				if tt.token == "invalid-format" {
					req.Header.Set("Authorization", tt.token)
				} else {
					req.Header.Set("Authorization", "Bearer "+tt.token)
				}
			}

			rr := httptest.NewRecorder()

			middleware := AuthMiddleware(validToken)
			middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedCode)
			}

			if tt.expectedError != "" {
				var response Response
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if response.Error != tt.expectedError {
					t.Errorf("handler returned wrong error message: got %v want %v",
						response.Error, tt.expectedError)
				}
			}
		})
	}
}
