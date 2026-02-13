// Package terminal renders transcripts as ANSI-colored tree views.
package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/charmbracelet/x/term"
	"github.com/sonnes/chitragupt/core"
)

const defaultWidth = 100

// Renderer pretty-prints a transcript as a tree to the terminal.
type Renderer struct {
	// Width overrides terminal width detection. Zero means auto-detect.
	Width int
}

// New creates a terminal Renderer.
func New() *Renderer {
	return &Renderer{}
}

// Render writes the transcript as an ANSI-colored tree to w.
func (r *Renderer) Render(w io.Writer, t *core.Transcript) error {
	width := r.termWidth()
	turns := groupTurns(t.Messages)
	root := r.buildTree(turns, width)

	if _, err := fmt.Fprint(w, root.String()); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	total := countLines(turns)
	_, err := fmt.Fprintln(w, styleCounter.Render(fmt.Sprintf("  (%d/%d)", total, total)))
	return err
}

func (r *Renderer) termWidth() int {
	if r.Width > 0 {
		return r.Width
	}
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		return w
	}
	return defaultWidth
}

// nodeKind distinguishes child items within a turn.
type nodeKind int

const (
	nodeToolUse nodeKind = iota
	nodeAssistantText
)

// node is a single item within a turn.
type node struct {
	kind nodeKind
	text string
}

// turn is a conversation turn starting with a user message.
type turn struct {
	prompt   string
	children []node
}

// groupTurns partitions flat messages into conversation turns.
// A new turn starts at each user message with a text block.
// Tool-result-only user messages are skipped.
func groupTurns(messages []core.Message) []turn {
	var turns []turn
	var current *turn

	for _, msg := range messages {
		switch msg.Role {
		case core.RoleUser:
			if isToolResultOnly(msg) {
				continue
			}
			if current != nil {
				turns = append(turns, *current)
			}
			current = &turn{prompt: extractUserPrompt(msg)}

		case core.RoleAssistant:
			if current == nil {
				current = &turn{prompt: "(start)"}
			}
			for _, block := range msg.Content {
				switch block.Type {
				case core.BlockToolUse:
					current.children = append(current.children, node{
						kind: nodeToolUse,
						text: summarizeToolUse(block),
					})
				case core.BlockText:
					text := strings.TrimSpace(block.Text)
					if text != "" {
						current.children = append(current.children, node{
							kind: nodeAssistantText,
							text: text,
						})
					}
				}
			}
		}
	}

	if current != nil {
		turns = append(turns, *current)
	}
	return turns
}

func isToolResultOnly(msg core.Message) bool {
	if len(msg.Content) == 0 {
		return false
	}
	for _, b := range msg.Content {
		if b.Type != core.BlockToolResult {
			return false
		}
	}
	return true
}

func extractUserPrompt(msg core.Message) string {
	for _, b := range msg.Content {
		if b.Type == core.BlockText {
			return strings.TrimSpace(b.Text)
		}
	}
	return ""
}

// buildTree constructs a lipgloss tree from the grouped turns.
func (r *Renderer) buildTree(turns []turn, width int) *tree.Tree {
	contentWidth := width - 8
	if contentWidth < 40 {
		contentWidth = 40
	}

	root := tree.New().
		EnumeratorStyle(styleConnector)

	for i, t := range turns {
		isLast := i == len(turns)-1

		userLabel := styleUser.Render("user: " + truncate(t.prompt, contentWidth-6))

		if isLast {
			userLabel = styleUser.Render("> ") + userLabel
		}

		if len(t.children) == 0 {
			root.Child(userLabel)
			continue
		}

		childWidth := contentWidth - 8
		turnNode := tree.Root(userLabel).
			Enumerator(childEnumerator).
			EnumeratorStyle(styleConnector)

		for _, child := range t.children {
			var styled string
			switch child.kind {
			case nodeToolUse:
				styled = styleTool.Render(truncate(child.text, childWidth))
			case nodeAssistantText:
				styled = styleAssistant.Render("assistant: " + truncate(child.text, childWidth-11))
			}
			turnNode.Child(styled)
		}

		root.Child(turnNode)
	}

	return root
}

// childEnumerator uses spaces for children within a turn.
func childEnumerator(_ tree.Children, _ int) string {
	return "    "
}

// truncate shortens text to maxWidth, appending "..." if needed.
// Multi-line text is reduced to the first line.
func truncate(s string, maxWidth int) string {
	if maxWidth < 4 {
		maxWidth = 4
	}
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)

	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Truncate by runes to avoid splitting multi-byte chars
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+3 > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "..."
}

// countLines counts total display lines (turns + their children).
func countLines(turns []turn) int {
	n := 0
	for _, t := range turns {
		n++ // the user prompt line
		n += len(t.children)
	}
	return n
}
