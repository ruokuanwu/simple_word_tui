package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("12")  // 蓝
	colorAccent  = lipgloss.Color("212") // 粉
	colorMuted   = lipgloss.Color("240") // 灰
	colorOK      = lipgloss.Color("42")  // 绿

	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	termStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	phoneticStyle = lipgloss.NewStyle().
			Foreground(colorPrimary)

	defStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	defBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{
			Top:         "╌",
			Bottom:      "╌",
			Left:        "┆",
			Right:       "┆",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
		}).
		BorderForeground(colorMuted).
		Padding(0, 1)

	okStyle = lipgloss.NewStyle().
		Foreground(colorOK).
		Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	congratsBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorOK).
			Padding(1, 3).
			Align(lipgloss.Center)
)
