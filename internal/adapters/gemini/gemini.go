package gemini

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hmepas/resume/internal/adapters/common"
	"github.com/hmepas/resume/internal/resume"
)

type Adapter struct{}

func (Adapter) ID() string { return "gemini" }

func (Adapter) Sessions(ctx resume.Context) ([]resume.Session, error) {
	root, err := common.HomePath(".gemini", "tmp")
	if err != nil {
		return nil, err
	}
	if !common.Exists(root) {
		return nil, nil
	}

	var sessions []resume.Session
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectDir := filepath.Join(root, entry.Name())
		project := readProjectRoot(projectDir)
		if project == "" {
			continue
		}
		if !ctx.All && !resume.PathMatches(ctx.Project, project) {
			continue
		}
		chats := filepath.Join(projectDir, "chats")
		_ = common.WalkFiles(chats, func(path string) {
			if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") {
				if s := parseChat(path, project); s.SourcePath != "" {
					sessions = append(sessions, s)
				}
			}
		})
	}
	return sessions, nil
}

func readProjectRoot(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".project_root"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func parseChat(path, project string) resume.Session {
	if strings.HasSuffix(path, ".jsonl") {
		return parseJSONLChat(path, project)
	}
	return parseJSONChat(path, project)
}

func parseJSONChat(path, project string) resume.Session {
	data, err := os.ReadFile(path)
	if err != nil {
		return resume.Session{}
	}
	var obj struct {
		SessionID   string `json:"sessionId"`
		StartTime   string `json:"startTime"`
		LastUpdated string `json:"lastUpdated"`
		Messages    []struct {
			Type    string `json:"type"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return resume.Session{}
	}

	title := ""
	for _, msg := range obj.Messages {
		if msg.Type == "user" {
			title = contentText(msg.Content)
			break
		}
	}
	updated := common.ParseTime(obj.LastUpdated)
	if updated.IsZero() {
		updated = common.FileModTime(path)
	}

	return resume.Session{
		Agent:      "gemini",
		ID:         obj.SessionID,
		Project:    project,
		StartedAt:  common.ParseTime(obj.StartTime),
		UpdatedAt:  updated,
		Title:      title,
		SourcePath: path,
		ResumeHint: "gemini",
		Confidence: "high",
	}
}

func parseJSONLChat(path, project string) resume.Session {
	id := sessionIDFromFilename(path)
	var title string
	var startedAt time.Time
	var updatedAt time.Time
	_ = common.JSONLLines(path, func(obj map[string]any) {
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
	return resume.Session{
		Agent:      "gemini",
		ID:         id,
		Project:    project,
		StartedAt:  startedAt,
		UpdatedAt:  updatedAt,
		Title:      title,
		SourcePath: path,
		ResumeHint: "gemini",
		Confidence: "medium",
	}
}

func sessionIDFromFilename(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return strings.TrimPrefix(base, "session-")
}

func contentText(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if text, _ := m["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}
