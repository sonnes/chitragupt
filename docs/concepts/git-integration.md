---
title: "Git Integration"
summary: "How cg install captures transcripts through hooks and optional transcript branches"
read_when:
  - Changing cg install or cg uninstall
  - Debugging automatic transcript capture
  - Working with transcript branches or worktrees
---

# Git Integration

Chitragupt can install hooks that capture transcripts when agent sessions end.
The integration is configured through `cg install` and removed through
`cg uninstall`.

## Local Transcript Capture

The basic install path writes rendered transcripts to a configured output
directory and updates the manifest:

```sh
cg install --agent claude --format html --out transcripts
```

The hook renders a transcript after a session ends and keeps the transcript
directory ignored by the main working tree.

## Branch-Backed Transcripts

When a branch is configured, Chitragupt can write transcripts into a separate
git branch or worktree:

```sh
cg install --agent claude --format html --out .transcripts --branch gh-pages
```

This keeps transcript history versioned while avoiding normal source branch
churn.

## Uninstall

`cg uninstall` removes installed hook configuration while preserving transcript
data. `cg uninstall --purge` also removes transcript data and the transcript
branch when configured.

## Related

- [Manifest](manifest.md)
- [Research: Git Session Log Storage](../research/git-session-log-storage.md)

