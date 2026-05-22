package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeSessionStats(t *testing.T) {
	tr := &Transcript{
		SubAgents: []*Transcript{
			{SessionID: "agent-1"},
			{SessionID: "agent-2"},
		},
		Messages: []Message{
			{
				Role:  RoleAssistant,
				Model: "claude-opus-4-6",
				Content: []ContentBlock{
					{
						Type: BlockToolUse,
						Name: "Read",
						Input: map[string]any{
							"file_path": "/work/login.go",
						},
					},
					{
						Type: BlockToolUse,
						Name: "Edit",
						Input: map[string]any{
							"file_path": "/work/login.go",
						},
					},
					{
						Type: BlockToolUse,
						Name: "Skill",
						Input: map[string]any{
							"skill": "gstack:investigate",
						},
					},
					{
						Type: BlockToolUse,
						Name: "Bash",
						Input: map[string]any{
							"command": "make test ./...",
						},
					},
				},
			},
			{
				Role:  RoleAssistant,
				Model: "claude-sonnet-4-6",
				Content: []ContentBlock{
					{
						Type: BlockToolUse,
						Name: "Grep",
						Input: map[string]any{
							"pattern": "func Login",
						},
					},
				},
			},
		},
	}

	stats := ComputeSessionStats(tr)
	require.NotNil(t, stats)

	assert.Equal(t, 5, stats.ToolUsage["Read"]+stats.ToolUsage["Edit"]+stats.ToolUsage["Skill"]+stats.ToolUsage["Bash"]+stats.ToolUsage["Grep"])
	assert.Equal(t, 1, stats.ToolUsage["Read"])
	assert.Equal(t, 1, stats.FileOps["read"])
	assert.Equal(t, 1, stats.FileOps["edit"])
	assert.Equal(t, 1, stats.FileOps["search"])
	assert.Equal(t, 1, stats.SkillsUsed["gstack:investigate"])
	assert.Equal(t, 1, stats.CommandsUsed["make"])
	assert.Equal(t, 2, stats.ModelUsage["claude-opus-4-6"]+stats.ModelUsage["claude-sonnet-4-6"])
	assert.Equal(t, 2, stats.WorkMode.Explore)
	assert.Equal(t, 1, stats.WorkMode.Build)
	assert.Equal(t, 1, stats.WorkMode.Test)
	assert.Equal(t, 2, stats.SubAgentCount)
}

func TestComputeSessionStatsEmpty(t *testing.T) {
	assert.Nil(t, ComputeSessionStats(&Transcript{}))
}
