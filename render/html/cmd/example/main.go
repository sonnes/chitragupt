// Generates an example HTML transcript and writes it to stdout.
// Usage: go run ./render/html/cmd/example > example.html
package main

import (
	"os"
	"time"

	"github.com/sonnes/chitragupt/core"
	htmlrender "github.com/sonnes/chitragupt/render/html"
)

func main() {
	now := time.Date(2026, 2, 13, 10, 15, 0, 0, time.UTC)
	t1 := now.Add(2 * time.Second)
	t2 := now.Add(15 * time.Second)
	t3 := now.Add(18 * time.Second)
	t4 := now.Add(25 * time.Second)
	t5 := now.Add(40 * time.Second)
	t6 := now.Add(55 * time.Second)
	t7 := now.Add(70 * time.Second)
	t8 := now.Add(90 * time.Second)
	end := now.Add(2 * time.Minute)

	tr := &core.Transcript{
		SessionID: "8397fc7c-39b9-4e25-81da-ed47a574a88a",
		Agent:     "claude",
		Model:     "claude-opus-4-6",
		Dir:       "/home/user/code/chitragupt",
		GitBranch: "feat/html-renderer",
		Title:     "Add HTML renderer with Tailwind styling",
		CreatedAt: now,
		UpdatedAt: &end,
		Usage: &core.Usage{
			InputTokens:         48520,
			OutputTokens:        12340,
			CacheReadTokens:     35000,
			CacheCreationTokens: 8200,
		},
		Messages: []core.Message{
			{
				UUID:      "msg-001",
				Role:      core.RoleUser,
				Timestamp: &now,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatPlain, Text: "Add an HTML renderer that produces standalone pages styled with Tailwind CSS v4. It should support all content block types and pair tool_use with tool_result."},
				},
			},
			{
				UUID:      "msg-002",
				Role:      core.RoleAssistant,
				Model:     "claude-opus-4-6",
				Timestamp: &t1,
				Content: []core.ContentBlock{
					{Type: core.BlockThinking, Text: "The user wants an HTML renderer for the chitragupt project. I need to:\n1. Check the existing render interface\n2. Look at the core transcript types\n3. Design the HTML output with Tailwind v4 CDN\n4. Implement block rendering for all types\n5. Add tool pairing logic\n\nLet me start by reading the existing code."},
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "I'll implement the HTML renderer. Let me first check the existing interface and types."},
					{Type: core.BlockToolUse, ToolUseID: "toolu_01", Name: "Read", Input: map[string]any{"file_path": "/home/user/code/chitragupt/render/render.go"}},
				},
				Usage: &core.Usage{InputTokens: 12000, OutputTokens: 350},
			},
			{
				UUID:      "msg-003",
				Role:      core.RoleUser,
				Timestamp: &t2,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "toolu_01", Content: "package render\n\nimport (\n\t\"io\"\n\t\"github.com/sonnes/chitragupt/core\"\n)\n\n// Renderer writes a transcript to the given writer in a specific format.\ntype Renderer interface {\n\tRender(w io.Writer, t *core.Transcript) error\n}", IsError: false},
				},
			},
			{
				UUID:      "msg-004",
				Role:      core.RoleAssistant,
				Model:     "claude-opus-4-6",
				Timestamp: &t3,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "Good, the interface is straightforward. Now let me check the core types."},
					{Type: core.BlockToolUse, ToolUseID: "toolu_02", Name: "Read", Input: map[string]any{"file_path": "/home/user/code/chitragupt/core/transcript.go"}},
				},
				Usage: &core.Usage{InputTokens: 8000, OutputTokens: 120},
			},
			{
				UUID:      "msg-005",
				Role:      core.RoleUser,
				Timestamp: &t4,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "toolu_02", Content: "package core\n\nimport \"time\"\n\ntype Transcript struct {\n\tSessionID string    `json:\"session_id\"`\n\tAgent     string    `json:\"agent\"`\n\tModel     string    `json:\"model,omitempty\"`\n\tTitle     string    `json:\"title,omitempty\"`\n\tCreatedAt time.Time `json:\"created_at\"`\n\tMessages  []Message `json:\"messages\"`\n}\n\ntype Message struct {\n\tRole    Role           `json:\"role\"`\n\tContent []ContentBlock `json:\"content\"`\n}\n\n// ... (truncated)", IsError: false},
				},
			},
			{
				UUID:      "msg-006",
				Role:      core.RoleAssistant,
				Model:     "claude-opus-4-6",
				Timestamp: &t5,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "Now I have everything I need. Let me create the HTML renderer.\n\n## Implementation Plan\n\nI'll create these files:\n\n| File | Purpose |\n|------|--------|\n| `html.go` | Renderer struct, `New()`, `Render()` |\n| `blocks.go` | Per-block-type rendering functions |\n| `embed.go` | Template embedding |\n| `templates/page.html` | Page layout with Tailwind v4 |\n\nLet me start with the main renderer."},
					{Type: core.BlockToolUse, ToolUseID: "toolu_03", Name: "Write", Input: map[string]any{"file_path": "/home/user/code/chitragupt/render/html/html.go", "content": "package html\n\n// ... renderer implementation"}},
				},
				Usage: &core.Usage{InputTokens: 15000, OutputTokens: 4500},
			},
			{
				UUID:      "msg-007",
				Role:      core.RoleUser,
				Timestamp: &t6,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "toolu_03", Content: "File created successfully at: /home/user/code/chitragupt/render/html/html.go", IsError: false},
				},
			},
			{
				UUID:      "msg-008",
				Role:      core.RoleAssistant,
				Model:     "claude-opus-4-6",
				Timestamp: &t7,
				Content: []core.ContentBlock{
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "Now let me run the tests to verify everything works."},
					{Type: core.BlockToolUse, ToolUseID: "toolu_04", Name: "Bash", Input: map[string]any{"command": "go test ./render/html/... -v -count=1"}},
				},
				Usage: &core.Usage{InputTokens: 10000, OutputTokens: 200},
			},
			{
				UUID:      "msg-009",
				Role:      core.RoleUser,
				Timestamp: &t8,
				Content: []core.ContentBlock{
					{Type: core.BlockToolResult, ToolUseID: "toolu_04", Content: "--- FAIL: TestRenderFullPage (0.01s)\n    html_test.go:89: expected page to contain Tailwind CDN link\nFAIL\nFAIL\tgithub.com/sonnes/chitragupt/render/html\t0.523s", IsError: true},
				},
			},
			{
				UUID:      "msg-010",
				Role:      core.RoleAssistant,
				Model:     "claude-opus-4-6",
				Timestamp: &end,
				Content: []core.ContentBlock{
					{Type: core.BlockThinking, Text: "The test is failing because I forgot to add the Tailwind CDN script tag to the page template. Let me fix the template."},
					{Type: core.BlockText, Format: core.FormatMarkdown, Text: "The test found a missing Tailwind CDN link in the template. Let me fix that and re-run.\n\n```html\n<script src=\"https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4\"></script>\n```\n\nAll **22 tests** now pass. Here's what was implemented:\n\n- Standalone HTML pages with Tailwind v4 CDN\n- Syntax highlighting via goldmark + chroma\n- Tool use/result pairing by `tool_use_id`\n- Collapsible thinking blocks\n- Dark mode via `prefers-color-scheme`\n- Error styling for failed tool results"},
				},
				Usage: &core.Usage{InputTokens: 3520, OutputTokens: 7170},
			},
		},
	}

	r := htmlrender.New()
	if err := r.Render(os.Stdout, tr); err != nil {
		os.Stderr.WriteString("error: " + err.Error() + "\n")
		os.Exit(1)
	}
}
