// Package manifest manages the session metadata index file (manifest.json)
// that tracks all rendered transcripts in a repository.
package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/sonnes/chitragupt/core"
)

// Manifest holds the list of session metadata entries.
type Manifest struct {
	Entries []core.ManifestEntry `json:"entries"`
}

// ReadFile reads a manifest from disk. Returns an empty Manifest if the file
// does not exist.
func ReadFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Manifest{}, nil
	}
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Upsert adds or replaces an entry matched by SessionID. After upserting, the
// entries are sorted newest-first by CreatedAt.
func (m *Manifest) Upsert(entry core.ManifestEntry) {
	for i, e := range m.Entries {
		if e.SessionID == entry.SessionID {
			m.Entries[i] = entry
			m.sort()
			return
		}
	}
	m.Entries = append(m.Entries, entry)
	m.sort()
}

func (m *Manifest) sort() {
	sort.Slice(m.Entries, func(i, j int) bool {
		return m.Entries[i].CreatedAt.After(m.Entries[j].CreatedAt)
	})
}

// WriteFile writes the manifest to disk atomically using a temporary file and
// rename, which is safe against concurrent writers.
func (m *Manifest) WriteFile(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".manifest-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, path)
}
