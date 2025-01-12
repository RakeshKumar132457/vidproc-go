package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := ensureDirectories(cfg); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	db, err := storage.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	srv := setupServer(cfg, db)

	go func() {
		log.Printf("Starting server on port %s in %s mode", cfg.Port, cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	gracefulShutdown(srv)
}

func setupServer(cfg config.Config, db *sql.DB) *http.Server {

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK - Environment: %s", cfg.Environment)
	})

	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func ensureDirectories(cfg config.Config) error {
	dirs := []string{
		filepath.Dir(cfg.DBPath),
		cfg.VideoStoragePath,
	}

	for _, dir := range dirs {

		if err := os.MkdirAll(dir, 0755); err != nil {
			if os.IsPermission(err) {

				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get user home directory: %v", err)
				}
				newDir := filepath.Join(homeDir, "videoapi", filepath.Base(dir))
				if err := os.MkdirAll(newDir, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %v", newDir, err)
				}

				if dir == cfg.VideoStoragePath {
					cfg.VideoStoragePath = newDir
				} else {
					cfg.DBPath = filepath.Join(filepath.Dir(newDir), "videos.db")
				}
			} else {
				return fmt.Errorf("failed to create directory %s: %v", dir, err)
			}
		}
	}

	return nil
}

func gracefulShutdown(srv *http.Server) {

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}
