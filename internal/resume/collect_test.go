package resume

import "testing"

func TestDedupeUsesSessionIDWhenAvailable(t *testing.T) {
	sessions := []Session{
		{Agent: "opencode", ID: "ses_one", SourcePath: "/tmp/opencode.db"},
		{Agent: "opencode", ID: "ses_two", SourcePath: "/tmp/opencode.db"},
		{Agent: "opencode", ID: "ses_one", SourcePath: "/tmp/opencode.db"},
	}

	got := dedupe(sessions)
	if len(got) != 2 {
		t.Fatalf("len(dedupe) = %d, want 2", len(got))
	}
	if got[0].ID != "ses_one" || got[1].ID != "ses_two" {
		t.Fatalf("dedupe IDs = %q, %q; want ses_one, ses_two", got[0].ID, got[1].ID)
	}
}
