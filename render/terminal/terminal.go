// Package terminal renders transcripts as ANSI-colored message cards.
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

// Renderer pretty-prints a transcript as message cards to the terminal.
type Renderer struct {
	// Width overrides terminal width detection. Zero means auto-detect.
	Width int
}

// New creates a terminal Renderer.
func New() *Renderer {
	return &Renderer{}
}

// Render writes the transcript as ANSI-colored message cards to w.
func (r *Renderer) Render(w io.Writer, t *core.Transcript) error {
	width := r.termWidth()

	writeHeader(w, t)

	// Build tool_result index: tool_use_id → tool_result block.
	consumed := make(map[string]bool)
	for _, msg := range t.Messages {
		for _, b := range msg.Content {
			if b.Type == core.BlockToolResult && b.ToolUseID != "" {
				// Pre-mark results that will be consumed by their tool_use.
				consumed[b.ToolUseID] = false
			}
		}
	}

	var prevTimestamp *time.Time

	for _, msg := range t.Messages {
		var duration string
		if msg.Timestamp != nil && prevTimestamp != nil {
			duration = formatDuration(msg.Timestamp.Sub(*prevTimestamp))
		}
		if msg.Timestamp != nil {
			prevTimestamp = msg.Timestamp
		}

		writeMessage(w, msg, duration, consumed, width)
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

// writeMessage renders a single message card: role badge, metadata, content blocks.
func writeMessage(w io.Writer, msg core.Message, duration string, consumed map[string]bool, width int) bool {
	contentWidth := width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	var lines []string
	for _, b := range msg.Content {
		switch b.Type {
		case core.BlockText:
			text := strings.TrimSpace(b.Text)
			if text != "" {
				lines = append(lines, truncate(text, contentWidth))
			}
		case core.BlockThinking:
			lines = append(lines, styleThinking.Render("▸ Thinking..."))
		case core.BlockToolUse:
			if b.ToolUseID != "" {
				consumed[b.ToolUseID] = true
			}
			name := b.Name
			if name == "" {
				name = "tool"
			}
			summary := extractToolSummary(strings.ToLower(name), b.Input)
			toolLine := styleToolName.Render("⚙ " + name)
			if summary != "" {
				nameWidth := lipgloss.Width("⚙ " + name + "  ")
				toolLine += "  " + styleToolDetail.Render(truncate(summary, contentWidth-nameWidth))
			}
			lines = append(lines, toolLine)
			if b.SubAgentRef != nil {
				label := b.SubAgentRef.AgentID
				if b.SubAgentRef.AgentName != "" {
					label = b.SubAgentRef.AgentName
				}
				subLine := "  → " + label
				if b.SubAgentRef.AgentType != "" {
					subLine += " (" + b.SubAgentRef.AgentType + ")"
				}
				lines = append(lines, styleToolDetail.Render(subLine))
			}
		case core.BlockToolResult:
			if consumed[b.ToolUseID] {
				continue
			}
			lines = append(lines, styleToolDetail.Render(truncate(b.Content, contentWidth)))
		}
	}

	if len(lines) == 0 {
		return false
	}

	writeSeparator(w, width)

	badge := roleBadge(msg.Role)
	var metaParts []string
	if msg.Timestamp != nil {
		metaParts = append(metaParts, formatTime(*msg.Timestamp))
	}
	if duration != "" {
		metaParts = append(metaParts, duration)
	}
	header := badge
	if len(metaParts) > 0 {
		header += "    " + styleMeta.Render(strings.Join(metaParts, "    "))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, " "+header)

	for _, line := range lines {
		fmt.Fprintln(w, "  "+line)
	}

	return true
}

func roleBadge(role core.Role) string {
	label := strings.ToUpper(string(role))
	switch role {
	case core.RoleUser:
		return styleUserBadge.Render(label)
	case core.RoleAssistant:
		return styleAssistantBadge.Render(label)
	case core.RoleSystem:
		return styleSystemBadge.Render(label)
	default:
		return styleMeta.Render(label)
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
