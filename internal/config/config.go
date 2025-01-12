package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath           string
	VideoStoragePath string
	MaxVideoSize     int64
	MaxDuration      int
	MinDuration      int
	APIToken         string
}

const (
	defaultMaxVideoSize     = 25 * 1024 * 1024
	defaultMaxVideoDuration = 25
	defaultMinVideoDuration = 5
)

var (
	ErrMissingDBPath           = errors.New("DB_PATH environment variable is required")
	ErrMissingVideoStoragePath = errors.New("VIDEO_STORAGE_PATH environment variable is required")
	ErrMissingAPIToken         = errors.New("API_TOKEN environment variable is required")
)

func Load() (Config, error) {

	_ = godotenv.Load()

	var cfg Config
	var missingVars []string

	if cfg.DBPath = os.Getenv("DB_PATH"); cfg.DBPath == "" {
		missingVars = append(missingVars, "DB_PATH")
	}
	if cfg.VideoStoragePath = os.Getenv("VIDEO_STORAGE_PATH"); cfg.VideoStoragePath == "" {
		missingVars = append(missingVars, "VIDEO_STORAGE_PATH")
	}
	if cfg.APIToken = os.Getenv("API_TOKEN"); cfg.APIToken == "" {
		missingVars = append(missingVars, "API_TOKEN")
	}

	if len(missingVars) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	cfg.MaxVideoSize = getEnvInt64WithDefault("MAX_VIDEO_SIZE", defaultMaxVideoSize)
	cfg.MaxDuration = getEnvIntWithDefault("MAX_VIDEO_DURATION", defaultMaxVideoDuration)
	cfg.MinDuration = getEnvIntWithDefault("MIN_VIDEO_DURATION", defaultMinVideoDuration)

	return cfg, nil
}

func getEnvIntWithDefault(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}

	return val
}

func getEnvInt64WithDefault(key string, defaultVal int64) int64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}

	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return defaultVal
	}

	return val
}

