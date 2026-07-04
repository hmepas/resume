package resume

import (
	"testing"
	"unicode/utf8"
)

func TestOneLineTruncatesUnicodeSafely(t *testing.T) {
	input := "куда тебе положить аналог CLAUDE.md чтобы это добавлялось в системный промпт? и есть ли какие-то режимы"
	got := oneLine(input)
	if got == "" {
		t.Fatal("oneLine returned empty string")
	}
	if got[len(got)-3:] != "..." {
		t.Fatalf("oneLine() = %q, want ellipsis suffix", got)
	}
	if !validUTF8(got) {
		t.Fatalf("oneLine() returned invalid UTF-8: %q", got)
	}
}

func TestShortPathTruncatesUnicodeSafely(t *testing.T) {
	got := shortPath("/Users/hmepas/projects/пример/очень/длинного/пути/который/надо/укоротить/без/битого/utf8/session.jsonl")
	if !validUTF8(got) {
		t.Fatalf("shortPath() returned invalid UTF-8: %q", got)
	}
}

func validUTF8(s string) bool {
	return utf8.ValidString(s)
}
