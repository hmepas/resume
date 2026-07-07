package tui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/hmepas/resume/internal/resume"
)

type PickResult struct {
	Session resume.Session
	OK      bool
}

type Picker struct {
	in       *os.File
	out      *os.File
	sessions []resume.Session
	filtered []resume.Session
	query    string
	search   bool
	selected int
	offset   int
	width    int
	height   int
	topLine  int
	status   string
	confirm  *resume.Session
}

var runOpenCodeSessionDelete = func(sessionID string) error {
	cmd := exec.Command("opencode", "session", "delete", sessionID)
	return cmd.Run()
}

func CanRun(in, out *os.File) bool {
	return isTerminal(in) && isTerminal(out)
}

func Pick(in, out *os.File, sessions []resume.Session) (PickResult, error) {
	if len(sessions) == 0 {
		return PickResult{}, nil
	}

	raw, err := makeRaw(in)
	if err != nil {
		return PickResult{}, err
	}
	defer raw.restore()

	p := &Picker{
		in:       in,
		out:      out,
		sessions: sessions,
		filtered: sessions,
	}

	fmt.Fprint(out, "\x1b[?1049h\x1b[?25l\x1b[?1000h\x1b[?1006h")
	defer fmt.Fprint(out, "\x1b[?1006l\x1b[?1000l\x1b[?25h\x1b[?1049l")

	p.render()
	buf := make([]byte, 64)
	for {
		n, err := in.Read(buf)
		if err != nil {
			return PickResult{}, err
		}
		if n == 0 {
			continue
		}
		action := p.handle(buf[:n])
		switch action {
		case "render":
			p.render()
		case "accept":
			if len(p.filtered) == 0 {
				p.render()
				continue
			}
			return PickResult{Session: p.filtered[p.selected], OK: true}, nil
		case "cancel":
			return PickResult{}, nil
		}
	}
}

func (p *Picker) handle(data []byte) string {
	if len(data) > 1 && !strings.HasPrefix(string(data), "\x1b[") {
		if _, size := utf8.DecodeRune(data); size == len(data) {
			goto single
		}
		for len(data) > 0 {
			r, size := utf8.DecodeRune(data)
			if r == utf8.RuneError && size == 1 {
				size = 1
			}
			action := p.handle(data[:size])
			if action == "accept" || action == "cancel" {
				return action
			}
			data = data[size:]
		}
		return "render"
	}

single:
	if p.confirm != nil {
		return p.handleConfirm(data)
	}

	if len(data) == 1 {
		switch data[0] {
		case 3, 27:
			if p.search || p.query != "" {
				p.search = false
				p.query = ""
				p.refilter()
				return "render"
			}
			return "cancel"
		case 4:
			p.askDelete()
			return "render"
		case 10:
			p.move(1)
			return "render"
		case 11:
			p.move(-1)
			return "render"
		case 13:
			p.search = false
			return "accept"
		case 14:
			p.move(1)
			return "render"
		case 16:
			p.move(-1)
			return "render"
		case 21:
			p.query = ""
			p.refilter()
			return "render"
		case 127, 8:
			if p.search {
				p.backspace()
			}
			return "render"
		case '/':
			p.search = true
			p.status = ""
			return "render"
		case 'j':
			if p.search {
				p.query += "j"
				p.refilter()
			} else {
				p.move(1)
			}
			return "render"
		case 'k':
			if p.search {
				p.query += "k"
				p.refilter()
			} else {
				p.move(-1)
			}
			return "render"
		case 'd':
			if p.search {
				p.query += "d"
				p.refilter()
			} else {
				p.askDelete()
			}
			return "render"
		}
	}

	if strings.HasPrefix(string(data), "\x1b[") {
		return p.handleEscape(string(data))
	}

	text := string(data)
	if !p.search {
		switch text {
		case "т", "Т", "о", "О":
			p.move(1)
		case "з", "З", "л", "Л":
			p.move(-1)
		}
		return "render"
	}
	switch text {
	case "т", "Т", "о", "О":
		p.query += text
		p.refilter()
	case "з", "З", "л", "Л":
		p.query += text
		p.refilter()
	default:
		if utf8.Valid(data) {
			p.query += text
			p.refilter()
		}
	}
	return "render"
}

func (p *Picker) handleConfirm(data []byte) string {
	text := string(data)
	if len(data) == 1 {
		switch data[0] {
		case 3, 27:
			p.confirm = nil
			p.status = "delete canceled"
			return "render"
		case 'y', 'Y':
			p.deleteConfirmed()
			return "render"
		case 'n', 'N':
			p.confirm = nil
			p.status = "delete canceled"
			return "render"
		}
	}
	switch text {
	case "н", "Н":
		p.deleteConfirmed()
	case "т", "Т":
		p.confirm = nil
		p.status = "delete canceled"
	default:
		p.status = "delete? press y/n"
	}
	return "render"
}

func (p *Picker) handleEscape(seq string) string {
	switch seq {
	case "\x1b[A":
		p.move(-1)
		return "render"
	case "\x1b[B":
		p.move(1)
		return "render"
	}

	if strings.HasPrefix(seq, "\x1b[<") {
		var button, col, row int
		var suffix byte
		if _, err := fmt.Sscanf(seq, "\x1b[<%d;%d;%d%c", &button, &col, &row, &suffix); err == nil && suffix == 'M' {
			index := p.offset + row - p.topLine
			if index >= 0 && index < len(p.filtered) {
				p.selected = index
				return "accept"
			}
		}
	}
	return "render"
}

func (p *Picker) move(delta int) {
	if len(p.filtered) == 0 {
		p.selected = 0
		p.offset = 0
		return
	}
	p.selected += delta
	if p.selected < 0 {
		p.selected = len(p.filtered) - 1
	}
	if p.selected >= len(p.filtered) {
		p.selected = 0
	}
	p.ensureVisible()
}

func (p *Picker) backspace() {
	if p.query == "" {
		return
	}
	_, size := utf8.DecodeLastRuneInString(p.query)
	p.query = p.query[:len(p.query)-size]
	p.refilter()
}

func (p *Picker) askDelete() {
	if len(p.filtered) == 0 {
		return
	}
	session := p.filtered[p.selected]
	if session.Agent == "opencode" {
		if session.ID == "" {
			p.status = "cannot delete: OpenCode session id is empty"
			return
		}
		p.confirm = &session
		p.status = "delete opencode session? y/N"
		return
	}
	if session.SourcePath == "" {
		p.status = "cannot delete: source path is empty"
		return
	}
	p.confirm = &session
	p.status = fmt.Sprintf("delete %s session file? y/N", session.Agent)
}

func (p *Picker) deleteConfirmed() {
	session := *p.confirm
	p.confirm = nil
	if err := deleteSession(session); err != nil {
		p.status = "delete failed: " + err.Error()
		return
	}

	p.sessions = removeSession(p.sessions, session)
	p.filtered = filterSessions(p.sessions, p.query)
	if p.selected >= len(p.filtered) {
		p.selected = len(p.filtered) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
	p.ensureVisible()
	p.status = "deleted " + session.Agent + " session"
}

func deleteSession(session resume.Session) error {
	if session.Agent == "opencode" {
		return runOpenCodeSessionDelete(session.ID)
	}
	return os.Remove(session.SourcePath)
}

func removeSession(sessions []resume.Session, deleted resume.Session) []resume.Session {
	out := sessions[:0]
	for _, session := range sessions {
		if sameSession(session, deleted) {
			continue
		}
		out = append(out, session)
	}
	return out
}

func sameSession(a, b resume.Session) bool {
	if a.Agent != b.Agent || a.SourcePath != b.SourcePath {
		return false
	}
	return b.ID == "" || a.ID == b.ID
}

func (p *Picker) refilter() {
	p.filtered = filterSessions(p.sessions, p.query)
	p.selected = 0
	p.offset = 0
	if p.query == "" {
		p.search = false
	}
}

func (p *Picker) ensureVisible() {
	visible := p.visibleRows()
	if p.selected < p.offset {
		p.offset = p.selected
	}
	if p.selected >= p.offset+visible {
		p.offset = p.selected - visible + 1
	}
	if p.offset < 0 {
		p.offset = 0
	}
}

func (p *Picker) render() {
	p.width, p.height = terminalSize(p.out)
	p.ensureVisible()

	var b strings.Builder
	b.WriteString("\x1b[H\x1b[2J")
	p.writeLine(&b, bold(fmt.Sprintf("resume"))+"  "+fmt.Sprintf("%d/%d  ", len(p.filtered), len(p.sessions))+p.queryText())
	help := "enter run  / search  esc clear/quit  ^n/^p ^j/^k j/k  d/^d del  click"
	if p.confirm != nil {
		help = "confirm delete: y yes / n no"
	}
	p.writeLine(&b, dim(help))
	if p.status != "" {
		p.writeLine(&b, p.status)
	} else {
		p.writeLine(&b, "")
	}

	p.topLine = 4
	visible := p.visibleRows()
	end := p.offset + visible
	if end > len(p.filtered) {
		end = len(p.filtered)
	}
	if len(p.filtered) == 0 {
		p.writeLine(&b, dim("No matching sessions"))
	} else {
		for i := p.offset; i < end; i++ {
			p.renderRow(&b, i, p.filtered[i])
		}
	}
	io.WriteString(p.out, b.String())
}

func (p *Picker) queryText() string {
	if !p.search && p.query == "" {
		return dim("/ to search")
	}
	return "/ " + p.query
}

func (p *Picker) renderRow(b *strings.Builder, index int, session resume.Session) {
	prefix := "  "
	if index == p.selected {
		prefix = "> "
	}
	timeCol := session.UpdatedAt.Local().Format("2006-01-02 15:04")
	agentCol := fmt.Sprintf("%-8s", truncate(session.Agent, 8))
	idCol := shortID(session.ID)
	metaWidth := runeWidth(prefix) + 16 + 2 + 8 + 2 + runeWidth(idCol) + 2
	titleWidth := p.width - metaWidth
	if titleWidth < 0 {
		titleWidth = 0
	}
	title := oneLine(session.Title, titleWidth)
	if index == p.selected {
		b.WriteString("\x1b[7m")
		fmt.Fprintf(b, "%s%s  %s  %-8s  %s",
			prefix,
			timeCol,
			colorAgent(agentCol, session.Agent, true),
			idCol,
			title,
		)
		b.WriteString("\x1b[0m")
	} else {
		fmt.Fprintf(b, "%s%s  %s  %-8s  %s",
			prefix,
			timeCol,
			colorAgent(agentCol, session.Agent, false),
			idCol,
			title,
		)
	}
	b.WriteString("\x1b[K\r\n")
}

func (p *Picker) visibleRows() int {
	rows := p.height - 4
	if rows < 1 {
		return 1
	}
	return rows
}

func (p *Picker) writeLine(b *strings.Builder, text string) {
	if strings.Contains(text, "\x1b[") {
		b.WriteString(text)
	} else {
		b.WriteString(truncate(text, max(1, p.width-1)))
	}
	b.WriteString("\x1b[K\r\n")
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		s = "(untitled)"
	}
	return truncate(s, max)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}

func truncateLeft(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return "…" + string(runes[len(runes)-max+1:])
}

func shortID(id string) string {
	if id == "" {
		return "-"
	}
	runes := []rune(id)
	if len(runes) <= 8 {
		return id
	}
	return string(runes[:3]) + ".." + string(runes[len(runes)-4:])
}

func runeWidth(s string) int {
	return utf8.RuneCountInString(s)
}

func dim(s string) string {
	return "\x1b[2m" + s + "\x1b[0m"
}

func bold(s string) string {
	return "\x1b[1m" + s + "\x1b[0m"
}

func colorAgent(text, agent string, selected bool) string {
	code := agentColorCode(agent)
	if code == "" {
		return text
	}
	if selected {
		return "\x1b[7;" + code + "m" + text + "\x1b[7m"
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func agentColorCode(agent string) string {
	switch strings.ToLower(agent) {
	case "claude":
		return "31"
	case "codex":
		return "32"
	case "cursor":
		return "34"
	case "gemini":
		return "35"
	case "pi":
		return "36"
	case "opencode":
		return "33"
	default:
		return ""
	}
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
