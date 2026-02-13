package html

import (
	"bytes"
	"testing"
	"time"

	"github.com/sonnes/chitragupt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestTranscript() *core.Transcript {
	now := time.Date(2026, 1, 22, 9, 8, 6, 0, time.UTC)
	later := now.Add(30 * time.Minute)
	return &core.Transcript{
		SessionID: "test-session-123",
		Agent:     "claude",
		Model:     "claude-opus-4-6",
		Dir:       "/home/user/project",
		GitBranch: "main",
		Title:     "Fix the authentication bug",
		CreatedAt: now,
		UpdatedAt: &later,
		Usage:     &core.Usage{InputTokens: 5000, OutputTokens: 2000},
		Messages: []core.Message{
			{
				Role:      core.RoleUser,
				Timestamp: &now,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatPlain, Text: "Fix the authentication bug"},
				},
			},
			{
				Role:      core.RoleAssistant,
				Model:     "claude-opus-4-6",
				Timestamp: &later,
				Content: []core.ContentBlock{
					{Type: core.BlockThinking, Text: "Let me analyze the auth code..."},
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "I'll fix the bug in `auth.go`."},
					{Type: core.BlockToolUse, ToolUseID: "t1", Name: "Bash", Input: map[string]any{"command": "grep -n 'func Login' auth.go"}},
				},
			},
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "t1", Content: "42: func Login(ctx context.Context) error {", IsError: false},
				},
			},
		},
	}
}

func TestRenderFullPage(t *testing.T) {
	tr := buildTestTranscript()
	r := New()
	var buf bytes.Buffer
	err := r.Render(&buf, tr)
	require.NoError(t, err)

	html := buf.String()

	t.Run("page structure", func(t *testing.T) {
		assert.Contains(t, html, "<!DOCTYPE html>")
		assert.Contains(t, html, "<html lang=\"en\">")
		assert.Contains(t, html, "</html>")
	})

	t.Run("tailwind CDN", func(t *testing.T) {
		assert.Contains(t, html, "@tailwindcss/browser@4")
	})

	t.Run("inter font", func(t *testing.T) {
		assert.Contains(t, html, "fonts.googleapis.com")
		assert.Contains(t, html, "Inter")
	})

	t.Run("title", func(t *testing.T) {
		assert.Contains(t, html, "<title>Fix the authentication bug")
	})

	t.Run("header metadata", func(t *testing.T) {
		assert.Contains(t, html, "Fix the authentication bug")
		assert.Contains(t, html, "claude")
		assert.Contains(t, html, "claude-opus-4-6")
		assert.Contains(t, html, "Jan 22, 2026")
	})

	t.Run("usage stats", func(t *testing.T) {
		assert.Contains(t, html, "5,000")
		assert.Contains(t, html, "2,000")
	})

	t.Run("working dir", func(t *testing.T) {
		assert.Contains(t, html, "/home/user/project")
		assert.Contains(t, html, "main")
	})
}

func TestRenderMessages(t *testing.T) {
	tr := buildTestTranscript()
	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	html := buf.String()

	t.Run("user message", func(t *testing.T) {
		assert.Contains(t, html, "User")
		assert.Contains(t, html, "border-l-blue-500")
		assert.Contains(t, html, "Fix the authentication bug")
	})

	t.Run("assistant message", func(t *testing.T) {
		assert.Contains(t, html, "Assistant")
		assert.Contains(t, html, "border-l-emerald-500")
	})

	t.Run("thinking block", func(t *testing.T) {
		assert.Contains(t, html, "<details")
		assert.Contains(t, html, "Let me analyze the auth code...")
	})

	t.Run("markdown text", func(t *testing.T) {
		assert.Contains(t, html, `class="prose`)
		assert.Contains(t, html, "<code>auth.go</code>")
	})
}

func TestRenderToolPairing(t *testing.T) {
	tr := buildTestTranscript()
	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	html := buf.String()

	t.Run("tool use shows name and input", func(t *testing.T) {
		assert.Contains(t, html, "Bash")
		assert.Contains(t, html, "grep -n")
	})

	t.Run("tool result is paired", func(t *testing.T) {
		assert.Contains(t, html, "42: func Login")
	})

	t.Run("consumed tool_result message is skipped", func(t *testing.T) {
		// The third message only contained a consumed tool_result,
		// so it should not produce a separate message card.
		// Count message cards by role badges.
		count := countOccurrences(html, "font-semibold uppercase")
		assert.Equal(t, 2, count, "should have 2 message cards (user + assistant), not 3")
	})
}

func TestRenderToolResultError(t *testing.T) {
	now := time.Now()
	tr := &core.Transcript{
		SessionID: "err-session",
		Agent:     "claude",
		CreatedAt: now,
		Messages: []core.Message{
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{Type: core.BlockToolUse, ToolUseID: "e1", Name: "Bash", Input: map[string]any{"command": "false"}},
				},
			},
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "e1", Content: "exit status 1", IsError: true},
				},
			},
		},
	}

	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))
	html := buf.String()
	assert.Contains(t, html, "bg-red-50")
	assert.Contains(t, html, "exit status 1")
}

func TestRenderOrphanToolResult(t *testing.T) {
	now := time.Now()
	tr := &core.Transcript{
		SessionID: "orphan-session",
		Agent:     "claude",
		CreatedAt: now,
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "orphan-1", Content: "some output", IsError: false},
				},
			},
		},
	}

	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))
	html := buf.String()
	assert.Contains(t, html, "some output")
}

func TestRenderMinimalTranscript(t *testing.T) {
	now := time.Now()
	tr := &core.Transcript{
		SessionID: "minimal",
		Agent:     "claude",
		CreatedAt: now,
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatPlain, Text: "hello"},
				},
			},
		},
	}

	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))
	html := buf.String()
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "hello")
	assert.Contains(t, html, "Session minimal") // no title → fallback to session ID
}

func TestRenderNoTitle(t *testing.T) {
	now := time.Now()
	tr := &core.Transcript{
		SessionID: "abc-123",
		Agent:     "claude",
		CreatedAt: now,
		Messages:  []core.Message{},
	}

	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))
	html := buf.String()
	assert.Contains(t, html, "<title>chitragupt</title>")
	assert.Contains(t, html, "Session abc-123")
}

func TestFormatTimeFuncMap(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		expect string
	}{
		{
			name:   "time.Time",
			input:  time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC),
			expect: "Mar 15, 2026 2:30 PM",
		},
		{
			name:   "nil pointer",
			input:  (*time.Time)(nil),
			expect: "",
		},
		{
			name: "time pointer",
			input: func() *time.Time {
				t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			expect: "Jan 1, 2026 12:00 AM",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, formatTime(tt.input))
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input  int
		expect string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{-500, "-500"},
		{-1500, "-1,500"},
	}
	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			assert.Equal(t, tt.expect, formatNumber(tt.input))
		})
	}
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
