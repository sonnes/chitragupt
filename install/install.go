// Package install sets up git infrastructure for storing agent session
// transcripts alongside a repository. It creates an orphan branch, a git
// worktree, Claude Code hooks for transcript capture, and a git post-commit
// hook for automatic commits.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config holds the settings for the install command.
type Config struct {
	Agent  string // agent name, e.g. "claude"
	Format string // transcript format, e.g. "jsonl"
	Branch string // orphan branch name, e.g. "transcripts"
	Dir    string // git repository root (auto-detected if empty)
}

// Run executes the full install sequence.
func Run(cfg Config) error {
	if cfg.Dir == "" {
		dir, err := gitRoot()
		if err != nil {
			return fmt.Errorf("not a git repository (run from inside a repo): %w", err)
		}
		cfg.Dir = dir
	}

	worktreeDir := filepath.Join(cfg.Dir, ".transcripts")

	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf(".transcripts/ already exists; run 'cg uninstall' first or remove it manually")
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"create orphan branch", func() error { return createOrphanBranch(cfg.Dir, cfg.Branch, cfg.Agent) }},
		{"add git worktree", func() error { return addWorktree(cfg.Dir, cfg.Branch, worktreeDir) }},
		{"update .gitignore", func() error { return ensureGitignore(cfg.Dir) }},
		{"install Claude Code hook", func() error { return installClaudeHook(cfg.Dir, cfg.Agent, cfg.Format) }},
		{"install git post-commit hook", func() error { return installPostCommitHook(cfg.Dir) }},
	}

	for _, s := range steps {
		if err := s.fn(); err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
	}

	return nil
}

// gitRoot returns the top-level directory of the current git repo.
func gitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// createOrphanBranch creates an empty orphan branch with an initial directory
// for the agent (e.g. claude/).
func createOrphanBranch(repoDir, branch, agent string) error {
	// Check if branch already exists
	if err := git(repoDir, "rev-parse", "--verify", branch); err == nil {
		return nil // branch exists, skip
	}

	// Create a temporary worktree to set up the orphan branch
	tmpDir, err := os.MkdirTemp("", "cg-orphan-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := git(repoDir, "worktree", "add", "--detach", tmpDir); err != nil {
		return fmt.Errorf("create temp worktree: %w", err)
	}
	defer func() { _ = git(repoDir, "worktree", "remove", "--force", tmpDir) }()

	// Inside the temp worktree, create the orphan branch
	if err := git(tmpDir, "checkout", "--orphan", branch); err != nil {
		return fmt.Errorf("checkout orphan: %w", err)
	}
	// Clear any tracked files from the index. Ignore errors when there are
	// no tracked files (e.g. the repo only has --allow-empty commits).
	_ = git(tmpDir, "rm", "-rf", "--ignore-unmatch", ".")

	// Create the agent directory with a .gitkeep
	agentDir := filepath.Join(tmpDir, agent)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(agentDir, ".gitkeep"), nil, 0o644); err != nil {
		return err
	}

	if err := git(tmpDir, "add", "."); err != nil {
		return fmt.Errorf("stage files: %w", err)
	}
	if err := git(tmpDir, "commit", "-m", "Initialize transcripts branch"); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	return nil
}

// addWorktree adds a git worktree at .transcripts/ pointing to the orphan branch.
func addWorktree(repoDir, branch, worktreeDir string) error {
	return git(repoDir, "worktree", "add", worktreeDir, branch)
}

// ensureGitignore adds .transcripts/ to .gitignore if not already present.
func ensureGitignore(repoDir string) error {
	path := filepath.Join(repoDir, ".gitignore")
	entry := ".transcripts/"

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == entry {
			return nil // already present
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline before entry if file doesn't end with one
	if len(data) > 0 && data[len(data)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(entry + "\n")
	return err
}

// claudeSettings represents the structure of .claude/settings.json relevant to hooks.
type claudeSettings struct {
	Hooks map[string][]matcherGroup `json:"hooks,omitempty"`
}

type matcherGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookHandler `json:"hooks"`
}

type hookHandler struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// installClaudeHook adds a SessionEnd hook to .claude/settings.json that renders
// the session transcript via `cg render` and writes it to .transcripts/<agent>/.
func installClaudeHook(repoDir, agent, format string) error {
	// Write the hook script
	hookDir := filepath.Join(repoDir, ".claude", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return err
	}

	scriptPath := filepath.Join(hookDir, "save-transcript.sh")
	script := buildSaveTranscriptScript(agent, format)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return err
	}

	// Update .claude/settings.json
	settingsPath := filepath.Join(repoDir, ".claude", "settings.json")
	var settings claudeSettings

	data, err := os.ReadFile(settingsPath)
	if err == nil {
		// Parse existing settings - preserve unknown fields by using a map
		_ = json.Unmarshal(data, &settings)
	}

	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]matcherGroup)
	}

	handler := hookHandler{
		Type:    "command",
		Command: `"$CLAUDE_PROJECT_DIR"/.claude/hooks/save-transcript.sh`,
	}

	// Check if hook already exists
	for _, mg := range settings.Hooks["SessionEnd"] {
		for _, h := range mg.Hooks {
			if h.Command == handler.Command {
				return nil // already installed
			}
		}
	}

	settings.Hooks["SessionEnd"] = append(settings.Hooks["SessionEnd"], matcherGroup{
		Hooks: []hookHandler{handler},
	})

	// Merge hooks into existing settings (preserve other fields)
	var fullSettings map[string]any
	if len(data) > 0 {
		_ = json.Unmarshal(data, &fullSettings)
	}
	if fullSettings == nil {
		fullSettings = make(map[string]any)
	}
	fullSettings["hooks"] = settings.Hooks

	out, err := json.MarshalIndent(fullSettings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0o644)
}

// installPostCommitHook installs or appends to the post-commit hook to
// auto-commit transcript files in the worktree when the user commits.
// Uses git rev-parse --git-common-dir to find the correct hooks directory,
// which works in both normal repos and worktrees.
func installPostCommitHook(repoDir string) error {
	gitDir, err := gitOutput(repoDir, "rev-parse", "--git-common-dir")
	if err != nil {
		return fmt.Errorf("find git dir: %w", err)
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoDir, gitDir)
	}
	hookPath := filepath.Join(gitDir, "hooks", "post-commit")

	hookDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(hookPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	marker := "# cg-transcripts-start"
	if strings.Contains(string(data), marker) {
		return nil // already installed
	}

	f, err := os.OpenFile(hookPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add shebang if file is new/empty
	if len(data) == 0 {
		if _, err := f.WriteString("#!/bin/bash\n"); err != nil {
			return err
		}
	} else if data[len(data)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	_, err = f.WriteString(postCommitHookScript)
	return err
}

// git runs a git command in the given directory.
func git(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr // show git output for debugging
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitOutput runs a git command and returns its stdout.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// formatExtension maps a render format to its file extension.
func formatExtension(format string) string {
	switch format {
	case "html":
		return ".html"
	case "markdown":
		return ".md"
	case "json":
		return ".json"
	default:
		return "." + format // e.g. "jsonl" → ".jsonl"
	}
}

// buildSaveTranscriptScript generates the hook script with the agent and format
// baked in so the SessionEnd hook calls `cg render` with the right flags.
func buildSaveTranscriptScript(agent, format string) string {
	ext := formatExtension(format)
	return fmt.Sprintf(`#!/bin/bash
# Installed by cg install — renders Claude Code session transcripts to .transcripts/
set -e

INPUT=$(cat)
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id')

if [ -z "$TRANSCRIPT_PATH" ] || [ "$TRANSCRIPT_PATH" = "null" ]; then
  exit 0
fi

if [ ! -f "$TRANSCRIPT_PATH" ]; then
  exit 0
fi

DEST_DIR="$CLAUDE_PROJECT_DIR/.transcripts/%s"
if [ ! -d "$DEST_DIR" ]; then
  exit 0
fi

DEST="$DEST_DIR/$SESSION_ID%s"
cg render --agent %s --file "$TRANSCRIPT_PATH" --format %s > "$DEST"
`, agent, ext, agent, format)
}

const postCommitHookScript = `
# cg-transcripts-start
# Auto-commit transcripts to the transcripts worktree.
# Installed by cg install.
REPO_ROOT="$(git rev-parse --show-toplevel)"
WORKTREE="$REPO_ROOT/.transcripts"
if [ -d "$WORKTREE/.git" ] || [ -f "$WORKTREE/.git" ]; then
  MAIN_SHA="$(git rev-parse --short HEAD)"
  # Unset GIT_DIR/GIT_INDEX_FILE so git -C operates on the worktree's own repo,
  # not the parent repo that triggered this hook.
  unset GIT_DIR GIT_INDEX_FILE GIT_WORK_TREE
  git -C "$WORKTREE" add -A 2>/dev/null
  git -C "$WORKTREE" diff --cached --quiet 2>/dev/null || \
    git -C "$WORKTREE" commit -m "transcripts @ $MAIN_SHA" --quiet 2>/dev/null || true
fi
# cg-transcripts-end
`
