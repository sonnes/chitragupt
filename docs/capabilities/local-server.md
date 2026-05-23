---
title: "Local Server"
summary: "Browse transcript sessions from a local HTTP server"
read_when:
  - Using cg serve
  - Browsing sessions without writing output files
  - Debugging local transcript browsing behavior
---

# Local Server

`cg serve` starts a local HTTP server that reads current-project sessions
through every supported agent reader by default and renders them as HTML.

The server is live: each index or transcript request rereads the agent session
stores, so newly written or updated sessions appear after a browser refresh.
Missing agent stores are treated as empty.

The HTML index and session header visually distinguish top-level sessions,
subagents, continuations, forks, and unknown lineage when readers can derive
that metadata.

Serve sessions for a project:

```sh
cg serve
cg serve --project .
```

Serve every discoverable session:

```sh
cg serve --all
```

Choose a port:

```sh
cg serve --port 3000
```

Filter to one agent when needed:

```sh
cg serve --agent claude --project .
cg serve --agent codex --all
```

Serving is intentionally local and ephemeral. It does not install hooks, write a
manifest, manage transcript storage, or publish files.

## Related

- [Output Formats](output-formats.md)
- [Agent Readers](agent-readers.md)
