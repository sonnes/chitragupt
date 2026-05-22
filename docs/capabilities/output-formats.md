---
title: "Output Formats"
summary: "Terminal, HTML, Markdown, and JSON transcript rendering"
read_when:
  - Choosing a render format
  - Changing renderer behavior
  - Updating examples or shareable output
---

# Output Formats

Chitragupt renders normalized transcripts in four formats.

| Format | Flag | Renderer | Use case |
| --- | --- | --- | --- |
| Terminal | `--format terminal` | `render/terminal` | Inspect a transcript in the CLI |
| HTML | `--format html` | `render/html` | Share a static transcript page |
| Markdown | `--format markdown` | `render/markdown` | Publish or archive plain text |
| JSON | `--format json` | `render/json` | Preserve the standard transcript format |

Terminal output is the default when no output directory is requested.

## Examples

```sh
cg render --agent claude --file session.jsonl --format terminal
cg render --agent claude --file session.jsonl --format html --out transcripts
cg render --agent claude --file session.jsonl --format markdown
cg render --agent claude --file session.jsonl --format json
```

## Related

- [Rendering](../concepts/rendering.md)
- [Compact Mode](compact-mode.md)
- [Redaction](redaction.md)

