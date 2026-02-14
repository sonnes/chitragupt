// Package compact provides a Transformer that replaces verbose tool content
// with short summaries for compact transcript viewing.
package compact

import (
	"fmt"
	"strings"

	"github.com/sonnes/chitragupt/core"
)

// Config controls the compact transformer behavior.
type Config struct {
	StripThinking bool
}

// Compactor replaces verbose tool content with line-count summaries.
type Compactor struct {
	stripThinking bool
}

// New creates a Compactor from the given config.
func New(cfg Config) *Compactor {
	return &Compactor{stripThinking: cfg.StripThinking}
}

// Transform implements core.Transformer.
func (c *Compactor) Transform(t *core.Transcript) error {
	for i := range t.Messages {
		c.compactMessage(&t.Messages[i])
	}
	for _, sub := range t.SubAgents {
		if err := c.Transform(sub); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compactor) compactMessage(m *core.Message) {
	if c.stripThinking {
		m.Content = filterThinking(m.Content)
	}
	for j := range m.Content {
		c.compactBlock(&m.Content[j])
	}
}

func filterThinking(blocks []core.ContentBlock) []core.ContentBlock {
	out := make([]core.ContentBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.Type != core.BlockThinking {
			out = append(out, b)
		}
	}
	return out
}

func (c *Compactor) compactBlock(b *core.ContentBlock) {
	switch b.Type {
	case core.BlockToolResult:
		c.compactToolResult(b)
	case core.BlockToolUse:
		c.compactToolUse(b)
	}
}

func (c *Compactor) compactToolResult(b *core.ContentBlock) {
	label := "output"
	if b.IsError {
		label = "error"
	}
	b.Content = lineSummary(label, b.Content)
}

func (c *Compactor) compactToolUse(b *core.ContentBlock) {
	m, ok := b.Input.(map[string]any)
	if !ok || m == nil {
		return
	}
	switch strings.ToLower(b.Name) {
	case "write":
		summarizeMapField(m, "content")
	case "edit":
		summarizeMapField(m, "old_string")
		summarizeMapField(m, "new_string")
	}
}

// lineSummary returns a summary like "[output: 245 lines]" or "[error: 12 lines]".
func lineSummary(label, s string) string {
	n := countLines(s)
	if n == 1 {
		return fmt.Sprintf("[%s: 1 line]", label)
	}
	return fmt.Sprintf("[%s: %d lines]", label, n)
}

// summarizeMapField replaces a string field in a map with a line-count summary.
func summarizeMapField(m map[string]any, key string) {
	v, ok := m[key]
	if !ok {
		return
	}
	s, ok := v.(string)
	if !ok {
		return
	}
	m[key] = lineSummary(key, s)
}

// countLines returns the number of lines in s.
// An empty string has 0 lines. A string with no newline has 1 line.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n") + 1
	if strings.HasSuffix(s, "\n") {
		n--
	}
	return n
}
