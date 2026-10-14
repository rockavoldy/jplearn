// Package epub parses EPUB archives: container -> OPF -> spine ordering,
// chapter XHTML extraction and image discovery.
package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
)

// BookMeta holds bibliographic metadata parsed from the OPF package.
type BookMeta struct {
	Title  string
	Author string
}

// ChapterFile is a single spine item with its raw XHTML.
type ChapterFile struct {
	Path  string // path within the archive
	Title string
	HTML  string
}

// ImageFile is an image resource discovered inside the EPUB.
type ImageFile struct {
	Path string // resolved path within the archive (for <img src> matching)
	Mime string
	Data []byte
}

// Parsed is the complete result of parsing an EPUB archive.
type Parsed struct {
	Meta     BookMeta
	Chapters []ChapterFile
	Images   []ImageFile
	// OPFDir is the directory of the OPF file, used to resolve relative paths.
	OPFDir string
}

// ---------------------------------------------------------------------------
// container.xml / OPF structures
// ---------------------------------------------------------------------------

type container struct {
	Rootfiles struct {
		Rootfile []struct {
			FullPath  string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type opfPackage struct {
	Metadata struct {
		Title    []string `xml:"title"`
		Creators []struct {
			Value string `xml:",chardata"`
			Role  string `xml:"role,attr"`
		} `xml:"creator"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID        string `xml:"id,attr"`
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
			Props     string `xml:"properties,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

// Parse reads an EPUB from a stream, returning metadata, ordered chapters and
// embedded images. It does not perform any network or database operations.
func Parse(r io.ReaderAt, size int64) (*Parsed, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("open epub zip: %w", err)
	}

	opfPath, err := findOPFPath(zr)
	if err != nil {
		return nil, err
	}
	opfDir := path.Dir(opfPath)
	if opfDir == "." {
		opfDir = ""
	}

	pkg, err := readOPF(zr, opfPath)
	if err != nil {
		return nil, err
	}

	out := &Parsed{OPFDir: opfDir}
	if len(pkg.Metadata.Title) > 0 {
		out.Meta.Title = strings.TrimSpace(pkg.Metadata.Title[0])
	}
	for _, c := range pkg.Metadata.Creators {
		if strings.TrimSpace(c.Value) != "" {
			out.Meta.Author = strings.TrimSpace(c.Value)
			break
		}
	}

	// Build manifest id -> item lookup.
	type manifestItem struct {
		href      string
		mediaType string
		props     string
	}
	items := make(map[string]manifestItem, len(pkg.Manifest.Items))
	orderedIDs := make([]string, 0, len(pkg.Manifest.Items))
	for _, it := range pkg.Manifest.Items {
		items[it.ID] = manifestItem{href: it.Href, mediaType: it.MediaType, props: it.Props}
		orderedIDs = append(orderedIDs, it.ID)
	}

	// Extract images from the manifest.
	for _, id := range orderedIDs {
		it := items[id]
		if !strings.HasPrefix(it.mediaType, "image/") {
			continue
		}
		fullPath := resolvePath(opfDir, it.href)
		data, err := readZipFile(zr, fullPath)
		if err != nil {
			// Image may be referenced but absent; skip gracefully.
			continue
		}
		out.Images = append(out.Images, ImageFile{
			Path: fullPath,
			Mime: it.mediaType,
			Data: data,
		})
	}

	// Walk the spine in reading order.
	for _, ref := range pkg.Spine.ItemRefs {
		it, ok := items[ref.IDRef]
		if !ok {
			continue
		}
		if !isContentDoc(it.mediaType) {
			continue
		}
		fullPath := resolvePath(opfDir, it.href)
		raw, err := readZipFile(zr, fullPath)
		if err != nil {
			continue
		}
		htmlContent := string(raw)
		out.Chapters = append(out.Chapters, ChapterFile{
			Path:  fullPath,
			Title: extractTitle(htmlContent),
			HTML:  htmlContent,
		})
	}

	if len(out.Chapters) == 0 {
		return nil, fmt.Errorf("epub contains no readable spine documents")
	}
	return out, nil
}

func findOPFPath(zr *zip.Reader) (string, error) {
	data, err := readZipFile(zr, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("read container.xml: %w", err)
	}
	var c container
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}
	for _, rf := range c.Rootfiles.Rootfile {
		if strings.Contains(rf.MediaType, "opf") || rf.FullPath != "" {
			return rf.FullPath, nil
		}
	}
	return "", fmt.Errorf("no rootfile found in container.xml")
}

func readOPF(zr *zip.Reader, opfPath string) (*opfPackage, error) {
	data, err := readZipFile(zr, opfPath)
	if err != nil {
		return nil, fmt.Errorf("read opf %s: %w", opfPath, err)
	}
	var pkg opfPackage
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse opf: %w", err)
	}
	return &pkg, nil
}

func readZipFile(zr *zip.Reader, name string) ([]byte, error) {
	name = strings.TrimPrefix(name, "/")
	var f io.ReadCloser
	if file, err := zr.Open(name); err == nil {
		f = file
	} else {
		// Try a case-insensitive fallback.
		for _, zf := range zr.File {
			if strings.EqualFold(zf.Name, name) {
				opened, oerr := zf.Open()
				if oerr != nil {
					return nil, oerr
				}
				f = opened
				break
			}
		}
		if f == nil {
			return nil, err
		}
	}
	defer f.Close()
	return io.ReadAll(f)
}

func resolvePath(dir, href string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "/") {
		return strings.TrimPrefix(href, "/")
	}
	joined := path.Join(dir, href)
	return strings.TrimPrefix(joined, "./")
}

func isContentDoc(mediaType string) bool {
	switch mediaType {
	case "application/xhtml+xml", "text/html", "application/x-dtbook+xml":
		return true
	}
	return strings.Contains(mediaType, "xhtml") || strings.Contains(mediaType, "html")
}

func extractTitle(htmlContent string) string {
	lower := strings.ToLower(htmlContent)
	if i := strings.Index(lower, "<title>"); i >= 0 {
		if j := strings.Index(lower[i:], "</title>"); j > 0 {
			start := i + len("<title>")
			end := i + j
			if end > start && end <= len(htmlContent) {
				return strings.TrimSpace(htmlContent[start:end])
			}
		}
	}
	return ""
}
