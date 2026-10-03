package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hex-tui/ai"
	"hex-tui/hex"
)

func levelName(m Model) string { return ai.Levels[m.levelIdx].Name }

// Le point du panneau de réglages : changer le niveau SANS perdre la partie.
func TestLevelChangeKeepsGame(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 1
	m.levelIdx = 1
	m = m.start(9)

	// Un coup humain, puis la réponse du moteur.
	m.cursorR, m.cursorC = 4, 4
	nm, cmd := m.Update(runeKey(' '))
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("le moteur n'a pas été déclenché")
	}
	nm, _ = m.Update(cmd())
	m = nm.(Model)

	wantMoves, wantSize := m.board.Moves, m.board.N
	stone := m.board.At(4, 4)

	// Ouverture du panneau et changement de niveau.
	nm, _ = m.Update(runeKey('s'))
	m = nm.(Model)
	if !m.showSettings {
		t.Fatal("le panneau de réglages ne s'est pas ouvert")
	}
	if m.settingsRow != 2 {
		t.Fatalf("ligne sélectionnée %d, attendu 2 (niveau)", m.settingsRow)
	}
	if levelName(m) != "Moyen" {
		t.Fatalf("niveau de départ %q, attendu Moyen", levelName(m))
	}
	nm, _ = m.Update(runeKey('l'))
	m = nm.(Model)
	if levelName(m) != "Difficile" {
		t.Fatalf("niveau %q après un appui, attendu Difficile", levelName(m))
	}

	// Fermeture : la partie doit être intacte, au coup près.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.showSettings {
		t.Fatal("le panneau ne s'est pas refermé")
	}
	if m.screen != scrPlay {
		t.Fatal("on n'est plus en train de jouer")
	}
	if m.board.N != wantSize {
		t.Fatalf("taille %d, attendu %d", m.board.N, wantSize)
	}
	if m.board.Moves != wantMoves {
		t.Fatalf("%d coups, attendu %d — la partie a été perdue", m.board.Moves, wantMoves)
	}
	if m.board.At(4, 4) != stone {
		t.Fatal("une pierre a disparu")
	}
	if levelName(m) != "Difficile" {
		t.Fatal("le niveau n'a pas été conservé après fermeture")
	}
}

// Échap ne doit plus jeter la partie : il ouvre les réglages.
func TestEscOpensSettingsNotMenu(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 0
	m = m.start(7)
	m.cursorR, m.cursorC = 2, 2
	nm, _ := m.Update(runeKey(' '))
	m = nm.(Model)

	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if !m.showSettings {
		t.Fatal("Échap n'a pas ouvert les réglages")
	}
	if m.screen != scrPlay {
		t.Fatal("Échap a quitté la partie")
	}
	if m.board.Moves != 1 {
		t.Fatal("Échap a effacé la partie")
	}

	// Échap de nouveau : on reprend la partie.
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.showSettings {
		t.Fatal("le panneau ne s'est pas refermé")
	}
}

// Passer à l'ordinateur en cours de partie doit faire jouer le moteur, même
// si c'est déjà à lui que revient la main.
func TestSwitchToAIResumesPlay(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 0 // deux joueurs
	m = m.start(7)

	m.cursorR, m.cursorC = 3, 3
	nm, cmd := m.Update(runeKey(' '))
	m = nm.(Model)
	if cmd != nil {
		t.Fatal("moteur déclenché à tort en mode deux joueurs")
	}
	if m.turn != hex.EstOuest {
		t.Fatal("la main devrait être à Est–Ouest")
	}

	nm, _ = m.Update(runeKey('s'))
	m = nm.(Model)
	m.settingsRow = 1
	nm, _ = m.Update(runeKey('l'))
	m = nm.(Model)
	if !m.vsAI {
		t.Fatal("l'adversaire n'est pas passé à l'ordinateur")
	}

	// En refermant, le moteur doit se mettre au travail immédiatement.
	nm, cmd = m.Update(runeKey('s'))
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("le moteur n'a pas démarré après le passage à l'ordinateur")
	}
	if !m.thinking {
		t.Fatal("l'état de réflexion est absent")
	}
	nm, _ = m.Update(cmd())
	m = nm.(Model)
	if m.board.Moves != 2 {
		t.Fatalf("%d pierres, attendu 2 après la réponse du moteur", m.board.Moves)
	}
}

// « Nouvelle partie » repart de zéro avec la taille choisie.
func TestNewGameFromSettings(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 0
	m = m.start(7)
	m.cursorR, m.cursorC = 1, 1
	nm, _ := m.Update(runeKey(' '))
	m = nm.(Model)

	nm, _ = m.Update(runeKey('s'))
	m = nm.(Model)
	m.sizeIdx = 0 // 7 × 7
	m.settingsRow = 0
	nm, _ = m.Update(runeKey('l')) // taille suivante
	m = nm.(Model)
	if m.cfgSize() != "9 × 9" {
		t.Fatalf("taille choisie %q, attendu 9 × 9", m.cfgSize())
	}

	m.settingsRow = rowNewGame
	nm, _ = m.Update(runeKey(' ')) // Nouvelle partie
	m = nm.(Model)
	if m.board.N != 9 {
		t.Fatalf("plateau %d × %d, attendu 9 × 9", m.board.N, m.board.N)
	}
	if m.board.Moves != 0 {
		t.Fatal("le plateau n'a pas été réinitialisé")
	}
	if m.showSettings {
		t.Fatal("le panneau devrait s'être refermé")
	}
	if m.screen != scrPlay {
		t.Fatal("on ne joue pas")
	}
}

// « Menu principal » reste le seul moyen de sortir, et il est explicite.
func TestMenuPrincipalFromSettings(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 0
	m = m.start(7)
	nm, _ := m.Update(runeKey('s'))
	m = nm.(Model)
	m.settingsRow = rowMainMenu
	nm, _ = m.Update(runeKey(' '))
	m = nm.(Model)
	if m.screen != scrMenu {
		t.Fatal("on n'est pas revenu au menu")
	}
	if m.showSettings {
		t.Fatal("le panneau est resté ouvert")
	}
}

// Le panneau doit tenir à l'écran et annoncer que la partie est conservée.
func TestSettingsRenders(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 1
	m = m.start(11)
	nm, _ := m.Update(runeKey('s'))
	m = nm.(Model)

	out := m.View()
	for _, want := range []string{"Réglages", "conservée", "Niveau", "Nouvelle partie"} {
		if !strings.Contains(out, want) {
			t.Fatalf("« %s » absent du panneau", want)
		}
	}
	for i, l := range strings.Split(out, "\n") {
		if len([]rune(l)) > m.width {
			t.Fatalf("ligne %d trop large en 120 colonnes", i)
		}
	}
}
