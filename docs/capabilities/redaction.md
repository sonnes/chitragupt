---
title: "Redaction"
summary: "Secrets and PII sanitization before transcript rendering"
read_when:
  - Sharing transcripts publicly
  - Changing redaction rules
  - Adding a renderer or transform that handles raw content
---

# Redaction

Redaction sanitizes transcript content before rendering. It removes secrets such
as API keys, tokens, and private keys, plus PII such as emails, phone numbers,
IP addresses, and filesystem paths.

Redaction is enabled by default:

```sh
cg render --agent claude --file session.jsonl
```

Disable it only for trusted local output:

```sh
cg render --agent claude --file session.jsonl --no-redact
```

Limit redaction to a category:

```sh
cg render --agent claude --file session.jsonl --redact secrets
cg render --agent claude --file session.jsonl --redact pii
```

## Pipeline Position

Redaction runs after reading and before compaction:

```text
Reader -> core.Transcript -> Redactor -> Compactor -> Renderer
```

This ensures secrets are removed before any transform strips or summarizes
content.

## Implementation

Redaction lives in `redact/`:

- `redact.go` defines the redactor.
- `rules.go` defines built-in rules.
- `walk.go` walks transcript string fields.

## Related

- [Transforms](../concepts/transforms.md)
- [Compact Mode](compact-mode.md)

