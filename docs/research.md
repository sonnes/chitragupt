# Research: Storing Agent Session Logs in Git Repositories

## Executive Summary

This document surveys the landscape of tools and approaches for preserving AI agent session logs alongside code in git repositories. The primary approaches fall into three categories:

1. **External export tools** (Simon Willison's claude-code-transcripts, Will Larson's internal tool) -- extract sessions from `~/.claude/projects/` and publish as HTML/Gists
2. **Git-native session capture** (Entire CLI) -- hooks into git workflow to store sessions on an orphan branch alongside commits
3. **Claude Code hooks-based auto-save** (Jerad Bitner, various community scripts) -- use Claude Code's hook system to copy transcripts on session events

**Top recommendation**: The Entire CLI's architecture (orphan branch + worktree + Claude Code hooks) is the closest match to our `cg install` requirements. We should adopt the same core pattern -- orphan branch as a git worktree, Claude Code hooks for event capture -- but with a simpler, more focused implementation.

**Confidence level**: High

---

## 1. Claude Code Session Storage (Internal Format)

### Where sessions live

Claude Code stores all session data locally at:

```
~/.claude/projects/<project-path-hash>/
├── <session-uuid>.jsonl          # Full conversation transcript
├── <session-uuid>/
│   ├── session-memory/
│   │   └── summary.md            # Session summary
│   ├── subagents/
│   │   └── agent-<id>.jsonl      # Subagent transcripts
│   ├── file-history/             # File change tracking
│   ├── todos/                    # Task tracking
│   ├── session-env/              # Environment snapshots
│   └── debug/                    # Debug logs
├── memory/
│   ├── MEMORY.md                 # Project memory entrypoint
│   └── <topic>.md                # Topic-specific memory files
```

### Session JSONL format

Each line in the JSONL file contains a JSON object with:

| Field          | Description                                                |
|:---------------|:-----------------------------------------------------------|
| `parentUuid`   | Parent message UUID (for threading)                        |
| `sessionId`    | Session identifier                                         |
| `version`      | Schema version                                             |
| `gitBranch`    | Git branch at time of message                              |
| `cwd`          | Working directory                                          |
| `message`      | Object with `role` (user/assistant) and `content` array    |

Content arrays support `text`, `tool_use`, and `tool_result` block types.

### Key insight

Claude Code already keeps a complete JSON transcript of every session. The problem is not generation -- it is **preservation and association with code**. Sessions live in a user-local directory that is not version controlled, not portable, and not associated with specific commits.

---

## 2. Approaches Surveyed

### 2.1 Simon Willison's claude-code-transcripts

**Repository**: [simonw/claude-code-transcripts](https://github.com/simonw/claude-code-transcripts)

**How it works**:
- Python CLI tool (`uv tool install claude-code-transcripts`)
- Reads JSONL session files from `~/.claude/projects/` (or imports from Claude web API)
- Converts them to paginated, mobile-friendly HTML (index.html + page-001.html, page-002.html, etc.)
- Can publish to GitHub Gists with `--gist` flag for shareable URLs

**Commands**:
- `local` -- Interactive picker from local sessions (default)
- `web` -- Import from Claude API
- `json` -- Convert specific JSON/JSONL files
- `all` -- Batch convert entire session archive

**Storage**: Output HTML files, either local filesystem or GitHub Gists. No git-native storage.

**Git integration**: Minimal. Can auto-detect associated GitHub repo to generate commit links with `--repo OWNER/NAME`. Published transcripts link to commits but are not stored in the repository itself.

**Pros**:
- Beautiful, shareable HTML output
- Works retroactively on any existing session
- Gist publishing for easy sharing
- Active maintenance, good community adoption

**Cons**:
- Manual, after-the-fact process (not automatic)
- Sessions are not stored alongside the code in git
- No association between sessions and specific commits
- Requires separate hosting/publishing step
- Python dependency

### 2.2 Will Larson's Internal Approach

**Source**: [Sharing Claude transcripts](https://lethain.com/sharing-claude-transcripts/) (lethain.com)

**How it works**:
- Built on Simon Willison's claude-code-transcripts tool
- Internal CLI command `imp claude share-session` selects and merges sessions
- Sessions stored in an internal repository with a Cloudflare Pages viewer behind SSO
- Focused on organizational knowledge sharing ("discovery of what's possible and what's good")

**Storage**: Internal git repository, but details of the git integration are not specified.

**Git integration**: Sessions are merged "into the holding repository" -- unclear if this is automated or manual.

**Pros**:
- Team-oriented: surfaces advanced prompting practices
- SSO-protected viewer
- Leverages existing tooling

**Cons**:
- Internal/proprietary, not open source
- Manual sharing process
- Separate repository from the code itself

### 2.3 Entire CLI

**Repository**: [entireio/cli](https://github.com/entireio/cli)  
**Blog post**: [Entire CLI: Version Control for Your Agent Sessions](https://www.mager.co/blog/2026-02-10-entire-cli/)

**How it works**:
- Go CLI tool (created by ex-GitHub CEO)
- `entire enable` installs git hooks and creates configuration
- Captures AI agent sessions (prompts, responses, files touched, token usage)
- Stores everything on a separate orphan branch `entire/checkpoints/v1`
- Agent-agnostic (Claude Code, Gemini CLI)

**Two strategies**:

| Strategy        | Behavior                                                              |
|:----------------|:----------------------------------------------------------------------|
| Manual-commit   | Default. Prompts to link sessions when pushing. Shadow branches per commit hash. |
| Auto-commit     | Creates a checkpoint after every agent response. Fine-grained save points.       |

**Directory structure on checkpoint branch**:
```
<checkpoint-id[:2]>/<checkpoint-id[2:]>/
├── metadata.json
├── 0/                    # First session in checkpoint
│   ├── full.jsonl        # Full transcript
│   └── context.md        # Condensed context
```

Temporary (pre-commit) metadata:
```
.entire/metadata/<session-id>/
├── full.jsonl
├── prompt.txt
└── tasks/<tool-use-id>/
```

**Git integration**: Deep. Uses orphan branch for storage, git hooks for capture, worktree-aware branching (each worktree gets its own shadow branch namespace). Session phase state machine (ACTIVE -> ACTIVE_COMMITTED -> IDLE -> ENDED) manages lifecycle.

**Internal architecture**:
- Go-based, uses `go-git` library (with CLI fallbacks for buggy operations)
- Strategy pattern for manual vs. auto commit
- Checkpoint storage abstraction layer
- Session state machine with event-driven transitions
- Settings package to avoid import cycles

**Pros**:
- Most complete solution in the ecosystem
- Git-native: everything stays in the repo, no external services
- Agent-agnostic
- Clean separation via orphan branch (no pollution of main history)
- Worktree-aware
- Active development, funded project

**Cons**:
- Complex: session state machine, multiple strategies, checkpoint sharding
- Heavy: full platform with `explain`, `rewind`, `resume` commands
- Opinionated about directory structure (.entire/)
- Significant accidental complexity for what is fundamentally "copy JSONL to a branch"
- External dependency (separate CLI tool to install)

### 2.4 Jerad Bitner's Auto-Save Approach

**Source**: [Never Lose a Claude Code Conversation Again](https://jeradbitner.com/blog/claude-code-auto-save-conversations)

**How it works**:
- Claude Code plugin with a Stop hook (~10 lines)
- Hook extracts `transcript_path` and `session_id` from Claude Code's event data
- Invokes a skill that saves transcripts as JSONL + Markdown + metadata JSON

**Storage**: `~/.claude/conversation-logs/` with three files per session (JSONL, markdown, metadata). Symlink to latest conversation.

**Git integration**: None. Local file storage only.

**Pros**:
- Very simple implementation
- Automatic (fires on every Stop event)
- Produces readable markdown alongside raw JSONL
- Skill/plugin separation for reusability

**Cons**:
- No git integration
- Sessions not associated with commits
- Local-only storage
- Requires Claude Code plugin system

### 2.5 Other Tools

| Tool | Description |
|:-----|:------------|
| [claude-code-log](https://github.com/daaain/claude-code-log) | Python CLI converting JSONL to HTML/Markdown |
| [claude-conversation-extractor](https://github.com/ZeroSumQuant/claude-conversation-extractor) | Extracts clean conversation logs from Claude storage |
| [clog](https://github.com/HillviewCap/clog) | Web-based viewer for Claude Code logs with real-time monitoring |
| [claude-sessions](https://github.com/iannuttall/claude-sessions) | Custom slash commands for session tracking |

None of these integrate with git for storage.

---

## 3. Git Orphan Branches + Worktrees

### Orphan branches

An orphan branch is a branch with no parent commits -- a completely separate history tree within the same repository.

```bash
# Create orphan branch
git checkout --orphan transcripts
git rm -rf .
git commit --allow-empty -m "Initialize transcripts branch"
git push origin transcripts
git checkout main
```

**Key properties**:
- Shares the same `.git` directory as the main branch
- Completely independent commit history
- Does not appear in `git log` of main branch
- Push/pull/fetch work normally
- Common use case: GitHub Pages (`gh-pages` branch)

### Git worktrees

A worktree allows checking out a different branch into a separate directory, backed by the same `.git` repository.

```bash
# Add worktree for the orphan branch
git worktree add .transcripts transcripts
```

**Key properties**:
- Separate working directory, same repo
- Changes in the worktree commit to the worktree's branch
- Multiple branches can be "active" simultaneously
- The worktree directory can be gitignored from the main branch
- Worktree metadata stored in `.git/worktrees/<name>/`

### Combined pattern (orphan + worktree)

This is the established pattern for storing separate content in the same repo:

```bash
# One-time setup
git checkout --orphan transcripts
git rm -rf .
git commit --allow-empty -m "Initialize transcripts branch"
git push origin transcripts
git checkout main

# Create worktree
git worktree add -B transcripts .transcripts origin/transcripts

# Add .transcripts to .gitignore
echo ".transcripts/" >> .gitignore
```

After this:
- `.transcripts/` is a fully functional working directory on the `transcripts` branch
- Files added/committed in `.transcripts/` go to the `transcripts` branch
- The main branch's history is unaffected
- `git push` from `.transcripts/` pushes to `origin/transcripts`

This is exactly the pattern used by GitHub Pages (`gh-pages`), Hugo static sites, and now Entire CLI for session storage.

---

## 4. Claude Code Hooks System

### Overview

Claude Code hooks are user-defined event handlers that fire at specific lifecycle points. They are configured in `.claude/settings.json` (project-level, committable) or `~/.claude/settings.json` (user-level).

### Relevant hook events for session capture

| Event        | When                           | Receives                                          | Can block? |
|:-------------|:-------------------------------|:--------------------------------------------------|:-----------|
| `Stop`       | Agent finishes responding      | `session_id`, `transcript_path`, `cwd`            | Yes        |
| `SessionEnd` | Session terminates             | `session_id`, `transcript_path`, `cwd`, `reason`  | No         |
| `PostToolUse`| After any tool succeeds        | `tool_name`, `tool_input`, `tool_response`        | No         |

### SessionEnd hook (recommended for transcript capture)

The `SessionEnd` event fires when Claude Code exits (user exit, sigint, error) and provides:

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../session-uuid.jsonl",
  "cwd": "/Users/user/project",
  "permission_mode": "default",
  "hook_event_name": "SessionEnd",
  "reason": "prompt_input_exit"
}
```

This is ideal because:
- Fires exactly once per session (at end)
- Provides `transcript_path` to the complete JSONL file
- Cannot block termination (safe -- no risk of hanging)
- `reason` field indicates clean vs. error exit

### Stop hook (alternative for mid-session capture)

The `Stop` event fires when Claude finishes a response turn. Useful for auto-commit strategies where you want checkpoints during a session, not just at the end.

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../session-uuid.jsonl",
  "cwd": "/Users/user/project",
  "hook_event_name": "Stop",
  "stop_hook_active": false
}
```

### Git hooks (standard git, not Claude Code hooks)

Standard git hooks can complement Claude Code hooks:

| Hook          | Trigger                  | Use case                                    |
|:--------------|:-------------------------|:--------------------------------------------|
| `post-commit` | After `git commit`       | Auto-commit transcripts when user commits   |
| `pre-push`    | Before `git push`        | Ensure transcripts are pushed alongside code|

**Note**: Claude Code does not have native `PreCommit`/`PostCommit` hook events (there is an [open feature request](https://github.com/anthropics/claude-code/issues/4834)). To hook into git commit events, use standard git hooks (`.git/hooks/post-commit`), not Claude Code hooks.

### Hook configuration format

```json
{
  "hooks": {
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/save-transcript.sh"
          }
        ]
      }
    ]
  }
}
```

---

## 5. Comparison Matrix

| Feature                          | Willison | Larson | Entire CLI | Bitner | **cg install (proposed)** |
|:---------------------------------|:---------|:-------|:-----------|:-------|:--------------------------|
| Automatic capture                | No       | No     | Yes        | Yes    | **Yes**                   |
| Git-native storage               | No       | Partial| Yes        | No     | **Yes**                   |
| Orphan branch isolation          | No       | No     | Yes        | No     | **Yes**                   |
| Worktree support                 | No       | No     | Yes        | No     | **Yes**                   |
| Commit association               | Links    | No     | Yes        | No     | **Yes**                   |
| Agent-agnostic                   | Partial  | No     | Yes        | No     | **Planned**               |
| Claude Code hooks integration    | No       | No     | Yes        | Yes    | **Yes**                   |
| Readable output (HTML/MD)        | Yes      | Yes    | No         | Yes    | **Yes (via cg render)**   |
| Complexity                       | Low      | Medium | High       | Low    | **Low-Medium**            |
| External dependency              | Python   | Custom | Go binary  | Plugin | **None (built into cg)**  |

---

## 6. Recommendation for `cg install`

### Architecture

Adopt the Entire CLI's core pattern (orphan branch + worktree) but with dramatically reduced complexity:

```
cg install --agent claude --format jsonl --branch transcripts
```

This command should:

1. **Create an orphan branch** named `transcripts` (configurable via `--branch`)
2. **Set up a git worktree** at `.transcripts/` pointing to that branch
3. **Add `.transcripts/` to `.gitignore`** (so the main branch ignores it)
4. **Install a Claude Code hook** (SessionEnd) in `.claude/settings.json` that copies the session JSONL to `.transcripts/claude/<session-id>.jsonl`
5. **Install a git post-commit hook** that auto-commits pending transcripts in the worktree

### Directory structure on the transcripts branch

```
.transcripts/
└── claude/
    ├── <session-id-1>.jsonl
    ├── <session-id-2>.jsonl
    └── ...
```

Keep it flat and simple. No checkpoint sharding, no metadata.json wrappers, no state machines. The JSONL files are already self-contained and timestamped.

### Hook implementation

**Claude Code SessionEnd hook** (`.claude/settings.json`):
```json
{
  "hooks": {
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/save-transcript.sh"
          }
        ]
      }
    ]
  }
}
```

**save-transcript.sh** (simplified pseudocode):
```bash
#!/bin/bash
INPUT=$(cat)
TRANSCRIPT_PATH=$(echo "$INPUT" | jq -r '.transcript_path')
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id')
DEST="$CLAUDE_PROJECT_DIR/.transcripts/claude/$SESSION_ID.jsonl"

# Copy transcript to worktree
mkdir -p "$(dirname "$DEST")"
cp "$TRANSCRIPT_PATH" "$DEST"
```

**Git post-commit hook** (`.git/hooks/post-commit`):
```bash
#!/bin/bash
WORKTREE="$(git rev-parse --show-toplevel)/.transcripts"
if [ -d "$WORKTREE" ]; then
  cd "$WORKTREE"
  git add -A
  MAIN_COMMIT=$(git -C "$(git rev-parse --show-toplevel)" rev-parse --short HEAD)
  git commit -m "Transcripts for commit $MAIN_COMMIT" --allow-empty 2>/dev/null || true
fi
```

### Why this approach

**Essential complexity only**:
- Orphan branch: necessary to isolate transcript history from code history
- Worktree: necessary to have a working directory for the branch without switching branches
- SessionEnd hook: necessary to capture transcripts automatically
- Post-commit hook: necessary to associate transcripts with commits

**Accidental complexity avoided** (things Entire CLI has that we do not need):
- Session state machine (ACTIVE -> COMMITTED -> IDLE -> ENDED)
- Checkpoint sharding (ID[:2]/ID[2:]/)
- Shadow branches per commit hash
- Strategy pattern (manual vs. auto)
- `rewind`, `resume`, `explain` commands
- Metadata.json wrappers
- Condensation pipeline
- go-git library with CLI fallbacks

**The 80/20**: The session JSONL file is already the complete record. Copying it to a worktree directory and committing it on a post-commit hook gives us 80% of the value with 20% of the complexity. Users who need `rewind`, `explain`, etc. can use `cg render` on the JSONL files.

### Open questions for the team

1. **Should `cg install` also push the transcripts branch to the remote?** Recommendation: not by default, but `cg install --push` could set up a post-push hook.

2. **Should we support a Stop hook (mid-session capture) in addition to SessionEnd?** Recommendation: start with SessionEnd only. Mid-session capture adds complexity (partial transcripts, deduplication) for marginal benefit.

3. **Should the post-commit hook also run `cg compact` to create a compact transcript alongside the raw JSONL?** Recommendation: yes, this adds value and `cg` already has the compact transformer.

4. **How do we handle multiple agents?** The `--agent claude` flag creates `claude/` subdirectory. Future `--agent cursor` or `--agent copilot` would create `cursor/`, `copilot/`, etc. Each agent would have its own hook mechanism.

5. **Should `.claude/settings.json` changes be committed?** Recommendation: yes, the hook configuration should be shared with the team. This matches Claude Code's design where `.claude/settings.json` is committable.

6. **What happens if the worktree is missing (e.g., fresh clone)?** Need a `cg install --restore` or automatic detection to re-create the worktree from the existing `transcripts` branch.

---

## 7. Sources & References

### Primary sources

- [simonw/claude-code-transcripts](https://github.com/simonw/claude-code-transcripts) -- Simon Willison's transcript export tool
- [Simon Willison's blog post](https://simonwillison.net/2025/Dec/25/claude-code-transcripts/) -- Original announcement
- [entireio/cli](https://github.com/entireio/cli) -- Entire CLI source code
- [Entire CLI blog post](https://www.mager.co/blog/2026-02-10-entire-cli/) -- Architecture overview
- [Sharing Claude transcripts (Will Larson)](https://lethain.com/sharing-claude-transcripts/) -- Internal tooling approach
- [Never Lose a Claude Code Conversation Again (Jerad Bitner)](https://jeradbitner.com/blog/claude-code-auto-save-conversations) -- Auto-save approach

### Claude Code documentation

- [Hooks reference](https://code.claude.com/docs/en/hooks) -- Full hook events, input schemas, decision control
- [Automate workflows with hooks](https://code.claude.com/docs/en/hooks-guide) -- Practical hook guide
- [Manage Claude's memory](https://code.claude.com/docs/en/memory) -- Session storage and CLAUDE.md

### Git techniques

- [git-worktree documentation](https://git-scm.com/docs/git-worktree) -- Official reference
- [Git orphan branches guide (Graphite)](https://graphite.com/guides/git-orphan-branches) -- Orphan branch explanation
- [Git orphan worktree gist](https://gist.github.com/fabito/1f28df3e775b4095b3abd5e4211f814e) -- Practical setup example
- [Git for Static Sites (Kris Jenkins)](https://blog.jenkster.com/2016/02/git-for-static-sites.html) -- Worktree + orphan pattern for gh-pages

### Community tools

- [claude-code-log](https://github.com/daaain/claude-code-log) -- JSONL to HTML/Markdown converter
- [clog](https://github.com/HillviewCap/clog) -- Web-based Claude Code log viewer
- [claude-conversation-extractor](https://github.com/ZeroSumQuant/claude-conversation-extractor) -- Session extraction tool
- [claude-code-hooks-mastery](https://github.com/disler/claude-code-hooks-mastery) -- Hooks examples and patterns

### Related discussions

- [PreCommit/PostCommit hook feature request](https://github.com/anthropics/claude-code/issues/4834) -- Proposed git lifecycle hooks for Claude Code
- [Claude Code settings.json guide](https://www.eesel.ai/blog/settings-json-claude-code) -- Settings configuration
