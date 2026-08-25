package adapters_test

// Profiling helpers against the real local session stores; opt-in via
// RESUME_PROFILE=1 (timing test) or -bench (benchmark).

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hmepas/resume/internal/adapters"
	"github.com/hmepas/resume/internal/resume"
)

func TestAdapterTiming(t *testing.T) {
	if os.Getenv("RESUME_PROFILE") == "" {
		t.Skip("profiling helper; set RESUME_PROFILE=1 to run")
	}
	project, err := resume.DetectProject(".")
	if err != nil {
		t.Fatal(err)
	}
	ctx := resume.Context{Project: project}
	for _, a := range adapters.Builtin() {
		start := time.Now()
		sessions, err := a.Sessions(ctx)
		fmt.Printf("%-10s %8.1fms  sessions=%d err=%v\n",
			a.ID(), float64(time.Since(start).Microseconds())/1000, len(sessions), err)
	}
}

func BenchmarkCollect(b *testing.B) {
	project, err := resume.DetectProject(".")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		resume.Collect(context.Background(), adapters.Builtin(),
			resume.CollectOptions{Project: project})
	}
}
