package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sonnes/chitragupt/manifest"
	"github.com/sonnes/chitragupt/reader/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testdataFixture = "../../reader/claude/testdata/simple.jsonl"

// setupClaudeDir creates a fake claude projects dir with the given session JSONL
// files. It rewrites the sessionId field in the fixture to match the target
// session ID. Returns a Reader pointed at the temp dir.
func setupClaudeDir(t *testing.T, sessions map[string]string) *claude.Reader {
	t.Helper()
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	for sessionID, fixturePath := range sessions {
		data, err := os.ReadFile(fixturePath)
		require.NoError(t, err)
		// Rewrite sessionId in the fixture so ReadSession+buildTranscript produce
		// a transcript whose SessionID matches the directory entry.
		patched := strings.ReplaceAll(string(data), `"sessionId":"sess-1"`, `"sessionId":"`+sessionID+`"`)
		err = os.WriteFile(filepath.Join(projectDir, sessionID+".jsonl"), []byte(patched), 0o644)
		require.NoError(t, err)
	}

	return &claude.Reader{Dir: dir}
}

// setupTranscriptsDir creates a .transcripts/ dir with session subdirectories
// containing the specified index files.
func setupTranscriptsDir(t *testing.T, sessions map[string][]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".transcripts")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	for sessionID, indexFiles := range sessions {
		sessionDir := filepath.Join(dir, sessionID)
		require.NoError(t, os.MkdirAll(sessionDir, 0o755))
		for _, name := range indexFiles {
			err := os.WriteFile(filepath.Join(sessionDir, name), []byte("<html>"), 0o644)
			require.NoError(t, err)
		}
	}

	return dir
}

func TestRepairManifest(t *testing.T) {
	tests := []struct {
		name           string
		transcriptDirs map[string][]string // sessionID → index files
		claudeSessions map[string]string   // sessionID → fixture path
		wantEntries    int
		wantSkipped    int
		wantHrefs      map[string]string // sessionID → expected href
	}{
		{
			name: "basic repair with two sessions",
			transcriptDirs: map[string][]string{
				"sess-1": {"index.html"},
				"sess-2": {"index.html"},
			},
			claudeSessions: map[string]string{
				"sess-1": testdataFixture,
				"sess-2": testdataFixture,
			},
			wantEntries: 2,
			wantSkipped: 0,
		},
		{
			name: "missing raw source",
			transcriptDirs: map[string][]string{
				"sess-1":   {"index.html"},
				"sess-999": {"index.html"},
			},
			claudeSessions: map[string]string{
				"sess-1": testdataFixture,
				// sess-999 has no raw source
			},
			wantEntries: 1,
			wantSkipped: 1,
		},
		{
			name: "session dir with only jsonl index",
			transcriptDirs: map[string][]string{
				"sess-1": {"index.jsonl"},
			},
			claudeSessions: map[string]string{
				"sess-1": testdataFixture,
			},
			wantEntries: 1,
			wantSkipped: 0,
			wantHrefs: map[string]string{
				"sess-1": "sess-1/index.jsonl",
			},
		},
		{
			name:           "no session dirs",
			transcriptDirs: map[string][]string{},
			claudeSessions: map[string]string{},
			wantEntries:    0,
			wantSkipped:    0,
		},
		{
			name: "session dir without index file",
			transcriptDirs: map[string][]string{
				"sess-1": {"other.txt"},
			},
			claudeSessions: map[string]string{
				"sess-1": testdataFixture,
			},
			wantEntries: 0,
			wantSkipped: 1,
		},
		{
			name: "href priority html over jsonl",
			transcriptDirs: map[string][]string{
				"sess-1": {"index.html", "index.jsonl"},
			},
			claudeSessions: map[string]string{
				"sess-1": testdataFixture,
			},
			wantEntries: 1,
			wantSkipped: 0,
			wantHrefs: map[string]string{
				"sess-1": "sess-1/index.html",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transcriptsDir := setupTranscriptsDir(t, tt.transcriptDirs)
			reader := setupClaudeDir(t, tt.claudeSessions)

			m, skipped, err := repairManifest(transcriptsDir, reader)
			require.NoError(t, err)

			assert.Equal(t, tt.wantEntries, len(m.Entries))
			assert.Equal(t, tt.wantSkipped, skipped)

			for _, entry := range m.Entries {
				if wantHref, ok := tt.wantHrefs[entry.SessionID]; ok {
					assert.Equal(t, wantHref, entry.Href)
				}
			}
		})
	}
}

func TestRepairManifestSkipsNonSessionEntries(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".transcripts")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	// Create non-session entries
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644))

	reader := setupClaudeDir(t, map[string]string{})

	m, skipped, err := repairManifest(dir, reader)
	require.NoError(t, err)
	assert.Equal(t, 0, len(m.Entries))
	assert.Equal(t, 0, skipped)
}

func TestRepairManifestWritesValidJSON(t *testing.T) {
	transcriptsDir := setupTranscriptsDir(t, map[string][]string{
		"sess-1": {"index.html"},
	})
	reader := setupClaudeDir(t, map[string]string{
		"sess-1": testdataFixture,
	})

	m, _, err := repairManifest(transcriptsDir, reader)
	require.NoError(t, err)

	manifestPath := filepath.Join(transcriptsDir, "manifest.json")
	require.NoError(t, m.WriteFile(manifestPath))

	// Read back and verify it's valid JSON
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var parsed manifest.Manifest
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, 1, len(parsed.Entries))
	assert.Equal(t, "sess-1", parsed.Entries[0].SessionID)
	assert.Equal(t, "sess-1/index.html", parsed.Entries[0].Href)
}

func TestDetectSessionHref(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantHref  string
	}{
		{
			name:     "html",
			files:    []string{"index.html"},
			wantHref: "test-session/index.html",
		},
		{
			name:     "jsonl",
			files:    []string{"index.jsonl"},
			wantHref: "test-session/index.jsonl",
		},
		{
			name:     "json",
			files:    []string{"index.json"},
			wantHref: "test-session/index.json",
		},
		{
			name:     "md",
			files:    []string{"index.md"},
			wantHref: "test-session/index.md",
		},
		{
			name:     "html wins over jsonl",
			files:    []string{"index.jsonl", "index.html"},
			wantHref: "test-session/index.html",
		},
		{
			name:     "no index file",
			files:    []string{"other.txt"},
			wantHref: "",
		},
		{
			name:     "empty dir",
			files:    nil,
			wantHref: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "test-session")
			require.NoError(t, os.MkdirAll(dir, 0o755))
			for _, f := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
			}

			got := detectSessionHref("test-session", dir)
			assert.Equal(t, tt.wantHref, got)
		})
	}
}
