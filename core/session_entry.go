package core

import "time"

// SessionEntry holds lightweight metadata for a transcript listing.
type SessionEntry struct {
	SessionID       string          `json:"session_id"`
	Title           string          `json:"title,omitempty"`
	Agent           string          `json:"agent"`
	Relation        SessionRelation `json:"relation,omitempty"`
	ParentSessionID string          `json:"parent_session_id,omitempty"`
	ForkedFrom      *ForkInfo       `json:"forked_from,omitempty"`
	Author          string          `json:"author,omitempty"`
	Model           string          `json:"model,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       *time.Time      `json:"updated_at,omitempty"`
	Usage           *Usage          `json:"usage,omitempty"`
	DiffStats       *DiffStats      `json:"diff_stats,omitempty"`
	Stats           *SessionStats   `json:"stats,omitempty"`
	MessageCount    int             `json:"message_count"`
	Href            string          `json:"href"`
}

// NewSessionEntry extracts listing metadata from a transcript.
func NewSessionEntry(t *Transcript, href string) SessionEntry {
	stats := t.Stats
	if stats == nil {
		stats = ComputeSessionStats(t)
	}

	return SessionEntry{
		SessionID:       t.SessionID,
		Title:           t.Title,
		Agent:           t.Agent,
		Relation:        t.Relation,
		ParentSessionID: t.ParentSessionID,
		ForkedFrom:      t.ForkedFrom,
		Author:          t.Author,
		Model:           t.Model,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		Usage:           t.Usage,
		DiffStats:       t.DiffStats,
		Stats:           stats,
		MessageCount:    len(t.Messages),
		Href:            href,
	}
}
