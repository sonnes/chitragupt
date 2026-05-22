---
title: "Local Server"
summary: "Browse rendered transcript sessions from a local HTTP server"
read_when:
  - Using cg serve
  - Changing local browsing behavior
  - Debugging transcript index or static file serving
---

# Local Server

`cg serve` starts a local HTTP server for browsing session transcripts.

Serve all known sessions:

```sh
cg serve --agent claude --all
```

Serve sessions for a project:

```sh
cg serve --agent claude --project <project-name>
```

Choose a port:

```sh
cg serve --agent claude --port 3000
```

## Implementation

Server behavior lives in `server/`. CLI wiring lives in `cmd/cg/serve.go`.

## Related

- [Output Formats](output-formats.md)
- [Manifest](../concepts/manifest.md)

