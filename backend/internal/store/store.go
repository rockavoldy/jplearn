// Package store contains SQL data-access methods for all domain entities.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"jplearn/internal/models"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// Store provides persistence operations backed by a pgx pool.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Store around an existing pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// ---------------------------------------------------------------------------
// Articles
// ---------------------------------------------------------------------------

// UpsertArticle inserts a new article or updates the existing one for the URL.
func (s *Store) UpsertArticle(ctx context.Context, a *models.Article) (*models.Article, error) {
	const q = `
		INSERT INTO articles (url, title, content_html)
		VALUES ($1, $2, $3)
		ON CONFLICT (url) DO UPDATE
			SET title = EXCLUDED.title,
			    content_html = EXCLUDED.content_html
		RETURNING id, url, title, content_html, created_at`
	out := &models.Article{}
	err := s.pool.QueryRow(ctx, q, a.URL, a.Title, a.ContentHTML).
		Scan(&out.ID, &out.URL, &out.Title, &out.ContentHTML, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert article: %w", err)
	}
	return out, nil
}

// GetArticle fetches an article by id.
func (s *Store) GetArticle(ctx context.Context, id string) (*models.Article, error) {
	const q = `SELECT id, url, title, content_html, created_at FROM articles WHERE id = $1`
	out := &models.Article{}
	err := s.pool.QueryRow(ctx, q, id).
		Scan(&out.ID, &out.URL, &out.Title, &out.ContentHTML, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}
	return out, nil
}

// ListArticles returns recent articles (without heavy HTML payloads).
func (s *Store) ListArticles(ctx context.Context, limit int) ([]models.Article, error) {
	const q = `SELECT id, url, title, '' AS content_html, created_at
	           FROM articles ORDER BY created_at DESC LIMIT $1`
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}
	defer rows.Close()

	var out []models.Article
	for rows.Next() {
		var a models.Article
		if err := rows.Scan(&a.ID, &a.URL, &a.Title, &a.ContentHTML, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Books
// ---------------------------------------------------------------------------

// CreateBook inserts a new book row.
func (s *Store) CreateBook(ctx context.Context, title, author string) (*models.Book, error) {
	const q = `INSERT INTO books (title, author) VALUES ($1, $2)
	           RETURNING id, title, COALESCE(author, ''), created_at`
	b := &models.Book{}
	if err := s.pool.QueryRow(ctx, q, title, author).
		Scan(&b.ID, &b.Title, &b.Author, &b.CreatedAt); err != nil {
		return nil, fmt.Errorf("create book: %w", err)
	}
	return b, nil
}

// ListBooks returns all books ordered by creation time.
func (s *Store) ListBooks(ctx context.Context) ([]models.Book, error) {
	const q = `SELECT id, title, COALESCE(author, ''), created_at FROM books ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list books: %w", err)
	}
	defer rows.Close()

	var out []models.Book
	for rows.Next() {
		var b models.Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan book: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetBook fetches a book by id.
func (s *Store) GetBook(ctx context.Context, id string) (*models.Book, error) {
	const q = `SELECT id, title, COALESCE(author, ''), created_at FROM books WHERE id = $1`
	b := &models.Book{}
	err := s.pool.QueryRow(ctx, q, id).Scan(&b.ID, &b.Title, &b.Author, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get book: %w", err)
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// Chapters
// ---------------------------------------------------------------------------

// CreateChapter inserts a chapter row.
func (s *Store) CreateChapter(ctx context.Context, c *models.Chapter) error {
	const q = `INSERT INTO chapters (book_id, chapter_index, title, content_html)
	           VALUES ($1, $2, $3, $4)`
	if _, err := s.pool.Exec(ctx, q, c.BookID, c.ChapterIndex, c.Title, c.ContentHTML); err != nil {
		return fmt.Errorf("create chapter: %w", err)
	}
	return nil
}

// ListChapters returns the table of contents for a book (no HTML).
func (s *Store) ListChapters(ctx context.Context, bookID string) ([]models.ChapterSummary, error) {
	const q = `SELECT id, chapter_index, COALESCE(title, '')
	           FROM chapters WHERE book_id = $1 ORDER BY chapter_index ASC`
	rows, err := s.pool.Query(ctx, q, bookID)
	if err != nil {
		return nil, fmt.Errorf("list chapters: %w", err)
	}
	defer rows.Close()

	var out []models.ChapterSummary
	for rows.Next() {
		var c models.ChapterSummary
		if err := rows.Scan(&c.ID, &c.ChapterIndex, &c.Title); err != nil {
			return nil, fmt.Errorf("scan chapter: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChapter fetches a chapter with its processed HTML.
func (s *Store) GetChapter(ctx context.Context, id string) (*models.Chapter, error) {
	const q = `SELECT id, book_id, chapter_index, COALESCE(title, ''), content_html, created_at
	           FROM chapters WHERE id = $1`
	c := &models.Chapter{}
	err := s.pool.QueryRow(ctx, q, id).
		Scan(&c.ID, &c.BookID, &c.ChapterIndex, &c.Title, &c.ContentHTML, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get chapter: %w", err)
	}
	return c, nil
}

// ---------------------------------------------------------------------------
// EPUB images
// ---------------------------------------------------------------------------

// CreateImage inserts an epub_images row.
func (s *Store) CreateImage(ctx context.Context, img *models.EPUBImage) error {
	const q = `INSERT INTO epub_images (book_id, original_path, mime_type, s3_key, s3_url)
	           VALUES ($1, $2, $3, $4, $5)`
	if _, err := s.pool.Exec(ctx, q, img.BookID, img.OriginalPath, img.MimeType, img.S3Key, img.S3URL); err != nil {
		return fmt.Errorf("create image: %w", err)
	}
	return nil
}

// GetImage fetches an image record by id.
func (s *Store) GetImage(ctx context.Context, id string) (*models.EPUBImage, error) {
	const q = `SELECT id, book_id, original_path, mime_type, s3_key, s3_url, created_at
	           FROM epub_images WHERE id = $1`
	img := &models.EPUBImage{}
	err := s.pool.QueryRow(ctx, q, id).
		Scan(&img.ID, &img.BookID, &img.OriginalPath, &img.MimeType, &img.S3Key, &img.S3URL, &img.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get image: %w", err)
	}
	return img, nil
}

// ---------------------------------------------------------------------------
// Dictionary
// ---------------------------------------------------------------------------

// LookupDictionary returns dictionary entries matching the lemma against the
// kanji and readings arrays using GIN-indexed overlap operators.
func (s *Store) LookupDictionary(ctx context.Context, lemma string, limit int) ([]models.DictEntry, error) {
	const q = `
		SELECT ent_seq, kanji, readings, meanings
		FROM jmdict_entries
		WHERE kanji && ARRAY[$1]::text[] OR readings && ARRAY[$1]::text[]
		ORDER BY ent_seq
		LIMIT $2`
	rows, err := s.pool.Query(ctx, q, lemma, limit)
	if err != nil {
		return nil, fmt.Errorf("dictionary lookup: %w", err)
	}
	defer rows.Close()

	var out []models.DictEntry
	for rows.Next() {
		var (
			ent      models.DictEntry
			meanings []byte
		)
		if err := rows.Scan(&ent.EntSeq, &ent.Kanji, &ent.Readings, &meanings); err != nil {
			return nil, fmt.Errorf("scan dict entry: %w", err)
		}
		if len(meanings) > 0 {
			if err := json.Unmarshal(meanings, &ent.Meanings); err != nil {
				return nil, fmt.Errorf("unmarshal meanings: %w", err)
			}
		}
		out = append(out, ent)
	}
	return out, rows.Err()
}

// InsertDictEntries bulk-inserts dictionary entries within a transaction.
func (s *Store) InsertDictEntries(ctx context.Context, entries []models.DictEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin dict tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `INSERT INTO jmdict_entries (ent_seq, kanji, readings, meanings)
	           VALUES ($1, $2, $3, $4)
	           ON CONFLICT (ent_seq) DO UPDATE
	             SET kanji = EXCLUDED.kanji,
	                 readings = EXCLUDED.readings,
	                 meanings = EXCLUDED.meanings`

	batch := &pgx.Batch{}
	for _, e := range entries {
		meanings, err := json.Marshal(e.Meanings)
		if err != nil {
			return fmt.Errorf("marshal meanings: %w", err)
		}
		kanji := e.Kanji
		if kanji == nil {
			kanji = []string{}
		}
		readings := e.Readings
		if readings == nil {
			readings = []string{}
		}
		batch.Queue(q, e.EntSeq, kanji, readings, meanings)
	}

	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("send dict batch: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit dict tx: %w", err)
	}
	return nil
}
