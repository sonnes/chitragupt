// Package codex reads OpenAI Codex CLI session logs (JSONL rollouts in ~/.codex/sessions/).
package codex

// Reader reads Codex CLI JSONL rollout files.
type Reader struct {
	// Dir overrides the default session directory (~/.codex/sessions/).
	Dir string
}
