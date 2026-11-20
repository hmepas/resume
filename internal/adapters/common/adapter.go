package common

import (
	"fmt"
	"os"

	"github.com/hmepas/resume/internal/resume"
)

// ResumeCommand builds "<agent> <args...> <session id>", run in the session's
// project directory. The agent ID doubles as the binary name.
func ResumeCommand(session resume.Session, args ...string) (resume.Command, error) {
	if session.ID == "" {
		return resume.Command{}, fmt.Errorf("%s session has no resume id", session.Agent)
	}
	return resume.Command{
		Name: session.Agent,
		Args: append(append([]string(nil), args...), session.ID),
		Dir:  session.Project,
	}, nil
}

// FileDeleter implements resume.Deleter for adapters that keep one file per
// session: it removes the session's source file. Embed it in the Adapter.
type FileDeleter struct{}

var _ resume.Deleter = FileDeleter{}

func (FileDeleter) DeleteScope() string { return "session file" }

func (FileDeleter) Delete(session resume.Session) error { return os.Remove(session.SourcePath) }
