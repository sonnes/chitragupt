# Chitragupt

## Overview

Chitragupt (`cg`) is a Go CLI that converts CLI agent session logs into
shareable transcripts. It reads session files from Claude Code, Codex,
OpenCode, and Cursor, normalizes them into a standard transcript model, then
renders them as HTML, Markdown, JSON, or terminal output.

## Tech Stack

- Go
- Makefile-driven build and test workflow
- JSONL/JSON session readers
- Static HTML, Markdown, JSON, and ANSI terminal renderers

## Code Quality Expectations

### Test Driven Development

- Prefer writing tests before implementation when changing behavior.
- Run the failing test to confirm it fails for the right reason.
- Write the minimum implementation needed to make the test pass.
- Run tests again to confirm they pass.
- Use `github.com/stretchr/testify` for assertions: `require` for fatal checks,
  `assert` for the rest.
- Use table-driven tests with `t.Run` subtests.
- Use `testdata/*.jsonl` files for synthetic session fixtures, not inline string
  builders.
- For tests that need directory structure, copy testdata files into
  `t.TempDir()`.

### Build Commands

- Use `make` as the single entry point for build, test, lint, examples, and
  cleanup commands.
- Always build with `make build`.
- Build binaries must be written to `.bin/`.
- Do not invoke `go build` directly for normal development; add or use a
  Makefile target instead.

### Go Style

Write vertical, readable Go code. Favor clarity over compression:

- Break long function calls with one argument per line.
- Use intermediate variables instead of deeply nested expressions.
- Break long conditionals into named booleans when that improves readability.
- Use early returns to keep control flow flat.
- Write struct literals vertically when they have more than a couple fields.
- Keep packages focused and avoid circular dependencies.

### Documentation

- Public functions and types should have concise GoDoc comments.
- Use simple, direct language.
- Prefer `doc.go` files for package-level documentation when a package needs
  more than a short package comment.
- Use docs front matter for repository docs:

```yaml
---
title: "Page Title"
summary: "Short description of what this document explains"
read_when:
  - Reason to read this document
---
```

## Makefile Targets

| Target          | Purpose                                      |
| --------------- | -------------------------------------------- |
| `make build`    | Build the CLI binary to `.bin/cg`            |
| `make install`  | Install the CLI binary with `go install`     |
| `make test`     | Run all tests                                |
| `make lint`     | Run `go vet ./...`                           |
| `make clean`    | Remove built binaries                        |
| `make examples` | Regenerate example transcript output files   |

## Project Structure

```text
cmd/cg/        CLI entrypoint and command wiring
core/          Standard transcript model, schema, turns, transforms
reader/        Agent-specific session readers
  claude/      Claude Code JSONL sessions
  codex/       Codex sessions
  cursor/      Cursor sessions
  opencode/    OpenCode sessions
redact/        Secrets and PII redaction transformer
compact/       Compact transcript transformer
render/        Output renderers
  html/        Static HTML renderer
  json/        Standard transcript JSON renderer
  markdown/    Markdown renderer
  terminal/    ANSI terminal renderer
server/        Local transcript browsing server
manifest/      Manifest read/write and repair logic
install/       Hook installation and removal
docs/          Research, concepts, capabilities, and implementation plans
examples/      Example session inputs and rendered outputs
```

## Transcript Pipeline

All agent formats flow through the same pipeline:

```text
Agent logs -> Reader -> core.Transcript -> Transforms -> Renderer -> Output
```

Readers should produce `core.Transcript` values and avoid renderer-specific
behavior. Renderers should consume `core.Transcript` values and avoid
agent-specific parsing.

Transform ordering matters:

1. Redaction runs first so secrets and PII are removed from full transcript
   content.
2. Compaction runs after redaction so verbose tool output can be summarized or
   stripped safely.
3. Rendering runs last.

## Development Rules

- Preserve the reader/core/transform/renderer boundaries.
- Keep agent-specific quirks inside the matching reader package.
- Keep output-format decisions inside renderer packages.
- Do not make redaction or compaction renderer-specific.
- Do not add inline JSONL fixture builders when a `testdata/*.jsonl` fixture
  can express the case.
- Update docs when changing user-facing CLI behavior, transcript schema,
  install hooks, manifest behavior, redaction, compaction, or rendering output.
- Run `make test` before committing behavior changes.
- Run `make build` before handing off CLI changes.
