package api

import (
	"database/sql"
	"net/http"
	"vidproc-go/internal/config"
	"vidproc-go/internal/storage"
	"vidproc-go/internal/video"
)

type Router struct {
	db           *sql.DB
	config       config.Config
	storage      storage.VideoStorage
	shareStorage storage.ShareLinkStorage
	processor    video.Processor
}

func NewRouter(db *sql.DB, cfg config.Config) *Router {
	videoStorage := storage.NewVideoStorage(db)
	shareStorage := storage.NewShareLinkStorage(db)
	videoProcessor := video.NewFFmpegProcessor()

	return &Router{
		db:           db,
		config:       cfg,
		storage:      videoStorage,
		shareStorage: shareStorage,
		processor:    videoProcessor,
	}
}

func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

func (r *Router) SetupRoutes() http.Handler {
	mux := http.NewServeMux()
	middleware := Chain(
		LoggingMiddleware,
		AuthMiddleware(r.config.APIToken),
	)

	videoHandler := NewVideoHandler(r.config, r.storage)
	shareHandler := NewShareHandler(r.config, r.storage, r.shareStorage)

	protected := http.NewServeMux()

	protected.HandleFunc("/api/videos", videoHandler.HandleVideos)
	protected.HandleFunc("/api/videos/trim/", videoHandler.HandleTrim)
	protected.HandleFunc("/api/videos/merge", videoHandler.HandleMerge)

	protected.HandleFunc("/api/shares", shareHandler.HandleShares)
	protected.HandleFunc("/api/shares/", shareHandler.HandleShareOperations)

	mux.HandleFunc("/api/health", r.handleHealth)

	mux.Handle("/api/", middleware(protected))

	return mux
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	SendSuccess(w, http.StatusOK, map[string]string{
		"status": "available",
	}, "service is healthy")
}
