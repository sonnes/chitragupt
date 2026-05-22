---
title: "Output Formats"
summary: "Terminal, HTML, and Markdown transcript rendering"
read_when:
  - Choosing a render format
  - Changing renderer behavior
  - Updating examples or shareable output
---

# Output Formats

Chitragupt renders normalized transcripts in three supported formats.
Transcript metadata may include derived `stats` such as tool usage, file
operation categories, command names, skills, work-mode counts, and sub-agent
count. Human-facing renderers can use these fields for summaries.

| Format | Flag | Renderer | Use case |
| --- | --- | --- | --- |
| Terminal | `--format terminal` | `render/terminal` | Inspect a transcript in the CLI |
| HTML | `--format html` | `render/html` | Share a static transcript page |
| Markdown | `--format markdown` | `render/markdown` | Publish or archive plain text |

Terminal output is the default when no output directory is requested.

## Examples

```sh
cg render --agent claude --file session.jsonl --format terminal
cg render --agent claude --file session.jsonl --format html --out transcripts
cg render --agent claude --file session.jsonl --format markdown
```

## Related

- [Rendering](../concepts/rendering.md)
- [Compact Mode](compact-mode.md)
- [Redaction](redaction.md)
