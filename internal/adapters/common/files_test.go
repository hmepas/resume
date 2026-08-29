package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONLLinesSkipsOversizedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.jsonl")
	data := "{\"n\":1}\n" +
		`{"pad":"` + strings.Repeat("x", maxJSONLLine) + `"}` + "\n" +
		"{\"n\":2}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	var got []float64
	if err := JSONLLines(path, func(obj map[string]any) {
		if n, ok := obj["n"].(float64); ok {
			got = append(got, n)
		} else {
			t.Fatal("oversized line was not skipped")
		}
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("parsed lines = %v, want [1 2]", got)
	}
}

func TestFirstUserTextSkipsMetaLocalCommand(t *testing.T) {
	obj := map[string]any{
		"isMeta": true,
		"message": map[string]any{
			"role":    "user",
			"content": "<local-command-caveat>ignore me",
		},
	}

	if got := FirstUserText(obj); got != "" {
		t.Fatalf("FirstUserText() = %q, want empty", got)
	}
}

func TestUsefulTitleRejectsLocalCommandMarkers(t *testing.T) {
	if UsefulTitle("<local-command-caveat>ignore me") {
		t.Fatal("UsefulTitle() accepted local command caveat")
	}
	if !UsefulTitle("empty-pls-delete") {
		t.Fatal("UsefulTitle() rejected custom title")
	}
}
