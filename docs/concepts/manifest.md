---
title: "Manifest"
summary: "Metadata index for rendered transcript sessions"
read_when:
  - Changing manifest read/write behavior
  - Debugging transcript index generation
  - Repairing or rebuilding transcript metadata
---

# Manifest

The manifest tracks metadata for rendered transcript sessions. It lets
Chitragupt regenerate the transcript index without reparsing every session on
every command.

## Location

The manifest lives in the transcript output directory:

```text
.transcripts/
  manifest.json
  index.html
  {session_id}/
    index.html
```

## Updates

Session capture updates the manifest after rendering a transcript. The
`cg index` command uses the manifest to regenerate the browseable index page.

## Repair

If the manifest is missing, corrupt, or out of sync, use:

```sh
cg manifest repair --dir .transcripts --agent claude
```

Repair scans session directories and rebuilds manifest entries from available
source files.

## Related

- [Manifest Repair](../capabilities/manifest-repair.md)
- [Git Integration](git-integration.md)

