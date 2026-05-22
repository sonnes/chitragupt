---
title: "Agent Readers"
summary: "Supported CLI agent session readers and their responsibilities"
read_when:
  - Rendering sessions from a specific agent
  - Adding support for a new agent log format
  - Debugging reader-specific parsing behavior
---

# Agent Readers

Chitragupt reads CLI agent session logs and normalizes them into
`core.Transcript`.

| Agent | Reader package | Status | Notes |
| --- | --- | --- | --- |
| Claude Code | `reader/claude` | Supported | Includes sub-agent parsing |
| OpenAI Codex | `reader/codex` | Supported | Reads rollout JSONL files from `~/.codex/sessions` |

Use the `--agent` flag to select a reader:

```sh
cg render --agent claude --file session.jsonl
cg render --agent codex --file rollout.jsonl
```

Codex session lookup reads `rollout-*.jsonl` files under
`$CODEX_HOME/sessions` when `CODEX_HOME` is set, otherwise
`~/.codex/sessions`.

## Related

- [Readers](../concepts/readers.md)
- [Sub-Agent Transcripts](subagents.md)
