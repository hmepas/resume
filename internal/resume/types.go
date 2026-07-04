package resume

import "time"

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

type Adapter interface {
	ID() string
	Sessions(ctx Context) ([]Session, error)
}

type Context struct {
	Project Project
	All     bool
}
