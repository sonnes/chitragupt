---
title: "Transforms"
summary: "Transcript mutations that run between readers and renderers"
read_when:
  - Adding a new transcript-wide behavior
  - Changing redaction or compaction order
  - Keeping renderer logic format-agnostic
---

# Transforms

Transforms mutate a normalized `core.Transcript` before rendering. They are the
right place for behavior that should apply to every reader and every renderer.

## Contract

Transforms implement the `core.Transformer` interface:

```go
type Transformer interface {
    Transform(t *Transcript) error
}
```

Transforms mutate in place to avoid copying large transcripts. `core.Chain`
applies multiple transforms in order.

## Current Transforms

| Package | Purpose |
| --- | --- |
| `redact` | Remove secrets and PII before sharing transcripts |
| `compact` | Strip or summarize verbose transcript content |

## Ordering

Run redaction before compaction:

```text
Reader -> core.Transcript -> Redactor -> Compactor -> Renderer
```

Redaction needs access to complete tool input and output. Compaction can remove
or summarize content only after redaction has sanitized it.

## Design Rule

If a behavior should affect every output format, it belongs in a transform. If
it should affect only one output format, it belongs in that renderer.

## Related

- [Redaction](../capabilities/redaction.md)
- [Compact Mode](../capabilities/compact-mode.md)

