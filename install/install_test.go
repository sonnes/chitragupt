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
		Agent:   "claude",
		Formats: []string{"html"},
		Branch:  "transcripts",
		OutDir:  ".transcripts",
		Dir:     dir,
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

	t.Run("gitkeep exists in worktree", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".transcripts", ".gitkeep"))
		assert.NoError(t, err)
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

		// Script calls cg render with the correct agent and format
		script, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		assert.Contains(t, string(script), "cg render --agent claude --file")
		assert.Contains(t, string(script), "--format html")
		assert.Contains(t, string(script), "--out")
		assert.Contains(t, string(script), "cg manifest upsert")

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
		assert.Contains(t, string(data), "cg index")

		info, err := os.Stat(hookPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&0o100 != 0, "hook should be executable")
	})
}

func TestRunIdempotent(t *testing.T) {
	dir := initRepo(t)

	cfg := Config{
		Agent:   "claude",
		Formats: []string{"html"},
		Branch:  "transcripts",
		OutDir:  ".transcripts",
		Dir:     dir,
	}

	require.NoError(t, Run(cfg))

	// Second run should succeed (idempotent)
	require.NoError(t, Run(cfg))
}

func TestRunSimpleDirectoryMode(t *testing.T) {
	dir := initRepo(t)

	cfg := Config{
		Agent:   "claude",
		Formats: []string{"html"},
		OutDir:  ".transcripts",
		Dir:     dir,
	}

	require.NoError(t, Run(cfg))

	t.Run("output directory created", func(t *testing.T) {
		info, err := os.Stat(filepath.Join(dir, ".transcripts"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("no orphan branch", func(t *testing.T) {
		cmd := exec.Command("git", "rev-parse", "--verify", "transcripts")
		cmd.Dir = dir
		assert.Error(t, cmd.Run(), "orphan branch should not exist")
	})

	t.Run("not a worktree", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".transcripts", ".git"))
		assert.True(t, os.IsNotExist(err), ".git pointer should not exist in simple mode")
	})

	t.Run("gitignore updated", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Contains(t, string(data), ".transcripts/")
	})

	t.Run("claude hook installed", func(t *testing.T) {
		scriptPath := filepath.Join(dir, ".claude", "hooks", "save-transcript.sh")
		_, err := os.Stat(scriptPath)
		assert.NoError(t, err)
	})

	t.Run("no post-commit hook", func(t *testing.T) {
		hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
		_, err := os.Stat(hookPath)
		assert.True(t, os.IsNotExist(err), "post-commit hook should not be installed in simple mode")
	})
}

func TestRunSimpleDirectoryModeIdempotent(t *testing.T) {
	dir := initRepo(t)

	cfg := Config{
		Agent:   "claude",
		Formats: []string{"html"},
		OutDir:  ".transcripts",
		Dir:     dir,
	}

	require.NoError(t, Run(cfg))
	require.NoError(t, Run(cfg))
}

func TestEnsureGitignore(t *testing.T) {
	t.Run("creates .gitignore if missing", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ensureGitignore(dir, ".transcripts"))

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

		require.NoError(t, ensureGitignore(dir, ".transcripts"))

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

		require.NoError(t, ensureGitignore(dir, ".transcripts"))

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

		require.NoError(t, ensureGitignore(dir, ".transcripts"))

		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Equal(t, "node_modules/\n.transcripts/\n", string(data))
	})

	t.Run("custom directory name", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, ensureGitignore(dir, "my-output"))

		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Equal(t, "my-output/\n", string(data))
	})
}

func TestInstallClaudeHook(t *testing.T) {
	t.Run("creates settings from scratch", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

		require.NoError(t, installClaudeHook(dir, "claude", []string{"jsonl"}, ".transcripts"))

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

		require.NoError(t, installClaudeHook(dir, "claude", []string{"html"}, ".transcripts"))

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

		require.NoError(t, installClaudeHook(dir, "claude", []string{"jsonl"}, ".transcripts"))
		require.NoError(t, installClaudeHook(dir, "claude", []string{"jsonl"}, ".transcripts"))

		data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
		require.NoError(t, err)

		// Should only have one SessionEnd hook entry
		count := strings.Count(string(data), "save-transcript.sh")
		assert.Equal(t, 1, count)
	})

	t.Run("bakes formats into script", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))

		require.NoError(t, installClaudeHook(dir, "claude", []string{"html", "jsonl"}, ".transcripts"))

		script, err := os.ReadFile(filepath.Join(dir, ".claude", "hooks", "save-transcript.sh"))
		require.NoError(t, err)
		assert.Contains(t, string(script), "cg render --agent claude --file")
		assert.Contains(t, string(script), "--format html")
		assert.Contains(t, string(script), "--format jsonl")
		assert.Contains(t, string(script), "cg manifest upsert")
		assert.Contains(t, string(script), "index.html")
	})
}

func TestBuildSaveTranscriptScript(t *testing.T) {
	t.Run("single format", func(t *testing.T) {
		script := buildSaveTranscriptScript("claude", []string{"jsonl"}, ".transcripts")
		assert.Contains(t, script, "cg render --agent claude --file")
		assert.Contains(t, script, "--format jsonl")
		assert.Contains(t, script, "--out")
		assert.Contains(t, script, `DEST_DIR="$CLAUDE_PROJECT_DIR/.transcripts"`)
		assert.Contains(t, script, "cg manifest upsert")
		assert.Contains(t, script, "index.jsonl")
	})

	t.Run("multiple formats", func(t *testing.T) {
		script := buildSaveTranscriptScript("claude", []string{"html", "jsonl"}, ".transcripts")
		assert.Contains(t, script, "--format html")
		assert.Contains(t, script, "--format jsonl")
		assert.Contains(t, script, "--out")
	})

	t.Run("html preferred for href", func(t *testing.T) {
		script := buildSaveTranscriptScript("claude", []string{"jsonl", "html"}, ".transcripts")
		assert.Contains(t, script, "index.html", "href should prefer html even if not first")
	})

	t.Run("non-html href uses first format", func(t *testing.T) {
		script := buildSaveTranscriptScript("claude", []string{"jsonl"}, ".transcripts")
		assert.Contains(t, script, "index.jsonl")
	})

	t.Run("custom output directory", func(t *testing.T) {
		script := buildSaveTranscriptScript("claude", []string{"html"}, "my-docs")
		assert.Contains(t, script, `DEST_DIR="$CLAUDE_PROJECT_DIR/my-docs"`)
		assert.Contains(t, script, `"$CLAUDE_PROJECT_DIR/my-docs/manifest.json"`)
	})
}

func TestInstallPostCommitHook(t *testing.T) {
	t.Run("creates new hook file", func(t *testing.T) {
		dir := initRepo(t)
		require.NoError(t, installPostCommitHook(dir, ".transcripts"))

		data, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "post-commit"))
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(data), "#!/bin/bash\n"))
		assert.Contains(t, string(data), "cg-transcripts-start")
		assert.Contains(t, string(data), "cg index")
	})

	t.Run("appends to existing hook", func(t *testing.T) {
		dir := initRepo(t)
		hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
		require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o755))
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'existing'\n"), 0o755))

		require.NoError(t, installPostCommitHook(dir, ".transcripts"))

		data, err := os.ReadFile(hookPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "echo 'existing'")
		assert.Contains(t, string(data), "cg-transcripts-start")
	})

	t.Run("idempotent", func(t *testing.T) {
		dir := initRepo(t)
		require.NoError(t, installPostCommitHook(dir, ".transcripts"))
		require.NoError(t, installPostCommitHook(dir, ".transcripts"))

		data, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "post-commit"))
		require.NoError(t, err)
		count := strings.Count(string(data), "cg-transcripts-start")
		assert.Equal(t, 1, count)
	})

	t.Run("custom output directory", func(t *testing.T) {
		dir := initRepo(t)
		require.NoError(t, installPostCommitHook(dir, "my-docs"))

		data, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "post-commit"))
		require.NoError(t, err)
		assert.Contains(t, string(data), `WORKTREE="$REPO_ROOT/my-docs"`)
	})
}

func TestPostCommitHookAutoCommits(t *testing.T) {
	dir := initRepo(t)

	cfg := Config{
		Agent:   "claude",
		Formats: []string{"html"},
		Branch:  "transcripts",
		OutDir:  ".transcripts",
		Dir:     dir,
	}
	require.NoError(t, Run(cfg))

	// Simulate a transcript file being copied to the worktree
	transcriptDir := filepath.Join(dir, ".transcripts", "test-session")
	require.NoError(t, os.MkdirAll(transcriptDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(transcriptDir, "index.html"),
		[]byte("<html>test</html>"),
		0o644,
	))

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
