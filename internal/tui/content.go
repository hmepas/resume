package tui

import (
	"os/exec"
	"strings"

	"github.com/hmepas/resume/internal/resume"
)

// contentSearcher greps session source files with the first available of
// rg/ag/grep so `/` also matches transcript content, not just titles.
type contentSearcher struct {
	tool  []string
	cache map[string]map[string]bool
}

func newContentSearcher() *contentSearcher {
	s := &contentSearcher{cache: map[string]map[string]bool{}}
	for _, candidate := range [][]string{
		{"rg", "-l", "-i", "-F", "--no-messages", "--"},
		{"ag", "-l", "-i", "-Q", "--silent", "--"},
		{"grep", "-l", "-i", "-F", "-s", "--"},
	} {
		if _, err := exec.LookPath(candidate[0]); err == nil {
			s.tool = candidate
			break
		}
	}
	return s
}

// matches returns the source paths whose file content contains query as a
// case-insensitive substring. Nil when no tool is available or the query is
// too short to be worth a scan.
func (s *contentSearcher) matches(sessions []resume.Session, query string) map[string]bool {
	query = strings.TrimSpace(query)
	if s == nil || s.tool == nil || len([]rune(query)) < 2 {
		return nil
	}
	if cached, ok := s.cache[query]; ok {
		return cached
	}

	seen := make(map[string]bool, len(sessions))
	paths := make([]string, 0, len(sessions))
	for _, session := range sessions {
		if session.SourcePath == "" || seen[session.SourcePath] {
			continue
		}
		seen[session.SourcePath] = true
		paths = append(paths, session.SourcePath)
	}
	if len(paths) == 0 {
		s.cache[query] = nil
		return nil
	}

	args := append(append([]string{}, s.tool[1:]...), query)
	args = append(args, paths...)
	out, _ := exec.Command(s.tool[0], args...).Output()

	matched := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			matched[line] = true
		}
	}
	s.cache[query] = matched
	return matched
}
