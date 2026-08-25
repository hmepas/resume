package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

type Adapter struct{}

func (Adapter) ID() string { return "claude" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := common.HomePath(".claude", "projects")
	if err != nil {
		return nil, err
	}
	if !common.Exists(root) {
		return nil, nil
	}

	activeNames := activeSessionNames()
	var sessions []resume.Session
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Cheap prefilter: dir names encode the project path. It is loose
		// (over-inclusive), the recorded cwd below filters precisely.
		if !ctx.All && !looseDirMatch(entry.Name(), ctx.Project) {
			continue
		}
		_ = common.WalkFiles(filepath.Join(root, entry.Name()), func(path string) {
			if strings.HasSuffix(path, ".jsonl") && !isClaudeChildSession(path) {
				session := parseSession(path, activeNames)
				if session.SourcePath == "" {
					return
				}
				if !ctx.All && !resume.PathMatches(ctx.Project, session.Project) {
					return
				}
				sessions = append(sessions, session)
			}
		})
	}
	return sessions, nil
}

// looseDirMatch reports whether an encoded project dir name could refer to a
// path inside the project. Claude replaces path separators and punctuation
// with "-", which is ambiguous, so both sides are normalized the same way and
// compared by prefix; over-matching is fine, under-matching is not.
func looseDirMatch(dirName string, project resume.Project) bool {
	dir := looseKey(dirName)
	for _, path := range []string{project.Root, project.Path} {
		key := looseKey(path)
		if key == "" {
			continue
		}
		if dir == key || strings.HasPrefix(dir, key+"-") {
			return true
		}
	}
	return false
}

func looseKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func activeSessionNames() map[string]string {
	dir, err := common.HomePath(".claude", "sessions")
	if err != nil || !common.Exists(dir) {
		return nil
	}
	names := make(map[string]string)
	_ = common.WalkFiles(dir, func(path string) {
		if !strings.HasSuffix(path, ".json") {
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return
		}
		id := common.String(obj, "sessionId")
		name := strings.TrimSpace(common.String(obj, "name"))
		if id != "" && name != "" {
			names[id] = name
		}
	})
	return names
}

func isClaudeChildSession(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if part == "subagents" || part == "tool-results" {
			return true
		}
	}
	return false
}

func decodeProjectPath(dir string) string {
	base := filepath.Base(dir)
	if !strings.HasPrefix(base, "-") {
		return ""
	}
	return string(filepath.Separator) + strings.ReplaceAll(strings.TrimPrefix(base, "-"), "-", string(filepath.Separator))
}

const (
	boundedHead = 64 * 1024
	boundedTail = 64 * 1024
)

func parseSession(path string, activeNames map[string]string) resume.Session {
	session := scanSession(path, activeNames, true)
	if session.SourcePath == "" {
		// The bounded scan found nothing useful (e.g. a title buried in the
		// middle of the file); fall back to a full scan for this file only.
		if info, err := os.Stat(path); err == nil && info.Size() > boundedHead+boundedTail {
			session = scanSession(path, activeNames, false)
		}
	}
	return session
}

func scanSession(path string, activeNames map[string]string, bounded bool) resume.Session {
	project := decodeProjectPath(filepath.Dir(path))
	var title string
	var aiTitle string
	var customTitle string
	var commandTitle string
	var startupTitle string
	var sessionID string
	var startedAt time.Time
	var updatedAt time.Time

	scan := common.JSONLLines
	if bounded {
		scan = func(path string, fn func(map[string]any)) error {
			return common.JSONLBounded(path, boundedHead, boundedTail, fn)
		}
	}
	_ = scan(path, func(obj map[string]any) {
		if sessionID == "" {
			sessionID = common.String(obj, "sessionId")
		}
		if aiTitle == "" {
			aiTitle = common.String(obj, "aiTitle")
		}
		if common.String(obj, "type") == "custom-title" {
			if value := common.String(obj, "customTitle"); value != "" {
				customTitle = value
			}
		}
		if commandTitle == "" {
			commandTitle = commandName(common.FirstUserText(obj))
		}
		if startupTitle == "" {
			startupTitle = startupHookTitle(obj)
		}
		if cwd := common.String(obj, "cwd"); cwd != "" {
			project = cwd
		}
		if ts := common.ParseTime(common.String(obj, "timestamp")); !ts.IsZero() {
			if startedAt.IsZero() || ts.Before(startedAt) {
				startedAt = ts
			}
			if ts.After(updatedAt) {
				updatedAt = ts
			}
		}
		if title == "" {
			if text := common.FirstUserText(obj); common.UsefulTitle(text) {
				title = text
			}
		}
	})

	if updatedAt.IsZero() {
		updatedAt = common.FileModTime(path)
	}
	if project == "" {
		return resume.Session{}
	}
	if sessionID == "" {
		sessionID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	displayName := strings.TrimSpace(activeNames[sessionID])
	if displayName == "" {
		displayName = customTitle
	}
	if displayName == "" {
		displayName = aiTitle
	}
	if displayName == "" {
		displayName = commandTitle
	}
	if displayName == "" {
		displayName = startupTitle
	}
	if displayName == "" && title == "" {
		return resume.Session{}
	}
	title = combineTitle(displayName, title)

	return resume.Session{
		Agent:      "claude",
		ID:         sessionID,
		Project:    project,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      title,
		SourcePath: path,
		ResumeHint: "claude --resume " + sessionID,
		Confidence: "high",
	}
}

var commandNamePattern = regexp.MustCompile(`(?s)<command-name>\s*([^<]+?)\s*</command-name>`)

func commandName(text string) string {
	matches := commandNamePattern.FindStringSubmatch(text)
	if len(matches) != 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func startupHookTitle(obj map[string]any) string {
	attachment, ok := obj["attachment"].(map[string]any)
	if !ok || common.String(attachment, "type") != "hook_success" {
		return ""
	}
	text := strings.TrimSpace(common.String(attachment, "content"))
	if !common.UsefulTitle(text) {
		return ""
	}
	return text
}

func combineTitle(name, text string) string {
	if name == "" {
		return text
	}
	if text == "" || text == name {
		return name
	}
	return name + ": " + text
}
