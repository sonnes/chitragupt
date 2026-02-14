// Package core defines the Standardized Transcript Format — a normalized
// representation of CLI agent session logs that all readers produce and all
// renderers consume.
package core

import "time"

// Transcript is the top-level container for a single session.
type Transcript struct {
	SessionID       string     `json:"session_id"`
	ParentSessionID string     `json:"parent_session_id,omitempty"`
	Agent           string     `json:"agent"`                // "claude", "codex", "opencode", "cursor"
	Author          string     `json:"author,omitempty"`     // git user.name from working directory
	Model           string     `json:"model,omitempty"`      // primary model used
	Dir             string     `json:"dir,omitempty"`        // working directory
	GitBranch       string     `json:"git_branch,omitempty"` // branch at session start
	Title           string     `json:"title,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	Usage           *Usage     `json:"usage,omitempty"`      // aggregate session usage
	DiffStats       *DiffStats `json:"diff_stats,omitempty"` // aggregate edit statistics
	Messages        []Message  `json:"messages"`
}

// Usage holds token counters. Used both at session level (aggregate) and per
// individual message.
type Usage struct {
	InputTokens         int `json:"input_tokens,omitempty"`
	OutputTokens        int `json:"output_tokens,omitempty"`
	CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
}

// DiffStats summarizes file-level edit statistics across the session.
type DiffStats struct {
	Added   int `json:"added,omitempty"`   // lines added (Write content + Edit new_string)
	Removed int `json:"removed,omitempty"` // lines removed (Edit old_string)
	Changed int `json:"changed,omitempty"` // unique files touched
}

// Add accumulates the counts from other into u.
func (u *Usage) Add(other Usage) {
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.CacheReadTokens += other.CacheReadTokens
	u.CacheCreationTokens += other.CacheCreationTokens
}

// Message is a single turn in the conversation.
type Message struct {
	UUID       string         `json:"uuid,omitempty"`
	ParentUUID string         `json:"parent_uuid,omitempty"`
	Role       Role           `json:"role"`
	Model      string         `json:"model,omitempty"` // set for assistant messages
	Timestamp  *time.Time     `json:"timestamp,omitempty"`
	Content    []ContentBlock `json:"content"`
	Usage      *Usage         `json:"usage,omitempty"`
}

// Role enumerates who produced a message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// ContentBlock is one piece of a message. The Type field determines which
// other fields are populated.
type ContentBlock struct {
	Type      BlockType  `json:"type"`
	Format    TextFormat `json:"format,omitempty"`      // "markdown" or "plain"; set for "text" blocks
	Text      string     `json:"text,omitempty"`        // set for "text" and "thinking"
	ToolUseID string     `json:"tool_use_id,omitempty"` // set for "tool_use" and "tool_result"
	Name      string     `json:"name,omitempty"`        // tool name, set for "tool_use"
	Input     any        `json:"input,omitempty"`       // tool input params, set for "tool_use"
	Content   string     `json:"content,omitempty"`     // tool output, set for "tool_result"
	IsError   bool       `json:"is_error,omitempty"`    // set for "tool_result"
}

// TextFormat indicates how a text block should be rendered.
type TextFormat string

const (
	FormatMarkdown TextFormat = "markdown"
	FormatPlain    TextFormat = "plain"
)

// BlockType enumerates content block kinds.
type BlockType string

const (
	BlockText       BlockType = "text"
	BlockThinking   BlockType = "thinking"
	BlockToolUse    BlockType = "tool_use"
	BlockToolResult BlockType = "tool_result"
)
