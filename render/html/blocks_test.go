package html

import (
	"testing"

	"github.com/sonnes/chitragupt/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRenderer() *Renderer {
	return New()
}

func TestRenderTextBlockMarkdown(t *testing.T) {
	r := testRenderer()
	tests := []struct {
		name     string
		block    core.ContentBlock
		contains []string
	}{
		{
			name:     "bold text",
			block:    core.ContentBlock{Type: core.BlockText, Format: core.FormatMarkdown, Text: "Hello **world**"},
			contains: []string{"<strong>world</strong>", `class="prose`},
		},
		{
			name:     "code fence",
			block:    core.ContentBlock{Type: core.BlockText, Format: core.FormatMarkdown, Text: "```go\nfmt.Println(\"hi\")\n```"},
			contains: []string{"<pre", "Println"},
		},
		{
			name:     "inline code",
			block:    core.ContentBlock{Type: core.BlockText, Format: core.FormatMarkdown, Text: "Use `git status` to check."},
			contains: []string{"<code>git status</code>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := r.renderTextBlock(tt.block)
			require.NoError(t, err)
			for _, s := range tt.contains {
				assert.Contains(t, string(out), s)
			}
		})
	}
}

func TestRenderTextBlockPlain(t *testing.T) {
	r := testRenderer()
	tests := []struct {
		name     string
		text     string
		contains string
		absent   string
	}{
		{
			name:     "simple text",
			text:     "hello world",
			contains: "hello world",
		},
		{
			name:     "html escaped",
			text:     "<script>alert('xss')</script>",
			contains: "&lt;script&gt;",
			absent:   "<script>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := core.ContentBlock{Type: core.BlockText, Format: core.FormatPlain, Text: tt.text}
			out, err := r.renderTextBlock(b)
			require.NoError(t, err)
			assert.Contains(t, string(out), tt.contains)
			if tt.absent != "" {
				assert.NotContains(t, string(out), tt.absent)
			}
			assert.NotContains(t, string(out), `class="prose"`, "plain text should not use prose wrapper")
		})
	}
}

func TestRenderThinkingBlock(t *testing.T) {
	b := core.ContentBlock{Type: core.BlockThinking, Text: "Let me analyze the code..."}
	out, err := renderThinkingBlock(b)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "<details")
	assert.Contains(t, s, "<summary")
	assert.Contains(t, s, "Thinking")
	assert.Contains(t, s, "Let me analyze the code...")
}

func TestRenderThinkingBlockEscaping(t *testing.T) {
	b := core.ContentBlock{Type: core.BlockThinking, Text: "Check if x < 10 && y > 5"}
	out, err := renderThinkingBlock(b)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "&lt;")
	assert.Contains(t, s, "&amp;&amp;")
}

func TestRenderToolUseBlockPaired(t *testing.T) {
	r := testRenderer()
	use := core.ContentBlock{
		Type:      core.BlockToolUse,
		ToolUseID: "t1",
		Name:      "Bash",
		Input:     map[string]any{"command": "git status"},
	}
	result := &core.ContentBlock{
		Type:      core.BlockToolResult,
		ToolUseID: "t1",
		Content:   "On branch main\nnothing to commit",
		IsError:   false,
	}

	out, err := r.renderToolUseBlock(use, result)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "Bash", "should show tool name")
	assert.Contains(t, s, "git status", "should show input")
	assert.Contains(t, s, "On branch main", "should show result")
	assert.NotContains(t, s, "border-red", "non-error should not have red styling")
}

func TestRenderToolUseBlockUnpaired(t *testing.T) {
	r := testRenderer()
	use := core.ContentBlock{
		Type:      core.BlockToolUse,
		ToolUseID: "t2",
		Name:      "Read",
		Input:     map[string]any{"file_path": "/tmp/test.go"},
	}

	out, err := r.renderToolUseBlock(use, nil)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "Read")
	assert.Contains(t, s, "test.go")
}

func TestRenderToolUseBlockNilInput(t *testing.T) {
	r := testRenderer()
	use := core.ContentBlock{
		Type:      core.BlockToolUse,
		ToolUseID: "t3",
		Name:      "TodoRead",
		Input:     nil,
	}

	out, err := r.renderToolUseBlock(use, nil)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "TodoRead")
}

func TestRenderToolResultBlockError(t *testing.T) {
	b := core.ContentBlock{
		Type:      core.BlockToolResult,
		ToolUseID: "t4",
		Content:   "command not found: foobar",
		IsError:   true,
	}

	out, err := renderToolResultBlock(b)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "command not found: foobar")
	assert.Contains(t, s, "border-red-500")
	assert.Contains(t, s, "text-red-700")
}

func TestRenderToolResultBlockNonError(t *testing.T) {
	b := core.ContentBlock{
		Type:      core.BlockToolResult,
		ToolUseID: "t5",
		Content:   "OK",
		IsError:   false,
	}

	out, err := renderToolResultBlock(b)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "OK")
	assert.NotContains(t, s, "border-red")
}

func TestRenderToolUseBlockErrorResult(t *testing.T) {
	r := testRenderer()
	use := core.ContentBlock{
		Type:      core.BlockToolUse,
		ToolUseID: "t6",
		Name:      "Bash",
		Input:     map[string]any{"command": "false"},
	}
	result := &core.ContentBlock{
		Type:      core.BlockToolResult,
		ToolUseID: "t6",
		Content:   "exit status 1",
		IsError:   true,
	}

	out, err := r.renderToolUseBlock(use, result)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "bg-red-50", "error result should have red background")
	assert.Contains(t, s, "text-red-700", "error result should have red text")
	assert.Contains(t, s, "exit status 1")
}

func TestRenderToolUseBlockSubAgentHref(t *testing.T) {
	r := testRenderer()
	r.SubAgentHref = func(agentID string) string {
		return "/session/" + agentID
	}

	use := core.ContentBlock{
		Type:      core.BlockToolUse,
		ToolUseID: "t7",
		Name:      "Task",
		Input:     map[string]any{"prompt": "do stuff"},
		SubAgentRef: &core.SubAgentRef{
			AgentID:   "abc123",
			AgentName: "researcher",
			AgentType: "Explore",
		},
	}

	out, err := r.renderToolUseBlock(use, nil)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `href="/session/abc123"`, "should use custom SubAgentHref")
	assert.NotContains(t, s, "agent-abc123.html", "should not use default file link")
	assert.Contains(t, s, "researcher", "should show agent name")
	assert.Contains(t, s, "(Explore)", "should show agent type")
}

func TestRenderToolUseBlockSubAgentDefaultHref(t *testing.T) {
	r := testRenderer()

	use := core.ContentBlock{
		Type:      core.BlockToolUse,
		ToolUseID: "t8",
		Name:      "Task",
		SubAgentRef: &core.SubAgentRef{
			AgentID: "def456",
		},
	}

	out, err := r.renderToolUseBlock(use, nil)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `href="agent-def456.html"`, "should use default file link when SubAgentHref is nil")
}

func TestFormatToolInput(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		expect string
	}{
		{name: "nil", input: nil, expect: ""},
		{name: "simple map", input: map[string]any{"key": "value"}, expect: `"key": "value"`},
		{name: "nested", input: map[string]any{"a": map[string]any{"b": 1}}, expect: `"b": 1`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := formatToolInput(tt.input)
			if tt.expect == "" {
				assert.Empty(t, out)
			} else {
				assert.Contains(t, out, tt.expect)
			}
		})
	}
}
