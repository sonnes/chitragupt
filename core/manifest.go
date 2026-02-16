package core

import "time"

// ManifestEntry holds lightweight metadata for a single session, used by the
// manifest file and the index template. It mirrors the fields of Transcript
// that the index page needs, without carrying the full message list.
type ManifestEntry struct {
	SessionID    string     `json:"session_id"`
	Title        string     `json:"title,omitempty"`
	Agent        string     `json:"agent"`
	Author       string     `json:"author,omitempty"`
	Model        string     `json:"model,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	Usage        *Usage     `json:"usage,omitempty"`
	DiffStats    *DiffStats `json:"diff_stats,omitempty"`
	MessageCount int        `json:"message_count"`
	Href         string     `json:"href"`
}

// NewManifestEntry extracts metadata from a Transcript and pairs it with the
// given href (relative link to the rendered page).
func NewManifestEntry(t *Transcript, href string) ManifestEntry {
	return ManifestEntry{
		SessionID:    t.SessionID,
		Title:        t.Title,
		Agent:        t.Agent,
		Author:       t.Author,
		Model:        t.Model,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		Usage:        t.Usage,
		DiffStats:    t.DiffStats,
		MessageCount: len(t.Messages),
		Href:         href,
	}
}
