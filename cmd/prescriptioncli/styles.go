package main

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary = lipgloss.Color("#7D56F4")
	colorSuccess = lipgloss.Color("#04B575")
	colorError   = lipgloss.Color("#FF5F56")
	colorMuted   = lipgloss.Color("#888888")
	colorWhite   = lipgloss.Color("#FFFFFF")

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	StyleSubtle = lipgloss.NewStyle().Foreground(colorMuted)

	StyleSuccess = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	StyleError   = lipgloss.NewStyle().Bold(true).Foreground(colorError)

	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	StyleFieldLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginTop(1)

	StyleSummaryKey   = lipgloss.NewStyle().Bold(true).Width(14)
	StyleSummaryValue = lipgloss.NewStyle().Foreground(colorWhite)
)
