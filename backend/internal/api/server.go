// Package api wires HTTP routes to services and implements the REST handlers.
package api

import (
	"log/slog"
	"net/http"

	"jplearn/internal/config"
	"jplearn/internal/nlp"
	"jplearn/internal/service"
	"jplearn/internal/storage"
	"jplearn/internal/store"
)

// Server holds shared dependencies for all HTTP handlers.
type Server struct {
	cfg    *config.Config
	store  *store.Store
	s3     *storage.S3Client
	nlp    *nlp.Processor
	epub   *service.EPUBService
	logger *slog.Logger
}

// NewServer constructs the API server and its dependencies.
func NewServer(
	cfg *config.Config,
	st *store.Store,
	s3c *storage.S3Client,
	proc *nlp.Processor,
	epubSvc *service.EPUBService,
	logger *slog.Logger,
) *Server {
	return &Server{
		cfg:    cfg,
		store:  st,
		s3:     s3c,
		nlp:    proc,
		epub:   epubSvc,
		logger: logger,
	}
}

// Handler returns the fully configured http.Handler, including middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)

	mux.HandleFunc("POST /api/articles", s.handleCreateArticle)
	mux.HandleFunc("GET /api/articles", s.handleListArticles)
	mux.HandleFunc("GET /api/articles/{id}", s.handleGetArticle)

	mux.HandleFunc("POST /api/epubs/upload", s.handleEPUBUpload)
	mux.HandleFunc("GET /api/books", s.handleListBooks)
	mux.HandleFunc("GET /api/books/{id}/chapters", s.handleListChapters)
	mux.HandleFunc("GET /api/chapters/{id}", s.handleGetChapter)

	mux.HandleFunc("GET /api/images/{id}", s.handleGetImage)
	mux.HandleFunc("GET /api/dictionary", s.handleDictionary)

	return recoverMiddleware(s.logger)(corsMiddleware(s.cfg.CORSOrigins)(loggingMiddleware(s.logger)(mux)))
}
