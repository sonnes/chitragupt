---
title: "Transcript Pipeline"
summary: "How agent logs become normalized transcripts and rendered output"
read_when:
  - Adding a new agent reader
  - Adding a transcript transform
  - Changing render behavior across output formats
---

# Transcript Pipeline

Chitragupt converts agent session logs into shareable output through one common
pipeline:

```text
Agent logs -> Reader -> core.Transcript -> Transforms -> Renderer -> Output
```

Each stage owns a narrow responsibility. Readers understand agent-specific log
formats. The core model provides the normalized transcript shape. Transforms
mutate normalized transcripts. Renderers produce the final user-facing output.

## Readers

Readers parse raw session files and return `core.Transcript` values. Agent
quirks stay inside the matching reader package:

- `reader/claude` parses Claude Code JSONL sessions.

Readers should not know whether the transcript will be rendered as HTML,
Markdown, or terminal output.

## Core Model

The core package defines the standard transcript format used by every reader
and renderer. Canonical definitions live in:

- [`core/transcript.go`](../../core/transcript.go)
- [`core/schema.json`](../../core/schema.json)

All shared behavior should flow through this model instead of agent-specific
types.

## Transforms

Transforms run after parsing and before rendering. They implement the
`core.Transformer` contract and mutate a transcript in place.

The current transform order is:

1. Redaction removes secrets and PII from full transcript content.
2. Compaction summarizes or strips verbose content after redaction has had a
   chance to inspect it.

Ordering matters. Redaction must see the complete transcript before compaction
removes tool output.

## Renderers

Renderers consume `core.Transcript` values and write output:

- `render/html` writes static HTML.
- `render/markdown` writes Markdown.
- `render/terminal` writes ANSI terminal output.

Renderers should not parse raw agent logs or duplicate transform behavior.

## Related

- [Standard Transcript Format](standard-transcript-format.md)
- [Readers](readers.md)
- [Transforms](transforms.md)
- [Rendering](rendering.md)
