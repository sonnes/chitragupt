# Manifest Repair — Design Specification

## Overview

The `cg manifest repair` command rebuilds `manifest.json` from scratch by scanning the `.transcripts/` directory for existing session directories and re-parsing their raw source files. This handles cases where the manifest is lost, corrupted, or out of sync with the actual rendered transcripts.

---

## Background

The SessionEnd hook calls `cg manifest upsert` after each render to incrementally maintain the manifest. This can get out of sync when:

- The hook fails partway (render succeeds but upsert fails)
- Someone manually adds or removes session directories
- The manifest file is deleted or corrupted
- The transcripts branch is checked out fresh without a manifest

---

## CLI Interface

```
cg manifest repair --dir .transcripts/ --agent claude
```

| Flag      | Alias | Required | Default   | Description                                    |
| --------- | ----- | -------- | --------- | ---------------------------------------------- |
| `--dir`   | `-d`  | yes      |           | Path to the `.transcripts/` directory          |
| `--agent` | `-a`  | no       | `claude`  | Agent name, selects the reader for raw sources |

Output: summary line to stdout (e.g. `Repaired manifest: 12 entries (2 skipped)`).

---

## Directory Structure

The repair command operates on this layout:

```
.transcripts/                  ← --dir points here
├── manifest.json              ← rebuilt by repair
├── index.html                 ← not touched by repair
├── .gitkeep
├── {session_id}/
│   ├── index.html
│   ├── index.jsonl            ← (if multiple formats configured)
│   └── agent-{id}.html
└── {session_id}/
    └── index.html
```

Session directories live directly under `.transcripts/` (flat, no agent subdirectory). Each directory name is the session UUID.

---

## Algorithm

1. **Scan** `--dir` for entries, keeping only directories. Skip non-session entries: `.git`, `.gitkeep`, files (`manifest.json`, `index.html`).

2. **For each session directory:**
   a. **Detect href** — check for `index.*` files in priority order: `.html` → `.jsonl` → `.json` → `.md`. If no index file exists, skip the directory.
   b. **Parse raw source** — call `reader.ReadSession(sessionID)` to locate and parse the raw JSONL from the agent's session store (e.g. `~/.claude/projects/`).
   c. If the raw source is not found, log a warning to stderr and skip.
   d. **Compute diff stats** — call `computeDiffStatsTree(t)` on the parsed transcript.
   e. **Build entry** — `core.NewManifestEntry(t, href)` with href as `{sessionID}/index.{ext}`.
   f. **Upsert** into the manifest.

3. **Write** the rebuilt `manifest.json` atomically via `Manifest.WriteFile`.

4. **Print** summary to stdout.

---

## Implementation

### Extracted repair function

The core logic lives in a standalone function for testability, separate from the CLI action:

```go
func repairManifest(dir string, r reader.Reader) (*manifest.Manifest, int, error)
```

Returns the rebuilt manifest, count of skipped sessions, and any fatal error. The CLI action wraps this with flag parsing and output.

### Href detection

```go
func detectSessionHref(sessionID, sessionDir string) string
```

Checks for `index.{html,jsonl,json,md}` in priority order. Returns `{sessionID}/index.{ext}` or empty string if nothing found.

### File placement

Both functions live in `cmd/cg/manifest.go` alongside the existing `manifestUpsertCmd`. The repair subcommand is registered in `manifestCmd()`.

---

## Error Handling

| Condition                        | Behavior                                      |
| -------------------------------- | --------------------------------------------- |
| `--dir` doesn't exist           | Fatal error                                   |
| Session dir has no index file    | Skip, count as skipped                        |
| `ReadSession` fails (not found) | Warn to stderr, skip, count as skipped        |
| `ReadSession` fails (parse err) | Warn to stderr, skip, count as skipped        |
| No sessions found at all         | Write empty manifest, print "0 entries"       |
| `WriteFile` fails                | Fatal error                                   |

Repair is best-effort: it rebuilds what it can and reports what it skipped. A non-zero skip count is not an error.

---

## Limitations

**Requires raw source files.** The reader needs the original JSONL files in the agent's session store (`~/.claude/projects/`). Repair will skip sessions whose raw files have been purged or are on a different machine. This is acceptable because:

- The primary use case is repairing a local installation where raw files exist
- Cloned transcripts branches are read-only archives — they already have a manifest committed

---

## Testing

### Unit tests in `cmd/cg/manifest_test.go`

Uses testdata JSONL fixtures and `t.TempDir()` for filesystem setup.

| Test case                     | Setup                                                                 | Assertion                                           |
| ----------------------------- | --------------------------------------------------------------------- | --------------------------------------------------- |
| Basic repair                  | Two session dirs with `index.html`, raw JSONL files in mock claude dir | Manifest has 2 entries, sorted newest-first         |
| Missing raw source            | Session dir exists but no corresponding raw file                       | Skipped count is 1, manifest has remaining entries  |
| Mixed formats                 | Session dir with only `index.jsonl`                                   | Href uses `.jsonl` extension                        |
| No session dirs               | Empty `.transcripts/` (only `.gitkeep`)                               | Empty manifest written                              |
| Skips non-session entries     | Dir contains `manifest.json`, `index.html`, `.git`                    | None of these treated as sessions                   |
| Href priority                 | Session dir with both `index.html` and `index.jsonl`                  | Href uses `.html`                                   |

Tests use `claude.Reader{Dir: tempDir}` to override the session directory, pointing to a temp dir with testdata fixtures.

---

## Design Decisions

**Re-parse over sidecar metadata.** We chose to re-parse raw source files rather than writing per-session `metadata.json` sidecars during render. This avoids adding another file per session and another step to the hook script. The tradeoff is that repair requires raw sources to be available locally, which is the common case for the primary use case (local installation repair).

**Best-effort with summary.** Rather than failing on the first unresolvable session, repair skips problematic entries and reports a summary. This is more useful for bulk repair where some sessions may have been cleaned up.

**Flat directory scan.** Session directories sit directly under `.transcripts/` with no agent subdirectory nesting. The scan reads one directory level and filters by directory type, which is simple and fast.
