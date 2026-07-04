package common

import "testing"

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
