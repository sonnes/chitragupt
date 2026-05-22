---
title: "Rendering"
summary: "How normalized transcripts become HTML, Markdown, JSON, and terminal output"
read_when:
  - Adding or changing an output format
  - Debugging rendered transcript output
  - Deciding whether behavior belongs in a renderer or transform
---

# Rendering

Renderers turn normalized transcripts into output files or terminal text. They
consume `core.Transcript` values and should not parse raw agent logs.

## Formats

| Package | Format | Notes |
| --- | --- | --- |
| `render/html` | HTML | Static, shareable transcript pages |
| `render/markdown` | Markdown | Plain text documentation-friendly output |
| `render/json` | JSON | Standard transcript JSON |
| `render/terminal` | Terminal | ANSI output for direct CLI viewing |

## Output Directories

When rendering to a directory, Chitragupt writes an `index` file for the main
session. HTML rendering may also write linked sub-agent pages beside it.

## Renderer Boundaries

Renderers can make presentation decisions such as layout, syntax highlighting,
and how to display tool calls. They should not perform redaction, compaction, or
agent-specific parsing.

## Examples

Example outputs are generated with:

```sh
make examples
```

This target rebuilds the CLI and regenerates the checked-in example transcript
outputs from source JSONL fixtures.

## Related

- [Output Formats](../capabilities/output-formats.md)
- [Transcript Pipeline](transcript-pipeline.md)

