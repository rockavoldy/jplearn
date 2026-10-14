package epub

import (
	"archive/zip"
	"bytes"
	"testing"
)

// buildTestEPUB creates an in-memory EPUB archive with a minimal structure:
// container.xml -> content.opf (2 chapters + 1 image).
func buildTestEPUB(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>テストの本</dc:title>
    <dc:creator>著者</dc:creator>
  </metadata>
  <manifest>
    <item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="img1" href="images/pic.png" media-type="image/png"/>
  </manifest>
  <spine>
    <itemref idref="c1"/>
    <itemref idref="c2"/>
  </spine>
</package>`,
		"OEBPS/ch1.xhtml":      `<html><head><title>第一章</title></head><body><p>猫が好きです。</p><img src="images/pic.png"/></body></html>`,
		"OEBPS/ch2.xhtml":      `<html><head><title>第二章</title></head><body><p>犬も好きです。</p></body></html>`,
		"OEBPS/images/pic.png": "\x89PNG\r\n\x1a\nFAKE",
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestParseEPUB(t *testing.T) {
	data := buildTestEPUB(t)
	parsed, err := Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if parsed.Meta.Title != "テストの本" {
		t.Errorf("title = %q, want %q", parsed.Meta.Title, "テストの本")
	}
	if parsed.Meta.Author != "著者" {
		t.Errorf("author = %q, want %q", parsed.Meta.Author, "著者")
	}
	if len(parsed.Chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(parsed.Chapters))
	}
	if parsed.Chapters[0].Path != "OEBPS/ch1.xhtml" {
		t.Errorf("first chapter path = %q", parsed.Chapters[0].Path)
	}
	if parsed.Chapters[0].Title != "第一章" {
		t.Errorf("first chapter title = %q", parsed.Chapters[0].Title)
	}
	if len(parsed.Images) != 1 {
		t.Fatalf("images = %d, want 1", len(parsed.Images))
	}
	if parsed.Images[0].Path != "OEBPS/images/pic.png" {
		t.Errorf("image path = %q", parsed.Images[0].Path)
	}
	if parsed.Images[0].Mime != "image/png" {
		t.Errorf("image mime = %q", parsed.Images[0].Mime)
	}
}

func TestResolvePath(t *testing.T) {
	cases := []struct{ dir, href, want string }{
		{"OEBPS", "ch1.xhtml", "OEBPS/ch1.xhtml"},
		{"OEBPS", "images/pic.png", "OEBPS/images/pic.png"},
		{"", "content.xhtml", "content.xhtml"},
		{"OEBPS/text", "../images/pic.png", "OEBPS/images/pic.png"},
	}
	for _, c := range cases {
		if got := resolvePath(c.dir, c.href); got != c.want {
			t.Errorf("resolvePath(%q,%q) = %q, want %q", c.dir, c.href, got, c.want)
		}
	}
}
