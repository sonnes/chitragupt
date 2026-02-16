package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sonnes/chitragupt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func entry(id string, created time.Time) core.ManifestEntry {
	return core.ManifestEntry{
		SessionID: id,
		Title:     "Session " + id,
		Agent:     "claude",
		CreatedAt: created,
		Href:      "claude/" + id + "/index.html",
	}
}

func TestReadFileNotExist(t *testing.T) {
	m, err := ReadFile(filepath.Join(t.TempDir(), "manifest.json"))
	require.NoError(t, err)
	assert.Empty(t, m.Entries)
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	e := entry("abc", now)
	e.Usage = &core.Usage{InputTokens: 5000, OutputTokens: 2000}
	e.DiffStats = &core.DiffStats{Added: 10, Removed: 3, Changed: 2}
	e.MessageCount = 8

	m := &Manifest{Entries: []core.ManifestEntry{e}}
	require.NoError(t, m.WriteFile(path))

	got, err := ReadFile(path)
	require.NoError(t, err)
	require.Len(t, got.Entries, 1)
	assert.Equal(t, "abc", got.Entries[0].SessionID)
	assert.Equal(t, 5000, got.Entries[0].Usage.InputTokens)
	assert.Equal(t, 10, got.Entries[0].DiffStats.Added)
	assert.Equal(t, 8, got.Entries[0].MessageCount)
}

func TestUpsertAppend(t *testing.T) {
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	m := &Manifest{}

	m.Upsert(entry("a", now))
	m.Upsert(entry("b", now.Add(time.Hour)))

	require.Len(t, m.Entries, 2)
	assert.Equal(t, "b", m.Entries[0].SessionID, "newest first")
	assert.Equal(t, "a", m.Entries[1].SessionID)
}

func TestUpsertReplace(t *testing.T) {
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	m := &Manifest{}

	m.Upsert(entry("a", now))
	m.Upsert(entry("b", now.Add(time.Hour)))

	updated := entry("a", now)
	updated.Title = "Updated title"
	m.Upsert(updated)

	require.Len(t, m.Entries, 2)
	// Find entry "a" and check title was replaced.
	var found bool
	for _, e := range m.Entries {
		if e.SessionID == "a" {
			assert.Equal(t, "Updated title", e.Title)
			found = true
		}
	}
	assert.True(t, found, "entry 'a' should exist")
}

func TestUpsertSortsNewestFirst(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	t2 := t0.Add(2 * time.Hour)

	m := &Manifest{}
	m.Upsert(entry("old", t0))
	m.Upsert(entry("new", t2))
	m.Upsert(entry("mid", t1))

	require.Len(t, m.Entries, 3)
	assert.Equal(t, "new", m.Entries[0].SessionID)
	assert.Equal(t, "mid", m.Entries[1].SessionID)
	assert.Equal(t, "old", m.Entries[2].SessionID)
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := &Manifest{Entries: []core.ManifestEntry{
		entry("x", time.Now()),
	}}
	require.NoError(t, m.WriteFile(path))

	// Verify no leftover temp files.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "manifest.json", entries[0].Name())
}

func TestWriteFileCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "manifest.json")

	m := &Manifest{}
	require.NoError(t, m.WriteFile(path))

	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestNewManifestEntry(t *testing.T) {
	now := time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	later := now.Add(30 * time.Minute)

	tr := &core.Transcript{
		SessionID: "sess-1",
		Title:     "Fix auth bug",
		Agent:     "claude",
		Author:    "ravi",
		Model:     "claude-opus-4-6",
		CreatedAt: now,
		UpdatedAt: &later,
		Usage:     &core.Usage{InputTokens: 1000, OutputTokens: 500},
		DiffStats: &core.DiffStats{Added: 5, Removed: 2, Changed: 1},
		Messages:  make([]core.Message, 3),
	}

	e := core.NewManifestEntry(tr, "claude/sess-1/index.html")

	assert.Equal(t, "sess-1", e.SessionID)
	assert.Equal(t, "Fix auth bug", e.Title)
	assert.Equal(t, "ravi", e.Author)
	assert.Equal(t, "claude-opus-4-6", e.Model)
	assert.Equal(t, now, e.CreatedAt)
	assert.Equal(t, &later, e.UpdatedAt)
	assert.Equal(t, 1000, e.Usage.InputTokens)
	assert.Equal(t, 5, e.DiffStats.Added)
	assert.Equal(t, 3, e.MessageCount)
	assert.Equal(t, "claude/sess-1/index.html", e.Href)
}
