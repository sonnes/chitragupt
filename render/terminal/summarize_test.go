package terminal

import (
	"testing"

	"github.com/sonnes/chitragupt/core"
	"github.com/stretchr/testify/assert"
)

func TestSummarizeToolUse(t *testing.T) {
	tests := []struct {
		name   string
		block  core.ContentBlock
		expect string
	}{
		{
			name:   "bash command",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "Bash", Input: map[string]any{"command": "git status"}},
			expect: "[bash: git status]",
		},
		{
			name:   "read file",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "Read", Input: map[string]any{"file_path": "/tmp/test.go"}},
			expect: "[read: /tmp/test.go]",
		},
		{
			name:   "write file",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "Write", Input: map[string]any{"file_path": "/tmp/out.go"}},
			expect: "[write: /tmp/out.go]",
		},
		{
			name:   "edit file",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "Edit", Input: map[string]any{"file_path": "/tmp/x.go", "old_string": "foo"}},
			expect: "[edit: /tmp/x.go]",
		},
		{
			name:   "glob pattern",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "Glob", Input: map[string]any{"pattern": "**/*.go"}},
			expect: "[glob: **/*.go]",
		},
		{
			name:   "grep pattern",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "Grep", Input: map[string]any{"pattern": "func main"}},
			expect: "[grep: func main]",
		},
		{
			name:   "nil input",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "TodoRead", Input: nil},
			expect: "[todoread]",
		},
		{
			name:   "unknown tool with fallback key",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "WebSearch", Input: map[string]any{"query": "golang testing"}},
			expect: "[websearch: golang testing]",
		},
		{
			name:   "unknown tool no matching keys",
			block:  core.ContentBlock{Type: core.BlockToolUse, Name: "Custom", Input: map[string]any{"foo": "bar"}},
			expect: "[custom]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeToolUse(tt.block)
			assert.Equal(t, tt.expect, got)
		})
	}
}
