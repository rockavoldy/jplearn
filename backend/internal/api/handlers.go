package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"jplearn/internal/models"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Articles
// ---------------------------------------------------------------------------

// handleCreateArticle receives a scraped article, annotates it with Kagome and
// persists the processed HTML.
func (s *Server) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxArticlePayloadBytes)
	defer r.Body.Close()

	var req models.CreateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload: "+err.Error())
		return
	}
	if strings.TrimSpace(req.URL) == "" || strings.TrimSpace(req.HTML) == "" {
		writeError(w, http.StatusBadRequest, "url and html are required")
		return
	}
	if len(req.HTML) > int(s.cfg.MaxArticlePayloadBytes) {
		writeError(w, http.StatusRequestEntityTooLarge, "article payload exceeds limit")
		return
	}

	annotated, err := s.nlp.AnnotateHTML(req.HTML)
	if err != nil {
		s.logger.Error("annotate article failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to process article")
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = req.URL
	}

	article, err := s.store.UpsertArticle(r.Context(), &models.Article{
		URL:         req.URL,
		Title:       title,
		ContentHTML: annotated,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, article)
}

func (s *Server) handleGetArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	article, err := s.store.GetArticle(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, article)
}

func (s *Server) handleListArticles(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 50)
	articles, err := s.store.ListArticles(r.Context(), limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articles)
}

// ---------------------------------------------------------------------------
// EPUB books & chapters
// ---------------------------------------------------------------------------

// handleEPUBUpload accepts a multipart .epub file, parses it and persists the
// resulting book, chapters and images.
func (s *Server) handleEPUBUpload(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.EnableEPUBUpload {
		writeError(w, http.StatusForbidden, "epub upload is disabled")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxEPUBUploadBytes+1024)
	if err := r.ParseMultipartForm(s.cfg.MaxEPUBUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	if header.Size > s.cfg.MaxEPUBUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "epub exceeds 50MB limit")
		return
	}
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".epub") {
		writeError(w, http.StatusBadRequest, "file must have .epub extension")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read upload: "+err.Error())
		return
	}

	book, err := s.epub.Ingest(r.Context(), bytes.NewReader(data))
	if err != nil {
		s.logger.Error("epub ingest failed", "error", err)
		writeError(w, http.StatusUnprocessableEntity, "failed to process epub: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, book)
}

func (s *Server) handleListBooks(w http.ResponseWriter, r *http.Request) {
	books, err := s.store.ListBooks(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, books)
}

func (s *Server) handleListChapters(w http.ResponseWriter, r *http.Request) {
	bookID := r.PathValue("id")
	chapters, err := s.store.ListChapters(r.Context(), bookID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, chapters)
}

func (s *Server) handleGetChapter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	chapter, err := s.store.GetChapter(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, chapter)
}

// ---------------------------------------------------------------------------
// Image proxy
// ---------------------------------------------------------------------------

// handleGetImage streams the S3 object for an image record with cache headers.
func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	img, err := s.store.GetImage(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	body, contentType, err := s.s3.Get(r.Context(), img.S3Key)
	if err != nil {
		s.logger.Error("s3 get image failed", "key", img.S3Key, "error", err)
		writeError(w, http.StatusBadGateway, "failed to fetch image from storage")
		return
	}
	defer body.Close()

	if contentType == "" || contentType == "application/octet-stream" {
		contentType = img.MimeType
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", img.S3Key)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil {
		s.logger.Warn("image stream interrupted", "error", err)
	}
}

// ---------------------------------------------------------------------------
// Dictionary
// ---------------------------------------------------------------------------

func (s *Server) handleDictionary(w http.ResponseWriter, r *http.Request) {
	lemma := strings.TrimSpace(r.URL.Query().Get("lemma"))
	if lemma == "" {
		writeError(w, http.StatusBadRequest, "lemma query parameter is required")
		return
	}
	limit := queryInt(r, "limit", 10)

	entries, err := s.store.LookupDictionary(r.Context(), lemma, limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lemma":   lemma,
		"count":   len(entries),
		"entries": entries,
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	if n > 500 {
		return 500
	}
	return n
}
