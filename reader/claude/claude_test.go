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
			name:       "tool results folded into assistant, Read results dropped",
			file:       "tool_loop.jsonl",
			wantMsgs:   2,
			wantBlocks: []int{1, 3},
			wantRoles:  []core.Role{core.RoleUser, core.RoleAssistant},
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
	// all_block_types.jsonl produces: user(text), assistant(thinking,text,tool_use,tool_result)
	tr := readTestdata(t, "all_block_types.jsonl")
	require.Len(t, tr.Messages, 2)

	tests := []struct {
		name     string
		msgIdx   int
		blockIdx int
		wantType core.BlockType
		wantText string
	}{
		{"user text", 0, 0, core.BlockText, "hello"},
		{"thinking", 1, 0, core.BlockThinking, "reasoning"},
		{"assistant text", 1, 1, core.BlockText, "response"},
		{"tool use", 1, 2, core.BlockToolUse, ""},
		{"tool result", 1, 3, core.BlockToolResult, ""},
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
		assert.Equal(t, core.FormatMarkdown, tr.Messages[1].Content[1].Format)
	})

	t.Run("tool use fields", func(t *testing.T) {
		b := tr.Messages[1].Content[2]
		assert.Equal(t, "toolu_1", b.ToolUseID)
		assert.Equal(t, "Bash", b.Name)
	})

	t.Run("tool result fields", func(t *testing.T) {
		b := tr.Messages[1].Content[3]
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
	require.Len(t, tr.Messages, 2)

	// Error tool_result is folded into the assistant message after the tool_use.
	b := tr.Messages[1].Content[1]
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

// --- Sub-agent tests ---

// setupSubagentDir creates a temp directory with a main session file and a
// subagents directory containing the child file, mimicking the on-disk layout:
//
//	<project>/<sessionID>.jsonl
//	<project>/<sessionID>/subagents/agent-<agentID>.jsonl
func setupSubagentDir(t *testing.T, mainFile, childFile, agentID string) (mainPath string) {
	t.Helper()
	dir := t.TempDir()

	mainData, err := os.ReadFile(testdataPath(mainFile))
	require.NoError(t, err)
	mainPath = filepath.Join(dir, "sess-main-1.jsonl")
	require.NoError(t, os.WriteFile(mainPath, mainData, 0o644))

	subDir := filepath.Join(dir, "sess-main-1", "subagents")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	childData, err := os.ReadFile(testdataPath(childFile))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-"+agentID+".jsonl"), childData, 0o644))

	return mainPath
}

func TestDiscoverSubagentFiles(t *testing.T) {
	t.Run("returns nil when no subagents dir", func(t *testing.T) {
		files, err := discoverSubagentFiles(testdataPath("simple.jsonl"))
		require.NoError(t, err)
		assert.Nil(t, files)
	})

	t.Run("discovers agent files", func(t *testing.T) {
		mainPath := setupSubagentDir(t, "subagent_main.jsonl", "subagent_child.jsonl", "ae267a1")
		files, err := discoverSubagentFiles(mainPath)
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Contains(t, files, "ae267a1")
	})

	t.Run("skips acompact files", func(t *testing.T) {
		mainPath := setupSubagentDir(t, "subagent_main.jsonl", "subagent_child.jsonl", "ae267a1")

		// Add an acompact file that should be skipped.
		subDir := filepath.Join(filepath.Dir(mainPath), "sess-main-1", "subagents")
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-acompact-xyz.jsonl"), []byte("{}"), 0o644))

		files, err := discoverSubagentFiles(mainPath)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.NotContains(t, files, "acompact-xyz")
	})
}

func TestScanSubagentEntries(t *testing.T) {
	f, err := os.Open(testdataPath("subagent_child.jsonl"))
	require.NoError(t, err)
	defer f.Close()

	entries, err := scanSubagentEntries(f)
	require.NoError(t, err)
	assert.Len(t, entries, 4)

	// All entries should have isSidechain=true and agentId set.
	for _, e := range entries {
		assert.True(t, e.IsSidechain)
		assert.Equal(t, "ae267a1", e.AgentID)
	}
}

func TestBuildSubagentTranscript(t *testing.T) {
	path := testdataPath("subagent_child.jsonl")
	sub, err := buildSubagentTranscript(path, "sess-main-1")
	require.NoError(t, err)
	require.NotNil(t, sub)

	assert.Equal(t, "ae267a1", sub.SessionID)
	assert.Equal(t, "sess-main-1", sub.ParentSessionID)
	assert.Equal(t, "claude", sub.Agent)
	assert.Equal(t, "claude-sonnet-4-5-20250929", sub.Model)
	assert.Equal(t, "Find all Go files", sub.Title)
	assert.False(t, sub.CreatedAt.IsZero())
	require.NotNil(t, sub.Usage)
	assert.Equal(t, 400, sub.Usage.InputTokens)
	assert.Equal(t, 200, sub.Usage.OutputTokens)
}

func TestExtractAgentIDFromResult(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "standalone agentId",
			content: "Here are the results...\n\nagentId: ae267a1",
			want:    "ae267a1",
		},
		{
			name:    "team agent_id",
			content: "Research complete.\n\nagent_id: researcher@auth-team",
			want:    "researcher@auth-team",
		},
		{
			name:    "no agent ID",
			content: "Some random output",
			want:    "",
		},
		{
			name:    "empty string",
			content: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractAgentIDFromResult(tt.content))
		})
	}
}

func TestExtractTaskAgentInfo(t *testing.T) {
	t.Run("extracts all fields", func(t *testing.T) {
		input := map[string]any{
			"subagent_type": "deep-researcher",
			"name":          "researcher",
			"team_name":     "auth-team",
			"prompt":        "Analyze auth module",
		}
		ref := extractTaskAgentInfo(input)
		assert.Equal(t, "deep-researcher", ref.AgentType)
		assert.Equal(t, "researcher", ref.AgentName)
		assert.Equal(t, "auth-team", ref.TeamName)
	})

	t.Run("handles missing fields", func(t *testing.T) {
		input := map[string]any{
			"subagent_type": "Explore",
			"prompt":        "Find files",
		}
		ref := extractTaskAgentInfo(input)
		assert.Equal(t, "Explore", ref.AgentType)
		assert.Empty(t, ref.AgentName)
		assert.Empty(t, ref.TeamName)
	})

	t.Run("handles nil input", func(t *testing.T) {
		ref := extractTaskAgentInfo(nil)
		assert.Empty(t, ref.AgentType)
	})
}

func TestAttachSubagents(t *testing.T) {
	t.Run("standalone subagent", func(t *testing.T) {
		mainPath := setupSubagentDir(t, "subagent_main.jsonl", "subagent_child.jsonl", "ae267a1")
		r := &Reader{}
		tr, err := r.ReadFile(mainPath)
		require.NoError(t, err)

		// Sub-agents should be attached.
		require.Len(t, tr.SubAgents, 1)
		sub := tr.SubAgents[0]
		assert.Equal(t, "ae267a1", sub.SessionID)
		assert.Equal(t, "sess-main-1", sub.ParentSessionID)
		assert.Equal(t, "Find all Go files", sub.Title)

		// The Task tool_use block should have a SubAgentRef.
		var found bool
		for _, msg := range tr.Messages {
			for _, b := range msg.Content {
				if b.Type == core.BlockToolUse && b.Name == "Task" {
					require.NotNil(t, b.SubAgentRef)
					assert.Equal(t, "ae267a1", b.SubAgentRef.AgentID)
					assert.Equal(t, "Explore", b.SubAgentRef.AgentType)
					found = true
				}
			}
		}
		assert.True(t, found, "expected to find Task tool_use block with SubAgentRef")
	})

	t.Run("team subagent", func(t *testing.T) {
		// Build directory structure manually for team session ID.
		dir := t.TempDir()

		mainData, err := os.ReadFile(testdataPath("subagent_team_main.jsonl"))
		require.NoError(t, err)
		teamMain := filepath.Join(dir, "sess-team-1.jsonl")
		require.NoError(t, os.WriteFile(teamMain, mainData, 0o644))

		subDir := filepath.Join(dir, "sess-team-1", "subagents")
		require.NoError(t, os.MkdirAll(subDir, 0o755))
		childData, err := os.ReadFile(testdataPath("subagent_child.jsonl"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "agent-researcher@auth-team.jsonl"), childData, 0o644))

		r := &Reader{}
		tr, err := r.ReadFile(teamMain)
		require.NoError(t, err)

		require.Len(t, tr.SubAgents, 1)

		// The Task tool_use block should have team-specific SubAgentRef fields.
		var found bool
		for _, msg := range tr.Messages {
			for _, b := range msg.Content {
				if b.Type == core.BlockToolUse && b.Name == "Task" {
					require.NotNil(t, b.SubAgentRef)
					assert.Equal(t, "researcher@auth-team", b.SubAgentRef.AgentID)
					assert.Equal(t, "deep-researcher", b.SubAgentRef.AgentType)
					assert.Equal(t, "researcher", b.SubAgentRef.AgentName)
					assert.Equal(t, "auth-team", b.SubAgentRef.TeamName)
					found = true
				}
			}
		}
		assert.True(t, found, "expected to find Task tool_use block with team SubAgentRef")
	})

	t.Run("no subagents directory is no-op", func(t *testing.T) {
		tr := readTestdata(t, "simple.jsonl")
		assert.Nil(t, tr.SubAgents)
	})
}
