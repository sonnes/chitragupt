// Package json renders transcripts as JSON (serializes the standardized format as-is).
package json

// Renderer renders a transcript to JSON.
type Renderer struct {
	// Indent controls pretty-printing. When true, output is indented.
	Indent bool
}
