// Package markdown renders transcripts as Markdown documents.
package markdown

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sonnes/chitragupt/core"
)

// Renderer renders a transcript to Markdown.
type Renderer struct{}

// New creates a Markdown Renderer.
func New() *Renderer {
	return &Renderer{}
}

// Render writes the transcript as a Markdown document to w.
func (r *Renderer) Render(w io.Writer, t *core.Transcript) error {
	title := t.Title
	if title == "" {
		title = "Session " + t.SessionID
	}

	writeLine(w, "# "+title)
	writeLine(w, "")
	writeMetadata(w, t)

	turns := core.GroupTurns(t.Messages)
	for i, turn := range turns {
		writeTurn(w, i+1, turn)
	}

	return nil
}

func writeMetadata(w io.Writer, t *core.Transcript) {
	var items []string
	if t.Agent != "" {
		items = append(items, "**Agent:** "+t.Agent)
	}
	if t.Author != "" {
		items = append(items, "**Author:** @"+t.Author)
	}
	if t.Model != "" {
		items = append(items, "**Model:** "+t.Model)
	}
	if !t.CreatedAt.IsZero() {
		items = append(items, "**Created:** "+t.CreatedAt.Format("2006-01-02 15:04 MST"))
	}
	if t.Dir != "" {
		dir := t.Dir
		if t.GitBranch != "" {
			dir += " (" + t.GitBranch + ")"
		}
		items = append(items, "**Directory:** `"+dir+"`")
	}
	if t.Usage != nil {
		usage := fmt.Sprintf(
			"**Usage:** %s input / %s output tokens",
			formatNumber(t.Usage.InputTokens),
			formatNumber(t.Usage.OutputTokens),
		)
		items = append(items, usage)
	}
	if t.DiffStats != nil {
		diff := fmt.Sprintf(
			"**Diff:** +%s / ~%s / -%s",
			formatNumber(t.DiffStats.Added),
			formatNumber(t.DiffStats.Changed),
			formatNumber(t.DiffStats.Removed),
		)
		items = append(items, diff)
	}

	for _, item := range items {
		writeLine(w, "- "+item)
	}
	if len(items) > 0 {
		writeLine(w, "")
	}
}

func writeTurn(w io.Writer, n int, turn core.Turn) {
	writeLine(w, fmt.Sprintf("## Turn %d", n))
	writeLine(w, "")

	if turn.UserMessage != nil {
		writeLine(w, "### User")
		writeLine(w, "")
		writeMessageBlocks(w, turn.UserMessage.Content)
	}

	steps, response := turn.SplitContent()
	if len(steps) > 0 {
		writeLine(w, "### Steps")
		writeLine(w, "")
		writeMessageBlocks(w, steps)
	}

	if len(response) > 0 {
		writeLine(w, "### Assistant")
		writeLine(w, "")
		writeMessageBlocks(w, response)
	}
}

func writeMessageBlocks(w io.Writer, blocks []core.ContentBlock) {
	for _, block := range blocks {
		switch block.Type {
		case core.BlockText:
			text := block.Text
			if block.Format == core.FormatPlain {
				text = core.CleanUserText(text)
			}
			writeParagraph(w, text)
		case core.BlockThinking:
			writeLine(w, "<details>")
			writeLine(w, "<summary>Thinking</summary>")
			writeLine(w, "")
			writeFence(w, block.Text, "")
			writeLine(w, "</details>")
			writeLine(w, "")
		case core.BlockToolUse:
			writeToolUse(w, block)
		case core.BlockToolResult:
			writeToolResult(w, block)
		}
	}
}

func writeToolUse(w io.Writer, block core.ContentBlock) {
	name := block.Name
	if name == "" {
		name = "tool"
	}

	writeLine(w, "#### "+name)
	writeLine(w, "")

	input := formatToolInput(block.Input)
	if input != "" {
		writeFence(w, input, "json")
	}

	if block.SubAgentRef != nil {
		label := block.SubAgentRef.AgentID
		if block.SubAgentRef.AgentName != "" {
			label = block.SubAgentRef.AgentName
		}
		writeLine(w, "Sub-agent: `"+label+"`")
		writeLine(w, "")
	}
}

func writeToolResult(w io.Writer, block core.ContentBlock) {
	label := "Tool result"
	if block.IsError {
		label = "Tool error"
	}
	writeLine(w, "#### "+label)
	writeLine(w, "")
	writeFence(w, block.Content, "")
}

func writeParagraph(w io.Writer, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	writeLine(w, text)
	writeLine(w, "")
}

func writeFence(w io.Writer, text, language string) {
	fence := "```"
	if strings.Contains(text, "```") {
		fence = "~~~"
	}

	writeLine(w, fence+language)
	writeLine(w, text)
	writeLine(w, fence)
	writeLine(w, "")
}

func writeLine(w io.Writer, text string) {
	fmt.Fprintln(w, text)
}

func formatToolInput(input any) string {
	if input == nil {
		return ""
	}
	data, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", input)
	}
	return string(data)
}

func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}
