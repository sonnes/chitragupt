// Package json renders transcripts as standard transcript JSON.
package json

import (
	stdjson "encoding/json"
	"fmt"
	"io"

	"github.com/sonnes/chitragupt/core"
)

// Renderer renders a transcript to JSON.
type Renderer struct {
	// Indent controls pretty-printing. When true, output is indented.
	Indent bool
}

// New creates a JSON Renderer.
func New() *Renderer {
	return &Renderer{Indent: true}
}

// Render writes the transcript as JSON to w.
func (r *Renderer) Render(w io.Writer, t *core.Transcript) error {
	encoder := stdjson.NewEncoder(w)
	if r.Indent {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(t); err != nil {
		return fmt.Errorf("encode transcript json: %w", err)
	}
	return nil
}
