package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONLBoundedSkipsMiddle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	var b strings.Builder
	pad := strings.Repeat("x", 100)
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&b, "{\"n\":%d,\"pad\":%q}\n", i, pad)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	var got []float64
	if err := JSONLBounded(path, 1024, 1024, func(obj map[string]any) {
		got = append(got, obj["n"].(float64))
	}); err != nil {
		t.Fatal(err)
	}

	if len(got) == 0 || len(got) >= 100 {
		t.Fatalf("expected a bounded subset of lines, got %d", len(got))
	}
	if got[0] != 0 {
		t.Fatalf("expected first line from head, got %v", got[0])
	}
	if got[len(got)-1] != 99 {
		t.Fatalf("expected last line from tail, got %v", got[len(got)-1])
	}
	for i := 1; i < len(got); i++ {
		if got[i] == got[i-1] {
			t.Fatalf("line %v parsed twice", got[i])
		}
	}
}

func TestJSONLBoundedSmallFileFull(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.jsonl")
	if err := os.WriteFile(path, []byte("{\"n\":1}\n{\"n\":2}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := JSONLBounded(path, 1024, 1024, func(map[string]any) { count++ }); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 lines, got %d", count)
	}
}
