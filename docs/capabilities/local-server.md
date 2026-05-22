---
title: "Local Server"
summary: "Browse transcript sessions from a local HTTP server"
read_when:
  - Using cg serve
  - Browsing sessions without writing output files
  - Debugging local transcript browsing behavior
---

# Local Server

`cg serve` starts a local HTTP server that reads sessions through the configured
reader and renders them on demand as HTML.

Serve sessions for a project:

```sh
cg serve --agent claude --project <project-name>
cg serve --agent codex --project .
```

Serve every discoverable session:

```sh
cg serve --agent claude --all
cg serve --agent codex --all
```

Choose a port:

```sh
cg serve --agent claude --port 3000
cg serve --agent codex --port 3000
```

Serving is intentionally local and ephemeral. It does not install hooks, write a
manifest, manage transcript storage, or publish files.

## Related

- [Output Formats](output-formats.md)
- [Agent Readers](agent-readers.md)
