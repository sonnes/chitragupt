package install

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initRepo creates a temporary git repo with an initial commit and returns its path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run(), "setup: %v", args)
	}
	return dir
}

func TestRun(t *testing.T) {
	dir := initRepo(t)

	cfg := Config{
		Agent:  "claude",
		Format: "jsonl",
		Branch: "transcripts",
		Dir:    dir,
	}

	require.NoError(t, Run(cfg))

	t.Run("orphan branch exists", func(t *testing.T) {
		cmd := exec.Command("git", "rev-parse", "--verify", "transcripts")
		cmd.Dir = dir
		assert.NoError(t, cmd.Run())
	})

	t.Run("worktree exists", func(t *testing.T) {
		wtDir := filepath.Join(dir, ".transcripts")
		info, err := os.Stat(wtDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// .git file (worktree pointer) should exist
		gitFile := filepath.Join(wtDir, ".git")
		info, err = os.Stat(gitFile)
		require.NoError(t, err)
		assert.False(t, info.IsDir()) // file, not directory
	})

	t.Run("agent directory exists in worktree", func(t *testing.T) {
		agentDir := filepath.Join(dir, ".transcripts", "claude")
		info, err := os.Stat(agentDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("gitignore updated", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Contains(t, string(data), ".transcripts/")
	})

	t.Run("claude hook installed", func(t *testing.T) {
		// Hook script exists and is executable
		scriptPath := filepath.Join(dir, ".claude", "hooks", "save-transcript.sh")
		info, err := os.Stat(scriptPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o100 != 0, "script should be executable")

		// settings.json has the hook
		data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "SessionEnd")
		assert.Contains(t, string(data), "save-transcript.sh")
	})

	t.Run("post-commit hook installed", func(t *testing.T) {
		hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
		data, err := os.ReadFile(hookPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "cg-transcripts-start")
		assert.Contains(t, string(data), "cg-transcripts-end")

		info, err := os.Stat(hookPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o100 != 0, "hook should be executable")
	})
}

func TestRunIdempotent(t *testing.T) {
	dir := initRepo(t)

	cfg := Config{
		Agent:  "claude",
		Format: "jsonl",
		Branch: "transcripts",
		Dir:    dir,
	}

	require.NoError(t, Run(cfg))

	// Second run should fail because .transcripts/ already exists
	err := Run(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestEnsureGitignore(t *testing.T) {
	t.Run("creates .gitignore if missing", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ensureGitignore(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Equal(t, ".transcripts/\n", string(data))
	})

	t.Run("appends to existing .gitignore", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, ".gitignore"),
			[]byte("node_modules/\n"),
			0o644,
		))

		require.NoError(t, ensureGitignore(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Equal(t, "node_modules/\n.transcripts/\n", string(data))
	})

	t.Run("skips if already present", func(t *testing.T) {
		dir := t.TempDir()
		content := "node_modules/\n.transcripts/\n"
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, ".gitignore"),
			[]byte(content),
			0o644,
		))

		require.NoError(t, ensureGitignore(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("handles file without trailing newline", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, ".gitignore"),
			[]byte("node_modules/"),
			0o644,
		))

		require.NoError(t, ensureGitignore(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Equal(t, "node_modules/\n.transcripts/\n", string(data))
	})
}

func TestInstallClaudeHook(t *testing.T) {
	t.Run("creates settings from scratch", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

		require.NoError(t, installClaudeHook(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))
		assert.Contains(t, settings, "hooks")
	})

	t.Run("preserves existing settings", func(t *testing.T) {
		dir := t.TempDir()
		claudeDir := filepath.Join(dir, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0o755))

		existing := `{"permissions":{"allow":["Bash"]}}`
		require.NoError(t, os.WriteFile(
			filepath.Join(claudeDir, "settings.json"),
			[]byte(existing),
			0o644,
		))

		require.NoError(t, installClaudeHook(dir))

		data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))
		assert.Contains(t, settings, "permissions")
		assert.Contains(t, settings, "hooks")
	})

	t.Run("idempotent", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

		require.NoError(t, installClaudeHook(dir))
		require.NoError(t, installClaudeHook(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
		require.NoError(t, err)

		// Should only have one SessionEnd hook entry
		count := strings.Count(string(data), "save-transcript.sh")
		assert.Equal(t, 1, count)
	})
}

func TestInstallPostCommitHook(t *testing.T) {
	t.Run("creates new hook file", func(t *testing.T) {
		dir := initRepo(t)
		require.NoError(t, installPostCommitHook(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "post-commit"))
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(data), "#!/bin/bash\n"))
		assert.Contains(t, string(data), "cg-transcripts-start")
	})

	t.Run("appends to existing hook", func(t *testing.T) {
		dir := initRepo(t)
		hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
		require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o755))
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'existing'\n"), 0o755))

		require.NoError(t, installPostCommitHook(dir))

		data, err := os.ReadFile(hookPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "echo 'existing'")
		assert.Contains(t, string(data), "cg-transcripts-start")
	})

	t.Run("idempotent", func(t *testing.T) {
		dir := initRepo(t)
		require.NoError(t, installPostCommitHook(dir))
		require.NoError(t, installPostCommitHook(dir))

		data, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "post-commit"))
		require.NoError(t, err)
		count := strings.Count(string(data), "cg-transcripts-start")
		assert.Equal(t, 1, count)
	})
}

func TestPostCommitHookAutoCommits(t *testing.T) {
	dir := initRepo(t)

	cfg := Config{
		Agent:  "claude",
		Format: "jsonl",
		Branch: "transcripts",
		Dir:    dir,
	}
	require.NoError(t, Run(cfg))

	// Simulate a transcript file being copied to the worktree
	transcriptFile := filepath.Join(dir, ".transcripts", "claude", "test-session.jsonl")
	require.NoError(t, os.WriteFile(transcriptFile, []byte(`{"type":"user"}`+"\n"), 0o644))

	// Make a commit on the main branch — this triggers the post-commit hook
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", "trigger hook")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	// Check that the transcript was committed on the transcripts branch
	wtDir := filepath.Join(dir, ".transcripts")
	out, err := exec.Command("git", "-C", wtDir, "log", "--oneline").Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "transcripts @")
}
