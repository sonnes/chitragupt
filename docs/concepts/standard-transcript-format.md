---
title: "Standard Transcript Format"
summary: "The normalized transcript model shared by every reader and renderer"
read_when:
  - Changing core transcript structs
  - Adding renderer support for new content blocks
  - Validating JSON transcript output
---

# Standard Transcript Format

The standard transcript format is the boundary between agent-specific parsing
and output rendering. Every reader produces a `core.Transcript`; every renderer
consumes one.

The canonical sources are:

- [`core/transcript.go`](../../core/transcript.go) for Go structs
- [`core/schema.json`](../../core/schema.json) for JSON schema

## Transcript

A transcript represents one agent session. It carries session metadata,
messages, and optional sub-agent transcripts.

Important metadata includes the session ID, agent name, working directory,
project, title, timestamps, and git branch when available.

## Messages

Messages preserve the conversation order. Each message has a role, timestamp,
and content blocks. Roles map the agent log into a stable shape rather than
exposing every raw provider detail to renderers.

## Content Blocks

Content blocks represent text, tool calls, tool results, thinking blocks, and
other structured transcript content. Readers should preserve as much useful
structure as possible so renderers can make deliberate display choices.

## Sub-Agents

Sub-agent transcripts are represented as linked `core.Transcript` values. The
main transcript can reference a sub-agent from the task/tool block that spawned
it, while HTML rendering can write sub-agent pages beside the main transcript.

## Compatibility Contract

When changing the format:

- Update `core/transcript.go`.
- Update `core/schema.json`.
- Add or update reader tests with JSONL fixtures.
- Add or update renderer tests for affected output formats.
- Update capability docs when user-facing behavior changes.

## Related

- [Transcript Pipeline](transcript-pipeline.md)
- [Sub-Agent Transcripts](../capabilities/subagents.md)

