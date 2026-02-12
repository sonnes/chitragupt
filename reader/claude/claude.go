// Package claude reads Claude Code session logs (JSONL in ~/.claude/projects/).
package claude

// Reader reads Claude Code JSONL session files.
type Reader struct {
	// Dir overrides the default session directory (~/.claude/projects/).
	Dir string
}
