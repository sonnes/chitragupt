package install

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstall(t *testing.T) {
	dir := initRepo(t)

	// Install in branch mode.
	require.NoError(t, Run(Config{
		Agent:   "claude",
		Formats: []string{"html"},
		Branch:  "transcripts",
		OutDir:  ".transcripts",
		Dir:     dir,
	}))

	// Uninstall without purge.
	require.NoError(t, Uninstall(UninstallConfig{
		OutDir: ".transcripts",
		Dir:    dir,
	}))

	t.Run("hook script removed", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "save-transcript.sh"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("settings.json cleaned", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "save-transcript.sh")
		assert.NotContains(t, string(data), "SessionEnd")
	})

	t.Run("post-commit hook removed", func(t *testing.T) {
		hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
		_, err := os.Stat(hookPath)
		assert.True(t, os.IsNotExist(err), "post-commit hook file should be deleted when only cg content")
	})

	t.Run("gitignore entry removed", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".gitignore"))
		// gitignore may be deleted if it only had our entry
		if err == nil {
			data, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
			assert.NotContains(t, string(data), ".transcripts/")
		}
	})

	t.Run("worktree removed", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".transcripts", ".git"))
		assert.True(t, os.IsNotExist(err), "worktree should be removed")
	})

	t.Run("branch still exists", func(t *testing.T) {
		cmd := exec.Command("git", "rev-parse", "--verify", "transcripts")
		cmd.Dir = dir
		assert.NoError(t, cmd.Run(), "branch should be preserved without --purge")
	})
}

func TestUninstallPurge(t *testing.T) {
	dir := initRepo(t)

	require.NoError(t, Run(Config{
		Agent:   "claude",
		Formats: []string{"html"},
		Branch:  "transcripts",
		OutDir:  ".transcripts",
		Dir:     dir,
	}))

	require.NoError(t, Uninstall(UninstallConfig{
		OutDir: ".transcripts",
		Purge:  true,
		Dir:    dir,
	}))

	t.Run("output directory deleted", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".transcripts"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("orphan branch deleted", func(t *testing.T) {
		cmd := exec.Command("git", "rev-parse", "--verify", "transcripts")
		cmd.Dir = dir
		assert.Error(t, cmd.Run(), "branch should be deleted with --purge")
	})
}

func TestUninstallSimpleMode(t *testing.T) {
	dir := initRepo(t)

	// Install in simple mode (no branch).
	require.NoError(t, Run(Config{
		Agent:   "claude",
		Formats: []string{"html"},
		OutDir:  ".transcripts",
		Dir:     dir,
	}))

	require.NoError(t, Uninstall(UninstallConfig{
		OutDir: ".transcripts",
		Dir:    dir,
	}))

	t.Run("hook script removed", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "save-transcript.sh"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("output directory preserved", func(t *testing.T) {
		info, err := os.Stat(filepath.Join(dir, ".transcripts"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestUninstallSimpleModePurge(t *testing.T) {
	dir := initRepo(t)

	require.NoError(t, Run(Config{
		Agent:   "claude",
		Formats: []string{"html"},
		OutDir:  ".transcripts",
		Dir:     dir,
	}))

	require.NoError(t, Uninstall(UninstallConfig{
		OutDir: ".transcripts",
		Purge:  true,
		Dir:    dir,
	}))

	t.Run("output directory deleted", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, ".transcripts"))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestUninstallIdempotent(t *testing.T) {
	dir := initRepo(t)

	require.NoError(t, Run(Config{
		Agent:   "claude",
		Formats: []string{"html"},
		Branch:  "transcripts",
		OutDir:  ".transcripts",
		Dir:     dir,
	}))

	require.NoError(t, Uninstall(UninstallConfig{OutDir: ".transcripts", Dir: dir}))
	require.NoError(t, Uninstall(UninstallConfig{OutDir: ".transcripts", Dir: dir}))
}

func TestUninstallNoInstall(t *testing.T) {
	dir := initRepo(t)
	require.NoError(t, Uninstall(UninstallConfig{OutDir: ".transcripts", Dir: dir}))
}

func TestRemovePostCommitHook(t *testing.T) {
	t.Run("preserves other hook content", func(t *testing.T) {
		dir := initRepo(t)

		hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
		require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o755))
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'existing'\n"), 0o755))

		require.NoError(t, installPostCommitHook(dir, ".transcripts"))
		require.NoError(t, removePostCommitHook(dir))

		data, err := os.ReadFile(hookPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "echo 'existing'")
		assert.NotContains(t, string(data), "cg-transcripts-start")
	})

	t.Run("deletes file when only cg content", func(t *testing.T) {
		dir := initRepo(t)
		require.NoError(t, installPostCommitHook(dir, ".transcripts"))
		require.NoError(t, removePostCommitHook(dir))

		hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
		_, err := os.Stat(hookPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestRemoveClaudeHook(t *testing.T) {
	t.Run("preserves other settings", func(t *testing.T) {
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
		require.NoError(t, removeClaudeHook(dir))

		data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))
		assert.Contains(t, settings, "permissions")
		assert.NotContains(t, settings, "hooks")
		assert.NotContains(t, string(data), "save-transcript.sh")
	})

	t.Run("removes hook script file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))
		require.NoError(t, installClaudeHook(dir, "claude", []string{"html"}, ".transcripts"))

		scriptPath := filepath.Join(dir, ".claude", "hooks", "save-transcript.sh")
		_, err := os.Stat(scriptPath)
		require.NoError(t, err)

		require.NoError(t, removeClaudeHook(dir))

		_, err = os.Stat(scriptPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestRemoveGitignoreEntry(t *testing.T) {
	t.Run("preserves other entries", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, ".gitignore"),
			[]byte("node_modules/\n.transcripts/\n"),
			0o644,
		))

		require.NoError(t, removeGitignoreEntry(dir, ".transcripts"))

		data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
		require.NoError(t, err)
		assert.Contains(t, string(data), "node_modules/")
		assert.NotContains(t, string(data), ".transcripts/")
	})

	t.Run("deletes file when only our entry", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, ".gitignore"),
			[]byte(".transcripts/\n"),
			0o644,
		))

		require.NoError(t, removeGitignoreEntry(dir, ".transcripts"))

		_, err := os.Stat(filepath.Join(dir, ".gitignore"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("no-op when file missing", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, removeGitignoreEntry(dir, ".transcripts"))
	})
}

func TestDetectWorktreeBranch(t *testing.T) {
	dir := initRepo(t)

	require.NoError(t, Run(Config{
		Agent:   "claude",
		Formats: []string{"html"},
		Branch:  "my-branch",
		OutDir:  ".transcripts",
		Dir:     dir,
	}))

	outPath := filepath.Join(dir, ".transcripts")
	branch := detectWorktreeBranch(dir, outPath)
	assert.Equal(t, "my-branch", branch)

	t.Run("returns empty for non-worktree", func(t *testing.T) {
		plainDir := t.TempDir()
		assert.Equal(t, "", detectWorktreeBranch(dir, plainDir))
	})
}
