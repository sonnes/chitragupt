package core

import (
	"fmt"
	"strings"
	"time"
)

// ComputeDiffStats walks all tool_use blocks in the transcript and computes
// aggregate line-level diff statistics. It must be called BEFORE compact
// transformation, which mutates tool input strings.
func ComputeDiffStats(t *Transcript) *DiffStats {
	files := make(map[string]bool)
	var added, removed int

	for _, msg := range t.Messages {
		for _, b := range msg.Content {
			if b.Type != BlockToolUse {
				continue
			}
			m, ok := b.Input.(map[string]any)
			if !ok || m == nil {
				continue
			}

			switch strings.ToLower(b.Name) {
			case "write":
				if fp := stringVal(m, "file_path"); fp != "" {
					files[fp] = true
				}
				if content := stringVal(m, "content"); content != "" {
					added += countLines(content)
				}
			case "edit":
				if fp := stringVal(m, "file_path"); fp != "" {
					files[fp] = true
				}
				if old := stringVal(m, "old_string"); old != "" {
					removed += countLines(old)
				}
				if ns := stringVal(m, "new_string"); ns != "" {
					added += countLines(ns)
				}
			}
		}
	}

	if added == 0 && removed == 0 && len(files) == 0 {
		return nil
	}

	return &DiffStats{
		Added:   added,
		Removed: removed,
		Changed: len(files),
	}
}

// RelativeTime formats a time.Time as a human-readable relative string.
func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

func stringVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// countLines returns the number of lines in s.
// An empty string has 0 lines. A string with no newline has 1 line.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n") + 1
	if strings.HasSuffix(s, "\n") {
		n--
	}
	return n
}
