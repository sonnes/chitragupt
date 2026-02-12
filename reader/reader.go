// Package reader defines the interface for parsing agent-specific session logs
// into the standardized transcript format.
package reader

import "github.com/sonnes/chitragupt/core"

// Reader parses agent session data into standardized transcripts.
type Reader interface {
	// ReadFile parses a single session file at the given path.
	ReadFile(path string) (*core.Transcript, error)

	// ReadSession locates and parses a session by its ID.
	ReadSession(sessionID string) (*core.Transcript, error)

	// ReadProject returns all session transcripts for a named project.
	ReadProject(project string) ([]*core.Transcript, error)

	// ReadAll returns every session transcript the agent has stored.
	ReadAll() ([]*core.Transcript, error)
}
