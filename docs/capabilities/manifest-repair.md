---
title: "Manifest Repair"
summary: "Rebuild manifest.json by scanning rendered transcript directories"
read_when:
  - Repairing a missing or corrupt manifest
  - Changing manifest repair behavior
  - Debugging transcript index entries
---

# Manifest Repair

`cg manifest repair` rebuilds `manifest.json` by scanning an existing transcript
directory.

```sh
cg manifest repair --dir .transcripts --agent claude
```

Use repair when:

- `manifest.json` is missing.
- The manifest is corrupt.
- The manifest is out of sync with rendered session directories.
- A transcript branch was checked out without a valid manifest.

## Directory Shape

Repair expects session directories directly under the transcript directory:

```text
.transcripts/
  manifest.json
  index.html
  {session_id}/
    index.html
    agent-{id}.html
```

Non-session entries such as `.git`, `.gitkeep`, `manifest.json`, and
`index.html` are ignored.

## Related

- [Manifest](../concepts/manifest.md)
- [Git Integration](../concepts/git-integration.md)

