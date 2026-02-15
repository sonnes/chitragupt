// Package terminal renders transcripts as ANSI-colored turn cards.
package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/sonnes/chitragupt/core"
)

const defaultWidth = 100

// Renderer pretty-prints a transcript as turn cards to the terminal.
type Renderer struct {
	// Width overrides terminal width detection. Zero means auto-detect.
	Width int
}

// New creates a terminal Renderer.
func New() *Renderer {
	return &Renderer{}
}

// Render writes the transcript as ANSI-colored turn cards to w.
func (r *Renderer) Render(w io.Writer, t *core.Transcript) error {
	width := r.termWidth()
	contentWidth := width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	writeHeader(w, t)

	turns := core.GroupTurns(t.Messages)
	var prevTimestamp *time.Time

	for _, turn := range turns {
		var ts *time.Time
		if turn.UserMessage != nil {
			ts = turn.UserMessage.Timestamp
		} else if len(turn.AssistantMessages) > 0 {
			ts = turn.AssistantMessages[0].Timestamp
		}

		var duration string
		if ts != nil && prevTimestamp != nil {
			duration = formatDuration(ts.Sub(*prevTimestamp))
		}
		if ts != nil {
			prevTimestamp = ts
		}

		writeTurn(w, turn, duration, contentWidth, width)
	}

	fmt.Fprintln(w)
	return nil
}

func (r *Renderer) termWidth() int {
	if r.Width > 0 {
		return r.Width
	}
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		return w
	}
	return defaultWidth
}

// writeHeader renders the session metadata block.
func writeHeader(w io.Writer, t *core.Transcript) {
	// Row 1: Title + diff stats
	title := t.Title
	if title == "" && t.SessionID != "" {
		title = "Session " + t.SessionID
	}
	row1 := styleTitle.Render(title)
	if t.DiffStats != nil {
		var stats []string
		if t.DiffStats.Added > 0 {
			stats = append(stats, styleAdded.Render(fmt.Sprintf("+%s", formatNumber(t.DiffStats.Added))))
		}
		if t.DiffStats.Changed > 0 {
			stats = append(stats, styleChanged.Render(fmt.Sprintf("~%s", formatNumber(t.DiffStats.Changed))))
		}
		if t.DiffStats.Removed > 0 {
			stats = append(stats, styleRemoved.Render(fmt.Sprintf("-%s", formatNumber(t.DiffStats.Removed))))
		}
		if len(stats) > 0 {
			row1 += "  " + strings.Join(stats, " ")
		}
	}
	fmt.Fprintln(w, row1)

	// Row 2: @author  relative_time  model  dir(branch)
	var parts []string
	if t.Author != "" {
		parts = append(parts, "@"+t.Author)
	} else if t.Agent != "" {
		parts = append(parts, "@"+t.Agent)
	}
	if !t.CreatedAt.IsZero() {
		parts = append(parts, core.RelativeTime(t.CreatedAt))
	}
	if t.Model != "" {
		parts = append(parts, t.Model)
	}
	if t.Dir != "" {
		dir := t.Dir
		if t.GitBranch != "" {
			dir += "(" + t.GitBranch + ")"
		}
		parts = append(parts, dir)
	}
	if len(parts) > 0 {
		fmt.Fprintln(w, styleMeta.Render(strings.Join(parts, "  ")))
	}

	// Usage stats
	if t.Usage != nil {
		fmt.Fprintln(w)
		writeUsage(w, t.Usage)
	}
}

// writeUsage renders token counters in two rows: values then labels.
func writeUsage(w io.Writer, u *core.Usage) {
	type stat struct {
		value int
		label string
	}
	stats := []stat{
		{u.InputTokens, "INPUT"},
		{u.OutputTokens, "OUTPUT"},
	}
	if u.CacheReadTokens > 0 {
		stats = append(stats, stat{u.CacheReadTokens, "CACHE READ"})
	}
	if u.CacheCreationTokens > 0 {
		stats = append(stats, stat{u.CacheCreationTokens, "CACHE WRITE"})
	}

	var values, labels []string
	for _, s := range stats {
		formatted := formatNumber(s.value)
		colWidth := max(len(formatted), len(s.label))
		values = append(values, fmt.Sprintf("%*s", colWidth, formatted))
		labels = append(labels, fmt.Sprintf("%-*s", colWidth, s.label))
	}

	fmt.Fprintln(w, "  "+styleStat.Render(strings.Join(values, "    ")))
	fmt.Fprintln(w, "  "+styleStatLabel.Render(strings.Join(labels, "    ")))
}

// writeSeparator renders a horizontal rule.
func writeSeparator(w io.Writer, width int) {
	n := min(width, 72)
	fmt.Fprintln(w)
	fmt.Fprintln(w, styleSeparator.Render(strings.Repeat("─", n)))
}

// writeTurn renders a full turn: user prompt, steps summary, and response.
func writeTurn(w io.Writer, turn core.Turn, duration string, contentWidth, width int) {
	// User message.
	if turn.UserMessage != nil {
		writeSeparator(w, width)

		header := styleUserBadge.Render("USER")
		var metaParts []string
		if turn.UserMessage.Timestamp != nil {
			metaParts = append(metaParts, formatTime(*turn.UserMessage.Timestamp))
		}
		if duration != "" {
			metaParts = append(metaParts, duration)
		}
		if len(metaParts) > 0 {
			header += "    " + styleMeta.Render(strings.Join(metaParts, "    "))
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, " "+header)

		for _, b := range turn.UserMessage.Content {
			if b.Type == core.BlockText {
				text := core.CleanUserText(b.Text)
				if text != "" {
					fmt.Fprintln(w, "  "+truncate(text, contentWidth))
				}
			}
		}
	}

	steps, response := turn.SplitContent()
	stepCount := turn.StepCount()

	// Steps summary line.
	if stepCount > 0 {
		var toolNames []string
		for _, b := range steps {
			if b.Type == core.BlockToolUse {
				toolNames = append(toolNames, summarizeToolUse(b))
			}
		}

		writeSeparator(w, width)
		fmt.Fprintln(w)

		label := fmt.Sprintf("  %d steps", stepCount)
		fmt.Fprintln(w, styleAssistantBadge.Render(label))

		for _, tl := range toolNames {
			fmt.Fprintln(w, "  "+styleToolDetail.Render(tl))
		}
	}

	// Response.
	if len(response) > 0 {
		if stepCount == 0 {
			writeSeparator(w, width)
			fmt.Fprintln(w)
			fmt.Fprintln(w, " "+styleAssistantBadge.Render("ASSISTANT"))
		}
		for _, b := range response {
			if b.Type == core.BlockText {
				text := strings.TrimSpace(b.Text)
				if text != "" {
					fmt.Fprintln(w, "  "+truncate(text, contentWidth))
				}
			}
		}
	}
}

// truncate shortens text to maxWidth, appending "..." if needed.
// Multi-line text is reduced to the first line.
func truncate(s string, maxWidth int) string {
	if maxWidth < 4 {
		maxWidth = 4
	}
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)

	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+3 > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

// Format helpers — mirrored from render/html/funcmap.go.

func formatTime(t time.Time) string {
	return t.Format("Jan 2, 2006 3:04 PM")
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	case m > 0 && s > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return formatNumber(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
}
