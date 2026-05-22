package json

import (
	"bytes"
	stdjson "encoding/json"
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
		},
	}

	var buf bytes.Buffer
	err := New().Render(&buf, tr)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "\n  \"session_id\": \"sess-1\"")

	var got core.Transcript
	require.NoError(t, stdjson.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "sess-1", got.SessionID)
	assert.Equal(t, "claude", got.Agent)
	assert.Equal(t, "Fix auth", got.Title)
	require.Len(t, got.Messages, 1)
	assert.Equal(t, core.RoleUser, got.Messages[0].Role)
}
