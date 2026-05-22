---
title: "Codex Reader Design Specification"
summary: "Plan for adding OpenAI Codex rollout JSONL support to Chitragupt"
read_when:
  - Adding Codex CLI session support
  - Debugging Codex rollout JSONL parsing
  - Updating agent reader documentation for Codex
---

# Codex Reader — Design Specification

## Overview

Add a `reader/codex` package that parses OpenAI Codex rollout JSONL files from
`~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` into `core.Transcript`, then
register it as `--agent codex`.

The implementation should preserve Chitragupt's existing boundaries:

```text
Codex rollout JSONL -> reader/codex -> core.Transcript -> transforms -> renderers
```

The first release should support terminal, Markdown, and HTML output through
the existing renderers without adding Codex-specific renderer behavior.

## Research Summary

### Local Session Observations

I sampled local `~/.codex/sessions` files without recording private prompt or
tool-output content. The observed row envelope is:

```json
{
  "timestamp": "RFC3339 timestamp",
  "type": "session_meta | turn_context | response_item | event_msg | compacted",
  "payload": {}
}
```

Across 28 local rollout files:

| Row type | Count | Role in reader |
| --- | ---: | --- |
| `response_item` | 3593 | Canonical assistant, reasoning, tool calls, and tool outputs |
| `event_msg` | 1752 | UI replay events and token/task metadata |
| `turn_context` | 177 | Per-turn model/cwd/sandbox metadata |
| `session_meta` | 52 | Session ID, cwd, git, source, CLI version |
| `compacted` | 1 | Conversation compaction marker |

Important `response_item.payload.type` values observed:

| Payload type | Count | Notes |
| --- | ---: | --- |
| `function_call` | 1153 | Tool call with `name`, JSON-string `arguments`, and `call_id` |
| `function_call_output` | 1153 | Tool output keyed by `call_id`; usually string output |
| `reasoning` | 558 | Reasoning summary plus encrypted payload; only summary is displayable |
| `message` | 504 | User/developer/assistant message content |
| `custom_tool_call` | 104 | `apply_patch` calls in local samples |
| `custom_tool_call_output` | 104 | Output for custom tool calls |
| `web_search_call` | 15 | Web search action/status |
| `tool_search_call` / `tool_search_output` | 1 each | Deferred tool discovery |

Important `event_msg.payload.type` values observed:

| Payload type | Count | Notes |
| --- | ---: | --- |
| `token_count` | 895 | Source for token usage totals |
| `agent_message` | 320 | UI replay copy of assistant text |
| `agent_reasoning` | 110 | UI replay reasoning text |
| `user_message` | 100 | Human-visible user prompt |
| `task_started` / `task_complete` | 193 total | Turn lifecycle metadata |
| `patch_apply_end` | 89 | Patch status and change summaries |
| `mcp_tool_call_end` | 32 | MCP result metadata |
| `web_search_end` | 15 | Web search query/action metadata |

The key display decision from local samples: a user turn may have two nearby
representations. `response_item.message` with role `user` can include injected
project and environment context, while `event_msg.user_message` contains the
short human-authored prompt. Prefer `event_msg.user_message` for rendered user
messages when it exists, with `response_item.message` user content as fallback.

### External Prior Art

- OpenAI Codex source defines rollout rows as `SessionMeta`, `ResponseItem`,
  `Compacted`, `TurnContext`, and `EventMsg` variants:
  <https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs>
- OpenAI issue discussion confirms Codex writes per-session JSONL logs under
  `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl`:
  <https://github.com/openai/codex/issues/2288>
- OpenAI discussion confirms rollout filenames and internal session IDs are
  auto-generated and `/resume` relies on IDs stored in JSONL:
  <https://github.com/openai/codex/discussions/3827>
- `codlogs` is a read-only Codex session export/browser project. Useful lessons:
  stream large JSONL files, expose `--codex-home`, handle Windows/WSL path
  aliases, and treat sanitization as a derived copy:
  <https://github.com/tobitege/codlogs>
- `codex-replay` converts Codex JSONL sessions to self-contained HTML replays,
  confirming demand for the same sharing workflow Chitragupt already supports:
  <https://github.com/zpdldhkdl/codex-replay>
- `coding_agent_session_search` normalizes Codex rollout JSONL alongside other
  agent formats, reinforcing Chitragupt's reader/core/renderer split:
  <https://github.com/Dicklesworthstone/coding_agent_session_search>

## Scope

### In Scope

- `cg render --agent codex --file PATH`
- `cg render --agent codex --session ID`
- `cg render --agent codex --project PATH`
- `cg render --agent codex --all`
- Codex metadata mapping: session ID, cwd, git branch, model, timestamps, usage
- Human prompt extraction with injected-context suppression
- Assistant commentary/final messages
- Reasoning summaries as `thinking` blocks
- Tool calls/results for shell, patch, browser, MCP, web search, tool search,
  image view, and unknown tools
- Existing redaction, compaction, diff stats, and renderer support
- Documentation and examples for Codex usage

### Out of Scope for First Release

- Reading Codex SQLite state as the source of truth
- Editing Codex session titles or `session_index.jsonl`
- Full Codex resume/replay compatibility
- Re-rendering Codex sub-agent sessions as linked `SubAgents`
- Decompressing `.jsonl.zst` files unless local fixtures show they are common

## Raw Data Model

Create raw types in `reader/codex`:

```go
type rawLine struct {
    Timestamp string          `json:"timestamp"`
    Type      string          `json:"type"`
    Payload   json.RawMessage `json:"payload"`
}

type sessionMeta struct {
    ID            string     `json:"id"`
    Timestamp     string     `json:"timestamp"`
    CWD           string     `json:"cwd"`
    Originator    string     `json:"originator"`
    CLIVersion    string     `json:"cli_version"`
    Source        string     `json:"source"`
    ModelProvider string     `json:"model_provider"`
    Git           *gitMeta   `json:"git"`
}

type responseItem struct {
    Type    string            `json:"type"`
    Role    string            `json:"role"`
    Phase   string            `json:"phase"`
    Content []rawContentItem  `json:"content"`
    Name    string            `json:"name"`
    Args    string            `json:"arguments"`
    CallID  string            `json:"call_id"`
    Output  json.RawMessage   `json:"output"`
}
```

Keep the raw structs permissive. Codex is changing quickly, and unknown fields
should not fail parsing.

## Reader Behavior

### File Discovery

Default root:

```text
$CODEX_HOME/sessions
~/.codex/sessions
```

Discovery rules:

1. `ReadFile(path)` parses the exact file.
2. `ReadSession(sessionID)` walks `sessions/**/rollout-*.jsonl` and returns the
   file whose `session_meta.id` matches, falling back to filename contains ID.
3. `ReadProject(project)` parses sessions whose `session_meta.cwd` is the
   requested absolute project path, or whose cwd is below it.
4. `ReadAll()` walks all rollout JSONL files under the root.

Sort bulk reads newest-first by `session_meta.timestamp` or file mtime.

### Scanning

Use `bufio.Scanner` with the same large line buffer pattern as
`reader/claude`, but consider increasing the cap from 10 MB to 64 MB or using
`bufio.Reader.ReadBytes('\n')` because Codex sessions can contain large tool
outputs and images.

Malformed lines should be skipped with best-effort behavior only if at least
one valid message remains. A completely unparseable file should return an
error.

### Message Reconstruction

Process rows in file order and build a flat `[]core.Message`.

1. Capture `session_meta` as transcript metadata.
2. Capture each `turn_context` by `turn_id`; use its `model`, `cwd`, and
   effort metadata where useful.
3. On `event_msg.user_message`, append a `core.RoleUser` message containing
   the visible human prompt. Include images as placeholder text until
   `core.ContentBlock` supports image blocks.
4. On `response_item.message`:
   - role `assistant`: append assistant text as markdown.
   - role `user`: use only as fallback when there is no nearby
     `event_msg.user_message`.
   - role `developer` or `system`: skip by default for shareable output, but
     keep a narrow fallback option to map to `core.RoleSystem` if future docs
     require internal-context transcripts.
5. On `response_item.reasoning`, append a `core.BlockThinking` with joined
   `summary_text` entries. Ignore `encrypted_content`.
6. On `response_item.function_call` or `custom_tool_call`, append a
   `core.BlockToolUse` with:
   - `ToolUseID`: `call_id`
   - `Name`: function/tool name
   - `Input`: parsed JSON object from `arguments` when valid; raw string
     fallback otherwise
7. On matching output rows, append a `core.BlockToolResult` with the same
   `call_id`.
8. On `web_search_call` and `tool_search_call`, map them to `BlockToolUse`
   with names `web_search` and `tool_search`.
9. On `event_msg.patch_apply_end`, prefer enriching the matching `apply_patch`
   result if no `custom_tool_call_output` exists; otherwise ignore to avoid
   duplicate patch output.
10. On `compacted`, append a brief assistant text or thinking block indicating
    context was compacted. Do not expand replacement history in first release.

Assistant commentary and final-answer phases should remain assistant messages.
The existing `core.GroupTurns` logic can then split turns using user messages
and fold tool-result-only rows naturally.

### Title Derivation

Prefer sources in this order:

1. `~/.codex/session_index.jsonl` row with matching `id` and `thread_name`.
2. First `event_msg.user_message` text.
3. First fallback user `response_item.message` after stripping obvious Codex
   scaffolding blocks such as AGENTS/environment context.

Truncate with the existing Claude-style word-boundary behavior.

### Usage Mapping

Use `event_msg.token_count.payload.info.last_token_usage` for per-message or
per-turn usage where possible, and `total_token_usage` for session aggregate.
Observed fields map cleanly:

| Codex field | `core.Usage` field |
| --- | --- |
| `input_tokens` | `InputTokens` |
| `output_tokens` | `OutputTokens` |
| `cached_input_tokens` | `CacheReadTokens` |
| `reasoning_output_tokens` | Include in `OutputTokens` only if Codex total does |

Do not invent a schema field for reasoning tokens in the first release. If
users need that distinction later, extend `core.Usage` and `core/schema.json`
in a separate change.

### Git Metadata

Map `session_meta.git.branch` to `Transcript.GitBranch`. Keep
`repository_url` and `commit_hash` out of `core.Transcript` for now because the
standard transcript model has no fields for them.

## CLI Integration

Update the app registry:

```go
readers: map[string]func() reader.Reader{
    "claude": func() reader.Reader { return &claude.Reader{} },
    "codex":  func() reader.Reader { return &codex.Reader{} },
}
```

Update `render` help text:

```text
cg render --agent codex --file ~/.codex/sessions/2026/05/22/rollout-...jsonl
cg render --agent codex --session 019e...
cg render --agent codex --project .
cg render --agent codex --all --format html --out transcripts
```

No renderer changes should be necessary for the first release.

## Test Plan

Write tests before implementation in `reader/codex`.

Use `reader/codex/testdata/*.jsonl` fixtures, not inline JSONL builders.

### Reader Tests

| Test | Fixture | Assertion |
| --- | --- | --- |
| Basic transcript | `simple.jsonl` | session metadata, one user prompt, one assistant final answer |
| Injected user context | `injected_context.jsonl` | rendered user message uses `event_msg.user_message`, not large `response_item.user` |
| Tool loop | `tool_loop.jsonl` | function call and output map to tool use/result blocks |
| Patch tool | `apply_patch.jsonl` | `custom_tool_call` maps to `apply_patch` tool use/result |
| Reasoning | `reasoning.jsonl` | `summary_text` maps to thinking block; encrypted content ignored |
| Token usage | `token_count.jsonl` | aggregate usage maps from token-count event |
| Session lookup | temp `sessions/YYYY/MM/DD` tree | `ReadSession` finds by `session_meta.id` |
| Project lookup | temp tree with multiple cwd values | `ReadProject` returns matching sessions only |
| Bulk ordering | temp tree with two sessions | `ReadAll` sorted newest-first |
| Large line | generated temp file or fixture | scanner handles >1 MB line |

### Integration Tests

- Add CLI registry test if one exists; otherwise add a focused test for
  `newApp().reader("codex")`.
- Run:

```sh
make test
make build
```

### Golden Render Smoke Tests

Use one synthetic Codex fixture and verify:

```sh
cg render --agent codex --file reader/codex/testdata/simple.jsonl --format markdown
cg render --agent codex --file reader/codex/testdata/simple.jsonl --format html --out /tmp/cg-codex
```

## Documentation Updates

Update:

- `README.md` agent examples
- `docs/capabilities/agent-readers.md`
- `docs/capabilities/output-formats.md` only if examples mention agents
- `docs/concepts/readers.md` if it enumerates supported readers

Add a short note that Codex support reads local rollout files and does not
modify Codex state.

## Implementation Steps

1. Add `reader/codex/testdata` fixtures for the minimum schema cases.
2. Add failing tests for scanning, message mapping, title derivation, usage,
   and discovery.
3. Implement raw structs and `scanLines`.
4. Implement `buildTranscript`.
5. Implement tool/result mapping helpers.
6. Implement `ReadSession`, `ReadProject`, and `ReadAll`.
7. Register the reader in `cmd/cg/app.go`.
8. Update CLI help and docs.
9. Run `make test`.
10. Run `make build`.
11. Optionally render one local private session manually to inspect layout, but
    do not commit generated output from private data.

## Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| Codex format changes rapidly | Permissive raw structs; skip unknown row types; fixture coverage for observed variants |
| Large sessions exhaust memory | Stream JSONL scanning; avoid loading all raw rows when possible |
| Injected context leaks into shareable transcripts | Prefer `event_msg.user_message`; skip developer/system rows by default |
| Tool output duplicates | Use `response_item` as canonical and treat `event_msg` tool end rows as enrichment only |
| Images lack a core block type | Represent as concise placeholder text in first release; plan a later core schema extension |
| `.jsonl.zst` appears in the wild | Detect and return a clear unsupported error first; add zstd support later if local samples need it |

## Acceptance Criteria

- `cg render --agent codex --file <rollout.jsonl>` produces readable terminal,
  Markdown, and HTML output.
- Human prompts do not include Codex-injected AGENTS/environment context when
  `event_msg.user_message` is present.
- Tool calls and outputs appear in chronological order.
- Usage totals are populated when token-count rows exist.
- `--session`, `--project`, and `--all` work against a temp Codex sessions
  tree in tests.
- `make test` and `make build` pass.
