// Package html renders transcripts as standalone HTML pages styled with
// Tailwind CSS v4 (CDN) and syntax highlighting via goldmark + chroma.
package html

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"time"

	"github.com/sonnes/chitragupt/core"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
)

// Renderer renders a transcript to a standalone HTML page.
type Renderer struct {
	md   goldmark.Markdown
	tmpl *template.Template

	// SubAgentHref, when non-nil, overrides the default agent-{id}.html link
	// pattern for sub-agent references. Used by the serve command to generate
	// server-routed URLs instead of static file links.
	SubAgentHref func(agentID string) string
}

// New creates an HTML Renderer with goldmark configured for GFM and syntax highlighting.
func New() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(false), // inline styles for standalone pages
				),
			),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(), // allow raw HTML in markdown
		),
	)

	tmpl := template.Must(
		template.New("page.html").
			Funcs(funcMap()).
			ParseFS(content, "templates/*.html"),
	)

	return &Renderer{md: md, tmpl: tmpl}
}

// pageData is the top-level template data passed to page.html.
type pageData struct {
	Transcript      *core.Transcript
	Turns           []turnData
	OverallDuration string // total session duration (e.g. "2m 30s")
}

// turnData groups a user prompt with its assistant response cycle.
type turnData struct {
	ID        string          // anchor ID (e.g. "turn-0")
	User      []template.HTML // rendered user message blocks (nil if turn starts with assistant)
	UserText  string          // raw user text for timeline summary
	Timestamp *time.Time      // user message timestamp (or first assistant timestamp)
	Duration  string          // time since previous turn
	Steps     []template.HTML // rendered intermediate blocks (collapsed)
	StepCount int             // number of tool invocations
	Response  []template.HTML // rendered final text blocks (visible)
}

// indexData is the template data passed to index.html.
type indexData struct {
	Transcripts []*core.Transcript
}

// RenderIndex writes an HTML index page listing the given transcripts to w.
// Transcripts are sorted newest-first by CreatedAt.
func (r *Renderer) RenderIndex(w io.Writer, transcripts []*core.Transcript) error {
	sorted := make([]*core.Transcript, len(transcripts))
	copy(sorted, transcripts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})
	return r.tmpl.ExecuteTemplate(w, "index.html", indexData{Transcripts: sorted})
}

// Render writes the transcript as a complete HTML page to w.
func (r *Renderer) Render(w io.Writer, t *core.Transcript) error {
	// Build tool_result index: tool_use_id → tool_result block.
	resultIndex := make(map[string]core.ContentBlock)
	for _, msg := range t.Messages {
		for _, b := range msg.Content {
			if b.Type == core.BlockToolResult && b.ToolUseID != "" {
				resultIndex[b.ToolUseID] = b
			}
		}
	}

	consumed := make(map[string]bool)

	turns := core.GroupTurns(t.Messages)
	var prevTimestamp *time.Time
	var turnDatas []turnData

	for i, turn := range turns {
		td := turnData{ID: fmt.Sprintf("turn-%d", i)}

		// Render user message blocks.
		if turn.UserMessage != nil {
			td.Timestamp = turn.UserMessage.Timestamp
			td.UserText = userTextSummary(*turn.UserMessage)
			for _, b := range turn.UserMessage.Content {
				rendered, err := r.renderBlock(b, nil)
				if err != nil {
					return fmt.Errorf("render user block: %w", err)
				}
				td.User = append(td.User, rendered)
			}
		} else if len(turn.AssistantMessages) > 0 {
			td.Timestamp = turn.AssistantMessages[0].Timestamp
		}

		if td.Timestamp != nil && prevTimestamp != nil {
			td.Duration = formatDuration(td.Timestamp.Sub(*prevTimestamp))
		}
		if td.Timestamp != nil {
			prevTimestamp = td.Timestamp
		}

		// Split assistant content into steps and response.
		steps, response := turn.SplitContent()
		td.StepCount = turn.StepCount()

		for _, b := range steps {
			rendered, err := r.renderContentBlock(b, resultIndex, consumed)
			if err != nil {
				return err
			}
			if rendered != "" {
				td.Steps = append(td.Steps, rendered)
			}
		}

		for _, b := range response {
			rendered, err := r.renderContentBlock(b, resultIndex, consumed)
			if err != nil {
				return err
			}
			if rendered != "" {
				td.Response = append(td.Response, rendered)
			}
		}

		if len(td.User) > 0 || len(td.Steps) > 0 || len(td.Response) > 0 {
			turnDatas = append(turnDatas, td)
		}
	}

	var overallDuration string
	if t.UpdatedAt != nil && !t.CreatedAt.IsZero() {
		overallDuration = formatDuration(t.UpdatedAt.Sub(t.CreatedAt))
	}

	data := pageData{
		Transcript:      t,
		Turns:           turnDatas,
		OverallDuration: overallDuration,
	}
	return r.tmpl.ExecuteTemplate(w, "page.html", data)
}

// renderContentBlock renders a single content block, handling tool_use/result pairing.
func (r *Renderer) renderContentBlock(b core.ContentBlock, resultIndex map[string]core.ContentBlock, consumed map[string]bool) (template.HTML, error) {
	switch b.Type {
	case core.BlockToolUse:
		var result *core.ContentBlock
		if tr, ok := resultIndex[b.ToolUseID]; ok {
			result = &tr
			consumed[b.ToolUseID] = true
		}
		return r.renderBlock(b, result)
	case core.BlockToolResult:
		if consumed[b.ToolUseID] {
			return "", nil
		}
		return r.renderBlock(b, nil)
	default:
		return r.renderBlock(b, nil)
	}
}

// userTextSummary extracts a short text summary from a user message for the timeline.
func userTextSummary(msg core.Message) string {
	for _, b := range msg.Content {
		if b.Type != core.BlockText {
			continue
		}
		text := core.CleanUserText(b.Text)
		if text == "" {
			continue
		}
		if len(text) > 80 {
			text = text[:77] + "..."
		}
		return text
	}
	return ""
}

