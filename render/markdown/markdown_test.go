package markdown

import (
	"bytes"
	"testing"
	"time"

	"github.com/sonnes/chitragupt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	now := time.Date(2026, 1, 22, 9, 8, 6, 0, time.UTC)
	tr := &core.Transcript{
		SessionID: "sess-1",
		Agent:     "claude",
		Model:     "claude-opus-4-6",
		Dir:       "/work/project",
		GitBranch: "main",
		Title:     "Fix auth",
		CreatedAt: now,
		Usage: &core.Usage{
			InputTokens:  1500,
			OutputTokens: 500,
		},
		Messages: []core.Message{
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{
						Type:   core.BlockText,
						Format: core.FormatPlain,
						Text:   "Fix auth",
					},
				},
			},
			{
				Role: core.RoleAssistant,
				Content: []core.ContentBlock{
					{
						Type:  core.BlockToolUse,
						Name:  "Bash",
						Input: map[string]any{"command": "go test ./..."},
					},
					{
						Type:   core.BlockText,
						Format: core.FormatMarkdown,
						Text:   "Done in `auth.go`.",
					},
				},
			},
			{
				Role: core.RoleUser,
				Content: []core.ContentBlock{
					{
						Type:      core.BlockToolResult,
						ToolUseID: "toolu-1",
						Content:   "ok ./...",
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := New().Render(&buf, tr)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "# Fix auth")
	assert.Contains(t, out, "- **Agent:** claude")
	assert.Contains(t, out, "- **Usage:** 1,500 input / 500 output tokens")
	assert.Contains(t, out, "## Turn 1")
	assert.Contains(t, out, "#### Bash")
	assert.Contains(t, out, "\"command\": \"go test ./...\"")
	assert.Contains(t, out, "Done in `auth.go`.")
	assert.Contains(t, out, "#### Tool result")
	assert.Contains(t, out, "ok ./...")
}
