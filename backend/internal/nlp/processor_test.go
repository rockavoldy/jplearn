package nlp

import (
	"strings"
	"testing"
)

func TestAnnotateTextWrapsKanjiWithFurigana(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	out := p.AnnotateText("私は本を読みます。")

	if !strings.Contains(out, `class="vocab-word"`) {
		t.Fatalf("expected vocab-word spans, got: %s", out)
	}
	if !strings.Contains(out, "<ruby>") || !strings.Contains(out, "<rt>") {
		t.Fatalf("expected ruby/rt furigana for kanji, got: %s", out)
	}
	if !strings.Contains(out, "data-lemma=") {
		t.Fatalf("expected data-lemma attribute, got: %s", out)
	}
}

func TestAnnotateTextLeavesNonJapaneseUntouched(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	out := p.AnnotateText("hello world 123")
	if strings.Contains(out, "vocab-word") {
		t.Fatalf("non-Japanese text should not be annotated, got: %s", out)
	}
	if out != "hello world 123" {
		t.Fatalf("expected passthrough, got: %s", out)
	}
}

func TestAnnotateHTMLEscapesAndPreservesStructure(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	in := `<p>猫が好き</p><script>var x = 1;</script>`
	out, err := p.AnnotateHTML(in)
	if err != nil {
		t.Fatalf("AnnotateHTML error: %v", err)
	}

	if !strings.Contains(out, "<p>") || !strings.Contains(out, "</p>") {
		t.Fatalf("paragraph structure not preserved: %s", out)
	}
	if !strings.Contains(out, "<script>") {
		t.Fatalf("script element should be preserved untouched: %s", out)
	}
	if !strings.Contains(out, "vocab-word") {
		t.Fatalf("expected Japanese text to be annotated: %s", out)
	}
}

func TestKataToHira(t *testing.T) {
	if got := kataToHira("カタカナ"); got != "かたかな" {
		t.Fatalf("kataToHira = %q, want %q", got, "かたかな")
	}
}
