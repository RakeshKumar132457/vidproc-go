package api

import (
	"database/sql"
	"net/http"
	"vidproc-go/internal/config"
)

type Router struct {
	db     *sql.DB
	config config.Config
}

func NewRouter(db *sql.DB, cfg config.Config) *Router {
	return &Router{
		db:     db,
		config: cfg,
	}
}

func (r *Router) SetupRoutes() http.Handler {
	mux := http.NewServeMux()
	middleware := Chain(
		LoggingMiddleware,
		AuthMiddleware(r.config.APIToken),
	)

	protected := http.NewServeMux()
	protected.HandleFunc("/api/videos", r.handleVideos)
	protected.HandleFunc("/api/videos/", r.handleVideoOperations)

	mux.HandleFunc("/health", r.handleHealth)
	mux.Handle("/api/", middleware(protected))

	return mux
}

func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	SendSuccess(w, http.StatusOK, map[string]string{
		"status": "available",
	}, "service is healthy")
}

func (r *Router) handleVideos(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		r.handleVideoUpload(w, req)
	case http.MethodGet:
		r.handleListVideos(w, req)
	default:
		SendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (r *Router) handleVideoOperations(w http.ResponseWriter, req *http.Request) {

	SendError(w, http.StatusNotImplemented, "not implemented yet")
}

func (r *Router) handleVideoUpload(w http.ResponseWriter, req *http.Request) {
	SendError(w, http.StatusNotImplemented, "video upload not implemented yet")
}

func (r *Router) handleListVideos(w http.ResponseWriter, req *http.Request) {
	SendError(w, http.StatusNotImplemented, "list videos not implemented yet")
}
