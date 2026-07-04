package resume

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

type RenderOptions struct {
	All bool
}

func RenderTable(w io.Writer, sessions []Session, diagnostics []Diagnostic, opts RenderOptions) error {
	if len(sessions) == 0 {
		if opts.All {
			fmt.Fprintln(w, "No sessions found.")
		} else {
			fmt.Fprintln(w, "No sessions found for this project.")
		}
		printDiagnostics(w, diagnostics)
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if opts.All {
		fmt.Fprintln(tw, "UPDATED\tAGENT\tID\tPROJECT\tTITLE\tSOURCE")
	} else {
		fmt.Fprintln(tw, "UPDATED\tAGENT\tID\tTITLE\tSOURCE")
	}
	for _, session := range sessions {
		updated := formatTime(session.UpdatedAt)
		title := oneLine(session.Title)
		if title == "" {
			title = "(untitled)"
		}
		source := shortPath(session.SourcePath)
		if opts.All {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", updated, session.Agent, session.ID, shortPath(session.Project), title, source)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", updated, session.Agent, session.ID, title, source)
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	printDiagnostics(w, diagnostics)
	return nil
}

func printDiagnostics(w io.Writer, diagnostics []Diagnostic) {
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(w, "# %s: %s\n", diagnostic.Agent, diagnostic.Error)
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 96
	return truncateRunes(s, max, "...")
}

func shortPath(path string) string {
	if path == "" {
		return ""
	}
	if home := homeDir(); home != "" {
		if path == home {
			return "~"
		}
		if rel, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
			path = "~" + string(filepath.Separator) + rel
		}
	}
	return truncateRunesLeft(path, 64, "...")
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func truncateRunes(s string, max int, suffix string) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= len([]rune(suffix)) {
		return string(runes[:max])
	}
	return string(runes[:max-len([]rune(suffix))]) + suffix
}

func truncateRunesLeft(s string, max int, prefix string) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	prefixRunes := []rune(prefix)
	if max <= len(prefixRunes) {
		return string(runes[len(runes)-max:])
	}
	return prefix + string(runes[len(runes)-max+len(prefixRunes):])
}
