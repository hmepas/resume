package launcher

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hmepas/resume/internal/resume"
)

func Run(command resume.Command) error {
	if _, err := exec.LookPath(command.Name); err != nil {
		return fmt.Errorf("%s not found in PATH", command.Name)
	}

	cmd := exec.Command(command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
