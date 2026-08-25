package resume

import (
	"context"
	"sort"
	"sync"
)

type CollectOptions struct {
	Project Project
	All     bool
	Limit   int
}

func Collect(ctx context.Context, adapters []Adapter, opts CollectOptions) ([]Session, []Diagnostic) {
	// Adapters run concurrently; results keep the adapter order for
	// deterministic output.
	found := make([][]Session, len(adapters))
	errs := make([]*Diagnostic, len(adapters))

	var wg sync.WaitGroup
	for i, adapter := range adapters {
		wg.Add(1)
		go func(i int, adapter Adapter) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				errs[i] = &Diagnostic{Agent: adapter.ID(), Error: ctx.Err().Error()}
				return
			default:
			}

			sessions, err := adapter.Sessions(Context{Project: opts.Project, All: opts.All})
			if err != nil {
				errs[i] = &Diagnostic{Agent: adapter.ID(), Error: err.Error()}
				return
			}
			found[i] = sessions
		}(i, adapter)
	}
	wg.Wait()

	var sessions []Session
	var diagnostics []Diagnostic
	for i := range adapters {
		if errs[i] != nil {
			diagnostics = append(diagnostics, *errs[i])
			continue
		}
		sessions = append(sessions, found[i]...)
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
