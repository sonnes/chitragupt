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
| Claude Code | `reader/claude` | Supported | Includes sub-agent, continuation, and fork relation metadata |
| OpenAI Codex | `reader/codex` | Supported | Reads rollout JSONL files from `~/.codex/sessions`; detects root vs subagent sessions; diff stats are derived from `apply_patch` payloads |

Use the `--agent` flag to select a reader:

```sh
cg render --agent claude --file session.jsonl
cg render --agent codex --file rollout.jsonl
```

`cg serve` is agent-agnostic by default and lists current-project sessions from
every supported reader. Use `cg serve --all` to include every discoverable
session, or `cg serve --agent claude` / `cg serve --agent codex` to filter the
live server to one reader.

Codex session lookup reads `rollout-*.jsonl` files under
`$CODEX_HOME/sessions` when `CODEX_HOME` is set, otherwise
`~/.codex/sessions`.

Readers set `Transcript.Relation` when the source format exposes lineage:

- `root` is a top-level user session.
- `subagent` is a spawned side session.
- `continuation` is a session with a direct parent session ID.
- `fork` is a branched session with `forked_from` source metadata.
- `unknown` is reserved for sessions with ambiguous lineage metadata.

Claude Code forks use the source `forkedFrom` field. Claude Code
continuations use `parentSessionId`. Codex currently exposes reliable
`root`/`subagent` metadata through `thread_source` and `source.subagent`; it
does not expose a fork point in the local rollout logs.

## Related

- [Readers](../concepts/readers.md)
- [Sub-Agent Transcripts](subagents.md)
