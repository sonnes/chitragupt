package terminal

import "github.com/charmbracelet/lipgloss"

var (
	colorUser      = lipgloss.AdaptiveColor{Light: "#b58900", Dark: "#b58900"} // yellow
	colorAssistant = lipgloss.AdaptiveColor{Light: "#859900", Dark: "#859900"} // green
	colorTool      = lipgloss.AdaptiveColor{Light: "#586e75", Dark: "#93a1a1"} // gray
	colorDim       = lipgloss.AdaptiveColor{Light: "#93a1a1", Dark: "#586e75"}
	colorCounter   = lipgloss.AdaptiveColor{Light: "#586e75", Dark: "#93a1a1"}
)

var (
	styleUser      = lipgloss.NewStyle().Foreground(colorUser).Bold(true)
	styleAssistant = lipgloss.NewStyle().Foreground(colorAssistant)
	styleTool      = lipgloss.NewStyle().Foreground(colorTool)
	styleConnector = lipgloss.NewStyle().Foreground(colorDim)
	styleCounter   = lipgloss.NewStyle().Foreground(colorCounter).Faint(true)
)
