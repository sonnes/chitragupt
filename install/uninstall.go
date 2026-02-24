package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UninstallConfig holds the settings for the uninstall command.
type UninstallConfig struct {
	OutDir string // output directory name, e.g. ".transcripts"
	Purge  bool   // also delete transcript data and orphan branch
	Dir    string // git repository root (auto-detected if empty)
}

// Uninstall reverses the actions of install.
// By default it removes hooks and configuration only.
// With Purge=true it also deletes transcript data and the orphan branch.
func Uninstall(cfg UninstallConfig) error {
	if cfg.Dir == "" {
		dir, err := gitRoot()
		if err != nil {
			return fmt.Errorf("not a git repository (run from inside a repo): %w", err)
		}
		cfg.Dir = dir
	}

	if cfg.OutDir == "" {
		cfg.OutDir = ".transcripts"
	}

	outPath := filepath.Join(cfg.Dir, cfg.OutDir)

	// Detect worktree branch before removing anything.
	branch := detectWorktreeBranch(cfg.Dir, outPath)

	if err := removeClaudeHook(cfg.Dir); err != nil {
		return fmt.Errorf("remove Claude Code hook: %w", err)
	}

	if err := removePostCommitHook(cfg.Dir); err != nil {
		return fmt.Errorf("remove post-commit hook: %w", err)
	}

	if err := removeGitignoreEntry(cfg.Dir, cfg.OutDir); err != nil {
		return fmt.Errorf("remove .gitignore entry: %w", err)
	}

	if err := removeWorktree(cfg.Dir, outPath); err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}

	if cfg.Purge {
		if err := os.RemoveAll(outPath); err != nil {
			return fmt.Errorf("remove output directory: %w", err)
		}

		if branch != "" {
			// Ignore error — branch may not exist or may have already been deleted.
			_ = git(cfg.Dir, "branch", "-D", branch)
		}
	}

	return nil
}

// detectWorktreeBranch returns the branch name for a worktree at outPath,
// or empty string if outPath is not a worktree.
func detectWorktreeBranch(repoDir, outPath string) string {
	out, err := gitOutput(repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return ""
	}

	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return ""
	}
	// Resolve symlinks so paths match git's canonical output (e.g. /tmp → /private/tmp on macOS).
	if resolved, err := filepath.EvalSymlinks(absOut); err == nil {
		absOut = resolved
	}

	// Parse porcelain output: blocks separated by blank lines.
	// Each block has: worktree <path>, HEAD <sha>, branch <ref>
	var currentWorktree string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			currentWorktree = strings.TrimPrefix(line, "worktree ")
		}
		if strings.HasPrefix(line, "branch ") && currentWorktree == absOut {
			ref := strings.TrimPrefix(line, "branch ")
			// refs/heads/transcripts → transcripts
			return strings.TrimPrefix(ref, "refs/heads/")
		}
	}

	return ""
}

// removeClaudeHook removes the save-transcript.sh script and its entry
// from .claude/settings.json.
func removeClaudeHook(repoDir string) error {
	// Remove the hook script.
	scriptPath := filepath.Join(repoDir, ".claude", "hooks", "save-transcript.sh")
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove the SessionEnd entry from settings.json.
	settingsPath := filepath.Join(repoDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var settings claudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil // can't parse, leave it alone
	}

	groups := settings.Hooks["SessionEnd"]
	if len(groups) == 0 {
		return nil
	}

	// Filter out matcher groups that contain our hook.
	var kept []matcherGroup
	for _, mg := range groups {
		var keptHooks []hookHandler
		for _, h := range mg.Hooks {
			if !strings.Contains(h.Command, "save-transcript.sh") {
				keptHooks = append(keptHooks, h)
			}
		}
		if len(keptHooks) > 0 {
			mg.Hooks = keptHooks
			kept = append(kept, mg)
		}
	}

	// Rebuild the full settings, preserving other fields.
	var fullSettings map[string]any
	_ = json.Unmarshal(data, &fullSettings)
	if fullSettings == nil {
		return nil
	}

	if len(kept) > 0 {
		settings.Hooks["SessionEnd"] = kept
	} else {
		delete(settings.Hooks, "SessionEnd")
	}

	if len(settings.Hooks) > 0 {
		fullSettings["hooks"] = settings.Hooks
	} else {
		delete(fullSettings, "hooks")
	}

	out, err := json.MarshalIndent(fullSettings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0o644)
}

// removePostCommitHook strips the cg-transcripts block from the post-commit hook.
// If the remaining file is empty or just a shebang, the file is deleted.
func removePostCommitHook(repoDir string) error {
	gitDir, err := gitOutput(repoDir, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil // not a git repo or no git dir — nothing to do
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoDir, gitDir)
	}
	hookPath := filepath.Join(gitDir, "hooks", "post-commit")

	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	startMarker := "# cg-transcripts-start"
	endMarker := "# cg-transcripts-end"

	content := string(data)
	startIdx := strings.Index(content, startMarker)
	if startIdx < 0 {
		return nil // not installed
	}

	endIdx := strings.Index(content, endMarker)
	if endIdx < 0 {
		return nil // malformed, leave it
	}

	// Find the start of the line containing the start marker.
	lineStart := strings.LastIndex(content[:startIdx], "\n")
	if lineStart < 0 {
		lineStart = 0
	}

	// Find the end of the line containing the end marker.
	lineEnd := endIdx + len(endMarker)
	if lineEnd < len(content) && content[lineEnd] == '\n' {
		lineEnd++
	}

	remaining := content[:lineStart] + content[lineEnd:]
	remaining = strings.TrimRight(remaining, "\n\t ")

	// If only a shebang (or empty) remains, delete the file.
	stripped := strings.TrimSpace(remaining)
	if stripped == "" || stripped == "#!/bin/bash" || stripped == "#!/bin/sh" {
		return os.Remove(hookPath)
	}

	return os.WriteFile(hookPath, []byte(remaining+"\n"), 0o755)
}

// removeGitignoreEntry removes the outDir entry from .gitignore.
func removeGitignoreEntry(repoDir, outDir string) error {
	path := filepath.Join(repoDir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	entry := strings.TrimSuffix(outDir, "/") + "/"
	lines := strings.Split(string(data), "\n")
	var kept []string
	for _, line := range lines {
		if strings.TrimSpace(line) != entry {
			kept = append(kept, line)
		}
	}

	result := strings.Join(kept, "\n")

	// If only whitespace remains, delete the file.
	if strings.TrimSpace(result) == "" {
		return os.Remove(path)
	}

	return os.WriteFile(path, []byte(result), 0o644)
}

// removeWorktree removes the git worktree at outPath if it is one.
func removeWorktree(repoDir, outPath string) error {
	gitFile := filepath.Join(outPath, ".git")
	info, err := os.Stat(gitFile)
	if err != nil || info.IsDir() {
		return nil // not a worktree
	}

	return git(repoDir, "worktree", "remove", "--force", outPath)
}
