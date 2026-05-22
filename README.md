# chitragupt (`cg`)

The agent's record-keeper — generate shareable transcripts from CLI agent session logs.

Named after [Chitragupta](https://en.wikipedia.org/wiki/Chitragupta), an Indian mythological figure who maintains the records of every human action.

## What it does

`cg` reads Claude Code session logs (JSONL/JSON) and produces clean, shareable transcripts. It does not install hooks, manage storage, or decide where transcripts belong. Generate or browse the transcript; use it however you want.

## Install

```sh
go install github.com/sonnes/chitragupt/cmd/cg@latest
```

Or build from source:

```sh
make build
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

Render all sessions across all projects:

```sh
cg render --agent claude --all
```

Write output to a directory:

```sh
cg render --agent claude --project ./ --format html --out transcripts
```

### Output Formats

```sh
cg render --agent claude --file session.jsonl --format terminal   # default
cg render --agent claude --file session.jsonl --format html
cg render --agent claude --file session.jsonl --format markdown
```

### Serve

Browse sessions locally without creating files:

```sh
cg serve --agent claude --project <project-name>
cg serve --agent claude --all
cg serve --agent claude --port 3000
```

### Redaction

Secrets (API keys, tokens, connection strings) and PII (emails, phone numbers, IP addresses, filesystem paths) are redacted by default. Home directory paths are replaced with `~/…` to strip usernames while keeping transcripts readable. To disable:

```sh
cg render --agent claude --file session.jsonl --no-redact
```

To redact only specific categories:

```sh
cg render --agent claude --file session.jsonl --redact secrets
cg render --agent claude --file session.jsonl --redact pii
```

### Compact Mode

Strip tool results for a shorter transcript:

```sh
cg render --agent claude --file session.jsonl --compact
```

Also strip thinking blocks:

```sh
cg render --agent claude --file session.jsonl --compact --strip-thinking
```

## Philosophy

`cg` converts agent logs into readable artifacts, with an optional local browser
for inspection, and leaves capture, storage, publishing, indexing, versioning,
and hosting to shell scripts, git, CI, static hosts, or whatever workflow the
user prefers.

## Architecture

```
reader/       Parse agent-specific logs → core.Transcript
  claude/       Claude Code JSONL sessions

core/         Standardized transcript format + transformer pipeline

redact/       Secrets & PII redaction transformer
compact/      Compact output transformer

render/       Render transcripts to output formats
  terminal/     ANSI terminal with tree view
  html/         Tailwind v4 + syntax highlighting
  markdown/     Markdown

cmd/cg/       CLI entrypoint
```

## Documentation

- [Capabilities](docs/capabilities/README.md) — feature-oriented docs for readers, output formats, local serving, redaction, compact mode, and sub-agents
- [Concepts](docs/concepts/transcript-pipeline.md) — architecture docs for the transcript pipeline and core boundaries
- [Research](docs/research/git-session-log-storage.md) — archived background research on storing agent sessions in git

## License

Apache 2.0 — see [LICENSE](LICENSE).
