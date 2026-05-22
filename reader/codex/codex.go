// Package codex reads OpenAI Codex rollout JSONL session logs.
package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sonnes/chitragupt/core"
)

// Reader reads Codex rollout JSONL files.
type Reader struct {
	// Dir overrides the default sessions directory (~/.codex/sessions).
	Dir string
}

const maxLineSize = 64 << 20

type rawLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type sessionMeta struct {
	ID            string   `json:"id"`
	Timestamp     string   `json:"timestamp"`
	CWD           string   `json:"cwd"`
	Source        string   `json:"source"`
	Originator    string   `json:"originator"`
	CLIVersion    string   `json:"cli_version"`
	ModelProvider string   `json:"model_provider"`
	Git           *gitMeta `json:"git"`
}

type gitMeta struct {
	Branch        string `json:"branch"`
	CommitHash    string `json:"commit_hash"`
	RepositoryURL string `json:"repository_url"`
}

type turnContext struct {
	TurnID string `json:"turn_id"`
	CWD    string `json:"cwd"`
	Model  string `json:"model"`
	Effort string `json:"effort"`
}

type responseItem struct {
	Type    string           `json:"type"`
	Role    string           `json:"role"`
	Phase   string           `json:"phase"`
	Content []rawContentItem `json:"content"`
	Summary []rawSummaryItem `json:"summary"`
	Name    string           `json:"name"`
	Args    string           `json:"arguments"`
	CallID  string           `json:"call_id"`
	Input   any              `json:"input"`
	Output  json.RawMessage  `json:"output"`
	Action  any              `json:"action"`
	Status  string           `json:"status"`
}

type rawContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	ImageURL string `json:"image_url"`
	Detail   string `json:"detail"`
}

type rawSummaryItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type eventMsg struct {
	Type        string         `json:"type"`
	Message     string         `json:"message"`
	Images      []string       `json:"images"`
	LocalImages []string       `json:"local_images"`
	Info        *tokenInfo     `json:"info"`
	LastMessage string         `json:"last_agent_message"`
	Text        string         `json:"text"`
	CallID      string         `json:"call_id"`
	Stdout      string         `json:"stdout"`
	Stderr      string         `json:"stderr"`
	Success     bool           `json:"success"`
	Changes     []patchChange  `json:"changes"`
	Result      map[string]any `json:"result"`
	Query       string         `json:"query"`
	Action      any            `json:"action"`
}

type patchChange struct {
	Type        string `json:"type"`
	MovePath    string `json:"move_path"`
	UnifiedDiff string `json:"unified_diff"`
	Content     string `json:"content"`
}

type tokenInfo struct {
	LastTokenUsage  *tokenUsage `json:"last_token_usage"`
	TotalTokenUsage *tokenUsage `json:"total_token_usage"`
}

type tokenUsage struct {
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	TotalTokens           int `json:"total_tokens"`
}

type sessionIndexEntry struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
}

// ReadFile parses a single Codex rollout JSONL session file.
func (r *Reader) ReadFile(path string) (*core.Transcript, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	lines, err := scanLines(f)
	if err != nil {
		return nil, fmt.Errorf("scan session file: %w", err)
	}

	t, err := buildTranscript(lines)
	if err != nil {
		return nil, err
	}

	if title := r.sessionIndexTitle(t.SessionID); title != "" {
		t.Title = truncate(title, 80)
	}

	return t, nil
}

// ReadSession locates and parses a Codex session by its ID.
func (r *Reader) ReadSession(sessionID string) (*core.Transcript, error) {
	paths, err := r.rolloutPaths()
	if err != nil {
		return nil, err
	}

	for _, path := range paths {
		id, err := readSessionID(path)
		if err != nil {
			continue
		}
		if id == sessionID {
			return r.ReadFile(path)
		}
	}

	for _, path := range paths {
		if strings.Contains(filepath.Base(path), sessionID) {
			return r.ReadFile(path)
		}
	}

	return nil, fmt.Errorf("session %s not found", sessionID)
}

// ReadProject returns all Codex sessions whose cwd matches project.
func (r *Reader) ReadProject(project string) ([]*core.Transcript, error) {
	project, err := filepath.Abs(project)
	if err != nil {
		return nil, fmt.Errorf("resolve project path: %w", err)
	}

	transcripts, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var matched []*core.Transcript
	for _, t := range transcripts {
		if pathMatchesProject(t.Dir, project) {
			matched = append(matched, t)
		}
	}
	return matched, nil
}

// ReadAll returns every Codex session transcript under the sessions directory.
func (r *Reader) ReadAll() ([]*core.Transcript, error) {
	paths, err := r.rolloutPaths()
	if err != nil {
		return nil, err
	}

	var transcripts []*core.Transcript
	for _, path := range paths {
		t, err := r.ReadFile(path)
		if err != nil {
			continue
		}
		transcripts = append(transcripts, t)
	}

	sort.SliceStable(transcripts, func(i, j int) bool {
		return transcripts[i].CreatedAt.After(transcripts[j].CreatedAt)
	})

	return transcripts, nil
}

func (r *Reader) dir() string {
	if r.Dir != "" {
		return r.Dir
	}
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		return filepath.Join(codexHome, "sessions")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "sessions")
}

func scanLines(r io.Reader) ([]rawLine, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), maxLineSize)

	var lines []rawLine
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var raw rawLine
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		if raw.Type == "" {
			continue
		}
		lines = append(lines, raw)
	}
	return lines, scanner.Err()
}

func buildTranscript(lines []rawLine) (*core.Transcript, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("no codex rows found in session")
	}

	var meta *sessionMeta
	visibleUserMessages := hasVisibleUserMessages(lines)
	state := transcriptState{
		visibleUserMessages: visibleUserMessages,
	}

	for _, line := range lines {
		switch line.Type {
		case "session_meta":
			var m sessionMeta
			if err := json.Unmarshal(line.Payload, &m); err == nil {
				meta = &m
			}
		case "turn_context":
			var ctx turnContext
			if err := json.Unmarshal(line.Payload, &ctx); err == nil {
				state.recordTurnContext(ctx)
			}
		case "event_msg":
			var ev eventMsg
			if err := json.Unmarshal(line.Payload, &ev); err == nil {
				state.applyEvent(line, ev)
			}
		case "response_item":
			var item responseItem
			if err := json.Unmarshal(line.Payload, &item); err == nil {
				state.applyResponseItem(line, item)
			}
		case "compacted":
			state.appendAssistantBlock(line, core.ContentBlock{
				Type:   core.BlockThinking,
				Text:   "Context was compacted.",
				Format: core.FormatPlain,
			})
		}
	}

	state.flushAssistant()

	if len(state.messages) == 0 {
		return nil, fmt.Errorf("no messages found in session")
	}

	first := firstTime(lines)
	last := lastTime(lines)
	if meta != nil && meta.Timestamp != "" {
		first = parseTime(meta.Timestamp)
	}

	var updatedAt *time.Time
	if !last.IsZero() && !last.Equal(first) {
		updatedAt = &last
	}

	t := &core.Transcript{
		Agent:     "codex",
		CreatedAt: first,
		UpdatedAt: updatedAt,
		Usage:     state.totalUsage,
		Messages:  state.messages,
		Title:     deriveTitle(state.messages),
		Model:     state.model,
	}

	if meta != nil {
		t.SessionID = meta.ID
		t.Dir = meta.CWD
		if meta.Git != nil {
			t.GitBranch = meta.Git.Branch
		}
	}
	if t.SessionID == "" {
		t.SessionID = "codex-session"
	}
	if t.Dir == "" {
		t.Dir = state.cwd
	}

	return t, nil
}

type transcriptState struct {
	messages            []core.Message
	currentAssistant    *core.Message
	visibleUserMessages bool
	model               string
	cwd                 string
	totalUsage          *core.Usage
}

func (s *transcriptState) recordTurnContext(ctx turnContext) {
	if s.model == "" {
		s.model = ctx.Model
	}
	if s.cwd == "" {
		s.cwd = ctx.CWD
	}
}

func (s *transcriptState) applyEvent(line rawLine, ev eventMsg) {
	switch ev.Type {
	case "user_message":
		s.flushAssistant()
		ts := parseTime(line.Timestamp)
		text := strings.TrimSpace(ev.Message)
		if text == "" {
			text = summarizeImages(ev.Images, ev.LocalImages)
		}
		if text == "" {
			return
		}
		s.messages = append(s.messages, core.Message{
			Role:      core.RoleUser,
			Timestamp: &ts,
			Content: []core.ContentBlock{
				{
					Type:   core.BlockText,
					Format: core.FormatPlain,
					Text:   text,
				},
			},
		})
	case "token_count":
		if ev.Info != nil && ev.Info.TotalTokenUsage != nil {
			usage := mapUsage(ev.Info.TotalTokenUsage)
			s.totalUsage = &usage
		}
	case "patch_apply_end":
		if ev.CallID == "" {
			return
		}
		if s.hasToolResult(ev.CallID) {
			return
		}
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolResult,
			ToolUseID: ev.CallID,
			Content:   patchApplyResult(ev),
			IsError:   !ev.Success,
		})
	case "mcp_tool_call_end":
		if ev.CallID == "" || s.hasToolResult(ev.CallID) {
			return
		}
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolResult,
			ToolUseID: ev.CallID,
			Content:   stringifyAny(ev.Result),
		})
	}
}

func (s *transcriptState) applyResponseItem(line rawLine, item responseItem) {
	switch item.Type {
	case "message":
		s.applyMessage(line, item)
	case "reasoning":
		text := reasoningSummary(item.Summary)
		if text == "" {
			return
		}
		s.appendAssistantBlock(line, core.ContentBlock{
			Type: core.BlockThinking,
			Text: text,
		})
	case "function_call":
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolUse,
			ToolUseID: item.CallID,
			Name:      item.Name,
			Input:     parseArguments(item.Args),
		})
	case "function_call_output":
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolResult,
			ToolUseID: item.CallID,
			Content:   extractOutput(item.Output),
		})
	case "custom_tool_call":
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolUse,
			ToolUseID: item.CallID,
			Name:      item.Name,
			Input:     item.Input,
		})
	case "custom_tool_call_output", "tool_search_output":
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolResult,
			ToolUseID: item.CallID,
			Content:   extractOutput(item.Output),
		})
	case "web_search_call":
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolUse,
			ToolUseID: item.CallID,
			Name:      "web_search",
			Input:     item.Action,
		})
	case "tool_search_call":
		s.appendAssistantBlock(line, core.ContentBlock{
			Type:      core.BlockToolUse,
			ToolUseID: item.CallID,
			Name:      "tool_search",
			Input:     parseArguments(item.Args),
		})
	}
}

func (s *transcriptState) applyMessage(line rawLine, item responseItem) {
	switch item.Role {
	case "assistant":
		for _, block := range item.Content {
			text := strings.TrimSpace(block.Text)
			if text == "" {
				continue
			}
			s.appendAssistantBlock(line, core.ContentBlock{
				Type:   core.BlockText,
				Format: core.FormatMarkdown,
				Text:   text,
			})
		}
	case "user":
		if s.visibleUserMessages {
			return
		}
		text := contentText(item.Content)
		if text == "" {
			return
		}
		s.flushAssistant()
		ts := parseTime(line.Timestamp)
		s.messages = append(s.messages, core.Message{
			Role:      core.RoleUser,
			Timestamp: &ts,
			Content: []core.ContentBlock{
				{
					Type:   core.BlockText,
					Format: core.FormatPlain,
					Text:   text,
				},
			},
		})
	}
}

func (s *transcriptState) appendAssistantBlock(line rawLine, block core.ContentBlock) {
	ts := parseTime(line.Timestamp)
	if s.currentAssistant == nil {
		msg := core.Message{
			Role:      core.RoleAssistant,
			Model:     s.model,
			Timestamp: &ts,
		}
		s.currentAssistant = &msg
	}
	if s.currentAssistant.Model == "" {
		s.currentAssistant.Model = s.model
	}
	s.currentAssistant.Content = append(s.currentAssistant.Content, block)
}

func (s *transcriptState) flushAssistant() {
	if s.currentAssistant == nil {
		return
	}
	if len(s.currentAssistant.Content) > 0 {
		s.messages = append(s.messages, *s.currentAssistant)
	}
	s.currentAssistant = nil
}

func (s *transcriptState) hasToolResult(toolUseID string) bool {
	if s.currentAssistant != nil {
		for _, block := range s.currentAssistant.Content {
			if block.Type == core.BlockToolResult && block.ToolUseID == toolUseID {
				return true
			}
		}
	}
	for _, msg := range s.messages {
		for _, block := range msg.Content {
			if block.Type == core.BlockToolResult && block.ToolUseID == toolUseID {
				return true
			}
		}
	}
	return false
}

func hasVisibleUserMessages(lines []rawLine) bool {
	for _, line := range lines {
		if line.Type != "event_msg" {
			continue
		}
		var ev eventMsg
		if err := json.Unmarshal(line.Payload, &ev); err == nil && ev.Type == "user_message" {
			return true
		}
	}
	return false
}

func contentText(content []rawContentItem) string {
	var parts []string
	for _, item := range content {
		switch item.Type {
		case "input_text", "output_text":
			text := strings.TrimSpace(item.Text)
			if text != "" {
				parts = append(parts, text)
			}
		case "input_image":
			if item.ImageURL != "" {
				parts = append(parts, "[image: "+item.ImageURL+"]")
			} else {
				parts = append(parts, "[image]")
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

func reasoningSummary(summary []rawSummaryItem) string {
	var parts []string
	for _, item := range summary {
		if item.Type != "summary_text" {
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func summarizeImages(images, localImages []string) string {
	count := len(images) + len(localImages)
	if count == 0 {
		return ""
	}
	if count == 1 {
		return "[image]"
	}
	return fmt.Sprintf("[%d images]", count)
}

func parseArguments(args string) any {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil
	}

	var v any
	if err := json.Unmarshal([]byte(args), &v); err != nil {
		return args
	}
	return v
}

func extractOutput(raw json.RawMessage) string {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	var parts []rawContentItem
	if err := json.Unmarshal(raw, &parts); err == nil {
		return contentText(parts)
	}

	var v any
	if err := json.Unmarshal(raw, &v); err == nil {
		return stringifyAny(v)
	}

	return string(raw)
}

func stringifyAny(v any) string {
	if v == nil {
		return ""
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func patchApplyResult(ev eventMsg) string {
	var parts []string
	if ev.Stdout != "" {
		parts = append(parts, ev.Stdout)
	}
	if ev.Stderr != "" {
		parts = append(parts, ev.Stderr)
	}
	for _, change := range ev.Changes {
		switch {
		case change.UnifiedDiff != "":
			parts = append(parts, change.UnifiedDiff)
		case change.Content != "":
			parts = append(parts, change.Content)
		case change.MovePath != "":
			parts = append(parts, change.Type+": "+change.MovePath)
		case change.Type != "":
			parts = append(parts, change.Type)
		}
	}
	if len(parts) == 0 {
		if ev.Success {
			return "Patch applied."
		}
		return "Patch failed."
	}
	return strings.Join(parts, "\n")
}

func mapUsage(raw *tokenUsage) core.Usage {
	return core.Usage{
		InputTokens:     raw.InputTokens,
		OutputTokens:    raw.OutputTokens,
		CacheReadTokens: raw.CachedInputTokens,
	}
}

func deriveTitle(messages []core.Message) string {
	for _, msg := range messages {
		if msg.Role != core.RoleUser {
			continue
		}
		for _, block := range msg.Content {
			if block.Type != core.BlockText {
				continue
			}
			text := strings.TrimSpace(block.Text)
			if text != "" {
				return truncate(text, 80)
			}
		}
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

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func firstTime(lines []rawLine) time.Time {
	for _, line := range lines {
		if t := parseTime(line.Timestamp); !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

func lastTime(lines []rawLine) time.Time {
	for i := len(lines) - 1; i >= 0; i-- {
		if t := parseTime(lines[i].Timestamp); !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

func pathMatchesProject(cwd, project string) bool {
	if cwd == "" {
		return false
	}

	absCWD, err := filepath.Abs(cwd)
	if err != nil {
		absCWD = cwd
	}
	if absCWD == project {
		return true
	}

	rel, err := filepath.Rel(project, absCWD)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func (r *Reader) rolloutPaths() ([]string, error) {
	dir := r.dir()

	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, "rollout-") && strings.HasSuffix(name, ".jsonl") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk codex sessions: %w", err)
	}

	sort.Strings(paths)
	return paths, nil
}

func readSessionID(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	lines, err := scanLines(f)
	if err != nil {
		return "", err
	}

	for _, line := range lines {
		if line.Type != "session_meta" {
			continue
		}
		var meta sessionMeta
		if err := json.Unmarshal(line.Payload, &meta); err != nil {
			continue
		}
		return meta.ID, nil
	}
	return "", fmt.Errorf("session_meta not found")
}

func (r *Reader) sessionIndexTitle(sessionID string) string {
	if sessionID == "" {
		return ""
	}

	path := filepath.Join(filepath.Dir(r.dir()), "session_index.jsonl")
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64<<10), maxLineSize)
	for scanner.Scan() {
		var entry sessionIndexEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.ID == sessionID {
			return strings.TrimSpace(entry.ThreadName)
		}
	}
	return ""
}
