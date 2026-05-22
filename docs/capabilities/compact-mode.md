---
title: "Compact Mode"
summary: "Shorter transcripts by summarizing or stripping verbose content"
read_when:
  - Sharing long sessions
  - Changing tool-output compaction behavior
  - Reducing transcript noise before rendering
---

# Compact Mode

Compact mode shortens transcripts dominated by verbose tool output. It keeps the
conversation flow readable by replacing noisy content with concise summaries.

Enable compact mode:

```sh
cg render --agent claude --file session.jsonl --compact
```

Also strip thinking blocks:

```sh
cg render --agent claude --file session.jsonl --compact --strip-thinking
```

## Pipeline Position

Compaction runs after redaction:

```text
Reader -> core.Transcript -> Redactor -> Compactor -> Renderer
```

This lets redaction inspect the full transcript before compaction removes
content.

## Implementation

Compact mode lives in `compact/` and implements `core.Transformer`.

Use `compact/testdata/*.jsonl` fixtures when changing behavior.

## Related

- [Transforms](../concepts/transforms.md)
- [Redaction](redaction.md)
