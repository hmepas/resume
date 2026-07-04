package common

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func HomePath(parts ...string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{home}, parts...)...), nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func JSONLLines(path string, fn func(map[string]any)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err == nil {
			fn(obj)
		}
	}
	return scanner.Err()
}

func FileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func WalkFiles(root string, fn func(string)) error {
	if !Exists(root) {
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type().IsRegular() {
			fn(path)
		}
		return nil
	})
}

func String(obj map[string]any, keys ...string) string {
	var cur any = obj
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[key]
	}
	switch v := cur.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15-04-05.000Z",
		"2006-01-02T15-04",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func FirstUserText(obj map[string]any) string {
	if isMeta, _ := obj["isMeta"].(bool); isMeta {
		return ""
	}

	if typ, _ := obj["type"].(string); typ == "event_msg" {
		payload, _ := obj["payload"].(map[string]any)
		if payloadType, _ := payload["type"].(string); payloadType != "user_message" {
			return ""
		}
		if text, _ := payload["message"].(string); text != "" {
			return text
		}
	}

	msg, ok := obj["message"].(map[string]any)
	if ok {
		if role, _ := msg["role"].(string); role == "user" {
			return contentText(msg["content"])
		}
	}
	if typ, _ := obj["type"].(string); typ == "user" {
		return contentText(obj["content"])
	}
	return ""
}

func UsefulTitle(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || text == "." {
		return false
	}
	localCommandPrefixes := []string{
		"<local-command-caveat>",
		"<local-command-stdout>",
		"<local-command-stderr>",
		"<command-name>",
	}
	for _, prefix := range localCommandPrefixes {
		if strings.HasPrefix(text, prefix) {
			return false
		}
	}
	return true
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

func Missing(path string) error {
	if Exists(path) {
		return nil
	}
	return errors.New(path + " not found")
}
