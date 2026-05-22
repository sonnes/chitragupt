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

Use the `--agent` flag to select a reader:

```sh
cg render --agent claude --file session.jsonl
```

## Related

- [Readers](../concepts/readers.md)
- [Sub-Agent Transcripts](subagents.md)
