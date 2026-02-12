// Package claude reads Claude Code session logs (JSONL in ~/.claude/projects/).
package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sonnes/chitragupt/core"
)

// Reader reads Claude Code JSONL session files.
type Reader struct {
	// Dir overrides the default session directory (~/.claude/projects/).
	Dir string
}

// maxLineSize is the maximum JSONL line size (1 MB). Claude Code tool results
// can exceed the default 64 KB bufio.Scanner buffer.
const maxLineSize = 1 << 20

// Raw JSON deserialization types. These mirror the JSONL structure on disk.

type rawEntry struct {
	Type        string     `json:"type"`
	UUID        string     `json:"uuid"`
	ParentUUID  *string    `json:"parentUuid"`
	SessionID   string     `json:"sessionId"`
	Timestamp   string     `json:"timestamp"`
	CWD         string     `json:"cwd"`
	GitBranch   string     `json:"gitBranch"`
	IsSidechain bool       `json:"isSidechain"`
	Message     rawMessage `json:"message"`
}

type rawMessage struct {
	ID      string            `json:"id"`
	Role    string            `json:"role"`
	Model   string            `json:"model"`
	Content []json.RawMessage `json:"content"`
	Usage   *rawUsage         `json:"usage"`
}

type rawUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

type rawContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Thinking  string `json:"thinking"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Input     any    `json:"input"`
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"`
	IsError   bool   `json:"is_error"`
}

// ReadFile parses a single Claude Code JSONL session file.
func (r *Reader) ReadFile(path string) (*core.Transcript, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	entries, err := scanEntries(f)
	if err != nil {
		return nil, fmt.Errorf("scan session file: %w", err)
	}

	return buildTranscript(entries)
}

// ReadSession locates and parses a session by its UUID across all projects.
func (r *Reader) ReadSession(sessionID string) (*core.Transcript, error) {
	dir := r.dir()
	fileName := sessionID + ".jsonl"

	projectDirs, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read projects directory: %w", err)
	}

	for _, d := range projectDirs {
		if !d.IsDir() {
			continue
		}
		path := filepath.Join(dir, d.Name(), fileName)
		if _, err := os.Stat(path); err == nil {
			return r.ReadFile(path)
		}
	}

	return nil, fmt.Errorf("session %s not found", sessionID)
}

// ReadProject returns all session transcripts for a named project directory.
func (r *Reader) ReadProject(project string) ([]*core.Transcript, error) {
	projectDir := filepath.Join(r.dir(), project)

	dirEntries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("read project directory: %w", err)
	}

	var transcripts []*core.Transcript
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".jsonl") {
			continue
		}
		t, err := r.ReadFile(filepath.Join(projectDir, de.Name()))
		if err != nil {
			continue
		}
		transcripts = append(transcripts, t)
	}

	return transcripts, nil
}

// ReadAll returns every session transcript across all projects.
func (r *Reader) ReadAll() ([]*core.Transcript, error) {
	dir := r.dir()
	projectDirs, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read projects directory: %w", err)
	}

	var all []*core.Transcript
	for _, d := range projectDirs {
		if !d.IsDir() {
			continue
		}
		transcripts, err := r.ReadProject(d.Name())
		if err != nil {
			continue
		}
		all = append(all, transcripts...)
	}

	return all, nil
}

func (r *Reader) dir() string {
	if r.Dir != "" {
		return r.Dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// scanEntries reads JSONL lines, keeping only user and assistant message entries.
func scanEntries(r io.Reader) ([]rawEntry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

	var entries []rawEntry
	for scanner.Scan() {
		var entry rawEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.IsSidechain {
			continue
		}
		if entry.Type != "user" && entry.Type != "assistant" {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

// buildTranscript assembles a core.Transcript from filtered raw entries.
func buildTranscript(entries []rawEntry) (*core.Transcript, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no messages found in session")
	}

	messages := groupAndMapMessages(entries)
	first := entries[0]
	last := entries[len(entries)-1]

	createdAt := parseTime(first.Timestamp)
	var updatedAt *time.Time
	if last.Timestamp != first.Timestamp {
		t := parseTime(last.Timestamp)
		updatedAt = &t
	}

	return &core.Transcript{
		SessionID: first.SessionID,
		Agent:     "claude",
		Model:     findPrimaryModel(entries),
		Dir:       first.CWD,
		GitBranch: first.GitBranch,
		Title:     deriveTitle(messages),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Usage:     aggregateUsage(messages),
		Messages:  messages,
	}, nil
}

// groupAndMapMessages merges streaming assistant chunks into single messages
// and maps all entries to core.Message values.
//
// Assistant messages arrive as multiple JSONL lines sharing the same message.id,
// each carrying one content block. Tool-result user entries can be interleaved
// between chunks of the same assistant message. This function handles that
// interleaving by tracking the current assistant message group.
func groupAndMapMessages(entries []rawEntry) []core.Message {
	var messages []core.Message
	var currentAssistant *core.Message
	var currentMsgID string

	emit := func() {
		if currentAssistant != nil {
			messages = append(messages, *currentAssistant)
			currentAssistant = nil
			currentMsgID = ""
		}
	}

	for _, entry := range entries {
		if entry.Type == "assistant" {
			msgID := entry.Message.ID
			if msgID == currentMsgID && currentAssistant != nil {
				// Same assistant message — append content blocks, update usage.
				currentAssistant.Content = append(currentAssistant.Content,
					mapContentBlocks(entry.Message.Content, core.RoleAssistant)...)
				if entry.Message.Usage != nil {
					u := mapUsage(entry.Message.Usage)
					currentAssistant.Usage = &u
				}
			} else {
				emit()
				currentMsgID = msgID
				msg := buildAssistantMessage(entry)
				currentAssistant = &msg
			}
		} else {
			// User entry.
			if !isToolResultOnly(entry) {
				// Real human turn — flush pending assistant.
				emit()
			}
			messages = append(messages, buildUserMessage(entry))
		}
	}
	emit()
	return messages
}

func buildAssistantMessage(entry rawEntry) core.Message {
	ts := parseTime(entry.Timestamp)
	m := core.Message{
		UUID:      entry.UUID,
		Role:      core.RoleAssistant,
		Model:     entry.Message.Model,
		Timestamp: &ts,
		Content:   mapContentBlocks(entry.Message.Content, core.RoleAssistant),
	}
	if entry.ParentUUID != nil {
		m.ParentUUID = *entry.ParentUUID
	}
	if entry.Message.Usage != nil {
		u := mapUsage(entry.Message.Usage)
		m.Usage = &u
	}
	return m
}

func buildUserMessage(entry rawEntry) core.Message {
	ts := parseTime(entry.Timestamp)
	m := core.Message{
		UUID:      entry.UUID,
		Role:      core.RoleUser,
		Timestamp: &ts,
		Content:   mapContentBlocks(entry.Message.Content, core.RoleUser),
	}
	if entry.ParentUUID != nil {
		m.ParentUUID = *entry.ParentUUID
	}
	return m
}

// mapContentBlocks decodes raw JSON content blocks into core.ContentBlock values.
func mapContentBlocks(raw []json.RawMessage, role core.Role) []core.ContentBlock {
	var blocks []core.ContentBlock
	for _, r := range raw {
		if b, ok := mapContentBlock(r, role); ok {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func mapContentBlock(raw json.RawMessage, role core.Role) (core.ContentBlock, bool) {
	var b rawContentBlock
	if err := json.Unmarshal(raw, &b); err != nil {
		return core.ContentBlock{}, false
	}

	switch b.Type {
	case "text":
		format := core.FormatPlain
		if role == core.RoleAssistant {
			format = core.FormatMarkdown
		}
		return core.ContentBlock{
			Type:   core.BlockText,
			Format: format,
			Text:   b.Text,
		}, true

	case "thinking":
		return core.ContentBlock{
			Type: core.BlockThinking,
			Text: b.Thinking,
		}, true

	case "tool_use":
		return core.ContentBlock{
			Type:      core.BlockToolUse,
			ToolUseID: b.ID,
			Name:      b.Name,
			Input:     b.Input,
		}, true

	case "tool_result":
		return core.ContentBlock{
			Type:      core.BlockToolResult,
			ToolUseID: b.ToolUseID,
			Content:   extractToolResultContent(b.Content),
			IsError:   b.IsError,
		}, true

	default:
		return core.ContentBlock{}, false
	}
}

// extractToolResultContent handles tool_result content which can be a string
// or an array of {"type":"text","text":"..."} objects.
func extractToolResultContent(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

func mapUsage(raw *rawUsage) core.Usage {
	return core.Usage{
		InputTokens:         raw.InputTokens,
		OutputTokens:         raw.OutputTokens,
		CacheReadTokens:     raw.CacheReadInputTokens,
		CacheCreationTokens: raw.CacheCreationInputTokens,
	}
}

// isToolResultOnly reports whether a user entry contains only tool_result blocks.
func isToolResultOnly(entry rawEntry) bool {
	if len(entry.Message.Content) == 0 {
		return false
	}
	for _, raw := range entry.Message.Content {
		var b struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			continue
		}
		if b.Type != "tool_result" {
			return false
		}
	}
	return true
}

// deriveTitle extracts a title from the first user text block, skipping
// IDE metadata tags. Truncated to 80 characters on a word boundary.
func deriveTitle(messages []core.Message) string {
	for _, m := range messages {
		if m.Role != core.RoleUser {
			continue
		}
		for _, b := range m.Content {
			if b.Type != core.BlockText {
				continue
			}
			text := strings.TrimSpace(b.Text)
			if text == "" || strings.Contains(text, "<ide_opened_file>") {
				continue
			}
			return truncate(text, 80)
		}
		break
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if i := strings.LastIndex(s[:maxLen], " "); i > 0 {
		return s[:i] + "..."
	}
	return s[:maxLen] + "..."
}

func aggregateUsage(messages []core.Message) *core.Usage {
	var total core.Usage
	for _, m := range messages {
		if m.Usage != nil {
			total.Add(*m.Usage)
		}
	}
	if total == (core.Usage{}) {
		return nil
	}
	return &total
}

func findPrimaryModel(entries []rawEntry) string {
	for _, e := range entries {
		if e.Type == "assistant" && e.Message.Model != "" {
			return e.Message.Model
		}
	}
	return ""
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}
