# cc-statusline

A custom status line for [Claude Code](https://docs.anthropic.com/en/docs/claude-code), built in Go.

Displays model, context window usage, git status, reasoning effort, and rate-limit budget — high-contrast, optimized for dark terminals. Two lines: orientation on top, session budget below.

```
Opus 4.8·1M | █████████░░░ 73% 270k left | main +1 ~2
effort:max | 5h:82% · 46m | 7d:91% · Mon 05:17
```

## Features

**Line 1 — orientation**
- **Model name** in bold cyan, compacted (e.g. `Opus 4.8 (1M context)` → `Opus 4.8·1M`)
- **Context bar** — color-coded fill (green → cyan → yellow → red) with percentage and **tokens remaining** (e.g. `185k left`); 1M vs 200k window auto-detected from model name
- **Git info** — current branch, staged/modified/untracked counts

**Line 2 — session budget** (rendered only when there's something to show)
- **Effort** — current reasoning effort (`low`/`medium`/`high`/`xhigh`/`max`), color scaling white → cyan → yellow → magenta → red with intensity. Absent when the model doesn't support the effort parameter.
- **Rate limits** — 5-hour and 7-day usage (color shifts at 50% and 80%), each with its **reset time**: a countdown when under a day (`46m`, `4h11m`), weekday + time otherwise (`Mon 05:17`). Only present for Pro/Max subscribers after the first API response.

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

Claude Code pipes session JSON to stdin. The binary parses it, formats one or two ANSI-colored lines, and writes them to stdout. Every optional field (`effort`, `rate_limits`, each window, `resets_at`) is guarded — the second line is omitted entirely when empty, so an early-session render is a single line and the layout fills in as the session warms up.
