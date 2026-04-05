package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"
)

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
)

type SessionData struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage float64 `json:"used_percentage"`
	} `json:"context_window"`
	Cost struct {
		TotalDurationMs float64 `json:"total_duration_ms"`
	} `json:"cost"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	RateLimits struct {
		FiveHour struct {
			UsedPercentage *float64 `json:"used_percentage"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage *float64 `json:"used_percentage"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
}

func contextBar(pct float64) string {
	width := 12
	filled := int(math.Round(pct / 100.0 * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	var color string
	switch {
	case pct >= 90:
		color = red
	case pct >= 70:
		color = yellow
	case pct >= 40:
		color = cyan
	default:
		color = green
	}

	bar := color + strings.Repeat("█", filled) + white + strings.Repeat("░", empty) + reset
	return fmt.Sprintf("%s %s%d%%%s", bar, color+bold, int(pct), reset)
}

func gitInfo(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		cmd = exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD")
		out, err = cmd.Output()
		if err != nil {
			return ""
		}
		branch = strings.TrimSpace(string(out))
	}

	cmd = exec.Command("git", "-C", dir, "status", "--porcelain")
	out, err = cmd.Output()
	if err != nil {
		return magenta + branch + reset
	}

	staged, modified, untracked := 0, 0, 0
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		if x == '?' {
			untracked++
		} else {
			if x != ' ' && x != '?' {
				staged++
			}
			if y != ' ' && y != '?' {
				modified++
			}
		}
	}

	parts := []string{magenta + branch + reset}
	if staged > 0 {
		parts = append(parts, fmt.Sprintf("%s+%d%s", green+bold, staged, reset))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%s~%d%s", yellow+bold, modified, reset))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%s?%d%s", white, untracked, reset))
	}

	return strings.Join(parts, " ")
}

func rateLimitBadge(label string, pct float64) string {
	color := white
	if pct >= 80 {
		color = red + bold
	} else if pct >= 50 {
		color = yellow
	}
	return fmt.Sprintf("%s%s:%d%%%s", color, label, int(pct), reset)
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}

	var s SessionData
	if err := json.Unmarshal(data, &s); err != nil {
		return
	}

	model := s.Model.DisplayName
	if model == "" {
		model = "Claude"
	}

	parts := []string{
		cyan + bold + model + reset,
		contextBar(s.ContextWindow.UsedPercentage),
	}

	git := gitInfo(s.Workspace.CurrentDir)
	if git != "" {
		parts = append(parts, git)
	}

	if h := s.RateLimits.FiveHour.UsedPercentage; h != nil && *h > 0 {
		parts = append(parts, rateLimitBadge("5h", *h))
	}
	if d := s.RateLimits.SevenDay.UsedPercentage; d != nil && *d > 0 {
		parts = append(parts, rateLimitBadge("7d", *d))
	}

	fmt.Print(strings.Join(parts, white+" | "+reset))
}
