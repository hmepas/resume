package resume

import (
	"fmt"
	"time"
)

type Project struct {
	Path string `json:"path"`
	Root string `json:"root"`
}

type Session struct {
	Agent      string    `json:"agent"`
	ID         string    `json:"id,omitempty"`
	Project    string    `json:"project"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
	Title      string    `json:"title,omitempty"`
	SourcePath string    `json:"source_path"`
	ResumeHint string    `json:"resume_hint,omitempty"`
	Confidence string    `json:"confidence"`
}

type Diagnostic struct {
	Agent string `json:"agent"`
	Error string `json:"error"`
}

// Color is the ANSI SGR foreground code that tints an agent's name in the picker.
type Color string

const (
	ColorRed           Color = "31"
	ColorGreen         Color = "32"
	ColorYellow        Color = "33"
	ColorBlue          Color = "34"
	ColorMagenta       Color = "35"
	ColorCyan          Color = "36"
	ColorBrightMagenta Color = "95"
	ColorBrightCyan    Color = "96"
)

type Adapter interface {
	// ID is the agent name recorded on every session.
	ID() string
	// Color tints the agent name in the picker; unique per adapter.
	Color() Color
	Sessions(ctx Context) ([]Session, error)
	// Command returns the command that resumes the session.
	Command(session Session) (Command, error)
}

// Deleter is implemented by adapters whose sessions can be deleted.
type Deleter interface {
	// DeleteScope names what Delete removes, for the confirmation prompt
	// "delete <agent> <scope>? y/N", e.g. "session file".
	DeleteScope() string
	Delete(session Session) error
}

// Registry routes a session back to the adapter that produced it, keyed by
// Adapter.ID (which equals Session.Agent).
type Registry map[string]Adapter

func NewRegistry(adapters []Adapter) Registry {
	registry := make(Registry, len(adapters))
	for _, adapter := range adapters {
		registry[adapter.ID()] = adapter
	}
	return registry
}

// Command returns the command that resumes the session.
func (r Registry) Command(session Session) (Command, error) {
	adapter, ok := r[session.Agent]
	if !ok {
		return Command{}, fmt.Errorf("no adapter for agent %q", session.Agent)
	}
	return adapter.Command(session)
}

type Context struct {
	Project Project
	All     bool
}
