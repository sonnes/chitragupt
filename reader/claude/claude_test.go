package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sonnes/chitragupt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}

func readTestdata(t *testing.T, name string) *core.Transcript {
	t.Helper()
	r := &Reader{}
	tr, err := r.ReadFile(testdataPath(name))
	require.NoError(t, err)
	return tr
}

// setupProjectDir copies a testdata file into a temp directory structured as
// ~/.claude/projects/<project>/<sessionID>.jsonl for directory-traversal tests.
func setupProjectDir(t *testing.T, testdataFile, project, sessionID string) *Reader {
	t.Helper()
	data, err := os.ReadFile(testdataPath(testdataFile))
	require.NoError(t, err)

	dir := t.TempDir()
	projectDir := filepath.Join(dir, project)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, sessionID+".jsonl"), data, 0o644))
	return &Reader{Dir: dir}
}

func TestScanEntries(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		wantCount int
	}{
		{
			name:      "filters non-message types and sidechain",
			file:      "mixed_entries.jsonl",
			wantCount: 2,
		},
		{
			name:      "simple pair",
			file:      "simple.jsonl",
			wantCount: 2,
		},
		{
			name:      "streaming chunks all kept",
			file:      "streaming_chunks.jsonl",
			wantCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(testdataPath(tt.file))
			require.NoError(t, err)
			defer f.Close()

			entries, err := scanEntries(f)
			require.NoError(t, err)
			assert.Len(t, entries, tt.wantCount)
		})
	}
}

func TestGroupAndMapMessages(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		wantMsgs   int
		wantBlocks []int
		wantRoles  []core.Role
	}{
		{
			name:       "simple user-assistant pair",
			file:       "simple.jsonl",
			wantMsgs:   2,
			wantBlocks: []int{1, 1},
			wantRoles:  []core.Role{core.RoleUser, core.RoleAssistant},
		},
		{
			name:       "streaming chunks merged into one assistant message",
			file:       "streaming_chunks.jsonl",
			wantMsgs:   2,
			wantBlocks: []int{1, 3},
			wantRoles:  []core.Role{core.RoleUser, core.RoleAssistant},
		},
		{
			name:       "interleaved tool results within assistant group",
			file:       "tool_loop.jsonl",
			wantMsgs:   4,
			wantBlocks: []int{1, 1, 1, 2},
			wantRoles:  []core.Role{core.RoleUser, core.RoleUser, core.RoleUser, core.RoleAssistant},
		},
		{
			name:       "new human turn flushes pending assistant",
			file:       "multi_turn.jsonl",
			wantMsgs:   4,
			wantBlocks: []int{1, 1, 1, 1},
			wantRoles:  []core.Role{core.RoleUser, core.RoleAssistant, core.RoleUser, core.RoleAssistant},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(testdataPath(tt.file))
			require.NoError(t, err)
			defer f.Close()

			entries, err := scanEntries(f)
			require.NoError(t, err)

			messages := groupAndMapMessages(entries)
			require.Len(t, messages, tt.wantMsgs)

			for i, m := range messages {
				assert.Len(t, m.Content, tt.wantBlocks[i], "msg[%d] block count", i)
				assert.Equal(t, tt.wantRoles[i], m.Role, "msg[%d] role", i)
			}
		})
	}
}

func TestContentBlockMapping(t *testing.T) {
	// all_block_types.jsonl produces: user(text), user(tool_result), assistant(thinking,text,tool_use)
	tr := readTestdata(t, "all_block_types.jsonl")
	require.Len(t, tr.Messages, 3)

	tests := []struct {
		name     string
		msgIdx   int
		blockIdx int
		wantType core.BlockType
		wantText string
	}{
		{"user text", 0, 0, core.BlockText, "hello"},
		{"tool result", 1, 0, core.BlockToolResult, ""},
		{"thinking", 2, 0, core.BlockThinking, "reasoning"},
		{"assistant text", 2, 1, core.BlockText, "response"},
		{"tool use", 2, 2, core.BlockToolUse, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tr.Messages[tt.msgIdx].Content[tt.blockIdx]
			assert.Equal(t, tt.wantType, b.Type)
			if tt.wantText != "" {
				assert.Equal(t, tt.wantText, b.Text)
			}
		})
	}

	t.Run("user text is plain, assistant text is markdown", func(t *testing.T) {
		assert.Equal(t, core.FormatPlain, tr.Messages[0].Content[0].Format)
		assert.Equal(t, core.FormatMarkdown, tr.Messages[2].Content[1].Format)
	})

	t.Run("tool use fields", func(t *testing.T) {
		b := tr.Messages[2].Content[2]
		assert.Equal(t, "toolu_1", b.ToolUseID)
		assert.Equal(t, "Bash", b.Name)
	})

	t.Run("tool result fields", func(t *testing.T) {
		b := tr.Messages[1].Content[0]
		assert.Equal(t, "toolu_1", b.ToolUseID)
		assert.Equal(t, "cmd output", b.Content)
		assert.False(t, b.IsError)
	})
}

func TestExtractToolResultContent(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string", "hello", "hello"},
		{"nil", nil, ""},
		{"array of text blocks", []any{
			map[string]any{"type": "text", "text": "line1"},
			map[string]any{"type": "text", "text": "line2"},
		}, "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractToolResultContent(tt.in))
		})
	}
}

func TestBuildTranscript(t *testing.T) {
	tr := readTestdata(t, "simple.jsonl")

	assert.Equal(t, "sess-1", tr.SessionID)
	assert.Equal(t, "claude", tr.Agent)
	assert.Equal(t, "claude-opus-4-6", tr.Model)
	assert.Equal(t, "/work", tr.Dir)
	assert.Equal(t, "main", tr.GitBranch)
	assert.Equal(t, "fix the bug", tr.Title)
	assert.False(t, tr.CreatedAt.IsZero())
	require.NotNil(t, tr.UpdatedAt)
	require.NotNil(t, tr.Usage)
	assert.Equal(t, 100, tr.Usage.InputTokens)
	assert.Equal(t, 50, tr.Usage.OutputTokens)
	assert.Equal(t, 5, tr.Usage.CacheReadTokens)
	assert.Equal(t, 10, tr.Usage.CacheCreationTokens)
}

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{"simple text", "simple.jsonl", "fix the bug"},
		{"skips ide metadata", "ide_title.jsonl", "real title here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := readTestdata(t, tt.file)
			assert.Equal(t, tt.want, tr.Title)
		})
	}
}

func TestToolResultError(t *testing.T) {
	tr := readTestdata(t, "tool_error.jsonl")
	require.True(t, len(tr.Messages) >= 2)

	b := tr.Messages[1].Content[0]
	assert.Equal(t, core.BlockToolResult, b.Type)
	assert.True(t, b.IsError)
	assert.Equal(t, "permission denied", b.Content)
}

func TestUsageAggregation(t *testing.T) {
	tr := readTestdata(t, "multi_turn.jsonl")
	require.NotNil(t, tr.Usage)
	assert.Equal(t, 110, tr.Usage.InputTokens)
	assert.Equal(t, 45, tr.Usage.OutputTokens)
}

func TestReadSession(t *testing.T) {
	r := setupProjectDir(t, "simple.jsonl", "-project-a", "abc-123")

	tr, err := r.ReadSession("abc-123")
	require.NoError(t, err)
	assert.Equal(t, "sess-1", tr.SessionID)

	_, err = r.ReadSession("nonexistent")
	assert.Error(t, err)
}

func TestReadProject(t *testing.T) {
	r := setupProjectDir(t, "simple.jsonl", "-my-project", "sess-1")
	data, _ := os.ReadFile(testdataPath("simple.jsonl"))
	os.WriteFile(filepath.Join(r.Dir, "-my-project", "sess-2.jsonl"), data, 0o644)

	transcripts, err := r.ReadProject("-my-project")
	require.NoError(t, err)
	assert.Len(t, transcripts, 2)
}

func TestReadAll(t *testing.T) {
	r := setupProjectDir(t, "simple.jsonl", "-project-a", "sess-1")
	data, _ := os.ReadFile(testdataPath("simple.jsonl"))
	projB := filepath.Join(r.Dir, "-project-b")
	os.MkdirAll(projB, 0o755)
	os.WriteFile(filepath.Join(projB, "sess-2.jsonl"), data, 0o644)

	transcripts, err := r.ReadAll()
	require.NoError(t, err)
	assert.Len(t, transcripts, 2)
}
