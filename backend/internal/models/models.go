// Package models defines the core domain types shared across the API.
package models

import "time"

// Article is a single web article captured by the browser extension and
// processed with furigana + vocabulary spans.
type Article struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	ContentHTML string    `json:"content_html"`
	CreatedAt   time.Time `json:"created_at"`
}

// Book is an EPUB book.
type Book struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Chapter is a single chapter within a book.
type Chapter struct {
	ID           string    `json:"id"`
	BookID       string    `json:"book_id"`
	ChapterIndex int       `json:"chapter_index"`
	Title        string    `json:"title,omitempty"`
	ContentHTML  string    `json:"content_html,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// EPUBImage is an image extracted from an EPUB and offloaded to object storage.
type EPUBImage struct {
	ID           string    `json:"id"`
	BookID       string    `json:"book_id"`
	OriginalPath string    `json:"original_path"`
	MimeType     string    `json:"mime_type"`
	S3Key        string    `json:"s3_key"`
	S3URL        string    `json:"s3_url"`
	CreatedAt    time.Time `json:"created_at"`
}

// DictEntry is a single JMdict entry.
type DictEntry struct {
	EntSeq   int         `json:"ent_seq"`
	Kanji    []string    `json:"kanji"`
	Readings []string    `json:"readings"`
	Meanings []DictSense `json:"meanings"`
}

// DictSense is a single sense (meaning group) of a dictionary entry.
type DictSense struct {
	PartOfSpeech []string `json:"pos,omitempty"`
	Glosses      []string `json:"glosses"`
}

// CreateArticleRequest is the payload sent by the extension / API client.
type CreateArticleRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	HTML  string `json:"html"`
}

// ChapterSummary is a lightweight chapter entry for table-of-contents views.
type ChapterSummary struct {
	ID           string `json:"id"`
	ChapterIndex int    `json:"chapter_index"`
	Title        string `json:"title,omitempty"`
}
