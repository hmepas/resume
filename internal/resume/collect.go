package resume

import (
	"context"
	"sort"
)

type CollectOptions struct {
	Project Project
	All     bool
	Limit   int
}

func Collect(ctx context.Context, adapters []Adapter, opts CollectOptions) ([]Session, []Diagnostic) {
	var sessions []Session
	var diagnostics []Diagnostic

	for _, adapter := range adapters {
		select {
		case <-ctx.Done():
			diagnostics = append(diagnostics, Diagnostic{Agent: adapter.ID(), Error: ctx.Err().Error()})
			continue
		default:
		}

		found, err := adapter.Sessions(Context{Project: opts.Project, All: opts.All})
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{Agent: adapter.ID(), Error: err.Error()})
			continue
		}
		sessions = append(sessions, found...)
	}

	sessions = dedupe(sessions)
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	if opts.Limit > 0 && len(sessions) > opts.Limit {
		sessions = sessions[:opts.Limit]
	}
	return sessions, diagnostics
}

func dedupe(in []Session) []Session {
	seen := make(map[string]bool, len(in))
	out := make([]Session, 0, len(in))
	for _, session := range in {
		key := session.Agent + "\x00" + session.SourcePath
		if session.ID != "" {
			key = session.Agent + "\x00" + session.ID
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, session)
	}
	return out
}
