// Command import-jmdict streams a JMdict dictionary from the
// github.com/scriptin/jmdict-simplified JSON distribution into PostgreSQL.
//
// It accepts any of the release archive formats published by
// jmdict-simplified (e.g. jmdict-eng-3.x.x.json, .json.gz, .tgz, .zip) and
// reads entries one at a time so even the full (~150MB) file is processed with
// bounded memory. No runtime XML parsing is involved.
//
// Usage:
//
//	go run ./cmd/import-jmdict -file jmdict-eng-3.5.0.json.tgz
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"jplearn/internal/config"
	"jplearn/internal/database"
	"jplearn/internal/models"
	"jplearn/internal/store"
)

// ---------------------------------------------------------------------------
// jmdict-simplified JSON structures (https://github.com/scriptin/jmdict-simplified)
// ---------------------------------------------------------------------------

// dictHeader captures the metadata fields that precede the "words" array.
// The decoder is positioned on "words" afterwards for streaming.
type dictHeader struct {
	Version string `json:"version"`
}

type word struct {
	ID    string      `json:"id"`
	Kanji []formText  `json:"kanji"`
	Kana  []formText  `json:"kana"`
	Sense []senseJSON `json:"sense"`
}

type formText struct {
	Text   string   `json:"text"`
	Common bool     `json:"common"`
	Tags   []string `json:"tags"`
}

type senseJSON struct {
	PartOfSpeech []string    `json:"partOfSpeech"`
	Gloss        []glossJSON `json:"gloss"`
}

type glossJSON struct {
	Lang string `json:"lang"`
	Text string `json:"text"`
}

const batchSize = 2000

func main() {
	filePath := flag.String("file", "", "path to a jmdict-simplified JSON archive (overrides JMDICT_PATH)")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Fall back to the JMDICT_PATH value loaded from .env / environment.
	if *filePath == "" {
		*filePath = cfg.JMDictPath
	}
	if *filePath == "" {
		flag.Usage()
		log.Fatal("no dictionary specified: pass -file or set JMDICT_PATH")
	}

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	st := store.New(db.Pool)

	reader, err := openDictionary(*filePath)
	if err != nil {
		log.Fatalf("open dictionary: %v", err)
	}
	defer reader.Close()

	start := time.Now()
	total, skipped, err := importJSON(ctx, st.InsertDictEntries, reader)
	if err != nil {
		log.Fatalf("import: %v", err)
	}

	log.Printf("done: imported %d entries, skipped %d, in %.0fs", total, skipped, time.Since(start).Seconds())
}

// insertFunc persists a batch of dictionary entries.
type insertFunc func(ctx context.Context, entries []models.DictEntry) error

// importJSON streams the "words" array from a jmdict-simplified document and
// inserts entries in batches via the provided callback.
func importJSON(ctx context.Context, insert insertFunc, r io.Reader) (int, int, error) {
	dec := json.NewDecoder(r)

	// Advance to the opening of the top-level object.
	if err := expectDelim(dec, '{'); err != nil {
		return 0, 0, fmt.Errorf("expected JSON object: %w", err)
	}

	// Skip metadata keys until we reach "words".
	for {
		tok, err := dec.Token()
		if err != nil {
			return 0, 0, fmt.Errorf("scan for words key: %w", err)
		}
		key, ok := tok.(string)
		if !ok {
			continue
		}
		if key != "words" {
			// Skip the value associated with this metadata key.
			var discard json.RawMessage
			if err := dec.Decode(&discard); err != nil {
				return 0, 0, fmt.Errorf("skip key %q: %w", key, err)
			}
			continue
		}
		break
	}

	// Position on the opening bracket of the words array.
	if err := expectDelim(dec, '['); err != nil {
		return 0, 0, fmt.Errorf("expected words array: %w", err)
	}

	var (
		batch   = make([]models.DictEntry, 0, batchSize)
		total   int
		skipped int
		start   = time.Now()
	)

	for dec.More() {
		var w word
		if err := dec.Decode(&w); err != nil {
			return total, skipped, fmt.Errorf("decode word: %w", err)
		}

		entry, ok := convert(w)
		if !ok {
			skipped++
			continue
		}
		batch = append(batch, entry)

		if len(batch) >= batchSize {
			if err := insert(ctx, batch); err != nil {
				return total, skipped, fmt.Errorf("insert batch: %w", err)
			}
			total += len(batch)
			log.Printf("imported %d entries (%.0fs)", total, time.Since(start).Seconds())
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := insert(ctx, batch); err != nil {
			return total, skipped, fmt.Errorf("insert final batch: %w", err)
		}
		total += len(batch)
	}
	return total, skipped, nil
}

// convert maps a jmdict-simplified word into our DictEntry model.
func convert(w word) (models.DictEntry, bool) {
	entSeq, err := strconv.Atoi(strings.TrimSpace(w.ID))
	if err != nil {
		return models.DictEntry{}, false
	}

	entry := models.DictEntry{EntSeq: entSeq}
	for _, k := range w.Kanji {
		if t := strings.TrimSpace(k.Text); t != "" {
			entry.Kanji = append(entry.Kanji, t)
		}
	}
	for _, k := range w.Kana {
		if t := strings.TrimSpace(k.Text); t != "" {
			entry.Readings = append(entry.Readings, t)
		}
	}
	if len(entry.Kanji) == 0 && len(entry.Readings) == 0 {
		return models.DictEntry{}, false
	}

	for _, s := range w.Sense {
		var glosses []string
		for _, g := range s.Gloss {
			// Prefer English glosses (lang empty or "eng").
			if g.Lang != "" && !strings.EqualFold(g.Lang, "eng") {
				continue
			}
			if t := strings.TrimSpace(g.Text); t != "" {
				glosses = append(glosses, t)
			}
		}
		if len(glosses) == 0 {
			continue
		}
		entry.Meanings = append(entry.Meanings, models.DictSense{
			PartOfSpeech: s.PartOfSpeech,
			Glosses:      glosses,
		})
	}
	return entry, true
}

// ---------------------------------------------------------------------------
// Archive handling
// ---------------------------------------------------------------------------

// openDictionary opens a jmdict-simplified distribution, transparently handling
// plain JSON, gzip-compressed JSON and .tgz / .zip archives.
func openDictionary(path string) (io.ReadCloser, error) {
	lower := strings.ToLower(path)

	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return openFromTarGz(path)
	case strings.HasSuffix(lower, ".zip"):
		return openFromZip(path)
	case strings.HasSuffix(lower, ".gz"):
		return openGzip(path)
	default:
		return os.Open(path)
	}
}

func openGzip(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("gzip: %w", err)
	}
	return &layeredReader{src: gz, closers: []io.Closer{gz, f}}, nil
}

func openFromTarGz(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("gzip: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			_ = gz.Close()
			_ = f.Close()
			return nil, fmt.Errorf("no JSON entry found in %s", filepath.Base(path))
		}
		if err != nil {
			_ = gz.Close()
			_ = f.Close()
			return nil, fmt.Errorf("tar: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && strings.HasSuffix(strings.ToLower(hdr.Name), ".json") {
			return &layeredReader{src: tr, closers: []io.Closer{gz, f}}, nil
		}
	}
}

func openFromZip(path string) (io.ReadCloser, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(zf.Name), ".json") {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			zr.Close()
			return nil, fmt.Errorf("open zip entry: %w", err)
		}
		return &layeredReader{src: rc, closers: []io.Closer{rc, zr}}, nil
	}
	zr.Close()
	return nil, fmt.Errorf("no JSON entry found in %s", filepath.Base(path))
}

// layeredReader reads from src and closes each layered reader in order when
// Close is called.
type layeredReader struct {
	src     io.Reader
	closers []io.Closer
}

func (l *layeredReader) Read(p []byte) (int, error) {
	return l.src.Read(p)
}

func (l *layeredReader) Close() error {
	var err error
	for _, c := range l.closers {
		if e := c.Close(); e != nil {
			err = e
		}
	}
	return err
}

// expectDelim reads the next token and asserts it is the given delimiter.
func expectDelim(dec *json.Decoder, want rune) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := tok.(json.Delim)
	if !ok || rune(d) != want {
		return fmt.Errorf("unexpected token %v, want %q", tok, want)
	}
	return nil
}
