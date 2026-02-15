package html

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/sonnes/chitragupt/core"
)

// renderBlock dispatches to the appropriate block renderer based on type.
// For tool_use blocks, result is the paired tool_result (may be nil).
func (r *Renderer) renderBlock(b core.ContentBlock, result *core.ContentBlock) (template.HTML, error) {
	switch b.Type {
	case core.BlockText:
		return r.renderTextBlock(b)
	case core.BlockThinking:
		return renderThinkingBlock(b)
	case core.BlockToolUse:
		return r.renderToolUseBlock(b, result)
	case core.BlockToolResult:
		return renderToolResultBlock(b)
	default:
		return "", fmt.Errorf("unknown block type: %s", b.Type)
	}
}

func (r *Renderer) renderTextBlock(b core.ContentBlock) (template.HTML, error) {
	if b.Format == core.FormatMarkdown {
		var buf bytes.Buffer
		if err := r.md.Convert([]byte(b.Text), &buf); err != nil {
			return "", fmt.Errorf("goldmark convert: %w", err)
		}
		return template.HTML(`<div class="prose dark:prose-invert max-w-none">` + buf.String() + `</div>`), nil
	}
	text := core.CleanUserText(b.Text)
	if text == "" {
		return "", nil
	}
	escaped := template.HTMLEscapeString(text)
	return template.HTML(`<p class="whitespace-pre-wrap text-sm">` + escaped + `</p>`), nil
}

func renderThinkingBlock(b core.ContentBlock) (template.HTML, error) {
	escaped := template.HTMLEscapeString(b.Text)
	h := `<details class="group">` +
		`<summary class="text-xs font-medium text-slate-400 dark:text-slate-500 cursor-pointer select-none">Thinking…</summary>` +
		`<pre class="mt-2 text-xs text-slate-500 dark:text-slate-400 whitespace-pre-wrap bg-slate-50 dark:bg-slate-900 rounded p-3 max-h-96 overflow-y-auto">` + escaped + `</pre>` +
		`</details>`
	return template.HTML(h), nil
}

func (r *Renderer) renderToolUseBlock(b core.ContentBlock, result *core.ContentBlock) (template.HTML, error) {
	inputJSON := formatToolInput(b.Input)

	var inputHTML string
	if inputJSON != "" {
		var buf bytes.Buffer
		fenced := "```json\n" + inputJSON + "\n```"
		if err := r.md.Convert([]byte(fenced), &buf); err != nil {
			inputHTML = `<pre class="px-4 py-3 text-xs font-mono overflow-x-auto">` + template.HTMLEscapeString(inputJSON) + `</pre>`
		} else {
			inputHTML = `<div class="px-4 py-3 text-xs overflow-x-auto">` + buf.String() + `</div>`
		}
	}

	var resultHTML string
	if result != nil {
		errorClass := ""
		textClass := ""
		if result.IsError {
			errorClass = " bg-red-50 dark:bg-red-950"
			textClass = " text-red-700 dark:text-red-400"
		}
		escaped := template.HTMLEscapeString(result.Content)
		resultHTML = `<div class="border-t border-slate-200 dark:border-slate-700` + errorClass + `">` +
			`<pre class="px-4 py-3 text-xs font-mono overflow-x-auto max-h-96 overflow-y-auto` + textClass + `">` + escaped + `</pre>` +
			`</div>`
	}

	var linkCardHTML string
	if b.SubAgentRef != nil {
		label := b.SubAgentRef.AgentID
		if b.SubAgentRef.AgentName != "" {
			label = b.SubAgentRef.AgentName
		}
		typeLabel := ""
		if b.SubAgentRef.AgentType != "" {
			typeLabel = ` <span class="text-slate-400 dark:text-slate-500">` +
				`(` + template.HTMLEscapeString(b.SubAgentRef.AgentType) + `)</span>`
		}
		href := "agent-" + template.HTMLEscapeString(b.SubAgentRef.AgentID) + ".html"
		if r.SubAgentHref != nil {
			href = r.SubAgentHref(b.SubAgentRef.AgentID)
		}
		linkCardHTML = `<div class="border-t border-slate-200 dark:border-slate-700 px-4 py-2 flex items-center gap-2 bg-indigo-50 dark:bg-indigo-950">` +
			`<span class="text-xs">&#128279;</span>` +
			`<a href="` + href + `" class="text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:underline">` +
			template.HTMLEscapeString(label) + typeLabel +
			`</a>` +
			`<span class="ml-auto text-xs text-indigo-400 dark:text-indigo-500">View &rarr;</span>` +
			`</div>`
	}

	toolName := template.HTMLEscapeString(b.Name)
	icon := string(toolIcon(b.Name))
	summaryDetail := ""
	if s := toolInputSummary(b.Name, b.Input); s != "" {
		summaryDetail = ` <span class="text-xs font-mono text-slate-500 dark:text-slate-400 truncate">` + template.HTMLEscapeString(s) + `</span>`
	}
	h := `<details class="bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">` +
		`<summary class="px-4 py-2 flex items-center gap-2 text-slate-900 dark:text-white cursor-pointer select-none min-w-0">` +
		icon +
		`<span class="text-xs font-semibold font-mono shrink-0">` + toolName + `</span>` +
		summaryDetail +
		`</summary>` +
		inputHTML +
		resultHTML +
		linkCardHTML +
		`</details>`
	return template.HTML(h), nil
}

// renderToolResultBlock renders an orphan tool_result with no matching tool_use.
func renderToolResultBlock(b core.ContentBlock) (template.HTML, error) {
	escaped := template.HTMLEscapeString(b.Content)
	classes := "text-xs font-mono bg-slate-50 dark:bg-slate-900 rounded p-3 overflow-x-auto"
	if b.IsError {
		classes += " border-l-4 border-red-500 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-400"
	}
	h := `<pre class="` + classes + `">` + escaped + `</pre>`
	return template.HTML(h), nil
}

// toolInputSummary extracts a short label from tool input for the header line.
func toolInputSummary(toolName string, input any) string {
	m, ok := input.(map[string]any)
	if !ok {
		return ""
	}
	var key string
	switch strings.ToLower(toolName) {
	case "read", "write", "edit":
		key = "file_path"
	case "bash":
		key = "command"
	case "glob":
		key = "pattern"
	case "grep":
		key = "pattern"
	case "task":
		key = "description"
	case "notebookedit":
		key = "notebook_path"
	default:
		return ""
	}
	v, ok := m[key].(string)
	if !ok || v == "" {
		return ""
	}
	return v
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
