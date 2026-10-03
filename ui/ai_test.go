package ui

import (
	"strings"
	"testing"

	"hex-tui/ai"
	"hex-tui/hex"
)

// Le moteur doit répondre au coup de l'humain, sans figer l'interface :
// le calcul passe par une commande Bubble Tea, exécutée ici à la main.
func TestAIResponds(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 1 // ordinateur
	m = m.start(7)
	if !m.vsAI {
		t.Fatal("le mode ordinateur n'est pas actif")
	}

	m.cursorR, m.cursorC = 3, 3
	nm, cmd := m.Update(runeKey(' '))
	m = nm.(Model)
	if cmd == nil {
		t.Fatal("aucune commande de moteur renvoyée")
	}
	if !m.thinking {
		t.Fatal("l'état « réflexion » n'est pas activé")
	}
	if m.board.Moves != 1 {
		t.Fatalf("%d pierres, attendu 1 après le coup humain", m.board.Moves)
	}

	// Le curseur ne doit plus rien pouvoir jouer pendant la réflexion.
	nm, _ = m.Update(runeKey(' '))
	m = nm.(Model)
	if m.board.Moves != 1 {
		t.Fatal("un coup a été accepté pendant la réflexion")
	}

	// Exécution du calcul, puis application du coup.
	nm, _ = m.Update(cmd())
	m = nm.(Model)
	if m.thinking {
		t.Fatal("l'état « réflexion » n'a pas été levé")
	}
	if m.board.Moves != 2 {
		t.Fatalf("%d pierres, attendu 2 après la réponse du moteur", m.board.Moves)
	}
	if m.turn != hex.NordSud {
		t.Fatal("la main n'est pas revenue à l'humain")
	}
	// L'ordinateur joue Est–Ouest : sa pierre doit être de cette couleur.
	if m.board.At(m.board.N-1, m.board.N-1) == hex.EstOuest &&
		m.board.At(3, 3) != hex.NordSud {
		t.Fatal("les pierres ne sont pas attribuées aux bons camps")
	}
}

// À deux joueurs, aucune commande de moteur ne doit être déclenchée.
func TestNoAIInTwoPlayerMode(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 0
	m = m.start(7)
	m.cursorR, m.cursorC = 3, 3
	nm, cmd := m.Update(runeKey(' '))
	m = nm.(Model)
	if cmd != nil {
		t.Fatal("un moteur a été déclenché en mode deux joueurs")
	}
	if m.thinking {
		t.Fatal("état « réflexion » actif en mode deux joueurs")
	}
}

// Le menu doit se parcourir et changer les valeurs.
func TestMenuNavigation(t *testing.T) {
	m := testModel(120, 40)
	if !strings.Contains(m.View(), "11 × 11") {
		t.Fatal("la taille par défaut devrait être 11 × 11")
	}

	// Ligne 0 = taille : ← → changent la valeur.
	m.menuRow = 0
	nm, _ := m.Update(runeKey('l'))
	m = nm.(Model)
	if Sizes[m.sizeIdx] != 13 {
		t.Fatalf("taille %d après un appui à droite, attendu 13", Sizes[m.sizeIdx])
	}
	nm, _ = m.Update(runeKey('h'))
	m = nm.(Model)
	if Sizes[m.sizeIdx] != 11 {
		t.Fatalf("taille %d après un appui à gauche, attendu 11", Sizes[m.sizeIdx])
	}

	// Ligne 1 = adversaire.
	m.menuRow = 1
	nm, _ = m.Update(runeKey('l'))
	m = nm.(Model)
	if opponents[m.opponentIdx] != "2 joueurs" {
		t.Fatalf("adversaire %q, attendu « 2 joueurs »", opponents[m.opponentIdx])
	}

	// Ligne 2 = niveau, seulement pertinent contre l'ordinateur.
	m.menuRow = 2
	m.opponentIdx = 1
	m.levelIdx = 0
	nm, _ = m.Update(runeKey('l'))
	m = nm.(Model)
	if ai.Levels[m.levelIdx].Name != "Moyen" {
		t.Fatalf("niveau %q, attendu Moyen", ai.Levels[m.levelIdx].Name)
	}

	// Le menu ne doit pas déborder.
	for _, w := range []int{60, 80, 120} {
		mm := testModel(w, 30)
		for i, l := range strings.Split(mm.View(), "\n") {
			if len([]rune(l)) > w {
				t.Fatalf("menu w=%d : ligne %d trop large", w, i)
			}
		}
	}
}
