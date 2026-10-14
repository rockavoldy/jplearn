package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"jplearn/internal/models"
)

// sampleJSON is a minimal jmdict-simplified document with metadata before the
// words array, matching the real distribution layout.
const sampleJSON = `{
  "version": "3.5.0",
  "dictDate": "2024-01-01",
  "languages": ["eng"],
  "words": [
    {
      "id": "1000000",
      "kanji": [{"text": "食べる", "common": true, "tags": []}],
      "kana": [{"text": "たべる", "common": true, "tags": []}],
      "sense": [
        {
          "partOfSpeech": ["v1", "vt"],
          "gloss": [{"lang": "eng", "text": "to eat"}]
        }
      ]
    },
    {
      "id": "1000001",
      "kanji": [],
      "kana": [{"text": "いる", "common": true, "tags": []}],
      "sense": [
        {
          "partOfSpeech": ["v1"],
          "gloss": [
            {"lang": "eng", "text": "to be"},
            {"lang": "dut", "text": "zijn"}
          ]
        }
      ]
    }
  ]
}`

func collect() (insertFunc, *[]models.DictEntry) {
	var all []models.DictEntry
	fn := func(_ context.Context, entries []models.DictEntry) error {
		all = append(all, entries...)
		return nil
	}
	return fn, &all
}

func TestImportJSONStreamsWords(t *testing.T) {
	insert, got := collect()
	total, skipped, err := importJSON(context.Background(), insert, bytes.NewReader([]byte(sampleJSON)))
	if err != nil {
		t.Fatalf("importJSON error: %v", err)
	}
	if total != 2 || skipped != 0 {
		t.Fatalf("total=%d skipped=%d, want 2/0", total, skipped)
	}

	entries := *got
	if entries[0].EntSeq != 1000000 {
		t.Errorf("first ent_seq = %d, want 1000000", entries[0].EntSeq)
	}
	if len(entries[0].Kanji) != 1 || entries[0].Kanji[0] != "食べる" {
		t.Errorf("kanji = %v, want [食べる]", entries[0].Kanji)
	}
	if len(entries[0].Readings) != 1 || entries[0].Readings[0] != "たべる" {
		t.Errorf("readings = %v, want [たべる]", entries[0].Readings)
	}
	if len(entries[0].Meanings) != 1 || entries[0].Meanings[0].Glosses[0] != "to eat" {
		t.Errorf("meanings = %+v", entries[0].Meanings)
	}
	if len(entries[0].Meanings[0].PartOfSpeech) != 2 {
		t.Errorf("pos = %v, want 2 entries", entries[0].Meanings[0].PartOfSpeech)
	}

	// Second entry has no kanji and a non-English gloss that must be dropped.
	if len(entries[1].Kanji) != 0 {
		t.Errorf("second entry kanji = %v, want empty", entries[1].Kanji)
	}
	if len(entries[1].Meanings[0].Glosses) != 1 || entries[1].Meanings[0].Glosses[0] != "to be" {
		t.Errorf("glosses = %v, want [to be] (dutch filtered)", entries[1].Meanings[0].Glosses)
	}
}

func TestImportJSONSkipsInvalidID(t *testing.T) {
	doc := `{"words":[{"id":"not-a-number","kana":[{"text":"あ"}],"sense":[]}]}`
	insert, got := collect()
	total, skipped, err := importJSON(context.Background(), insert, bytes.NewReader([]byte(doc)))
	if err != nil {
		t.Fatalf("importJSON error: %v", err)
	}
	if total != 0 || skipped != 1 {
		t.Fatalf("total=%d skipped=%d, want 0/1", total, skipped)
	}
	if len(*got) != 0 {
		t.Fatalf("expected no entries, got %d", len(*got))
	}
}

func TestOpenDictionaryPlainJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.json")
	if err := os.WriteFile(path, []byte(sampleJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	assertOpensAndParses(t, path)
}

func TestOpenDictionaryGzip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.json.gz")
	writeGzip(t, path, []byte(sampleJSON))
	assertOpensAndParses(t, path)
}

func TestOpenDictionaryTarGz(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.json.tgz")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	payload := []byte(sampleJSON)
	if err := tw.WriteHeader(&tar.Header{
		Name: "jmdict-eng-3.5.0.json",
		Mode: 0o644,
		Size: int64(len(payload)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	assertOpensAndParses(t, path)
}

func TestOpenDictionaryZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dict.json.zip")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("jmdict-eng-3.5.0.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(sampleJSON)); err != nil {
		t.Fatal(err)
	}
	zw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	assertOpensAndParses(t, path)
}

func assertOpensAndParses(t *testing.T, path string) {
	t.Helper()
	rc, err := openDictionary(path)
	if err != nil {
		t.Fatalf("openDictionary(%s): %v", path, err)
	}
	defer rc.Close()

	insert, got := collect()
	total, _, err := importJSON(context.Background(), insert, rc)
	if err != nil {
		t.Fatalf("importJSON from %s: %v", path, err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(*got) != 2 {
		t.Fatalf("entries = %d, want 2", len(*got))
	}
}

// TestImportRealDictionary runs against a real jmdict-simplified archive when
// JMDICT_TEST_FILE points at one; otherwise it is skipped. Useful for catching
// upstream schema drift.
func TestImportRealDictionary(t *testing.T) {
	path := os.Getenv("JMDICT_TEST_FILE")
	if path == "" {
		t.Skip("set JMDICT_TEST_FILE to a jmdict-simplified archive to run")
	}

	rc, err := openDictionary(path)
	if err != nil {
		t.Fatalf("openDictionary: %v", err)
	}
	defer rc.Close()

	insert, got := collect()
	total, skipped, err := importJSON(context.Background(), insert, rc)
	if err != nil {
		t.Fatalf("importJSON: %v", err)
	}
	if total == 0 {
		t.Fatal("expected at least one imported entry")
	}
	t.Logf("imported=%d skipped=%d", total, skipped)

	// Spot-check that a well-known common word resolved with English glosses.
	for _, e := range *got {
		if len(e.Kanji) > 0 && e.Kanji[0] == "食べる" {
			if len(e.Meanings) == 0 || len(e.Meanings[0].Glosses) == 0 {
				t.Fatalf("食べる entry has no glosses: %+v", e)
			}
			return
		}
	}
	t.Log("note: 食べる not present in this dictionary slice (e.g. common-only subset)")
}

func writeGzip(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}
