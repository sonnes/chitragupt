# Sub-Agent Transcripts — Design Specification

## Overview

Claude Code sessions can spawn sub-agents via the `Task` tool (standalone or team-based). These sub-agent conversations are stored as separate JSONL files alongside the main session but are currently discarded during parsing. This spec adds support for parsing sub-agent files into full `Transcript` objects and rendering them as separate linked pages.

---

## On-Disk Layout

Claude Code stores sub-agent conversations in a `subagents/` directory nested under the session ID:

```
~/.claude/projects/<project>/
  <sessionID>.jsonl
  <sessionID>/subagents/
    agent-<agentId>.jsonl
    agent-<agentId>.jsonl
    agent-acompact-<id>.jsonl       # internal, skip
```

Each sub-agent file contains JSONL entries identical in structure to the main session, with two differences:

- Every entry has `"isSidechain": true`
- Every entry has an `"agentId"` field identifying the sub-agent

### Sub-agent types

| Type                | How spawned                                             | Example file             | Notes                                 |
| ------------------- | ------------------------------------------------------- | ------------------------ | ------------------------------------- |
| Standalone Task     | `Task` tool with `subagent_type` + `prompt`             | `agent-ae267a1.jsonl`    | No team context                       |
| Team agent          | `Task` tool with `team_name` + `name` + `subagent_type` | `agent-afc4669.jsonl`    | Part of a team                        |
| Message delivery    | `SendMessage` tool                                      | `agent-aae8e77.jsonl`    | Duplicates main session content, skip |
| Context compression | Internal                                                | `agent-acompact-*.jsonl` | Internal, skip                        |

### Linking main → sub-agent

The `Task` tool_result text in the main session contains the agent ID:

```
agentId: ae267a1                          # standalone agents
agent_id: researcher@git-integration      # team agents
```

This ID matches the filename `agent-<agentId>.jsonl` and the `agentId` field in the sub-agent's entries.

---

## Data Model Changes

### `core/transcript.go`

New struct:

```go
// SubAgentRef links a Task tool_use block to its sub-agent transcript.
type SubAgentRef struct {
    AgentID   string `json:"agent_id"`            // matches a SubAgents[].SessionID
    AgentName string `json:"agent_name,omitempty"` // from Task input "name"
    AgentType string `json:"agent_type,omitempty"` // from Task input "subagent_type"
    TeamName  string `json:"team_name,omitempty"`  // from Task input "team_name"
}
```

New field on `ContentBlock`:

```go
SubAgentRef *SubAgentRef `json:"sub_agent_ref,omitempty"`
```

New field on `Transcript`:

```go
SubAgents []*Transcript `json:"sub_agents,omitempty"`
```

Sub-agent transcripts are full `Transcript` objects:

| Field                     | Value                                              |
| ------------------------- | -------------------------------------------------- |
| `SessionID`               | The agentId (e.g. `"ae267a1"`)                     |
| `ParentSessionID`         | Parent's SessionID                                 |
| `Agent`                   | `"claude"`                                         |
| `Model`                   | From first assistant message in the sub-agent file |
| `Title`                   | Derived from first user message (the prompt)       |
| `CreatedAt` / `UpdatedAt` | From first/last entries                            |
| `Usage`                   | Aggregated from all messages                       |
| `Messages`                | Parsed from the sub-agent JSONL                    |

### `core/schema.json`

Add `sub_agent_ref` to `ToolUseBlock` properties and `sub_agents` to the top-level `Transcript` as a recursive `$ref`.

---

## Reader Changes — `reader/claude/claude.go`

### New field on `rawEntry`

```go
AgentID string `json:"agentId"`
```

### New functions

**`discoverSubagentFiles(mainPath string) (map[string]string, error)`**

Given `<project>/<sessionID>.jsonl`, scans `<project>/<sessionID>/subagents/` for `agent-*.jsonl` files. Returns `agentID → filepath` map. Skips files matching `agent-acompact-*`. Returns `nil, nil` when the directory doesn't exist.

**`scanSubagentEntries(r io.Reader) ([]rawEntry, error)`**

Like `scanEntries` but does NOT filter `isSidechain` (all sub-agent entries have it set). Filters to `user` and `assistant` types only.

**`buildSubagentTranscript(path, parentSessionID string) (*core.Transcript, error)`**

Reads a sub-agent JSONL file and returns a full `Transcript`. Reuses `groupAndMapMessages`, `aggregateUsage`, `findPrimaryModel`, `deriveTitle`.

**`extractAgentIDFromResult(content string) string`**

Parses the agent ID from a Task tool_result. Handles both formats:

- `"agentId: ae267a1"` → `"ae267a1"`
- `"agent_id: researcher@git-integration"` → `"researcher@git-integration"`

**`extractTaskAgentInfo(input any) SubAgentRef`**

Extracts `subagent_type`, `name`, `team_name` from a Task tool_use input map.

**`attachSubagents(mainPath string, t *core.Transcript) error`**

Orchestrator:

1. Call `discoverSubagentFiles` to find sub-agent files
2. Parse each into a `Transcript` via `buildSubagentTranscript`
3. Append all to `t.SubAgents`
4. Build a tool_result index from main transcript messages (`tool_use_id → content`)
5. Walk all `Task` tool_use blocks: extract agentID from the paired result, find the matching sub-transcript, set `ContentBlock.SubAgentRef`

### Modified methods

`ReadFile` and `ReadSession`: after `buildTranscript`, always call `attachSubagents`. If the subagents directory doesn't exist, it's a no-op.

---

## Transformer Changes

Both `Redactor` and `Compactor` recurse into sub-agents at the `Transform` level:

```go
func (r *Redactor) Transform(t *core.Transcript) error {
    // ... existing message redaction ...
    for _, sub := range t.SubAgents {
        if err := r.Transform(sub); err != nil {
            return err
        }
    }
    return nil
}
```

Same pattern for `Compactor`. `ComputeDiffStats` is called per-transcript in the CLI.

---

## Rendering

### Separate files with links

Sub-agent transcripts render as **separate HTML pages** using the same `Renderer.Render()`. The parent transcript links to them.

Output directory structure:

```
<out-dir>/
  index.html                    # main transcript
  agent-ae267a1.html            # standalone sub-agent
  agent-afc4669.html            # team sub-agent
```

### Task tool_use block — link card

When a `Task` tool_use block has a `SubAgentRef`, `renderToolUseBlock` appends a link card after the tool result:

```
┌──────────────────────────────────────────────┐
│ ⚙ Task                                      │
├──────────────────────────────────────────────┤
│ { "subagent_type": "Explore", ... }          │
├──────────────────────────────────────────────┤
│ agentId: ae267a1 (internal ID ...)           │  ← tool result (existing)
├──────────────────────────────────────────────┤
│ 🔗 researcher (deep-researcher)  View →      │  ← link card (new)
└──────────────────────────────────────────────┘
```

The link card shows the agent name and type from `SubAgentRef`, linking to `agent-{AgentID}.html`.

### Terminal renderer

Adds a single styled line below the Task tool summary:

```
⚙ Task  Explore codebase structure
  → researcher (deep-researcher)
```

No file linking in terminal mode.

---

## CLI Changes — `cmd/cg/render.go`

### New flag

```
--out-dir <path>  Output directory (writes index.html + agent-*.html)
```

### Behavior

Sub-agents are always parsed when present — no opt-in flag needed. Transformers (redaction, compaction) and enrichment (DiffStats, Author) apply to all transcripts in the tree.

When `--out-dir` is set with `--format html`:

- Create the output directory
- Write main transcript as `index.html`
- Write each sub-agent transcript as `agent-{SessionID}.html`

When `--out-dir` is not set, render to stdout as before. Sub-agent link cards render but point to non-existent files.

### Example

```
cg render --agent claude \
  --session db215088-db9e-4c3d-b99b-40600bc02892 \
  --out-dir ./transcript --format html
```

---

## What Is Skipped

- **`acompact-*` agents** — internal context compression, no user-visible content
- **Message delivery agents** — spawned by `SendMessage`, content already visible in the main session's `SendMessage` tool_use/result
- **Sub-agents with no `user`/`assistant` entries** — empty after filtering

---

## Design Decisions

**Separate files, not inline nesting.** Sub-agent conversations can be large (100+ messages). Embedding them inline makes the parent transcript unwieldy. Separate pages keep each transcript focused and self-contained. The link card provides navigation without clutter.

**Full `Transcript` objects.** Sub-agents are treated as first-class transcripts with their own title, usage stats, timeline, and header. This reuses all existing rendering infrastructure with zero special-casing in renderers.

**`SubAgentRef` on `ContentBlock`.** Placing the reference on the `Task` tool_use block co-locates the link with the spawn point. Renderers check one field — no separate index or lookup needed.

**`SubAgents` on `Transcript`.** The tree structure is explicit. Transformers recurse naturally. The CLI walks the tree to write multiple files. JSON output preserves the full tree.

**Always-on.** Sub-agent discovery is a single directory read — negligible cost. No opt-in flag needed. Sessions without sub-agents are unaffected (`discoverSubagentFiles` returns nil).
