# resume

Find and resume your last AI coding session, even when you forgot which agent you used.

`resume` is a small terminal tool that scans local session history for multiple AI coding agents, filters it to the current project, and opens an fzf-like picker sorted by recent activity.

```sh
cd ~/projects/my-app
resume
```

It answers the boring question that keeps stealing time:

> Was this project last touched in Claude Code, Codex, Cursor, Gemini, Pi, or something else?

## Features

- Cross-agent session discovery for the current Git repository.
- Interactive fuzzy picker with Vim-style navigation and mouse support.
- Native resume launching for supported agents.
- Stable agent colors using your terminal theme palette.
- JSON and table output for scripts.
- Read-only scanning by default; session deletion is explicit and confirmed.
- No daemon, database, telemetry, or cloud sync.
- Single Go binary for macOS and Linux.

## Install

### Homebrew

```sh
brew tap hmepas/resume
brew install resume
```

The Homebrew formula should install prebuilt GitHub Release artifacts by default. Users should not need Go or a compiler unless they explicitly choose a source build.

### curl

```sh
curl -fsSL https://raw.githubusercontent.com/hmepas/resume/main/scripts/install.sh | sh
```

Install a specific release:

```sh
curl -fsSL https://raw.githubusercontent.com/hmepas/resume/main/scripts/install.sh | RESUME_VERSION=v0.1.0 sh
```

Install somewhere else:

```sh
curl -fsSL https://raw.githubusercontent.com/hmepas/resume/main/scripts/install.sh | INSTALL_DIR="$HOME/bin" sh
```

The installer downloads a prebuilt binary from GitHub Releases and verifies it with `checksums.txt` when `sha256sum` or `shasum` is available.

### From Source

```sh
go install github.com/hmepas/resume/cmd/resume@latest
```

## Usage

Run from a project directory:

```sh
resume
```

By default, `resume` opens the interactive picker when stdin/stdout are terminals. Select a session and press Enter to launch the matching native agent command.

Print the command instead of running it:

```sh
resume --print-command
```

Print a table:

```sh
resume --no-interactive
```

Show every discovered session, not just the current project:

```sh
resume --all
```

Machine-readable output:

```sh
resume --json
```

Limit results:

```sh
resume --limit 20
```

Show version:

```sh
resume --version
```

## Interactive Controls

| Key | Action |
| --- | --- |
| `Enter` | launch selected session |
| `/` | enter search mode |
| `Esc` | clear search, then quit |
| `Ctrl-C` | quit |
| `Ctrl-N`, `Ctrl-J`, `j` | next session |
| `Ctrl-P`, `Ctrl-K`, `k` | previous session |
| `d` | delete selected session file, with `y/N` confirmation |
| `Ctrl-D` | delete selected session file, including from search mode |
| mouse click | select and launch |

Russian keyboard fallback outside search mode:

- `т` / `о` behave like next.
- `з` / `л` behave like previous.

## Supported Agents

Current adapters:

| Agent | Discovery | Launch |
| --- | --- | --- |
| Claude Code | `~/.claude/projects`, `~/.claude/sessions` | `claude --resume <id>` |
| Codex | `~/.codex/sessions`, `~/.codex/archived_sessions` | `codex resume <id>` |
| Cursor | workspace storage | `cursor <project>` |
| Gemini CLI | `~/.gemini/tmp` | `gemini` |
| OpenCode | local OpenCode storage | `opencode` |
| Pi | known Pi session locations | `pi` |

Adapter quality varies because each agent stores local history differently. Claude Code and Codex have the richest session metadata today; Cursor, Gemini, OpenCode, and Pi are progressively improving.

## Privacy

`resume` reads local agent metadata and session files. It does not upload anything, start a background service, or write to agent history.

The only mutating action is interactive deletion:

```text
d      delete selected source file after y/N confirmation
Ctrl-D same, also works while searching
```

## Release Artifacts

GitHub Releases should publish:

```text
resume_Darwin_arm64.tar.gz
resume_Darwin_x86_64.tar.gz
resume_Linux_arm64.tar.gz
resume_Linux_x86_64.tar.gz
checksums.txt
```

Build them with:

```sh
VERSION=v0.1.0 scripts/build-release.sh
```

Each tarball contains:

```text
resume
README.md
LICENSE
```

## Homebrew Tap

Recommended tap layout:

```text
homebrew-resume/
  Formula/
    resume.rb
```

Formula skeleton:

```ruby
class Resume < Formula
  desc "Cross-agent AI coding session picker"
  homepage "https://github.com/hmepas/resume"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_arm64.tar.gz"
      sha256 "<darwin-arm64-sha256>"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Darwin_x86_64.tar.gz"
      sha256 "<darwin-x86_64-sha256>"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_arm64.tar.gz"
      sha256 "<linux-arm64-sha256>"
    else
      url "https://github.com/hmepas/resume/releases/download/v#{version}/resume_Linux_x86_64.tar.gz"
      sha256 "<linux-x86_64-sha256>"
    end
  end

  def install
    bin.install "resume"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/resume --version")
  end
end
```

The same template lives in `packaging/homebrew/resume.rb`; copy it into the tap repo and replace the SHA-256 placeholders from `dist/checksums.txt` for each release.

## Development

Run locally:

```sh
go run ./cmd/resume
```

Run tests:

```sh
go test ./...
```

Build:

```sh
go build -o resume ./cmd/resume
```

If your environment cannot write to the default Go build cache:

```sh
GOCACHE="$PWD/.gocache" go test ./...
GOCACHE="$PWD/.gocache" go build -o resume ./cmd/resume
```

## Adapter Architecture

The core stays agent-agnostic:

```go
type Adapter interface {
    ID() string
    Sessions(ctx resume.Context) ([]resume.Session, error)
}
```

Adapters live under `internal/adapters/<agent>` and normalize each agent's local history into:

```text
agent
id
project
started_at
updated_at
title
source_path
resume_hint
confidence
```

Adding a new agent should usually mean:

1. Add `internal/adapters/<agent>`.
2. Implement `Sessions`.
3. Register it in `internal/adapters/builtin.go`.
4. Add focused parser tests with local fixtures or synthetic JSON.

No external runtime plugin system exists yet. That is intentional: local agent storage formats are still changing quickly, and built-in adapters are easier to test and ship safely.

## Non-Goals

- Windows support.
- Uploading or syncing session history.
- Replacing native agent CLIs.
- Full transcript search.
- Writing back into agent state.

## Status

Early but usable. The most important work before a polished public release is adding more fixture coverage for real agent storage formats and keeping adapter behavior aligned with the agents' own resume UIs.
