# cc-statusline

A custom status line for [Claude Code](https://docs.anthropic.com/en/docs/claude-code), built in Go.

Displays model name, context window usage, git branch status, and rate limit info — all in a single high-contrast line optimized for dark terminals.

## Features

- **Model name** in bold cyan, compacted (e.g. `Opus 4.7 (1M context)` → `Opus 4.7·1M`)
- **Context bar** — color-coded fill (green → cyan → yellow → red) with percentage and **tokens remaining** (e.g. `185k left`); 1M vs 200k window auto-detected from model name
- **Git info** — current branch, staged/modified/untracked counts
- **Rate limits** — 5-hour and 7-day usage (color shifts at 50% and 80%)

## Install

```bash
go build -o cc-statusline .
cp cc-statusline ~/.claude/
```

Then configure Claude Code to use it as the status line command:

```json
// ~/.claude/settings.json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/cc-statusline"
  }
}
```

## How it works

Claude Code pipes session JSON to stdin. The binary parses it, formats a single ANSI-colored line, and writes it to stdout.
