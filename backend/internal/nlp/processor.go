// Package nlp wraps the Kagome Japanese morphological analyzer to inject
// furigana (<ruby>) and interactive vocabulary spans into HTML.
package nlp

import (
	"fmt"
	stdhtml "html"
	"strings"
	"unicode"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Processor tokenizes Japanese text and rewrites it into annotated HTML.
type Processor struct {
	tok *tokenizer.Tokenizer
}

// New constructs a Processor backed by the IPA dictionary.
func New() (*Processor, error) {
	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		return nil, fmt.Errorf("create kagome tokenizer: %w", err)
	}
	return &Processor{tok: t}, nil
}

// AnnotateHTML walks an HTML fragment/document and annotates every Japanese
// text node. Script/style/ruby subtrees are preserved untouched.
func (p *Processor) AnnotateHTML(input string) (string, error) {
	ctx := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(input), ctx)
	if err != nil {
		return "", fmt.Errorf("parse html fragment: %w", err)
	}

	var buf strings.Builder
	for _, n := range nodes {
		p.annotateNode(n)
		if err := html.Render(&buf, n); err != nil {
			return "", fmt.Errorf("render node: %w", err)
		}
	}
	return buf.String(), nil
}

func (p *Processor) annotateNode(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		p.replaceTextNode(n)
		return
	case html.ElementNode:
		switch n.Data {
		case "script", "style", "ruby", "rt", "rp":
			return
		}
	}

	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		p.annotateNode(c)
		c = next
	}
}

func (p *Processor) replaceTextNode(n *html.Node) {
	if !containsJapanese(n.Data) {
		return
	}
	annotated := p.AnnotateText(n.Data)
	parent := n.Parent
	if parent == nil {
		return
	}
	fragCtx := &html.Node{Type: html.ElementNode, Data: "span", DataAtom: atom.Span}
	frag, err := html.ParseFragment(strings.NewReader(annotated), fragCtx)
	if err != nil {
		return
	}
	for _, fn := range frag {
		parent.InsertBefore(fn, n)
	}
	parent.RemoveChild(n)
}

// AnnotateText tokenizes a raw text string and returns annotated HTML.
// Japanese tokens become <span class="vocab-word"> elements; kanji-bearing
// tokens additionally get <ruby> furigana.
func (p *Processor) AnnotateText(text string) string {
	if !containsJapanese(text) {
		return stdhtml.EscapeString(text)
	}

	tokens := p.tok.Tokenize(text)
	var b strings.Builder
	for _, t := range tokens {
		surface := t.Surface
		if surface == "" {
			continue
		}
		if !containsJapanese(surface) {
			b.WriteString(stdhtml.EscapeString(surface))
			continue
		}

		lemma := surface
		if bf, ok := t.BaseForm(); ok && bf != "" && bf != "*" {
			lemma = bf
		}

		reading := ""
		if r, ok := t.Reading(); ok && r != "*" {
			reading = kataToHira(r)
		}

		b.WriteString(`<span class="vocab-word" data-lemma="`)
		b.WriteString(stdhtml.EscapeString(lemma))
		b.WriteString(`" data-reading="`)
		b.WriteString(stdhtml.EscapeString(reading))
		b.WriteString(`" data-surface="`)
		b.WriteString(stdhtml.EscapeString(surface))
		b.WriteString(`">`)

		if containsKanji(surface) && reading != "" && reading != surface {
			b.WriteString(`<ruby>`)
			b.WriteString(stdhtml.EscapeString(surface))
			b.WriteString(`<rt>`)
			b.WriteString(stdhtml.EscapeString(reading))
			b.WriteString(`</rt></ruby>`)
		} else {
			b.WriteString(stdhtml.EscapeString(surface))
		}
		b.WriteString(`</span>`)
	}
	return b.String()
}

// kataToHira converts full-width katakana to hiragana.
func kataToHira(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0x30A1 && r <= 0x30F6 {
			r -= 0x60
		}
		b.WriteRune(r)
	}
	return b.String()
}

func containsJapanese(s string) bool {
	for _, r := range s {
		if isKanji(r) || isKana(r) {
			return true
		}
	}
	return false
}

func containsKanji(s string) bool {
	for _, r := range s {
		if isKanji(r) {
			return true
		}
	}
	return false
}

func isKanji(r rune) bool {
	return unicode.In(r, unicode.Han)
}

func isKana(r rune) bool {
	return unicode.In(r, unicode.Hiragana, unicode.Katakana)
}
