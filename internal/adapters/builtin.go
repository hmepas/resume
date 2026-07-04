package adapters

import (
	"github.com/hmepas/resume/internal/adapters/claude"
	"github.com/hmepas/resume/internal/adapters/codex"
	"github.com/hmepas/resume/internal/adapters/cursor"
	"github.com/hmepas/resume/internal/adapters/gemini"
	"github.com/hmepas/resume/internal/adapters/opencode"
	"github.com/hmepas/resume/internal/adapters/pi"
	"github.com/hmepas/resume/internal/resume"
)

func Builtin() []resume.Adapter {
	return []resume.Adapter{
		codex.Adapter{},
		claude.Adapter{},
		gemini.Adapter{},
		cursor.Adapter{},
		opencode.Adapter{},
		pi.Adapter{},
	}
}
