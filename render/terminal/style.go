package terminal

import "github.com/charmbracelet/lipgloss"

var (
	// Role colors — blue for user, emerald for assistant, slate for system.
	colorUser      = lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#60a5fa"}
	colorAssistant = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34d399"}
	colorSystem    = lipgloss.AdaptiveColor{Light: "#64748b", Dark: "#94a3b8"}

	// UI colors.
	colorBright = lipgloss.AdaptiveColor{Light: "#0f172a", Dark: "#f1f5f9"}
	colorDim    = lipgloss.AdaptiveColor{Light: "#94a3b8", Dark: "#64748b"}
	colorTool   = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#a78bfa"} // purple
)

var (
	styleUserBadge      = lipgloss.NewStyle().Foreground(colorUser).Bold(true)
	styleAssistantBadge = lipgloss.NewStyle().Foreground(colorAssistant).Bold(true)
	styleSystemBadge    = lipgloss.NewStyle().Foreground(colorSystem).Bold(true)

	styleTitle    = lipgloss.NewStyle().Foreground(colorBright).Bold(true)
	styleMeta     = lipgloss.NewStyle().Foreground(colorDim)
	styleDuration = lipgloss.NewStyle().Foreground(colorAssistant)

	styleStat      = lipgloss.NewStyle().Foreground(colorBright).Bold(true)
	styleStatLabel = lipgloss.NewStyle().Foreground(colorDim)

	styleToolName   = lipgloss.NewStyle().Foreground(colorTool).Bold(true)
	styleToolDetail = lipgloss.NewStyle().Foreground(colorDim)
	styleThinking   = lipgloss.NewStyle().Foreground(colorDim).Italic(true)

	styleSeparator = lipgloss.NewStyle().Foreground(colorDim)
)
