package compact

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sonnes/chitragupt/core"
	"github.com/sonnes/chitragupt/reader/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readTestdata(t *testing.T, name string) *core.Transcript {
	t.Helper()
	r := &claude.Reader{}
	tr, err := r.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return tr
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line no newline", "hello", 1},
		{"single line with newline", "hello\n", 1},
		{"multiple lines", "a\nb\nc", 3},
		{"multiple lines trailing newline", "a\nb\nc\n", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, countLines(tt.input))
		})
	}
}

func TestLineSummary(t *testing.T) {
	tests := []struct {
		name  string
		label string
		input string
		want  string
	}{
		{"empty", "output", "", "[output: 0 lines]"},
		{"single line", "output", "hello", "[output: 1 line]"},
		{"multiple lines", "output", "a\nb\nc", "[output: 3 lines]"},
		{"error label", "error", "a\nb", "[error: 2 lines]"},
		{"field label", "content", "a\nb\nc\nd\n", "[content: 4 lines]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, lineSummary(tt.label, tt.input))
		})
	}
}

func TestFilterThinking(t *testing.T) {
	blocks := []core.ContentBlock{
		{Type: core.BlockThinking, Text: "reasoning"},
		{Type: core.BlockText, Text: "visible"},
		{Type: core.BlockToolUse, Name: "Bash"},
	}

	filtered := filterThinking(blocks)
	require.Len(t, filtered, 2)
	assert.Equal(t, core.BlockText, filtered[0].Type)
	assert.Equal(t, core.BlockToolUse, filtered[1].Type)
}

func TestFilterThinkingNoop(t *testing.T) {
	blocks := []core.ContentBlock{
		{Type: core.BlockText, Text: "hello"},
	}
	filtered := filterThinking(blocks)
	require.Len(t, filtered, 1)
	assert.Equal(t, "hello", filtered[0].Text)
}

func TestCompactToolResult(t *testing.T) {
	longContent := strings.Repeat("line\n", 50)

	tests := []struct {
		name    string
		content string
		isError bool
		want    string
	}{
		{"success output", longContent, false, "[output: 50 lines]"},
		{"error output", longContent, true, "[error: 50 lines]"},
		{"short output", "ok\n", false, "[output: 1 line]"},
		{"empty output", "", false, "[output: 0 lines]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &core.Transcript{
				SessionID: "test",
				Agent:     "claude",
				CreatedAt: time.Now(),
				Messages: []core.Message{{
					Role: core.RoleUser,
					Content: []core.ContentBlock{
						{Type: core.BlockToolResult, Content: tt.content, IsError: tt.isError},
					},
				}},
			}

			c := New(Config{})
			require.NoError(t, c.Transform(tr))
			assert.Equal(t, tt.want, tr.Messages[0].Content[0].Content)
		})
	}
}

func TestCompactToolUseInputs(t *testing.T) {
	longContent := strings.Repeat("line\n", 50)

	tests := []struct {
		name       string
		toolName   string
		input      map[string]any
		wantFields map[string]string // field → expected value substring
		keepFields []string          // fields that must NOT contain summary markers
	}{
		{
			name:       "write content summarized",
			toolName:   "Write",
			input:      map[string]any{"file_path": "/tmp/f.go", "content": longContent},
			wantFields: map[string]string{"content": "[content: 50 lines]"},
			keepFields: []string{"file_path"},
		},
		{
			name:     "edit old_string and new_string summarized",
			toolName: "Edit",
			input:    map[string]any{"file_path": "/tmp/f.go", "old_string": longContent, "new_string": longContent},
			wantFields: map[string]string{
				"old_string": "[old_string: 50 lines]",
				"new_string": "[new_string: 50 lines]",
			},
			keepFields: []string{"file_path"},
		},
		{
			name:       "bash command unchanged",
			toolName:   "Bash",
			input:      map[string]any{"command": "ls -la"},
			keepFields: []string{"command"},
		},
		{
			name:       "read file_path unchanged",
			toolName:   "Read",
			input:      map[string]any{"file_path": "/tmp/f.go"},
			keepFields: []string{"file_path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &core.Transcript{
				SessionID: "test",
				Agent:     "claude",
				CreatedAt: time.Now(),
				Messages: []core.Message{{
					Role: core.RoleAssistant,
					Content: []core.ContentBlock{
						{Type: core.BlockToolUse, Name: tt.toolName, Input: tt.input},
					},
				}},
			}

			c := New(Config{})
			require.NoError(t, c.Transform(tr))

			m := tr.Messages[0].Content[0].Input.(map[string]any)
			for field, want := range tt.wantFields {
				assert.Equal(t, want, m[field], "field %q", field)
			}
			for _, field := range tt.keepFields {
				s := m[field].(string)
				assert.NotContains(t, s, "[", "field %q should not be summarized", field)
			}
		})
	}
}

func TestCompactKeepThinkingByDefault(t *testing.T) {
	tr := &core.Transcript{
		SessionID: "test",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{{
			Role: core.RoleAssistant,
			Content: []core.ContentBlock{
				{Type: core.BlockThinking, Text: "deep thoughts"},
				{Type: core.BlockText, Text: "response"},
			},
		}},
	}

	c := New(Config{})
	require.NoError(t, c.Transform(tr))
	require.Len(t, tr.Messages[0].Content, 2)
	assert.Equal(t, core.BlockThinking, tr.Messages[0].Content[0].Type)
}

func TestCompactStripThinking(t *testing.T) {
	tr := &core.Transcript{
		SessionID: "test",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{{
			Role: core.RoleAssistant,
			Content: []core.ContentBlock{
				{Type: core.BlockThinking, Text: "deep thoughts"},
				{Type: core.BlockText, Text: "response"},
			},
		}},
	}

	c := New(Config{StripThinking: true})
	require.NoError(t, c.Transform(tr))
	require.Len(t, tr.Messages[0].Content, 1)
	assert.Equal(t, core.BlockText, tr.Messages[0].Content[0].Type)
}

func TestCompactVerboseSession(t *testing.T) {
	tr := readTestdata(t, "verbose_session.jsonl")

	c := New(Config{})
	require.NoError(t, c.Transform(tr))

	// Thinking block should still be present (default: keep)
	var hasThinking bool
	for _, msg := range tr.Messages {
		for _, b := range msg.Content {
			if b.Type == core.BlockThinking {
				hasThinking = true
			}
			if b.Type == core.BlockToolResult {
				assert.Contains(t, b.Content, "[output:")
			}
		}
	}
	assert.True(t, hasThinking, "thinking blocks should be preserved by default")

	// Write tool content field should be summarized
	for _, msg := range tr.Messages {
		for _, b := range msg.Content {
			if b.Type == core.BlockToolUse && strings.EqualFold(b.Name, "Write") {
				m := b.Input.(map[string]any)
				assert.Contains(t, m["content"], "[content:")
			}
		}
	}
}

func TestCompactVerboseSessionNoThinking(t *testing.T) {
	tr := readTestdata(t, "verbose_session.jsonl")

	c := New(Config{StripThinking: true})
	require.NoError(t, c.Transform(tr))

	for _, msg := range tr.Messages {
		for _, b := range msg.Content {
			assert.NotEqual(t, core.BlockThinking, b.Type)
		}
	}
}

func TestCompactErrorSession(t *testing.T) {
	tr := readTestdata(t, "error_session.jsonl")

	c := New(Config{})
	require.NoError(t, c.Transform(tr))

	for _, msg := range tr.Messages {
		for _, b := range msg.Content {
			if b.Type == core.BlockToolResult && b.IsError {
				assert.Contains(t, b.Content, "[error:")
			}
		}
	}
}
