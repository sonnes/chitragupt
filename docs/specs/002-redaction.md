# Redaction — Design Specification

## Overview

`cg` transcripts may contain secrets (API keys, tokens, private keys) and PII (emails, phone numbers, IP addresses) captured from tool inputs and outputs during agent sessions. The redaction layer sanitizes `core.Transcript` data before rendering, ensuring transcripts are safe to share.

---

## Pipeline Position

Redaction is a transform on the normalized transcript, between Reader and Renderer:

```
Agent Logs → Reader → core.Transcript → [Redactor] → Renderer → Output
```

This placement means:

- Works for all agents (Claude, Codex, OpenCode, Cursor) automatically
- Works for all output formats (HTML, Markdown, JSON, terminal) automatically
- Readers and renderers have no knowledge of redaction

---

## Transformer Interface

A general-purpose transform concept in `core/` that redaction implements:

```go
// core/transform.go

// Transformer mutates a Transcript in place.
type Transformer interface {
    Transform(t *Transcript) error
}

// Chain applies transformers in order.
func Chain(t *Transcript, transformers ...Transformer) error {
    for _, tr := range transformers {
        if err := tr.Transform(t); err != nil {
            return err
        }
    }
    return nil
}
```

In-place mutation avoids deep-copying large transcripts. The `Chain` helper enables composing redaction with future transforms (e.g., strip thinking blocks, anonymize usernames).

---

## Project Structure

```
redact/
├── redact.go       # Redactor struct, implements core.Transformer
├── rules.go        # Rule interface + built-in rule definitions
└── walk.go         # Transcript walker that visits all string fields in ContentBlocks
```

---

## Rule Interface

Each detection pattern is a self-contained rule:

```go
type Rule interface {
    Name() string              // e.g. "aws_secret_key"
    Kind() string              // "secret" or "pii"
    Detect(s string) []Match   // find all matches in a string
    Replacement(m Match) string // what to substitute
}

type Match struct {
    Start, End int
    Value      string
}
```

Rules are independent and testable in isolation with plain string inputs.

---

## Redactor

The `Redactor` composes rules and implements `core.Transformer`:

```go
type Redactor struct {
    rules []Rule
}

func New(cfg Config) *Redactor { ... }

func (r *Redactor) Transform(t *core.Transcript) error {
    t.Dir = r.redactString(t.Dir)
    t.Title = r.redactString(t.Title)
    for i := range t.Messages {
        for j := range t.Messages[i].Content {
            r.redactBlock(&t.Messages[i].Content[j])
        }
    }
    for _, sub := range t.SubAgents {
        if err := r.Transform(sub); err != nil {
            return err
        }
    }
    return nil
}
```

Transcript metadata fields (`Dir`, `Title`) are redacted first since they may contain filesystem paths with usernames. Sub-agent transcripts are processed recursively.

### Block-level redaction

`redactBlock` switches on block type and applies rules to the relevant string fields:

| Block type    | Fields redacted                                                   |
| ------------- | ----------------------------------------------------------------- |
| `text`        | `.Text`                                                           |
| `thinking`    | `.Text`                                                           |
| `tool_use`    | `.Input` — recursively walk the `any` value, redact string leaves |
| `tool_result` | `.Content`                                                        |

Tool inputs require recursive walking because `.Input` is `any` (typically `map[string]any` from JSON). The walker in `walk.go` traverses maps, slices, and nested structures, applying rules to every string value encountered.

---

## Built-in Rules

### Secrets (on by default)

| Rule              | Pattern                                                              |
| ----------------- | -------------------------------------------------------------------- |
| AWS access key    | `AKIA[0-9A-Z]{16}`                                                   |
| AWS secret key    | 40-char base64 adjacent to `aws_secret_access_key`                   |
| Generic API key   | `sk-[a-zA-Z0-9]{32,}`, `ghp_`, `gho_`, `glpat-`                      |
| Private key block | `-----BEGIN (RSA\|EC\|OPENSSH) PRIVATE KEY-----`                     |
| Connection string | `postgres://…`, `mongodb://…`, `mysql://…` with embedded credentials |
| JWT               | `eyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_.+/=]+`           |
| High-entropy      | Long hex/base64 near keywords `token`, `secret`, `password`, `key`   |

### PII (on by default)

| Rule            | Pattern                                           |
| --------------- | ------------------------------------------------- |
| Email           | RFC 5322 simplified                               |
| IPv4 address    | `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`              |
| Phone number    | Common formats with country codes                 |
| Filesystem path | Absolute home directory paths (replaced with `~/…`) |

PII detection is on by default. False positives in code-heavy transcripts (e.g., version numbers matching IP patterns, test data matching email patterns) can be handled via the allowlist.

---

## Replacement Strategy

Use typed placeholders so the transcript remains readable:

```
sk-abc123...xyz                → [REDACTED:api_key]
AKIAIOSFODNN7EXAMPLE           → [REDACTED:aws_key]
user@example.com               → [REDACTED:email]
-----BEGIN RSA PRIVATE KEY---- → [REDACTED:private_key]
/Users/alice/code/project      → ~/code/project
```

Most rules use the `[REDACTED:name]` pattern, where the rule's `Name()` populates the label. The filesystem path rule is an exception — it replaces the home directory prefix with `~` while preserving the relative path, keeping transcripts readable without exposing the local username.

---

## Configuration

```go
type Config struct {
    Secrets    bool     // redact secrets (default: true)
    PII        bool     // redact PII (default: true)
    ExtraRules []Rule   // user-supplied rules
    Allowlist  []string // patterns to never redact (e.g., known-safe test keys)
}
```

### CLI flags

```
cg render --agent claude --file session.jsonl                       # secrets + PII (default)
cg render --agent claude --file session.jsonl --redact=secrets      # secrets only
cg render --agent claude --file session.jsonl --no-redact           # disable entirely
```

Redaction is on by default. Use `--redact=secrets` to limit to secrets only, or `--no-redact` to disable entirely.

---

## Integration

In the command handler, the pipeline becomes:

```go
transcript, err := reader.ReadFile(path)
if err != nil {
    return err
}

if redactionEnabled {
    redactor := redact.New(redact.Config{Secrets: true, PII: piiFlag})
    if err := redactor.Transform(transcript); err != nil {
        return err
    }
}

return renderer.Render(os.Stdout, transcript)
```

---

## Design Risks

**False positives in code-heavy transcripts.** Source code contains strings that resemble secrets and PII patterns (test fixtures, example values, version strings). Mitigations:

- Keyword-proximity heuristic for high-entropy rules (only flag base64 blobs near `secret`/`token`/`password`)
- PII can be disabled with `--redact=secrets`
- Allowlist for known-safe patterns

**Tool input walking.** The `any`-typed `.Input` field requires recursive traversal. Malformed or deeply nested inputs could cause issues. The walker should enforce a max depth and handle non-string leaves gracefully.
