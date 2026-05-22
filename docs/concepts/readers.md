---
title: "Readers"
summary: "Agent-specific log parsers that produce core.Transcript values"
read_when:
  - Adding support for a new CLI agent
  - Debugging session parsing
  - Changing how raw logs map to transcript content
---

# Readers

Readers convert raw agent session logs into the standard transcript format.
They are the only layer that should understand agent-specific on-disk formats.

## Responsibilities

A reader should:

- Locate or open raw session files.
- Parse the agent's native JSONL or JSON format.
- Map messages and content blocks to `core.Transcript`.
- Preserve useful metadata such as session ID, project, cwd, branch, and
  timestamps.
- Return clear errors for malformed or missing input.

A reader should not:

- Redact content.
- Compact tool output.
- Render output.
- Depend on a specific output format.

## Current Readers

| Package | Agent | Notes |
| --- | --- | --- |
| `reader/claude` | Claude Code | Handles main sessions and sub-agent files |
| `reader/codex` | Codex | Parses Codex session logs |
| `reader/cursor` | Cursor | Parses Cursor session logs |
| `reader/opencode` | OpenCode | Parses OpenCode session logs |

## Test Fixtures

Use `testdata/*.jsonl` files for synthetic fixtures. Avoid inline string
builders for session logs. For directory traversal tests, copy fixture files
into `t.TempDir()` and build the directory shape the reader expects.

## Related

- [Standard Transcript Format](standard-transcript-format.md)
- [Agent Readers](../capabilities/agent-readers.md)

