// Package opencode reads OpenCode session data (SQLite in ~/.opencode/).
package opencode

// Reader reads OpenCode sessions from a SQLite database.
type Reader struct {
	// DBPath overrides the default database path (~/.opencode/).
	DBPath string
}
