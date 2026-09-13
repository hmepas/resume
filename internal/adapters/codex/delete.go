package codex

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

func isGuardian(obj map[string]any) bool {
	return common.String(obj, "payload", "thread_source") == "guardian_review" ||
		common.String(obj, "payload", "source", "subagent", "other") == "guardian"
}

// DeleteSession removes guardian journals first, keeping the parent available
// for a retry if deleting a review fails. Ordinary subagent journals are retained.
func DeleteSession(session resume.Session) error {
	root, err := common.HomePath(".codex")
	if err != nil {
		return err
	}
	type journal struct {
		path, id, owner, parent string
		guardian                bool
	}
	var journals []journal
	for _, dir := range []string{"sessions", "archived_sessions"} {
		err = filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) && path == filepath.Join(root, dir) {
					return nil
				}
				return walkErr
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			if !strings.HasSuffix(path, ".jsonl") {
				return nil
			}
			f, e := os.Open(path)
			if e != nil {
				return e
			}
			defer f.Close()
			var obj map[string]any
			if json.NewDecoder(f).Decode(&obj) != nil || common.String(obj, "type") != "session_meta" {
				return nil
			}
			journals = append(journals, journal{path, common.String(obj, "payload", "id"), common.String(obj, "payload", "session_id"), common.String(obj, "payload", "parent_thread_id"), isGuardian(obj)})
			return nil
		})
		if err != nil {
			return err
		}
	}
	related := map[string]bool{}
	if session.ID != "" {
		related[session.ID] = true
	}
	for changed := true; changed; {
		changed = false
		for _, j := range journals {
			if j.id != "" && !related[j.id] && (related[j.owner] || related[j.parent]) {
				related[j.id] = true
				changed = true
			}
		}
	}
	for _, j := range journals {
		if j.guardian && related[j.id] && j.path != session.SourcePath {
			if err := os.Remove(j.path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return os.Remove(session.SourcePath)
}
