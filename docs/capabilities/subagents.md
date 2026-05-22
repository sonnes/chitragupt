---
title: "Sub-Agent Transcripts"
summary: "Parsing and rendering linked transcripts for spawned Claude Code sub-agents"
read_when:
  - Rendering Claude Code sessions with Task tool calls
  - Debugging missing sub-agent pages
  - Changing sub-agent parsing or linking
---

# Sub-Agent Transcripts

Claude Code can spawn sub-agents through task-style tool calls. Chitragupt
parses supported sub-agent session files and renders them as linked transcript
pages beside the main session.

## On-Disk Layout

Claude Code stores sub-agents beside the main session:

```text
~/.claude/projects/<project>/
  <sessionID>.jsonl
  <sessionID>/subagents/
    agent-<agentID>.jsonl
```

Internal compaction agents and message-delivery sidechains are skipped.

## Linking

The main transcript links task/tool blocks to the matching sub-agent transcript
by agent ID. HTML output writes separate sub-agent pages and links to them from
the main session page.

## Implementation

Sub-agent parsing is part of `reader/claude`. The normalized representation
lives in `core.Transcript` so renderers can handle sub-agents without knowing
Claude Code's raw file layout.

## Related

- [Standard Transcript Format](../concepts/standard-transcript-format.md)
- [Agent Readers](agent-readers.md)

