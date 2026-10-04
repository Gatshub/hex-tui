package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"hex-tui/hex"
)

// damierTest construit une petite partie avec quelques pierres, pour que les
// aperçus montrent autre chose qu'un plateau vide.
func damierTest(style int, coords bool) Model {
	m := testModel(120, 40)
	m = m.start(7)
	m.styleIdx = style
	m.showCoords = coords
	m.board.Set(0, 0, hex.NordSud)
	m.board.Set(1, 0, hex.NordSud)
	m.board.Set(2, 0, hex.NordSud)
	m.board.Set(2, 1, hex.NordSud)
	m.board.Set(1, 3, hex.EstOuest)
	m.board.Set(2, 3, hex.EstOuest)
	m.board.Set(3, 2, hex.EstOuest)
	m.cursorR, m.cursorC = 4, 4
	return m
}

// TestApercuTroisDamiers imprime les trois damiers, avec et sans coordonnées.
// C'est l'aperçu que l'on regarde pour juger de l'allure du plateau.
//
//	HEX_PREVIEW=1 go test ./ui/ -run TestApercuTroisDamiers -v -count=1
func TestApercuTroisDamiers(t *testing.T) {
	if os.Getenv("HEX_PREVIEW") == "" {
		t.Skip("définir HEX_PREVIEW=1 pour afficher les aperçus")
	}
	for style, nom := range boardStyles {
		for _, coords := range []bool{false, true} {
			entete := fmt.Sprintf("%s — coordonnées : %v", nom, ouiNon(coords))
			fmt.Println("\n===== " + entete + " =====")
			for _, l := range damierTest(style, coords).boardLines() {
				fmt.Println(l)
			}
		}
	}
}

// TestStylesRespectentLaGeometrie vérifie, pour CHAQUE damier, les deux
// invariants qui font qu'une image ne ment pas sur le jeu :
//
//   - toutes les lignes du bloc ont la même largeur (sinon le centrage dérape) ;
//   - le décalage entre deux lignes de cases voisines vaut exactement le
//     DEMI-PAS, ni un pas entier (mon erreur du premier dessin : les diagonales
//     ne se touchaient plus) ni autre chose.
//
// Le second contrôle compare la position du premier marqueur, mesurée sur le
// rendu réel : c'est l'image qu'on juge, pas une formule.
func TestStylesRespectentLaGeometrie(t *testing.T) {
	for style, nom := range boardStyles {
		for _, coords := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/coords=%v", nom, coords), func(t *testing.T) {
				m := damierTest(style, coords)
				lines := m.boardLines()

				// Lignes attendues : une par case (deux pour le nid d'abeilles),
				// plus les deux bordures du damier n° 1, plus l'en-tête des
				// lettres si les coordonnées sont affichées.
				attendu := 7 * linesPerCell(style)
				if style == styleBordered {
					attendu += 2 // les deux lignes de bords Nord–Sud
				}
				if coords {
					attendu++
				}
				if len(lines) != attendu {
					t.Fatalf("%d lignes, attendu %d", len(lines), attendu)
				}

				want := m.boardWidth()
				for i, l := range lines {
					if got := lipgloss.Width(l); got != want {
						t.Errorf("ligne %d : largeur %d, attendu %d", i, got, want)
					}
				}

				// Décalage des premières cases : on repère la colonne du
				// premier caractère non blanc, ligne de cases par ligne de
				// cases, et on vérifie l'écart.
				per := linesPerCell(style)
				debut := borderLines(style)
				if coords {
					debut++ // l'en-tête des lettres
				}
				prev := -1
				for r := 0; r < 7; r++ {
					// On mesure sur la ligne BRUTE, sans rien retirer : le
					// premier caractère visible est celui de la case (ou le
					// numéro de ligne, qui suit le même décalage). L'écart
					// entre deux lignes doit valoir le demi-pas.
					l := lines[debut+r*per]
					col := strings.IndexFunc(l, func(c rune) bool { return c != ' ' })
					if col < 0 {
						t.Errorf("ligne de cases %d : aucun caractère visible", r)
						continue
					}
					if prev >= 0 && col-prev != rowOff {
						t.Errorf("ligne de cases %d : décalage %d, attendu %d (un demi-pas)",
							r, col-prev, rowOff)
					}
					prev = col
				}
			})
		}
	}
}

// TestBorduresNordSud verrouille la longueur et la position des deux lignes
// pleines du damier à bordures.
//
// Ce sont les bords NORD et SUD (ceux que relie le camp NordSud, r=0 et r=n-1),
// donc les deux bords HORIZONTAUX. Le défaut corrigé : elles étaient tracées à
// la largeur du BLOC entier et dépassaient de n-1 colonnes — celle du haut à
// droite, celle du bas à gauche — puisque chaque ligne de cases s'incline.
//
// Le contrôle est fait sur le RENDU réel : on compare la ligne de bordure à la
// ligne de cases qu'elle coiffe, caractère par caractère, plutôt qu'à une
// formule.
func TestBorduresNordSud(t *testing.T) {
	m := damierTest(styleBordered, false)
	lines := m.boardLines()
	n := m.board.N

	// Étendue visible d'une ligne, en COLONNES D'AFFICHAGE : les glyphes ■ ⬢ ·
	// font plusieurs octets, donc len() mentirait. On passe par lipgloss.Width,
	// comme le reste du fichier.
	etendue := func(l string) (debut, fin int) {
		l = strings.TrimRight(l, " ")
		debut = len(l) - len(strings.TrimLeft(l, " ")) // l'indentation est faite d'espaces
		return debut, lipgloss.Width(l)
	}

	cas := []struct {
		nom    string
		bord   int
		lignes int // index de la ligne de cases coiffée
	}{
		{"Nord", 0, 0},
		{"Sud", len(lines) - 1, n - 1},
	}
	for _, c := range cas {
		l := lines[c.bord]
		got := strings.Count(l, "■")
		bordDebut, bordFin := etendue(l)
		rowDebut, rowFin := etendue(lines[borderLines(styleBordered)+c.lignes])

		if bordFin-bordDebut != rowFin-rowDebut {
			t.Errorf("bord %s : %d colonnes, la ligne %d en fait %d — la bordure dépasse le losange",
				c.nom, bordFin-bordDebut, c.lignes+1, rowFin-rowDebut)
		}
		if got != rowFin-rowDebut {
			t.Errorf("bord %s : %d cases de bordure, la ligne %d en fait %d",
				c.nom, got, c.lignes+1, rowFin-rowDebut)
		}
		if bordDebut != rowDebut {
			t.Errorf("bord %s : commence en colonne %d, la ligne %d commence en %d",
				c.nom, bordDebut, c.lignes+1, rowDebut)
		}
	}
}

// TestCoordonnees verifie que les coordonnées apparaissent ou disparaissent
// vraiment, et qu'elles nomment les cases attendues.
func TestCoordonnees(t *testing.T) {
	sans := strings.Join(damierTest(styleRhombus, false).boardLines(), "\n")
	avec := strings.Join(damierTest(styleRhombus, true).boardLines(), "\n")

	if strings.Contains(sans, "A") {
		t.Error("les coordonnées apparaissent alors qu'elles sont désactivées")
	}
	for _, lettre := range []string{"A", "B", "G"} {
		if !strings.Contains(avec, lettre) {
			t.Errorf("colonne %q absente des coordonnées", lettre)
		}
	}
	if !strings.Contains(avec, "7") {
		t.Error("la ligne 7 est absente des coordonnées")
	}
}
