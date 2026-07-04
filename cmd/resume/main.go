package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hmepas/resume/internal/adapters"
	"github.com/hmepas/resume/internal/launcher"
	"github.com/hmepas/resume/internal/resume"
	"github.com/hmepas/resume/internal/tui"
)

var version = "dev"

func main() {
	var (
		all      bool
		jsonOut  bool
		limit    int
		noTUI    bool
		printCmd bool
		showHelp bool
		showVer  bool
	)

	flag.BoolVar(&all, "all", false, "show sessions for all projects")
	flag.BoolVar(&jsonOut, "json", false, "print JSON")
	flag.IntVar(&limit, "limit", 50, "maximum sessions to print")
	flag.BoolVar(&noTUI, "no-interactive", false, "print table instead of interactive picker")
	flag.BoolVar(&printCmd, "print-command", false, "print selected launch command instead of running it")
	flag.BoolVar(&showHelp, "help", false, "show help")
	flag.BoolVar(&showVer, "version", false, "show version")
	flag.Parse()

	if showHelp {
		usage()
		return
	}
	if showVer {
		fmt.Println(version)
		return
	}

	project, err := resume.DetectProject(".")
	if err != nil {
		fatal(err)
	}

	opts := resume.CollectOptions{
		Project: project,
		All:     all,
		Limit:   limit,
	}

	sessions, diagnostics := resume.Collect(context.Background(), adapters.Builtin(), opts)

	if jsonOut {
		out := struct {
			Project     resume.Project      `json:"project"`
			Sessions    []resume.Session    `json:"sessions"`
			Diagnostics []resume.Diagnostic `json:"diagnostics,omitempty"`
		}{Project: project, Sessions: sessions, Diagnostics: diagnostics}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fatal(err)
		}
		return
	}

	if !jsonOut && !noTUI && tui.CanRun(os.Stdin, os.Stdout) && len(sessions) > 0 {
		result, err := tui.Pick(os.Stdin, os.Stdout, sessions)
		if err != nil {
			fatal(err)
		}
		if !result.OK {
			return
		}
		command, err := launcher.ForSession(result.Session)
		if err != nil {
			fatal(err)
		}
		if printCmd {
			fmt.Println(launcher.ShellString(command))
			return
		}
		if err := launcher.Run(command); err != nil {
			fatal(err)
		}
		return
	}

	if err := resume.RenderTable(os.Stdout, sessions, diagnostics, resume.RenderOptions{All: all}); err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  resume [--all] [--json] [--limit N] [--no-interactive] [--print-command]

Shows recent AI coding sessions for the current project across supported agents.

Options:`)
	flag.PrintDefaults()
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "resume: %v\n", err)
	os.Exit(1)
}
