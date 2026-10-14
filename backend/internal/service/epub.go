// Package service contains application orchestration logic that combines
// parsing, NLP, object storage and persistence.
package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"jplearn/internal/epub"
	"jplearn/internal/models"
	"jplearn/internal/nlp"
	"jplearn/internal/storage"
	"jplearn/internal/store"
)

// EPUBService orchestrates EPUB ingestion end to end.
type EPUBService struct {
	store  *store.Store
	s3     *storage.S3Client
	nlp    *nlp.Processor
	logger *slog.Logger
}

// NewEPUBService wires the service dependencies.
func NewEPUBService(st *store.Store, s3c *storage.S3Client, proc *nlp.Processor, logger *slog.Logger) *EPUBService {
	return &EPUBService{store: st, s3: s3c, nlp: proc, logger: logger}
}

// Ingest parses an EPUB archive, offloads images to S3, annotates chapter text
// and persists everything. It returns the created book.
func (s *EPUBService) Ingest(ctx context.Context, r *bytes.Reader) (*models.Book, error) {
	parsed, err := epub.Parse(r, r.Size())
	if err != nil {
		return nil, err
	}

	title := parsed.Meta.Title
	if strings.TrimSpace(title) == "" {
		title = "Untitled Book"
	}

	book, err := s.store.CreateBook(ctx, title, parsed.Meta.Author)
	if err != nil {
		return nil, err
	}

	// Map resolved archive image path -> created image id (for HTML rewriting).
	imageIDByPath := make(map[string]string, len(parsed.Images))
	for _, img := range parsed.Images {
		key := fmt.Sprintf("books/%s/images/%s", book.ID, path.Base(img.Path))
		s3Key, s3URL, err := s.s3.PutBytes(ctx, key, img.Mime, img.Data)
		if err != nil {
			s.logger.Warn("failed to upload epub image", "path", img.Path, "error", err)
			continue
		}
		rec := &models.EPUBImage{
			BookID:       book.ID,
			OriginalPath: img.Path,
			MimeType:     img.Mime,
			S3Key:        s3Key,
			S3URL:        s3URL,
		}
		if err := s.store.CreateImage(ctx, rec); err != nil {
			s.logger.Warn("failed to persist epub image", "path", img.Path, "error", err)
			continue
		}
		imageIDByPath[img.Path] = rec.ID
	}

	for idx, ch := range parsed.Chapters {
		rewritten := rewriteImageSources(ch.HTML, ch.Path, imageIDByPath)
		annotated, err := s.nlp.AnnotateHTML(rewritten)
		if err != nil {
			s.logger.Warn("failed to annotate chapter", "path", ch.Path, "error", err)
			annotated = rewritten
		}

		chapterTitle := ch.Title
		if strings.TrimSpace(chapterTitle) == "" {
			chapterTitle = fmt.Sprintf("Chapter %d", idx+1)
		}

		if err := s.store.CreateChapter(ctx, &models.Chapter{
			BookID:       book.ID,
			ChapterIndex: idx,
			Title:        chapterTitle,
			ContentHTML:  annotated,
		}); err != nil {
			return nil, err
		}
	}

	return book, nil
}

// rewriteImageSources replaces relative <img src> values with backend proxy URLs
// (/api/images/{id}) when the referenced resource was offloaded to S3.
func rewriteImageSources(content, chapterPath string, imageIDByPath map[string]string) string {
	ctx := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(content), ctx)
	if err != nil {
		return content
	}

	chapterDir := path.Dir(chapterPath)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Img {
			for i, attr := range n.Attr {
				if attr.Key != "src" {
					continue
				}
				resolved := resolveSrc(chapterDir, attr.Val)
				if id, ok := imageIDByPath[resolved]; ok {
					n.Attr[i].Val = "/api/images/" + id
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	var buf strings.Builder
	for _, n := range nodes {
		walk(n)
		if err := html.Render(&buf, n); err != nil {
			return content
		}
	}
	return buf.String()
}

func resolveSrc(dir, src string) string {
	if src == "" {
		return ""
	}
	// Strip any query/fragment and leading slash.
	if i := strings.IndexAny(src, "?#"); i >= 0 {
		src = src[:i]
	}
	src = strings.TrimPrefix(src, "/")
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src
	}
	return strings.TrimPrefix(path.Join(dir, src), "./")
}
