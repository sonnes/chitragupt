package core

import "strings"

var fileOpByTool = map[string]string{
	"Read":           "read",
	"LS":             "read",
	"Write":          "write",
	"Edit":           "edit",
	"StrReplace":     "edit",
	"Delete":         "delete",
	"Glob":           "search",
	"Grep":           "search",
	"SemanticSearch": "search",
}

// ComputeSessionStats derives aggregate metadata from a transcript.
func ComputeSessionStats(t *Transcript) *SessionStats {
	if t == nil {
		return nil
	}

	stats := &SessionStats{
		ModelUsage:    make(map[string]int),
		ToolUsage:     make(map[string]int),
		FileOps:       make(map[string]int),
		SkillsUsed:    make(map[string]int),
		CommandsUsed:  make(map[string]int),
		WorkMode:      &WorkModeStats{},
		SubAgentCount: len(t.SubAgents),
	}

	for _, msg := range t.Messages {
		if msg.Model != "" {
			stats.ModelUsage[msg.Model]++
		}

		for _, block := range msg.Content {
			if block.Type != BlockToolUse {
				continue
			}

			stats.ToolUsage[block.Name]++
			recordWorkMode(stats.WorkMode, block.Name)

			if op, ok := fileOpByTool[block.Name]; ok {
				stats.FileOps[op]++
			}

			if block.Name == "Skill" {
				if skill := inputString(block.Input, "skill"); skill != "" {
					stats.SkillsUsed[skill]++
				}
			}

			if block.Name == "Bash" {
				if command := commandName(inputString(block.Input, "command")); command != "" {
					stats.CommandsUsed[command]++
				}
			}
		}
	}

	pruneEmptyStats(stats)
	if isEmptyStats(stats) {
		return nil
	}
	return stats
}

func recordWorkMode(work *WorkModeStats, toolName string) {
	switch toolName {
	case "Read", "Grep", "Glob", "WebFetch", "WebSearch", "LS", "SemanticSearch":
		work.Explore++
	case "Write", "Edit", "StrReplace":
		work.Build++
	case "Bash", "Agent", "Task", "TaskCreate", "TaskUpdate":
		work.Test++
	}
}

func inputString(input any, key string) string {
	m, ok := input.(map[string]any)
	if !ok {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func commandName(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func pruneEmptyStats(stats *SessionStats) {
	if len(stats.ModelUsage) == 0 {
		stats.ModelUsage = nil
	}
	if len(stats.ToolUsage) == 0 {
		stats.ToolUsage = nil
	}
	if len(stats.FileOps) == 0 {
		stats.FileOps = nil
	}
	if len(stats.SkillsUsed) == 0 {
		stats.SkillsUsed = nil
	}
	if len(stats.CommandsUsed) == 0 {
		stats.CommandsUsed = nil
	}
	if stats.WorkMode != nil && *stats.WorkMode == (WorkModeStats{}) {
		stats.WorkMode = nil
	}
}

func isEmptyStats(stats *SessionStats) bool {
	return stats.SubAgentCount == 0 &&
		len(stats.ModelUsage) == 0 &&
		len(stats.ToolUsage) == 0 &&
		len(stats.FileOps) == 0 &&
		len(stats.SkillsUsed) == 0 &&
		len(stats.CommandsUsed) == 0 &&
		stats.WorkMode == nil
}
