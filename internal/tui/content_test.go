package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hmepas/resume/internal/resume"
)

func TestFilterSessionsContentUnion(t *testing.T) {
	titleHit := resume.Session{Title: "fix payment bug", SourcePath: "/a"}
	contentHit := resume.Session{Title: "unrelated", SourcePath: "/b", UpdatedAt: time.Now()}
	miss := resume.Session{Title: "nothing here", SourcePath: "/c"}
	sessions := []resume.Session{miss, contentHit, titleHit}

	got := filterSessions(sessions, "payment", map[string]bool{"/b": true})
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
	if got[0].SourcePath != "/a" {
		t.Errorf("title match should rank first, got %q", got[0].SourcePath)
	}
	if got[1].SourcePath != "/b" {
		t.Errorf("content match should be included, got %q", got[1].SourcePath)
	}
}

func TestContentSearcher(t *testing.T) {
	s := newContentSearcher()
	if s.tool == nil {
		t.Skip("no rg/ag/grep available")
	}
	dir := t.TempDir()
	hit := filepath.Join(dir, "hit.jsonl")
	miss := filepath.Join(dir, "miss.jsonl")
	if err := os.WriteFile(hit, []byte("{\"text\":\"NeEdLe here\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(miss, []byte("{\"text\":\"nothing\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sessions := []resume.Session{{SourcePath: hit}, {SourcePath: miss}}

	got := s.matches(sessions, "needle")
	if !got[hit] || got[miss] {
		t.Fatalf("expected only %q to match, got %v", hit, got)
	}
	if s.matches(sessions, "x") != nil {
		t.Fatal("single-rune query should not trigger a scan")
	}
}

func TestFilterSessionsNoContentSet(t *testing.T) {
	sessions := []resume.Session{{Title: "hello world", SourcePath: "/a"}}
	if got := filterSessions(sessions, "zzz", nil); len(got) != 0 {
		t.Fatalf("expected no matches, got %d", len(got))
	}
}
