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
		Relation:  core.RelationRoot,
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
		assert.Contains(t, html, "project\n        </h1>")
		assert.Contains(t, html, "claude")
		assert.Contains(t, html, "Root")
		assert.NotContains(t, html, "@claude")
		assert.Contains(t, html, "claude-opus-4-6")
		assert.Contains(t, html, "ago")
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

	t.Run("user message in blue bubble", func(t *testing.T) {
		assert.Contains(t, html, "bg-blue-50")
		assert.Contains(t, html, "Fix the authentication bug")
	})

	t.Run("steps collapsed", func(t *testing.T) {
		assert.Contains(t, html, "Steps Completed")
	})

	t.Run("thinking block in steps", func(t *testing.T) {
		assert.Contains(t, html, "Let me analyze the auth code...")
	})

	t.Run("markdown text", func(t *testing.T) {
		assert.Contains(t, html, `class="prose`)
		assert.Contains(t, html, "<code>auth.go</code>")
	})
}

func TestRenderTimelineRail(t *testing.T) {
	tr := buildTestTranscript()
	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	html := buf.String()

	assert.Contains(t, html, `aria-label="Session timeline"`)
	assert.Contains(t, html, `class="timeline-list"`)
	assert.Contains(t, html, `class="timeline-item"`)
	assert.Contains(t, html, `class="timeline-link-text"`)
	assert.Contains(t, html, `.timeline-duration`)
	assert.Contains(t, html, `href="#turn-0"`)
	assert.Contains(t, html, `1 turns`)
}

func TestRenderIndexPage(t *testing.T) {
	tr := buildTestTranscript()
	tr.Author = "ravi"
	tr.DiffStats = &core.DiffStats{
		Added:   120,
		Removed: 12,
		Changed: 4,
	}
	tr.Stats = &core.SessionStats{
		ToolUsage: map[string]int{
			"Bash": 2,
			"Read": 1,
		},
		SubAgentCount: 1,
	}

	entry := core.NewSessionEntry(
		tr,
		"/session/claude/test-session-123",
	)

	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.RenderIndex(&buf, []core.SessionEntry{entry}))

	html := buf.String()

	assert.Contains(t, html, "<title>Sessions")
	assert.Contains(t, html, "Session archive")
	assert.Contains(t, html, "Live transcript index")
	assert.Contains(t, html, `class="index-row`)
	assert.Contains(t, html, "relation-root")
	assert.Contains(t, html, "Root")
	assert.Contains(t, html, `class="index-title`)
	assert.Contains(t, html, `href="/session/claude/test-session-123"`)
	assert.Contains(t, html, "Fix the authentication bug")
	assert.Contains(t, html, "claude-opus-4-6")
	assert.Contains(t, html, "3 messages")
	assert.Contains(t, html, "2 tools")
	assert.Contains(t, html, "1 sub-agent")
	assert.Contains(t, html, "+120")
	assert.Contains(t, html, "~4")
	assert.Contains(t, html, "-12")
	assert.Contains(t, html, "5,000")
	assert.Contains(t, html, "2,000")
	assert.NotContains(t, html, "@ravi")
	assert.NotContains(t, html, "@claude")
}

func TestRenderRelationDistinction(t *testing.T) {
	now := time.Date(2026, 1, 22, 9, 8, 6, 0, time.UTC)
	entries := []core.SessionEntry{
		core.NewSessionEntry(
			&core.Transcript{
				SessionID: "fork-session",
				Agent:     "claude",
				Relation:  core.RelationFork,
				ForkedFrom: &core.ForkInfo{
					SessionID:   "source-session",
					MessageUUID: "source-message",
				},
				Title:     "Forked work",
				CreatedAt: now,
			},
			"/session/claude/fork-session",
		),
		core.NewSessionEntry(
			&core.Transcript{
				SessionID:       "child-session",
				ParentSessionID: "parent-session",
				Agent:           "codex",
				Relation:        core.RelationSubagent,
				Title:           "Review patch",
				CreatedAt:       now,
			},
			"/session/codex/child-session",
		),
	}

	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.RenderIndex(&buf, entries))

	html := buf.String()
	assert.Contains(t, html, "relation-fork")
	assert.Contains(t, html, "Fork")
	assert.Contains(t, html, "forked from source-s")
	assert.Contains(t, html, "relation-subagent")
	assert.Contains(t, html, "Subagent")
	assert.Contains(t, html, "parent parent-s")
}

func TestRenderHeaderRelationDetails(t *testing.T) {
	tr := buildTestTranscript()
	tr.Relation = core.RelationFork
	tr.ForkedFrom = &core.ForkInfo{
		SessionID:   "source-session",
		MessageUUID: "source-message",
	}

	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.Render(&buf, tr))

	html := buf.String()
	assert.Contains(t, html, "session-header relation-fork")
	assert.Contains(t, html, "Fork")
	assert.Contains(t, html, "forked from source-session")
	assert.Contains(t, html, "at source-m")
}

func TestRenderIndexEmptyState(t *testing.T) {
	r := New()
	var buf bytes.Buffer
	require.NoError(t, r.RenderIndex(&buf, nil))

	html := buf.String()

	assert.Contains(t, html, "0 sessions")
	assert.Contains(t, html, "No sessions found.")
	assert.Contains(t, html, "--all")
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

	t.Run("consumed tool_result does not create extra turn", func(t *testing.T) {
		// The third message only contained a consumed tool_result.
		// It should be folded into the first turn's steps, not create a separate turn.
		count := countOccurrences(html, `id="turn-`)
		assert.Equal(t, 1, count, "should have 1 turn, not 2")
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

func TestProjectName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/project", "project"},
		{"/home/user/project/", "project"},
		{"relative/path/app", "app"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, projectName(tt.path))
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
