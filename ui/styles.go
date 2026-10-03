package ui

import (
	"github.com/charmbracelet/lipgloss"

	"hex-tui/hex"
)

// Palette adaptative (clair / sombre), d'inspiration « Tokyo Night ».
var (
	colBG       = lipgloss.AdaptiveColor{Light: "#e9e9ef", Dark: "#1a1b26"}
	colFG       = lipgloss.AdaptiveColor{Light: "#343b58", Dark: "#c0caf5"}
	colMuted    = lipgloss.AdaptiveColor{Light: "#8a8fa3", Dark: "#565f89"}
	colNordSud  = lipgloss.AdaptiveColor{Light: "#0b5b86", Dark: "#7dcfff"}
	colEstOuest = lipgloss.AdaptiveColor{Light: "#8a5a12", Dark: "#e0af68"}
	colAccent   = lipgloss.AdaptiveColor{Light: "#6a4fa3", Dark: "#bb9af7"}
	colGreen    = lipgloss.AdaptiveColor{Light: "#2f6b3f", Dark: "#9ece6a"}
	colWarn     = lipgloss.AdaptiveColor{Light: "#8a5a12", Dark: "#e0af68"}
	colCursorBG = lipgloss.AdaptiveColor{Light: "#b9c6f0", Dark: "#3d59a1"}
	colCursorFG = lipgloss.AdaptiveColor{Light: "#1a1b26", Dark: "#ffffff"}
)

// Styles.
var (
	styleTitle    = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	styleSubtitle = lipgloss.NewStyle().Foreground(colMuted)
	styleMuted    = lipgloss.NewStyle().Foreground(colMuted)
	styleFG       = lipgloss.NewStyle().Foreground(colFG)
	styleGood     = lipgloss.NewStyle().Foreground(colGreen).Bold(true)
	styleWarn     = lipgloss.NewStyle().Foreground(colWarn).Bold(true)
	styleNordSud  = lipgloss.NewStyle().Foreground(colNordSud)
	styleEstOuest = lipgloss.NewStyle().Foreground(colEstOuest)
	styleKey      = lipgloss.NewStyle().Foreground(colAccent)
	styleHelpKey  = lipgloss.NewStyle().Foreground(colNordSud).Bold(true)
	styleSelItem  = lipgloss.NewStyle().Foreground(colBG).Background(colNordSud).Bold(true).Padding(0, 2)
	styleMenuItem = lipgloss.NewStyle().Foreground(colFG).Padding(0, 2)
	styleBox      = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colAccent).
			Padding(1, 3)
)

// playerStyle renvoie le style de couleur d'un camp.
func playerStyle(p hex.Player) lipgloss.Style {
	if p == hex.NordSud {
		return styleNordSud
	}
	return styleEstOuest
}
