# chitragupt (`cg`)

The agent's record-keeper — generate shareable transcripts from CLI agent session logs.

Named after [Chitragupta](https://en.wikipedia.org/wiki/Chitragupta), an Indian mythological figure who maintains the records of every human action.

## What it does

`cg` reads Claude Code session logs (JSONL/JSON) and produces clean, shareable transcripts in multiple formats — static HTML, Markdown, or pretty-printed terminal output.

## Install

```sh
go install github.com/sonnes/chitragupt/cmd/cg@latest
```

Or build from source:

```sh
create an Integration with git.

Create  build
```

## Usage

Render a single session file to the terminal:

```sh
cg render --agent claude --file ~/.claude/projects/.../session.jsonl
```

Render by session ID:

```sh
cg render --agent claude --session <session-id>
```

Render all sessions in a project:

```sh
cg render --agent claude --project <project-name>
```

### Output formats

```sh
cg render --agent claude --file session.jsonl --format terminal   # default
cg render --agent claude --file session.jsonl --format html
cg render --agent claude --file session.jsonl --format markdown
cg render --agent claude --file session.jsonl --format json
```

### Redaction

Secrets and PII are redacted by default. To disable:

```sh
cg render --agent claude --file session.jsonl --no-redact
```

To redact only specific categories:

```sh
cg render --agent claude --file session.jsonl --redact secrets
cg render --agent claude --file session.jsonl --redact pii
```

### Compact mode

Strip tool results for a shorter transcript:

```sh
cg render --agent claude --file session.jsonl --compact
```

Also strip thinking blocks:

```sh
cg render --agent claude --file session.jsonl --compact=no-thinking
```

## Architecture

```
reader/       Parse agent-specific logs → core.Transcript
  claude/       Claude Code JSONL sessions
  codex/        Codex sessions
  cursor/       Cursor sessions
  opencode/     OpenCode sessions

core/         Standardized transcript format + transformer pipeline

redact/       Secrets & PII redaction transformer
compact/      Compact output transformer

render/       Render transcripts to output formats
  terminal/     ANSI terminal with tree view
  html/         Tailwind v4 + syntax highlighting
  markdown/     Markdown
  json/         JSON

server/       Local HTTP server for browsing sessions
cmd/cg/       CLI entrypoint
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
