---
title: "Agent Readers"
summary: "Supported CLI agent session readers and their responsibilities"
read_when:
  - Rendering sessions from a specific agent
  - Adding support for a new agent log format
  - Debugging reader-specific parsing behavior
---

# Agent Readers

Chitragupt reads session logs from multiple CLI agents and normalizes them into
`core.Transcript`.

| Agent | Reader package | Status | Notes |
| --- | --- | --- | --- |
| Claude Code | `reader/claude` | Supported | Includes sub-agent parsing |
| Codex | `reader/codex` | Supported | Parses Codex session logs |
| Cursor | `reader/cursor` | Supported | Parses Cursor session logs |
| OpenCode | `reader/opencode` | Supported | Parses OpenCode session logs |

Use the `--agent` flag to select a reader:

```sh
cg render --agent claude --file session.jsonl
```

## Related

- [Readers](../concepts/readers.md)
- [Sub-Agent Transcripts](subagents.md)

