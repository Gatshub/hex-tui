package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hex-tui/hex"
)

func testModel(w, h int) Model {
	m := New()
	m.width, m.height = w, h
	m.ready = true
	return m
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// firstVisible renvoie l'index du premier caractère non espace.
func firstVisible(s string) int {
	for i, r := range s {
		if r != ' ' {
			return i
		}
	}
	return -1
}

func TestMenuRenders(t *testing.T) {
	out := testModel(120, 40).View()
	if !strings.Contains(out, "H E X") {
		t.Fatal("le titre est absent du menu")
	}
	if !strings.Contains(out, "11 × 11") {
		t.Fatal("les tailles proposées sont absentes")
	}
}

// Les lignes impaires doivent être décalées de 2 colonnes, et toutes les
// cases alignées sur une grille de 4 colonnes.
func TestBoardGeometry(t *testing.T) {
	m := testModel(120, 40)
	m = m.start(9)
	lines := m.boardLines()
	if len(lines) != 9 {
		t.Fatalf("%d lignes, attendu 9", len(lines))
	}
	want := m.boardWidth()
	for i, l := range lines {
		// Toutes les lignes sont complétées à la largeur maximale pour un
		// centrage homogène (sinon le plateau apparaîtrait en escalier).
		if got := lipgloss.Width(l); got != want {
			t.Fatalf("ligne %d : largeur %d, attendu %d", i, got, want)
		}
		// Le marqueur occupe une case d'UNE colonne : le premier caractère
		// visible est donc en colonne indent(i), plus le décalage de la ligne.
		wantFirst := indent(i)
		if got := firstVisible(l); got != wantFirst {
			t.Fatalf("ligne %d : premier marqueur en colonne %d, attendu %d",
				i, got, wantFirst)
		}
	}
}

// TestCellAtRoundTrip vérifie que cliquer sur une case DESSINÉE la sélectionne,
// dans les trois damiers.
//
// La colonne cliquée est relevée SUR LE RENDU — là où les cases sont réellement
// dessinées — et non recalculée avec les formules de cellAt : sinon le test ne
// vérifierait que la cohérence du code avec lui-même, et laisserait passer
// précisément le défaut qui nous occupe, une souris restée sur la géométrie du
// losange pendant que le damier dessinait autre chose.
func TestCellAtRoundTrip(t *testing.T) {
	const n = 7
	for style, nom := range boardStyles {
		for _, coords := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/coords=%v", nom, coords), func(t *testing.T) {
				m := testModel(120, 40)
				m = m.start(n)
				m.styleIdx = style
				m.showCoords = coords

				lines := m.buildPlay()
				top := (m.height - len(lines)) / 2
				gridY := top + 2 + borderLines(style)
				// gridX reste la gauche du BLOC : les colonnes relevées sur le
				// rendu se comptent depuis elle, marge des coordonnées
				// comprise. cellAt, lui, part de la gauche des CASES.
				gridX := (m.width - m.boardWidth()) / 2
				if coords {
					gridY++
				}
				per := linesPerCell(style)
				bloc := m.boardLines()
				debut := borderLines(style)
				if coords {
					debut++ // l'en-tête des lettres
				}

				for r := 0; r < n; r++ {
					dessin := []rune(bloc[debut+r*per])
					// Colonnes où une case est réellement dessinée.
					var cases []int
					if style == styleHoney {
						for i, ch := range dessin {
							if ch == '/' {
								cases = append(cases, i+1) // l'intérieur de l'alvéole
							}
						}
					} else {
						for i, ch := range dessin {
							if ch == '·' || ch == '⬢' || ch == '⬡' {
								cases = append(cases, i)
							}
						}
					}
					if len(cases) != n {
						t.Fatalf("ligne %d : %d cases dessinées, attendu %d", r, len(cases), n)
					}
					for q, col := range cases {
						gr, gq, ok := m.cellAt(gridX+col, gridY+r*per)
						if !ok {
							t.Errorf("clic sur la case dessinée (%d,%d) refusé", r, q)
							continue
						}
						if gr != r || gq != q {
							t.Errorf("clic sur la case (%d,%d) → (%d,%d)", r, q, gr, gq)
						}
					}
				}
				if _, _, ok := m.cellAt(0, 0); ok {
					t.Error("un clic hors plateau devrait être refusé")
				}
			})
		}
	}
}

// Poser une pierre, annuler, et détecter la victoire.
func TestPlaceUndoWin(t *testing.T) {
	m := testModel(120, 40)
	m.opponentIdx = 0 // deux joueurs : pas de moteur pour interférer
	m = m.start(7)

	m.cursorR, m.cursorC = 3, 3
	nm, _ := m.Update(runeKey(' '))
	m = nm.(Model)
	if m.board.At(3, 3) != hex.NordSud {
		t.Fatal("la pierre n'a pas été posée")
	}
	if m.turn != hex.EstOuest {
		t.Fatal("le tour n'a pas changé")
	}

	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = nm.(Model)
	if m.board.At(3, 3) != hex.None {
		t.Fatal("l'annulation n'a pas retiré la pierre")
	}
	if m.turn != hex.NordSud {
		t.Fatal("l'annulation n'a pas rendu la main")
	}

	// Position gagnante : Nord–Sud occupe toute la colonne 0 sauf la dernière.
	for r := 0; r < 6; r++ {
		m.board.Set(r, 0, hex.NordSud)
	}
	m.turn = hex.NordSud
	m.cursorR, m.cursorC = 6, 0
	nm, _ = m.Update(runeKey(' '))
	m = nm.(Model)
	if m.winner != hex.NordSud {
		t.Fatalf("gagnant = %v, attendu Nord–Sud", m.winner)
	}
	if m.screen != scrWin {
		t.Fatal("l'écran de victoire ne s'affiche pas")
	}
	if len(m.winPath) == 0 {
		t.Fatal("la chaîne gagnante n'est pas identifiée")
	}
	if !strings.Contains(m.View(), "victoire") {
		t.Fatal("la bannière de victoire est absente")
	}
}

// Le déplacement suit le losange : les six directions sont les MÊMES sur toutes
// les lignes. C'est précisément l'intérêt du losange — sur l'ancien tracé en
// quinconce, il fallait savoir si l'on était sur une ligne paire ou impaire
// pour savoir où menait « nord-ouest ». Le test vérifie donc l'UNIFORMITÉ,
// c'est-à-dire que la parité ne change plus rien.
func TestMovementIsUniform(t *testing.T) {
	attendu := map[string][2]int{
		"W":  {0, -1},
		"E":  {0, 1},
		"NW": {-1, 0},
		"NE": {-1, 1},
		"SW": {1, -1},
		"SE": {1, 0},
	}
	for _, ligne := range []int{4, 5} {
		for dir, d := range attendu {
			m := testModel(120, 40)
			m = m.start(9)
			m.cursorR, m.cursorC = ligne, 4
			m.move(dir)
			if m.cursorR != ligne+d[0] || m.cursorC != 4+d[1] {
				t.Errorf("ligne %d, %s → (%d,%d), attendu (%d,%d)",
					ligne, dir, m.cursorR, m.cursorC, ligne+d[0], 4+d[1])
			}
		}
	}

	// Les bords ne débordent pas.
	m := testModel(120, 40)
	m = m.start(9)
	m.cursorR, m.cursorC = 0, 0
	m.move("NW")
	m.move("W")
	if m.cursorR != 0 || m.cursorC != 0 {
		t.Fatalf("débordement : (%d,%d)", m.cursorR, m.cursorC)
	}
}

// TestTouchesDeDeplacement confronte l'aide au comportement réel : CHAQUE
// touche annoncée doit vraiment déplacer le curseur.
//
// Ce test existe à cause d'un vrai défaut : l'aide promettait « hjkl » alors
// que le code ne gérait que h et l. j et k ne faisaient rien, en silence. Un
// test qui se contente de vérifier ce que le code gère ne voit pas ce genre de
// manque ; celui-ci part de la LISTE DES TOUCHES, pas des cas implémentés.
func TestTouchesDeDeplacement(t *testing.T) {
	cas := []struct {
		nom string
		msg tea.KeyMsg
		dr  int
		dc  int
	}{
		{"h", runeKey('h'), 0, -1},
		{"l", runeKey('l'), 0, 1},
		{"j", runeKey('j'), 1, 0},
		{"k", runeKey('k'), -1, 0},
		{"y", runeKey('y'), -1, 0},
		{"n", runeKey('n'), 1, 0},
		{"u", runeKey('u'), -1, 1},
		{"b", runeKey('b'), 1, -1},
		{"←", tea.KeyMsg{Type: tea.KeyLeft}, 0, -1},
		{"→", tea.KeyMsg{Type: tea.KeyRight}, 0, 1},
		{"↑", tea.KeyMsg{Type: tea.KeyUp}, -1, 0},
		{"↓", tea.KeyMsg{Type: tea.KeyDown}, 1, 0},
		{"Maj+↑", tea.KeyMsg{Type: tea.KeyShiftUp}, -1, 1},
		{"Maj+↓", tea.KeyMsg{Type: tea.KeyShiftDown}, 1, -1},
	}
	for _, c := range cas {
		m := testModel(120, 40)
		m = m.start(9)
		m.cursorR, m.cursorC = 4, 4
		nm, _ := m.Update(c.msg)
		got := nm.(Model)
		if got.cursorR != 4+c.dr || got.cursorC != 4+c.dc {
			t.Errorf("touche %s : curseur en (%d,%d), attendu (%d,%d)",
				c.nom, got.cursorR, got.cursorC, 4+c.dr, 4+c.dc)
		}
	}
}

// Aucune ligne ne doit dépasser la largeur du terminal.
func TestNoOverflow(t *testing.T) {
	for _, w := range []int{50, 60, 80, 100, 120, 160} {
		for _, h := range []int{20, 30, 40, 50} {
			for _, n := range Sizes {
				m := testModel(w, h)
				m = m.start(n)
				if m.boardWidth()+2 > w || m.board.N+8 > h {
					continue // la fenêtre est trop petite : message dédié
				}
				for i, l := range strings.Split(m.View(), "\n") {
					if got := lipgloss.Width(l); got > w {
						t.Fatalf("w=%d n=%d : ligne %d fait %d colonnes", w, n, i, got)
					}
				}
			}
		}
	}
}

// Aperçu textuel, pour vérifier la mise en page sans terminal interactif.
//
//	HEX_PREVIEW=1 go test ./ui/ -run TestPreview -v
func TestPreview(t *testing.T) {
	if os.Getenv("HEX_PREVIEW") == "" {
		t.Skip("définir HEX_PREVIEW=1 pour afficher l'aperçu")
	}

	fmt.Printf("\n===== menu =====\n%s\n", testModel(100, 30).View())

	m := testModel(100, 30)
	m = m.start(9)
	// Quelques pierres pour juger le rendu.
	m.board.Set(0, 0, hex.NordSud)
	m.board.Set(1, 0, hex.NordSud)
	m.board.Set(2, 0, hex.NordSud)
	m.board.Set(3, 1, hex.EstOuest)
	m.board.Set(4, 1, hex.EstOuest)
	m.board.Set(4, 2, hex.EstOuest)
	m.cursorR, m.cursorC = 5, 2
	fmt.Printf("\n===== partie 9x9 =====\n%s\n", m.View())

	// Écran de victoire : Nord–Sud relie la colonne 0 de haut en bas.
	w := testModel(100, 30)
	w = w.start(7)
	for r := 0; r < 7; r++ {
		w.board.Set(r, 0, hex.NordSud)
	}
	w.winner = hex.NordSud
	w.winPath = w.board.WinningCells(hex.NordSud)
	w.screen = scrWin
	fmt.Printf("\n===== victoire =====\n%s\n", w.View())
}

// finDePartie construit l'état d'une partie terminée, victoire Nord–Sud (la
// colonne 0 relie le haut au bas). C'est l'écran sur lequel « s » ne faisait
// rien.
func finDePartie() Model {
	m := testModel(100, 30)
	m = m.start(7)
	for r := 0; r < 7; r++ {
		m.board.Set(r, 0, hex.NordSud)
	}
	m.winner = hex.NordSud
	m.winPath = m.board.WinningCells(hex.NordSud)
	m.screen = scrWin
	return m
}

// TestTouchesAnnonceesALaFin part de la LISTE des touches que l'écran de fin
// annonce (annoncesFin) et exige que CHACUNE fasse quelque chose.
//
// Le test ne recopie pas la liste, il la lit : une touche annoncée mais inerte
// ne peut donc plus passer. C'est exactement le défaut trouvé avec « s »,
// affiché à l'écran alors que winKey ne le gérait pas.
//
// Une touche annoncée sans attente déclarée fait également échouer le test : on
// ne peut pas ajouter une annonce sans dire ce qu'elle doit produire.
func TestTouchesAnnonceesALaFin(t *testing.T) {
	attendu := map[string]struct {
		msg  tea.KeyMsg
		quoi string
		fait func(apres Model, cmd tea.Cmd) bool
	}{
		"enter": {
			tea.KeyMsg{Type: tea.KeyEnter}, "commencer une nouvelle partie",
			func(m Model, _ tea.Cmd) bool { return m.screen == scrPlay },
		},
		"s": {
			runeKey('s'), "ouvrir les réglages",
			func(m Model, _ tea.Cmd) bool { return m.showSettings },
		},
		"esc": {
			tea.KeyMsg{Type: tea.KeyEsc}, "revenir au menu",
			func(m Model, _ tea.Cmd) bool { return m.screen == scrMenu },
		},
		"q": {
			runeKey('q'), "quitter",
			func(_ Model, cmd tea.Cmd) bool { return cmd != nil },
		},
	}

	for _, a := range annoncesFin {
		c, declare := attendu[a.key]
		if !declare {
			t.Errorf("la touche %q est annoncée « %s », mais le test ne dit pas ce qu'elle doit faire",
				a.key, a.label)
			continue
		}
		apres, cmd := finDePartie().Update(c.msg)
		if !c.fait(apres.(Model), cmd) {
			t.Errorf("touche %q (annoncée « %s ») : elle devrait %s — mais rien ne se passe",
				a.key, a.label, c.quoi)
		}
	}
}
