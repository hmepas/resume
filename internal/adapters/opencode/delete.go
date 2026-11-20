package opencode

import (
	"errors"
	"os/exec"

	"github.com/hmepas/resume/internal/resume"
)

// runSessionDelete is a test seam for the opencode CLI call.
var runSessionDelete = func(sessionID string) error {
	cmd := exec.Command("opencode", "session", "delete", sessionID)
	return cmd.Run()
}

func (Adapter) DeleteScope() string { return "session" }

// Delete removes the session through the opencode CLI.
func (Adapter) Delete(session resume.Session) error {
	if session.ID == "" {
		return errors.New("opencode session has no id")
	}
	return runSessionDelete(session.ID)
}
