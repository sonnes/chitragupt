# chitragupt — Design Specification

## Overview

`cg` is a Go CLI that converts CLI agent session logs into shareable, human-readable transcripts. It reads session files from Claude Code, Codex, OpenCode, and Cursor, normalizes them into a standardized transcript format, and renders them as HTML, Markdown, JSON, or terminal output.

---

## CLI Interface

### Binary name

`cg`

### Commands

Single session

```
cg render --agent <name> --file <path>             Convert a session file to transcript
cg render --agent <name> --session <id>            Convert a session to transcript
```

Project

```
cg render --agent <name> --project <name>          Convert all sessions in a project to transcript
```

All sessions

```
cg render --agent <name> --all                     Convert all sessions to transcript
```

Output formats:

- HTML
- Markdown
- JSON (Standardized Transcript Format)
- Terminal (Pretty-printed output directly in the terminal using ANSI colors. No files written.)

When no `-o` is given, writes to standard output.

### `cg serve`

```
cg serve --agent <name> --all                serve all sessions browseable locally
cg serve --agent <name> --dir <dir>          serve a directory of sessions browseable locally

--port <port>   serve on a specific port
```

---

## Standardized Transcript Format

Canonical definitions: `core/schema.json` (JSON Schema) and `core/transcript.go` (Go structs).

All readers produce a `core.Transcript` and all renderers consume one.

### Transcript (top-level)

| Field              | Type       | Required | Description                                       |
| ------------------ | ---------- | -------- | ------------------------------------------------- |
| `session_id`       | string     | yes      | Unique session identifier (UUID)                  |
| `parent_session_id`| string     |          | Parent session ID (sub-agent sessions)            |
| `agent`            | enum       | yes      | `claude`, `codex`, `opencode`, `cursor` |
| `model`            | string     |          | Primary model used (e.g. `claude-opus-4-5-20251101`) |
| `dir`              | string     |          | Working directory                                 |
| `git_branch`       | string     |          | Git branch at session start                       |
| `title`            | string     |          | Human-readable title (first user prompt, truncated) |
| `created_at`       | datetime   | yes      | Session creation timestamp                        |
| `updated_at`       | datetime   |          | Last activity timestamp                           |
| `usage`            | Usage      |          | Aggregate token usage for the session             |
| `messages`         | Message[]  | yes      | Ordered conversation messages                     |

### Message

| Field         | Type           | Required | Description                              |
| ------------- | -------------- | -------- | ---------------------------------------- |
| `uuid`        | string         |          | Message ID from source log               |
| `parent_uuid` | string         |          | Parent message UUID (conversation tree)  |
| `role`        | enum           | yes      | `user`, `assistant`, `system`            |
| `model`       | string         |          | Model that produced this message         |
| `timestamp`   | datetime       |          | When the message was created             |
| `content`     | ContentBlock[] | yes      | Ordered content blocks                   |
| `usage`       | Usage          |          | Token usage for this message             |

### ContentBlock (discriminated by `type`)

**TextBlock** (`type: "text"`)
- `text` — the text content
- `format` — `"markdown"` (assistant output) or `"plain"` (user input)

**ThinkingBlock** (`type: "thinking"`)
- `text` — model reasoning (always plain text)

**ToolUseBlock** (`type: "tool_use"`)
- `tool_use_id` — links this call to its result
- `name` — tool name (e.g. `Bash`, `Read`, `Write`, `Edit`)
- `input` — tool-specific parameters (object)

**ToolResultBlock** (`type: "tool_result"`)
- `tool_use_id` — matches the corresponding `tool_use`
- `content` — textual tool output
- `is_error` — whether the tool invocation failed

### Usage

| Field                  | Type | Description              |
| ---------------------- | ---- | ------------------------ |
| `input_tokens`         | int  | Prompt tokens            |
| `output_tokens`        | int  | Completion tokens        |
| `cache_read_tokens`    | int  | Tokens read from cache   |
| `cache_creation_tokens`| int  | Tokens written to cache  |

### Example

```json
{
  "session_id": "8397fc7c-39b9-4e25-81da-ed47a574a88a",
  "agent": "claude",
  "model": "claude-opus-4-5-20251101",
  "dir": "/home/user/code/my-project",
  "git_branch": "main",
  "title": "Merge v2-spa worktree",
  "created_at": "2026-01-22T09:08:06Z",
  "updated_at": "2026-01-22T09:39:22Z",
  "usage": {
    "input_tokens": 5000,
    "output_tokens": 2000
  },
  "messages": [
    {
      "uuid": "e7ffa05c",
      "role": "user",
      "timestamp": "2026-01-22T09:08:06Z",
      "content": [
        { "type": "text", "format": "plain", "text": "Merge v2-spa worktree" }
      ]
    },
    {
      "uuid": "fe04a60f",
      "parent_uuid": "e7ffa05c",
      "role": "assistant",
      "model": "claude-opus-4-5-20251101",
      "timestamp": "2026-01-22T09:08:10Z",
      "content": [
        { "type": "thinking", "text": "The user wants to merge the v2-spa worktree..." },
        { "type": "text", "format": "markdown", "text": "I'll help you merge the v2-spa worktree." },
        { "type": "tool_use", "tool_use_id": "toolu_016Pe3ZR", "name": "Bash", "input": { "command": "git worktree list" } }
      ],
      "usage": { "input_tokens": 1500, "output_tokens": 200 }
    },
    {
      "uuid": "a124b572",
      "parent_uuid": "fe04a60f",
      "role": "user",
      "timestamp": "2026-01-22T09:08:14Z",
      "content": [
        { "type": "tool_result", "tool_use_id": "toolu_016Pe3ZR", "content": "/home/user/project  dd12a8d [main]\n/home/user/project/.worktrees/v2-spa  4034528 [v2-spa]", "is_error": false }
      ]
    }
  ]
}
```

---

## Project Structure

- `cmd/cg/main.go` — Entry point
- `core/` — Standardized Transcript Format types and JSON Schema
  - `transcript.go` — Go structs (`Transcript`, `Message`, `ContentBlock`, `Usage`)
  - `schema.json` — JSON Schema definition
- `reader/` — Parse agent-specific session files into `core.Transcript`
  - `reader/claude/` — Claude Code (JSONL in `~/.claude/projects/`)
  - `reader/codex/` — OpenAI Codex CLI (JSONL rollouts in `~/.codex/sessions/`)
  - `reader/opencode/` — OpenCode (SQLite in `~/.opencode/`)
  - `reader/cursor/` — Cursor (SQLite `state.vscdb` key-value store)
- `render/` — Render a `core.Transcript` to an output format
  - `render/html/` — HTML
  - `render/markdown/` — Markdown
  - `render/json/` — JSON (serialize transcript as-is)
  - `render/terminal/` — Pretty-print with ANSI colors
- `server/` — Local server to browse and render sessions on the fly

---

## Dependencies

| Library    | Purpose                            |
| ---------- | ---------------------------------- |
| `cli`      | CLI framework                      |
| `glamour`  | Terminal markdown rendering        |
| `goldmark` | Markdown to HTML conversion        |
| `chroma`   | Syntax highlighting                |
| `embed`    | Embed templates in binary (stdlib) |

---

## Build & Release

- `Makefile` for local dev: `make build`, `make test`, `make lint`
- GoReleaser for cross-platform binaries and Homebrew tap
- GitHub Actions CI: test on push, release on tag
