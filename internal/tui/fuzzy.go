package tui

import (
	"sort"
	"strings"
	"unicode"

	"github.com/hmepas/resume/internal/resume"
)

type match struct {
	session resume.Session
	score   int
}

func filterSessions(sessions []resume.Session, query string) []resume.Session {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return sessions
	}

	matches := make([]match, 0, len(sessions))
	for _, session := range sessions {
		score, ok := sessionScore(session, query)
		if ok {
			matches = append(matches, match{session: session, score: score})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].session.UpdatedAt.After(matches[j].session.UpdatedAt)
	})

	out := make([]resume.Session, len(matches))
	for i, match := range matches {
		out[i] = match.session
	}
	return out
}

func sessionScore(session resume.Session, query string) (int, bool) {
	agent := strings.ToLower(session.Agent)
	title := strings.ToLower(session.Title)
	project := strings.ToLower(session.Project)

	best := -1
	if score, ok := fuzzyScore(agent, query); ok {
		score *= 8
		if agent == query {
			score += 1000
		} else if strings.HasPrefix(agent, query) {
			score += 500
		}
		best = max(best, score)
	}
	if score, ok := fuzzyScore(title, query); ok {
		best = max(best, score*4)
	}
	if score, ok := fuzzyScore(project, query); ok {
		best = max(best, score)
	}
	if best < 0 {
		return 0, false
	}
	return best, true
}

func fuzzyScore(text, query string) (int, bool) {
	if query == "" {
		return 0, true
	}

	textRunes := []rune(text)
	queryRunes := []rune(query)
	score := 0
	last := -1
	for _, q := range queryRunes {
		found := -1
		for i := last + 1; i < len(textRunes); i++ {
			if textRunes[i] == q {
				found = i
				break
			}
		}
		if found < 0 {
			return 0, false
		}

		score += 10
		if found == 0 || unicode.IsSpace(textRunes[found-1]) || strings.ContainsRune("/_-:.", textRunes[found-1]) {
			score += 8
		}
		if last >= 0 {
			gap := found - last
			if gap == 1 {
				score += 12
			} else {
				score -= gap
			}
		}
		last = found
	}
	return score, true
}
