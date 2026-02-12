// Package server provides a local HTTP server for browsing and rendering
// agent sessions on the fly.
package server

import "github.com/sonnes/chitragupt/reader"

// Server serves session transcripts over HTTP for local browsing.
type Server struct {
	// Reader provides access to session data.
	Reader reader.Reader
	// Port is the TCP port to listen on.
	Port int
}
