package terminal

import (
	"fmt"
	"strings"

	"github.com/sonnes/chitragupt/core"
)

// summarizeToolUse produces a compact one-liner like "[bash: git status]".
func summarizeToolUse(block core.ContentBlock) string {
	name := strings.ToLower(block.Name)
	summary := extractToolSummary(name, block.Input)
	if summary == "" {
		return fmt.Sprintf("[%s]", name)
	}
	return fmt.Sprintf("[%s: %s]", name, summary)
}

// extractToolSummary extracts the most relevant field from the tool input.
func extractToolSummary(name string, input any) string {
	m, ok := input.(map[string]any)
	if !ok || m == nil {
		return ""
	}

	switch name {
	case "bash":
		return stringField(m, "command")
	case "read":
		return stringField(m, "file_path")
	case "write":
		return stringField(m, "file_path")
	case "edit":
		return stringField(m, "file_path")
	case "glob":
		return stringField(m, "pattern")
	case "grep":
		return stringField(m, "pattern")
	default:
		for _, key := range []string{"command", "file_path", "path", "pattern", "query", "url"} {
			if v := stringField(m, key); v != "" {
				return v
			}
		}
		return ""
	}
}

// stringField safely extracts a string value from a map.
func stringField(m map[string]any, key string) string {
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
