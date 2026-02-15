package core

// Turn groups a user prompt with all subsequent assistant messages,
// representing one request-response cycle in the conversation.
type Turn struct {
	UserMessage       *Message  // nil if the turn starts with an assistant message
	AssistantMessages []Message // all assistant messages in this turn
}

// GroupTurns splits a flat message list into turns. A new turn starts at each
// user message that contains human-authored content (text blocks). User
// messages that contain only tool_result blocks are folded into the current
// turn as part of the assistant's work.
func GroupTurns(messages []Message) []Turn {
	var turns []Turn
	var current *Turn

	for i := range messages {
		msg := &messages[i]
		if msg.Role == RoleUser {
			if isToolResultOnly(msg) {
				// Tool-result-only user messages are part of the agentic loop,
				// not a new human turn. Fold into the current turn.
				if current == nil {
					current = &Turn{}
				}
				current.AssistantMessages = append(current.AssistantMessages, *msg)
			} else {
				if current != nil {
					turns = append(turns, *current)
				}
				current = &Turn{UserMessage: msg}
			}
		} else {
			if current == nil {
				current = &Turn{}
			}
			current.AssistantMessages = append(current.AssistantMessages, *msg)
		}
	}
	if current != nil {
		turns = append(turns, *current)
	}
	return turns
}

// isToolResultOnly reports whether a message contains only tool_result blocks.
func isToolResultOnly(msg *Message) bool {
	if len(msg.Content) == 0 {
		return false
	}
	for _, b := range msg.Content {
		if b.Type != BlockToolResult {
			return false
		}
	}
	return true
}

// SplitContent classifies all content blocks from the turn's assistant
// messages into steps (intermediate work) and response (final output).
//
// Steps include tool_use, tool_result, thinking blocks, and text blocks that
// appear before or between tool calls. Response is the trailing run of text
// blocks after the last tool call in the last assistant message.
func (t Turn) SplitContent() (steps []ContentBlock, response []ContentBlock) {
	// Collect all blocks from all assistant messages.
	var allBlocks []ContentBlock
	for _, msg := range t.AssistantMessages {
		allBlocks = append(allBlocks, msg.Content...)
	}

	if len(allBlocks) == 0 {
		return nil, nil
	}

	// Find the index of the last non-text block (tool_use, tool_result, thinking).
	lastNonText := -1
	for i, b := range allBlocks {
		if b.Type != BlockText {
			lastNonText = i
		}
	}

	// If there are no tool/thinking blocks, everything is response.
	if lastNonText == -1 {
		return nil, allBlocks
	}

	// Everything up to and including lastNonText is steps;
	// everything after is response.
	steps = allBlocks[:lastNonText+1]
	response = allBlocks[lastNonText+1:]
	return steps, response
}

// StepCount returns the number of tool_use blocks across all assistant
// messages in this turn.
func (t Turn) StepCount() int {
	n := 0
	for _, msg := range t.AssistantMessages {
		for _, b := range msg.Content {
			if b.Type == BlockToolUse {
				n++
			}
		}
	}
	return n
}
