package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupTurns(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		want     int // number of turns
		checks   func(t *testing.T, turns []Turn)
	}{
		{
			name:     "empty",
			messages: nil,
			want:     0,
		},
		{
			name: "single user message",
			messages: []Message{
				{Role: RoleUser, Content: []ContentBlock{{Type: BlockText, Text: "hello"}}},
			},
			want: 1,
			checks: func(t *testing.T, turns []Turn) {
				assert.NotNil(t, turns[0].UserMessage)
				assert.Equal(t, "hello", turns[0].UserMessage.Content[0].Text)
				assert.Empty(t, turns[0].AssistantMessages)
			},
		},
		{
			name: "single assistant message",
			messages: []Message{
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockText, Text: "hi"}}},
			},
			want: 1,
			checks: func(t *testing.T, turns []Turn) {
				assert.Nil(t, turns[0].UserMessage)
				require.Len(t, turns[0].AssistantMessages, 1)
			},
		},
		{
			name: "user then assistant",
			messages: []Message{
				{Role: RoleUser, Content: []ContentBlock{{Type: BlockText, Text: "do stuff"}}},
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockText, Text: "done"}}},
			},
			want: 1,
			checks: func(t *testing.T, turns []Turn) {
				assert.NotNil(t, turns[0].UserMessage)
				require.Len(t, turns[0].AssistantMessages, 1)
			},
		},
		{
			name: "multi turn",
			messages: []Message{
				{Role: RoleUser, Content: []ContentBlock{{Type: BlockText, Text: "first"}}},
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockText, Text: "reply1"}}},
				{Role: RoleUser, Content: []ContentBlock{{Type: BlockText, Text: "second"}}},
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockText, Text: "reply2"}}},
			},
			want: 2,
			checks: func(t *testing.T, turns []Turn) {
				assert.Equal(t, "first", turns[0].UserMessage.Content[0].Text)
				assert.Equal(t, "second", turns[1].UserMessage.Content[0].Text)
			},
		},
		{
			name: "multiple assistant messages per turn",
			messages: []Message{
				{Role: RoleUser, Content: []ContentBlock{{Type: BlockText, Text: "do stuff"}}},
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockToolUse, Name: "Bash"}}},
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockText, Text: "done"}}},
			},
			want: 1,
			checks: func(t *testing.T, turns []Turn) {
				require.Len(t, turns[0].AssistantMessages, 2)
			},
		},
		{
			name: "assistant before first user",
			messages: []Message{
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockText, Text: "init"}}},
				{Role: RoleUser, Content: []ContentBlock{{Type: BlockText, Text: "hello"}}},
				{Role: RoleAssistant, Content: []ContentBlock{{Type: BlockText, Text: "reply"}}},
			},
			want: 2,
			checks: func(t *testing.T, turns []Turn) {
				assert.Nil(t, turns[0].UserMessage)
				require.Len(t, turns[0].AssistantMessages, 1)
				assert.NotNil(t, turns[1].UserMessage)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turns := GroupTurns(tt.messages)
			assert.Len(t, turns, tt.want)
			if tt.checks != nil {
				tt.checks(t, turns)
			}
		})
	}
}

func TestSplitContent(t *testing.T) {
	tests := []struct {
		name         string
		turn         Turn
		wantSteps    int
		wantResponse int
	}{
		{
			name: "no assistant messages",
			turn: Turn{
				UserMessage: &Message{Role: RoleUser},
			},
			wantSteps:    0,
			wantResponse: 0,
		},
		{
			name: "text only becomes response",
			turn: Turn{
				AssistantMessages: []Message{
					{Role: RoleAssistant, Content: []ContentBlock{
						{Type: BlockText, Text: "hello"},
						{Type: BlockText, Text: "world"},
					}},
				},
			},
			wantSteps:    0,
			wantResponse: 2,
		},
		{
			name: "tool then text splits correctly",
			turn: Turn{
				AssistantMessages: []Message{
					{Role: RoleAssistant, Content: []ContentBlock{
						{Type: BlockToolUse, Name: "Bash"},
						{Type: BlockToolResult, Content: "ok"},
						{Type: BlockText, Text: "done"},
					}},
				},
			},
			wantSteps:    2, // tool_use + tool_result
			wantResponse: 1, // trailing text
		},
		{
			name: "text between tools is step",
			turn: Turn{
				AssistantMessages: []Message{
					{Role: RoleAssistant, Content: []ContentBlock{
						{Type: BlockText, Text: "let me check"},
						{Type: BlockToolUse, Name: "Read"},
						{Type: BlockToolResult, Content: "file content"},
						{Type: BlockText, Text: "now editing"},
						{Type: BlockToolUse, Name: "Edit"},
						{Type: BlockToolResult, Content: "edited"},
						{Type: BlockText, Text: "all done"},
					}},
				},
			},
			wantSteps:    6, // text + tool + result + text + tool + result
			wantResponse: 1, // trailing "all done"
		},
		{
			name: "thinking block is step",
			turn: Turn{
				AssistantMessages: []Message{
					{Role: RoleAssistant, Content: []ContentBlock{
						{Type: BlockThinking, Text: "reasoning..."},
						{Type: BlockText, Text: "answer"},
					}},
				},
			},
			wantSteps:    1, // thinking
			wantResponse: 1, // text after thinking
		},
		{
			name: "tools only no response",
			turn: Turn{
				AssistantMessages: []Message{
					{Role: RoleAssistant, Content: []ContentBlock{
						{Type: BlockToolUse, Name: "Bash"},
						{Type: BlockToolResult, Content: "ok"},
					}},
				},
			},
			wantSteps:    2,
			wantResponse: 0,
		},
		{
			name: "multiple assistant messages",
			turn: Turn{
				AssistantMessages: []Message{
					{Role: RoleAssistant, Content: []ContentBlock{
						{Type: BlockToolUse, Name: "Bash"},
						{Type: BlockToolResult, Content: "ok"},
					}},
					{Role: RoleAssistant, Content: []ContentBlock{
						{Type: BlockText, Text: "finished"},
					}},
				},
			},
			wantSteps:    2, // tool_use + tool_result
			wantResponse: 1, // trailing text from second message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, response := tt.turn.SplitContent()
			assert.Len(t, steps, tt.wantSteps)
			assert.Len(t, response, tt.wantResponse)
		})
	}
}

func TestStepCount(t *testing.T) {
	turn := Turn{
		AssistantMessages: []Message{
			{Role: RoleAssistant, Content: []ContentBlock{
				{Type: BlockThinking, Text: "thinking"},
				{Type: BlockToolUse, Name: "Bash"},
				{Type: BlockToolResult, Content: "ok"},
				{Type: BlockToolUse, Name: "Read"},
				{Type: BlockToolResult, Content: "content"},
				{Type: BlockText, Text: "done"},
			}},
		},
	}
	assert.Equal(t, 2, turn.StepCount())
}
