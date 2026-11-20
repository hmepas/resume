package resume

import "strings"

type Command struct {
	Name string
	Args []string
	Dir  string
}

// String returns the shell-quoted command line without the directory.
func (c Command) String() string {
	parts := make([]string, 0, len(c.Args)+1)
	parts = append(parts, quote(c.Name))
	for _, arg := range c.Args {
		parts = append(parts, quote(arg))
	}
	return strings.Join(parts, " ")
}

// ShellString returns the command line prefixed with "cd <dir> && " when Dir
// is set.
func (c Command) ShellString() string {
	if c.Dir != "" {
		return "cd " + quote(c.Dir) + " && " + c.String()
	}
	return c.String()
}

func quote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool {
		return !(r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == '+' ||
			r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
	}) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
