---
title: "Capabilities"
summary: "Feature-oriented documentation for Chitragupt behavior"
read_when:
  - Choosing which feature document to read
  - Auditing current cg behavior
  - Planning user-facing CLI changes
---

# Capabilities

This folder documents Chitragupt's user-facing capabilities. Concept docs in
[`../concepts`](../concepts) explain the underlying architecture; capability
docs explain what the CLI can do today and where each feature lives.

## Pages

### Input and Parsing

- [Agent Readers](agent-readers.md) — supported CLI agent log readers
- [Sub-Agent Transcripts](subagents.md) — linked transcripts for spawned agents

### Output

- [Output Formats](output-formats.md) — terminal, HTML, Markdown, and JSON output
- [Local Server](local-server.md) — browsing rendered sessions locally

### Transcript Safety and Size

- [Redaction](redaction.md) — secrets and PII sanitization
- [Compact Mode](compact-mode.md) — shorter transcripts by removing noisy output

### Transcript Management

- [Manifest Repair](manifest-repair.md) — rebuilding `manifest.json`

