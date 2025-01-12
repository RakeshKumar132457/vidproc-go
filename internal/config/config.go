package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Environment string
	APIToken string
	DBPath           string
	VideoStoragePath string
	MaxVideoSize int64
	MaxDuration  int
	MinDuration  int
}

const (
	defaultPort         = "8080"
	defaultEnvironment  = "development"
	defaultMaxVideoSize = 25 * 1024 * 1024
	defaultMaxVideoDuration  = 25
	defaultMinVideoDuration  = 5
)

func Load() (Config, error) {

	_ = godotenv.Load()

	var cfg Config
	var missingVars []string

	if cfg.APIToken = os.Getenv("API_TOKEN_SECRET"); cfg.APIToken == "" {
		missingVars = append(missingVars, "API_TOKEN_SECRET")
	}

	if cfg.DBPath = os.Getenv("DB_PATH"); cfg.DBPath == "" {
		missingVars = append(missingVars, "DB_PATH")
	}

	if cfg.VideoStoragePath = os.Getenv("VIDEO_STORAGE_PATH"); cfg.VideoStoragePath == "" {
		missingVars = append(missingVars, "VIDEO_STORAGE_PATH")
	}

	if len(missingVars) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	cfg.Port = getEnvWithDefault("PORT", defaultPort)
	cfg.Environment = getEnvWithDefault("ENVIRONMENT", defaultEnvironment)
	cfg.MaxVideoSize = getEnvInt64WithDefault("MAX_VIDEO_SIZE", defaultMaxVideoSize)
	cfg.MaxDuration = getEnvIntWithDefault("MAX_VIDEO_DURATION", defaultMaxVideoDuration)
	cfg.MinDuration = getEnvIntWithDefault("MIN_VIDEO_DURATION", defaultMinVideoDuration)

	return cfg, nil
}

func getEnvWithDefault(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
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
