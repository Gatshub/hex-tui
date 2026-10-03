// Commande hex-tui : le jeu de Hex pour le terminal, en plein écran.
// Bâti avec Bubble Tea et Lip Gloss (projet Charm).
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"hex-tui/ui"
)

func main() {
	p := tea.NewProgram(
		ui.New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "hex-tui :", err)
		os.Exit(1)
	}
}
