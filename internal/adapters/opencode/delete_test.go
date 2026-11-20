package opencode

import (
	"testing"

	"github.com/hmepas/resume/internal/resume"
)

func stubSessionDelete(t *testing.T) *[]string {
	t.Helper()
	var calls []string
	original := runSessionDelete
	runSessionDelete = func(sessionID string) error {
		calls = append(calls, sessionID)
		return nil
	}
	t.Cleanup(func() { runSessionDelete = original })
	return &calls
}

func TestDeleteRunsCLIWithSessionID(t *testing.T) {
	calls := stubSessionDelete(t)
	if err := (Adapter{}).Delete(resume.Session{Agent: "opencode", ID: "ses_123"}); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0] != "ses_123" {
		t.Fatalf("calls = %q, want [ses_123]", *calls)
	}
}

func TestDeleteRejectsEmptyID(t *testing.T) {
	calls := stubSessionDelete(t)
	if err := (Adapter{}).Delete(resume.Session{Agent: "opencode"}); err == nil {
		t.Fatal("Delete with empty ID returned nil error")
	}
	if len(*calls) != 0 {
		t.Fatalf("calls = %q, want none", *calls)
	}
}
