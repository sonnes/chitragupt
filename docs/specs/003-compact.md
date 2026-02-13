# Compact Mode — Design Specification

## Overview

Agent session transcripts are dominated by verbose tool outputs (file contents from Read, command output from Bash, search results from Grep/Glob). When sharing transcripts, this noise obscures the conversation flow. The `--compact` flag replaces tool outputs with short summaries, producing concise transcripts focused on what the agent did and why.

---

## Pipeline Position

Compact is a transform on the normalized transcript, after redaction and before rendering:

```
Agent Logs → Reader → core.Transcript → [Redactor] → [Compactor] → Renderer → Output
```

Ordering matters: redaction must scan full content before compaction removes text that might contain secrets.

This placement means:

- Works for all agents (Claude, Codex, OpenCode, Cursor) automatically
- Works for all output formats (HTML, Markdown, JSON, terminal) automatically
- Readers and renderers have no knowledge of compaction

---

## Project Structure

```
compact/
├── compact.go       # Compactor struct, implements core.Transformer
└── compact_test.go  # Table-driven tests with testdata fixtures
    testdata/
    ├── verbose_session.jsonl    # Long tool results and tool inputs
    └── error_session.jsonl      # Error tool results
```

---

## Compactor

The `Compactor` implements `core.Transformer`:

```go
type Config struct {
    StripThinking bool // remove thinking blocks (default: false)
}

type Compactor struct { ... }

func New(cfg Config) *Compactor { ... }

func (c *Compactor) Transform(t *core.Transcript) error {
    for i := range t.Messages {
        c.compactMessage(&t.Messages[i])
    }
    return nil
}
```

### Block-level compaction

| Block type    | Action                                                                                          |
| ------------- | ----------------------------------------------------------------------------------------------- |
| `tool_result` | `.Content` replaced with summary like `[output: 245 lines]`                                     |
| `tool_use`    | Verbose `.Input` fields replaced with summary (Write `content`, Edit `old_string`/`new_string`) |
| `thinking`    | Kept by default; removed only when `StripThinking` is true                                      |
| `text`        | Unchanged                                                                                       |

### Tool result compaction

Tool result content is replaced with a line-count summary:

```
[output: 245 lines]
```

For error results:

```
[error: 12 lines]
```

Both success and error results are compacted — the summary distinguishes them via the label.

### Tool input compaction

Only tools with bulk content in their inputs are compacted:

| Tool    | Fields compacted           | Summary format                                     |
| ------- | -------------------------- | -------------------------------------------------- |
| `Write` | `content`                  | `[content: 84 lines]`                              |
| `Edit`  | `old_string`, `new_string` | `[old_string: 15 lines]`, `[new_string: 20 lines]` |

Other tool inputs (Bash `command`, Read `file_path`, Grep `pattern`) are inherently short and left unchanged. Tool name matching is case-insensitive.

---

## Configuration

```go
type Config struct {
    StripThinking bool // default: false
}
```

### CLI flags

```
cg render --agent claude --file session.jsonl --compact                    # compact mode, thinking preserved
cg render --agent claude --file session.jsonl --compact=no-thinking      # compact mode, thinking stripped
cg render --agent claude --file session.jsonl                              # full output (default)
```

Compact mode is off by default. The `--compact` flag accepts an optional `no-thinking` value to strip thinking blocks.

---

## Integration

In the render command handler, after redaction:

```go
if v := cmd.String("compact"); v != "" {
    cfg := compact.Config{}
    if v == "no-thinking" {
        cfg.StripThinking = true
    }
    compactor := compact.New(cfg)
    for _, t := range transcripts {
        if err := core.Chain(t, compactor); err != nil {
            return fmt.Errorf("compact: %w", err)
        }
    }
}
```

---

## Design Decisions

**Summaries, not truncation.** Replacing content with `[output: 245 lines]` is cleaner than keeping the first N lines. It tells you something was there and how large it was, without any of the noise. The transcript reads as a log of actions rather than a dump of outputs.

**Thinking blocks preserved by default.** Thinking blocks reveal the agent's reasoning and are valuable context when reviewing transcripts. They are only stripped with `--compact=no-thinking`.

**No compaction of Bash commands.** The `command` field tells you what the agent ran, which is essential context even in compact mode. Only the _outputs_ (tool results) and _bulk inputs_ (Write content, Edit strings) are compacted.

**Line count in summaries.** Showing the line count gives a sense of scale — `[output: 3 lines]` vs `[output: 2400 lines]` tells you whether the output was trivial or massive.
