// Package cursor reads Cursor session data (SQLite state.vscdb key-value store).
package cursor

// Reader reads Cursor sessions from a state.vscdb SQLite database.
type Reader struct {
	// DBPath overrides the default database path.
	DBPath string
}
