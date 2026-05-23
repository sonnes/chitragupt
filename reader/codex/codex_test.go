package codex

import (
	"os"
	"path/filepath"
	"strings"
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

func TestReadFileSimple(t *testing.T) {
	tr := readTestdata(t, "simple.jsonl")

	assert.Equal(t, "codex-simple", tr.SessionID)
	assert.Equal(t, "codex", tr.Agent)
	assert.Equal(t, core.RelationRoot, tr.Relation)
	assert.Equal(t, "/work/chitragupt", tr.Dir)
	assert.Equal(t, "main", tr.GitBranch)
	assert.Equal(t, "gpt-5.3-codex", tr.Model)
	assert.Equal(t, "Implement codex reader", tr.Title)
	assert.False(t, tr.CreatedAt.IsZero())
	require.NotNil(t, tr.UpdatedAt)
	require.NotNil(t, tr.Usage)
	assert.Equal(t, 20, tr.Usage.InputTokens)
	assert.Equal(t, 8, tr.Usage.OutputTokens)
	assert.Equal(t, 3, tr.Usage.CacheReadTokens)

	require.Len(t, tr.Messages, 2)
	assert.Equal(t, core.RoleUser, tr.Messages[0].Role)
	assert.Equal(t, "Implement codex reader", tr.Messages[0].Content[0].Text)
	assert.Equal(t, core.RoleAssistant, tr.Messages[1].Role)
	assert.Equal(t, "Implemented the reader.", tr.Messages[1].Content[0].Text)
}

func TestSessionMetaRelation(t *testing.T) {
	tr := readTestdata(t, "subagent_meta.jsonl")

	assert.Equal(t, "codex-subagent", tr.SessionID)
	assert.Equal(t, core.RelationSubagent, tr.Relation)
	assert.Equal(t, "/work/chitragupt", tr.Dir)
	assert.Equal(t, "feature", tr.GitBranch)
}

func TestInjectedContextUsesVisibleUserMessage(t *testing.T) {
	tr := readTestdata(t, "injected_context.jsonl")

	require.Len(t, tr.Messages, 2)
	require.Len(t, tr.Messages[0].Content, 1)
	assert.Equal(t, "What should be rendered", tr.Messages[0].Content[0].Text)
	assert.NotContains(t, tr.Messages[0].Content[0].Text, "AGENTS.md")
	assert.Equal(t, "What should be rendered", tr.Title)
}

func TestToolMapping(t *testing.T) {
	tr := readTestdata(t, "tool_loop.jsonl")

	require.Len(t, tr.Messages, 2)
	assistant := tr.Messages[1]
	require.Len(t, assistant.Content, 5)

	assert.Equal(t, core.BlockToolUse, assistant.Content[0].Type)
	assert.Equal(t, "exec_command", assistant.Content[0].Name)
	assert.Equal(t, "call-1", assistant.Content[0].ToolUseID)
	input, ok := assistant.Content[0].Input.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ls", input["cmd"])

	assert.Equal(t, core.BlockToolResult, assistant.Content[1].Type)
	assert.Equal(t, "call-1", assistant.Content[1].ToolUseID)
	assert.Equal(t, "README.md\n", assistant.Content[1].Content)

	assert.Equal(t, core.BlockToolUse, assistant.Content[2].Type)
	assert.Equal(t, "apply_patch", assistant.Content[2].Name)
	assert.Equal(t, core.BlockToolResult, assistant.Content[3].Type)
	assert.Equal(t, "Success", assistant.Content[3].Content)
}

func TestApplyPatchDiffStats(t *testing.T) {
	tr := readTestdata(t, "apply_patch.jsonl")

	stats := core.ComputeDiffStats(tr)
	require.NotNil(t, stats)
	assert.Equal(t, 4, stats.Added)
	assert.Equal(t, 1, stats.Removed)
	assert.Equal(t, 2, stats.Changed)
}

func TestReasoningSummary(t *testing.T) {
	tr := readTestdata(t, "reasoning.jsonl")

	require.Len(t, tr.Messages, 2)
	require.Len(t, tr.Messages[1].Content, 2)
	assert.Equal(t, core.BlockThinking, tr.Messages[1].Content[0].Type)
	assert.Equal(t, "I considered the reader boundaries.", tr.Messages[1].Content[0].Text)
}

func TestSessionIndexTitle(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	require.NoError(t, os.MkdirAll(sessionsDir, 0o755))

	data, err := os.ReadFile(testdataPath("simple.jsonl"))
	require.NoError(t, err)
	path := filepath.Join(sessionsDir, "rollout-simple.jsonl")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "session_index.jsonl"),
		[]byte(`{"id":"codex-simple","thread_name":"Indexed title"}`+"\n"),
		0o644,
	))

	r := &Reader{Dir: sessionsDir}
	tr, err := r.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Indexed title", tr.Title)
}

func TestReadSessionProjectAndAll(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions", "2026", "05", "22")
	require.NoError(t, os.MkdirAll(sessionsDir, 0o755))

	data, err := os.ReadFile(testdataPath("simple.jsonl"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(sessionsDir, "rollout-2026-05-22T10-00-00-codex-simple.jsonl"),
		data,
		0o644,
	))

	other := strings.ReplaceAll(string(data), "codex-simple", "codex-other")
	other = strings.ReplaceAll(other, "/work/chitragupt", "/other/project")
	other = strings.ReplaceAll(other, "2026-05-22T10:00:00Z", "2026-05-22T11:00:00Z")
	require.NoError(t, os.WriteFile(
		filepath.Join(sessionsDir, "rollout-2026-05-22T11-00-00-codex-other.jsonl"),
		[]byte(other),
		0o644,
	))

	r := &Reader{Dir: filepath.Join(dir, "sessions")}

	tr, err := r.ReadSession("codex-simple")
	require.NoError(t, err)
	assert.Equal(t, "codex-simple", tr.SessionID)

	project, err := r.ReadProject("/work/chitragupt")
	require.NoError(t, err)
	require.Len(t, project, 1)
	assert.Equal(t, "codex-simple", project[0].SessionID)

	all, err := r.ReadAll()
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "codex-other", all[0].SessionID)
	assert.Equal(t, "codex-simple", all[1].SessionID)
}
