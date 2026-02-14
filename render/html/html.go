// Package html renders transcripts as standalone HTML pages styled with
// Tailwind CSS v4 (CDN) and syntax highlighting via goldmark + chroma.
package html

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"
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
	Messages        []messageData
	OverallDuration string // total session duration (e.g. "2m 30s")
}

// messageData is the per-message template data passed to message.html.
type messageData struct {
	ID          string // anchor ID for timeline links (e.g. "msg-0")
	Message     core.Message
	RoleLabel   string
	BorderClass string
	BadgeClass  string
	DotClass    string // timeline dot color class
	Timestamp   *time.Time
	Duration    string   // time since previous message (e.g. "4s")
	Summary     string   // short text description for timeline sidebar
	Tools       []string // tool names used in this message (for timeline icons)
	Blocks      []template.HTML
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

	var prevTimestamp *time.Time
	var messages []messageData
	for i, msg := range t.Messages {
		md := messageData{
			ID:          fmt.Sprintf("msg-%d", i),
			Message:     msg,
			RoleLabel:   roleLabel(msg.Role),
			BorderClass: borderClass(msg.Role),
			BadgeClass:  badgeClass(msg.Role),
			DotClass:    dotClass(msg.Role),
			Timestamp:   msg.Timestamp,
		}
		if msg.Timestamp != nil && prevTimestamp != nil {
			md.Duration = formatDuration(msg.Timestamp.Sub(*prevTimestamp))
		}
		if msg.Timestamp != nil {
			prevTimestamp = msg.Timestamp
		}
		md.Summary, md.Tools = messageSummary(msg)

		hasContent := false
		for _, b := range msg.Content {
			switch b.Type {
			case core.BlockToolUse:
				var result *core.ContentBlock
				if tr, ok := resultIndex[b.ToolUseID]; ok {
					result = &tr
					consumed[b.ToolUseID] = true
				}
				rendered, err := r.renderBlock(b, result)
				if err != nil {
					return fmt.Errorf("render tool_use block: %w", err)
				}
				md.Blocks = append(md.Blocks, rendered)
				hasContent = true

			case core.BlockToolResult:
				if consumed[b.ToolUseID] {
					continue
				}
				rendered, err := r.renderBlock(b, nil)
				if err != nil {
					return fmt.Errorf("render tool_result block: %w", err)
				}
				md.Blocks = append(md.Blocks, rendered)
				hasContent = true

			default:
				rendered, err := r.renderBlock(b, nil)
				if err != nil {
					return fmt.Errorf("render %s block: %w", b.Type, err)
				}
				md.Blocks = append(md.Blocks, rendered)
				hasContent = true
			}
		}

		if hasContent {
			messages = append(messages, md)
		}
	}

	var overallDuration string
	if t.UpdatedAt != nil && !t.CreatedAt.IsZero() {
		overallDuration = formatDuration(t.UpdatedAt.Sub(t.CreatedAt))
	}

	data := pageData{
		Transcript:      t,
		Messages:        messages,
		OverallDuration: overallDuration,
	}
	return r.tmpl.ExecuteTemplate(w, "page.html", data)
}

func roleLabel(role core.Role) string {
	switch role {
	case core.RoleUser:
		return "User"
	case core.RoleAssistant:
		return "Assistant"
	case core.RoleSystem:
		return "System"
	default:
		return string(role)
	}
}

func borderClass(role core.Role) string {
	switch role {
	case core.RoleUser:
		return "border-l-4 border-l-blue-500"
	case core.RoleAssistant:
		return "border-l-4 border-l-emerald-500"
	case core.RoleSystem:
		return "border-l-4 border-l-slate-400"
	default:
		return ""
	}
}

func badgeClass(role core.Role) string {
	switch role {
	case core.RoleUser:
		return "text-blue-700 dark:text-blue-400 bg-blue-50 dark:bg-blue-950"
	case core.RoleAssistant:
		return "text-emerald-700 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950"
	case core.RoleSystem:
		return "text-slate-600 dark:text-slate-400 bg-slate-100 dark:bg-slate-800"
	default:
		return ""
	}
}

func dotClass(role core.Role) string {
	switch role {
	case core.RoleUser:
		return "bg-blue-500"
	case core.RoleAssistant:
		return "bg-emerald-500"
	case core.RoleSystem:
		return "bg-slate-400"
	default:
		return "bg-slate-300"
	}
}

// messageSummary returns a short text summary and list of tool names for the timeline.
func messageSummary(msg core.Message) (string, []string) {
	var summary string
	var tools []string
	for _, b := range msg.Content {
		switch b.Type {
		case core.BlockText:
			if summary == "" {
				text := strings.TrimSpace(b.Text)
				if text == "" || strings.HasPrefix(text, "<ide_") || strings.HasPrefix(text, "<system-reminder>") {
					continue
				}
				if len(text) > 50 {
					text = text[:47] + "..."
				}
				summary = text
			}
		case core.BlockToolUse:
			tools = append(tools, b.Name)
		}
	}
	if summary == "" && len(tools) > 0 {
		summary = strings.Join(tools, ", ")
	}
	return summary, tools
}
