package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	origEnv := make(map[string]string)
	envVars := []string{"DB_PATH", "VIDEO_STORAGE_PATH", "API_TOKEN", "MAX_VIDEO_SIZE", "MAX_VIDEO_DURATION", "MIN_VIDEO_DURATION"}
	for _, env := range envVars {
		origEnv[env] = os.Getenv(env)
	}

	defer func() {
		for k, v := range origEnv {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		wantErr  bool
		expected Config
	}{
		{
			name: "valid configuration from env vars",
			envVars: map[string]string{
				"DB_PATH":            "/app/data/db/videos.db",
				"VIDEO_STORAGE_PATH": "/app/data/videos",
				"MAX_VIDEO_SIZE":     "25000000",
				"MAX_VIDEO_DURATION": "25",
				"MIN_VIDEO_DURATION": "5",
				"API_TOKEN":          "test-token",
			},
			wantErr: false,
			expected: Config{
				DBPath:           "/app/data/db/videos.db",
				VideoStoragePath: "/app/data/videos",
				MaxVideoSize:     25000000,
				MaxDuration:      25,
				MinDuration:      5,
				APIToken:         "test-token",
			},
		},
		{
			name:    "missing required fields",
			envVars: map[string]string{},
			wantErr: true,
		},
		{
			name: "invalid numeric values",
			envVars: map[string]string{
				"DB_PATH":            "/app/data/db/videos.db",
				"VIDEO_STORAGE_PATH": "/app/data/videos",
				"API_TOKEN":          "test-token",
				"MAX_VIDEO_SIZE":     "invalid",
				"MAX_VIDEO_DURATION": "invalid",
				"MIN_VIDEO_DURATION": "invalid",
			},
			wantErr: false,
			expected: Config{
				DBPath:           "/app/data/db/videos.db",
				VideoStoragePath: "/app/data/videos",
				APIToken:         "test-token",
				MaxVideoSize:     defaultMaxVideoSize,
				MaxDuration:      defaultMaxVideoDuration,
				MinDuration:      defaultMinVideoDuration,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, env := range envVars {
				os.Unsetenv(env)
			}

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			config, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if config.DBPath != tt.expected.DBPath {
					t.Errorf("DBPath = %v, want %v", config.DBPath, tt.expected.DBPath)
				}
				if config.VideoStoragePath != tt.expected.VideoStoragePath {
					t.Errorf("VideoStoragePath = %v, want %v", config.VideoStoragePath, tt.expected.VideoStoragePath)
				}
				if config.MaxVideoSize != tt.expected.MaxVideoSize {
					t.Errorf("MaxVideoSize = %v, want %v", config.MaxVideoSize, tt.expected.MaxVideoSize)
				}
				if config.MaxDuration != tt.expected.MaxDuration {
					t.Errorf("MaxDuration = %v, want %v", config.MaxDuration, tt.expected.MaxDuration)
				}
				if config.MinDuration != tt.expected.MinDuration {
					t.Errorf("MinDuration = %v, want %v", config.MinDuration, tt.expected.MinDuration)
				}
				if config.APIToken != tt.expected.APIToken {
					t.Errorf("APIToken = %v, want %v", config.APIToken, tt.expected.APIToken)
				}
			}
		})
	}
}

