package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"
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
	Effort struct {
		Level string `json:"level"`
	} `json:"effort"`
	RateLimits struct {
		FiveHour struct {
			UsedPercentage *float64 `json:"used_percentage"`
			ResetsAt       *int64   `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage *float64 `json:"used_percentage"`
			ResetsAt       *int64   `json:"resets_at"`
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

func trimModel(name string) string {
	i := strings.Index(name, " (")
	if i == -1 {
		return name
	}
	suffix := strings.TrimSuffix(name[i+2:], " context)")
	suffix = strings.TrimSuffix(suffix, ")")
	return name[:i] + "·" + suffix
}

func contextWindowSize(modelName string) int {
	if strings.Contains(modelName, "1M") {
		return 1_000_000
	}
	return 200_000
}

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%dk", n/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// formatReset turns a Unix-epoch reset time into a glanceable string,
// adaptive to how far away it is: a countdown when under a day (the
// actionable number for the 5h window), a weekday + time otherwise
// (glanceable for the weekly cap).
func formatReset(epoch int64, now time.Time) string {
	d := time.Unix(epoch, 0).Sub(now)
	if d <= 0 {
		return "now"
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if h > 0 {
			return fmt.Sprintf("%dh%02dm", h, m)
		}
		return fmt.Sprintf("%dm", m)
	}
	return time.Unix(epoch, 0).Format("Mon 15:04")
}

func effortBadge(level string) string {
	if level == "" {
		return ""
	}
	var color string
	switch level {
	case "low":
		color = white
	case "medium":
		color = cyan
	case "high":
		color = yellow
	case "xhigh":
		color = magenta
	case "max":
		color = red
	default:
		color = white
	}
	return fmt.Sprintf("%seffort:%s%s", color+bold, level, reset)
}

func rateLimitBadge(label string, pct float64, resetsAt *int64, now time.Time) string {
	color := white
	if pct >= 80 {
		color = red + bold
	} else if pct >= 50 {
		color = yellow
	}
	badge := fmt.Sprintf("%s%s:%d%%%s", color, label, int(pct), reset)
	if resetsAt != nil {
		badge += white + " · " + reset + cyan + formatReset(*resetsAt, now) + reset
	}
	return badge
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

	rawModel := s.Model.DisplayName
	model := trimModel(rawModel)
	if model == "" {
		model = "Claude"
	}

	pct := s.ContextWindow.UsedPercentage
	total := contextWindowSize(rawModel)
	remaining := int(float64(total) * (100 - pct) / 100)

	sep := white + " | " + reset

	// Line 1 — orientation: model, context budget, git state.
	line1 := []string{
		cyan + bold + model + reset,
		contextBar(pct) + " " + white + formatTokens(remaining) + " left" + reset,
	}
	if git := gitInfo(s.Workspace.CurrentDir); git != "" {
		line1 = append(line1, git)
	}

	// Line 2 — session config + rate-limit budget with reset times.
	now := time.Now()
	var line2 []string
	if eb := effortBadge(s.Effort.Level); eb != "" {
		line2 = append(line2, eb)
	}
	if h := s.RateLimits.FiveHour.UsedPercentage; h != nil {
		line2 = append(line2, rateLimitBadge("5h", *h, s.RateLimits.FiveHour.ResetsAt, now))
	}
	if d := s.RateLimits.SevenDay.UsedPercentage; d != nil {
		line2 = append(line2, rateLimitBadge("7d", *d, s.RateLimits.SevenDay.ResetsAt, now))
	}

	out := strings.Join(line1, sep)
	if len(line2) > 0 {
		out += "\n" + strings.Join(line2, sep)
	}
	fmt.Print(out)
}
