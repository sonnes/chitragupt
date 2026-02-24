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

Write output to a directory (generates `index.html` + per-agent files):

```sh
cg render --agent claude --project ./ --format html --out .transcripts
```

### Output formats

```sh
cg render --agent claude --file session.jsonl --format terminal   # default
cg render --agent claude --file session.jsonl --format html
cg render --agent claude --file session.jsonl --format markdown
cg render --agent claude --file session.jsonl --format json
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

### Compact mode

Strip tool results for a shorter transcript:

```sh
cg render --agent claude --file session.jsonl --compact
```

Also strip thinking blocks:

```sh
cg render --agent claude --file session.jsonl --compact=no-thinking
```

### Serve

Browse sessions in a local web UI:

```sh
cg serve --agent claude --project <project-name>
cg serve --agent claude --all
cg serve --agent claude --port 3000
```

### Git integration

Set up automatic transcript capture when a session ends:

```sh
cg install --agent claude --format html --out transcripts
```

This creates a `transcripts/` directory, installs a `SessionEnd` hook that renders transcripts on session end, and adds `transcripts/` to `.gitignore`.

To also version transcripts on an orphan branch (creates a git worktree and auto-commits when you run `git commit`):

```sh
cg install --agent claude --format html --out .transcripts --branch gh-pages
```

To remove hooks and configuration:

```sh
cg uninstall            # keeps transcript data
cg uninstall --purge    # also deletes data and orphan branch
```

### Manifest

The manifest (`manifest.json`) tracks metadata for all rendered sessions. It is updated automatically by the SessionEnd hook.

Rebuild the manifest from scratch if it gets out of sync:

```sh
cg manifest repair --dir transcripts --agent claude
```

Regenerate the index page from the manifest:

```sh
cg index --dir transcripts
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
