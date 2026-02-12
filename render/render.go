// Package render defines the interface for rendering standardized transcripts
// into various output formats.
package render

import (
	"io"

	"github.com/sonnes/chitragupt/core"
)

// Renderer writes a transcript to the given writer in a specific format.
type Renderer interface {
	Render(w io.Writer, t *core.Transcript) error
}
