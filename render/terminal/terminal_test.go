package terminal

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/sonnes/chitragupt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderHeader(t *testing.T) {
	now := time.Now()
	later := now.Add(72*time.Hour + 44*time.Minute)
	tr := &core.Transcript{
		SessionID: "abc-123",
		Agent:     "claude",
		Model:     "claude-opus-4-5-20251101",
		Dir:       "/Users/test",
		GitBranch: "main",
		CreatedAt: now,
		UpdatedAt: &later,
		Usage: &core.Usage{
			InputTokens:         229,
			OutputTokens:        1273,
			CacheReadTokens:     1228873,
			CacheCreationTokens: 202896,
		},
	}

	r := &Renderer{Width: 100}
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	out := ansi.Strip(buf.String())

	assert.Contains(t, out, "Session abc-123")
	assert.Contains(t, out, "@claude")
	assert.Contains(t, out, "claude-opus-4-5-20251101")
	assert.Contains(t, out, "just now")
	assert.Contains(t, out, "/Users/test(main)")
	assert.Contains(t, out, "229")
	assert.Contains(t, out, "1,273")
	assert.Contains(t, out, "1,228,873")
	assert.Contains(t, out, "202,896")
	assert.Contains(t, out, "INPUT")
	assert.Contains(t, out, "OUTPUT")
	assert.Contains(t, out, "CACHE READ")
	assert.Contains(t, out, "CACHE WRITE")
}

func TestRenderBasicTranscript(t *testing.T) {
	now := time.Now()
	tr := &core.Transcript{
		SessionID: "test-basic",
		Agent:     "claude",
		CreatedAt: now,
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatPlain, Text: "Fix the auth bug"},
				},
			},
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{Type: core.BlockToolUse, ToolUseID: "t1", Name: "Bash", Input: map[string]any{"command": "grep -rn auth src/"}},
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "Found the issue in the auth module."},
				},
			},
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "t1", Content: "auth.go:12: func Auth()"},
				},
			},
		},
	}

	r := &Renderer{Width: 80}
	var buf bytes.Buffer
	err := r.Render(&buf, tr)
	require.NoError(t, err)

	out := ansi.Strip(buf.String())

	assert.Contains(t, out, "USER")
	assert.Contains(t, out, "Fix the auth bug")
	assert.Contains(t, out, "ASSISTANT")
	assert.Contains(t, out, "Bash")
	assert.Contains(t, out, "grep -rn auth src/")
	assert.Contains(t, out, "Found the issue in the auth module.")
}

func TestRenderSkipsToolResultMessages(t *testing.T) {
	now := time.Now()
	tr := &core.Transcript{
		SessionID: "test-skip-toolresult",
		Agent:     "claude",
		CreatedAt: now,
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatPlain, Text: "Hello"},
				},
			},
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{Type: core.BlockToolUse, ToolUseID: "t1", Name: "Read", Input: map[string]any{"file_path": "main.go"}},
				},
			},
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "t1", Content: "package main"},
				},
			},
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "Done."},
				},
			},
		},
	}

	r := &Renderer{Width: 80}
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	out := ansi.Strip(buf.String())
	// Tool-result-only user message should be skipped (consumed by tool_use).
	count := strings.Count(out, "USER")
	assert.Equal(t, 1, count, "should have exactly 1 USER card, got output:\n%s", out)
}

func TestRenderTruncation(t *testing.T) {
	tr := &core.Transcript{
		SessionID: "test-truncate",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatPlain, Text: strings.Repeat("a", 300)},
				},
			},
		},
	}

	r := &Renderer{Width: 60}
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	out := ansi.Strip(buf.String())
	assert.Contains(t, out, "...")
}

func TestRenderMultiTurn(t *testing.T) {
	now := time.Now()
	tr := &core.Transcript{
		SessionID: "test-multi",
		Agent:     "claude",
		CreatedAt: now,
		Messages: []core.Message{
			{
				Role:    core.RoleUser,
				Content: []core.ContentBlock{{Type: core.BlockText, Text: "First question"}},
			},
			{
				Role:    core.RoleAssistant,
				Content: []core.ContentBlock{{Type: core.BlockText, Text: "First answer"}},
			},
			{
				Role:    core.RoleUser,
				Content: []core.ContentBlock{{Type: core.BlockText, Text: "Second question"}},
			},
			{
				Role:    core.RoleAssistant,
				Content: []core.ContentBlock{{Type: core.BlockText, Text: "Second answer"}},
			},
		},
	}

	r := &Renderer{Width: 80}
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	out := ansi.Strip(buf.String())
	assert.Contains(t, out, "First question")
	assert.Contains(t, out, "First answer")
	assert.Contains(t, out, "Second question")
	assert.Contains(t, out, "Second answer")
	assert.Equal(t, 2, strings.Count(out, "USER"))
	assert.Equal(t, 2, strings.Count(out, "ASSISTANT"))
}

func TestRenderEmptyTranscript(t *testing.T) {
	tr := &core.Transcript{
		SessionID: "empty",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages:  []core.Message{},
	}

	r := &Renderer{Width: 80}
	var buf bytes.Buffer
	err := r.Render(&buf, tr)
	require.NoError(t, err)

	out := ansi.Strip(buf.String())
	assert.Contains(t, out, "Session empty")
	assert.Contains(t, out, "claude")
	assert.NotContains(t, out, "USER")
	assert.NotContains(t, out, "ASSISTANT")
}

func TestRenderThinkingBlocks(t *testing.T) {
	tr := &core.Transcript{
		SessionID: "test-thinking",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{
			{
				Role:    core.RoleUser,
				Content: []core.ContentBlock{{Type: core.BlockText, Text: "Help"}},
			},
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{Type: core.BlockThinking, Text: "Let me think about this..."},
					{Type: core.BlockText, Text: "Here's the answer."},
				},
			},
		},
	}

	r := &Renderer{Width: 80}
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	out := ansi.Strip(buf.String())
	assert.NotContains(t, out, "Let me think about this")
	assert.Contains(t, out, "Thinking...")
	assert.Contains(t, out, "Here's the answer.")
}

func TestRenderMessageTimestamps(t *testing.T) {
	t1 := time.Date(2026, 2, 3, 3, 26, 0, 0, time.UTC)
	t2 := t1.Add(5 * time.Second)

	tr := &core.Transcript{
		SessionID: "test-timestamps",
		Agent:     "claude",
		CreatedAt: t1,
		Messages: []core.Message{
			{
				Role:      core.RoleUser,
				Timestamp: &t1,
				Content:   []core.ContentBlock{{Type: core.BlockText, Text: "Hello"}},
			},
			{
				Role:      core.RoleAssistant,
				Timestamp: &t2,
				Content:   []core.ContentBlock{{Type: core.BlockText, Text: "Hi there"}},
			},
		},
	}

	r := &Renderer{Width: 80}
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	out := ansi.Strip(buf.String())
	assert.Contains(t, out, "Feb 3, 2026")
	assert.Contains(t, out, "5s")
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1273, "1,273"},
		{1228873, "1,228,873"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatNumber(tt.in), "formatNumber(%d)", tt.in)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{500 * time.Millisecond, "<1s"},
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m 30s"},
		{5 * time.Minute, "5m"},
		{72*time.Hour + 44*time.Minute, "72h 44m"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatDuration(tt.in), "formatDuration(%s)", tt.in)
	}
}
