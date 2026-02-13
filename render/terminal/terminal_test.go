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
					{Type: core.BlockToolUse, Name: "Bash", Input: map[string]any{"command": "grep -rn auth src/"}},
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

	assert.Contains(t, out, "user: Fix the auth bug")
	assert.Contains(t, out, "[bash: grep -rn auth src/]")
	assert.Contains(t, out, "assistant: Found the issue in the auth module.")
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
					{Type: core.BlockToolUse, Name: "Read", Input: map[string]any{"file_path": "main.go"}},
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
	// Only one "user:" line — the tool_result message should not create a turn
	count := strings.Count(out, "user:")
	assert.Equal(t, 1, count, "should have exactly 1 user turn, got output:\n%s", out)
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
	assert.Contains(t, out, "user: First question")
	assert.Contains(t, out, "user: Second question")
	assert.Contains(t, out, "assistant: First answer")
	assert.Contains(t, out, "assistant: Second answer")
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
	assert.Contains(t, out, "(0/0)")
}

func TestRenderLineCount(t *testing.T) {
	tr := &core.Transcript{
		SessionID: "test-count",
		Agent:     "claude",
		CreatedAt: time.Now(),
		Messages: []core.Message{
			{
				Role:    core.RoleUser,
				Content: []core.ContentBlock{{Type: core.BlockText, Text: "Hello"}},
			},
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{Type: core.BlockToolUse, Name: "Bash", Input: map[string]any{"command": "ls"}},
					{Type: core.BlockToolUse, Name: "Read", Input: map[string]any{"file_path": "a.go"}},
					{Type: core.BlockText, Text: "Done"},
				},
			},
		},
	}

	r := &Renderer{Width: 80}
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	out := ansi.Strip(buf.String())
	// 1 user line + 3 children = 4 lines
	assert.Contains(t, out, "(4/4)")
}

func TestRenderSkipsThinkingBlocks(t *testing.T) {
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
	assert.Contains(t, out, "assistant: Here's the answer.")
}
