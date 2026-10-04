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
				attendu := 7*linesPerCell(style) + borderLines(style)
				if style == styleBordered {
					attendu++ // la bordure Sud, après les cases
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
				// Un demi-pas par ligne de cases. Les deux valeurs sont
				// écrites ICI, volontairement : c'est la géométrie attendue,
				// et non celle que le code calcule — sinon le test ne
				// vérifierait rien. 1 colonne pour le losange (case d'une
				// colonne), 2 pour le nid d'abeilles (alvéole de quatre).
				pas := 1
				if style == styleHoney {
					pas = 2
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
					if prev >= 0 && col-prev != pas {
						t.Errorf("ligne de cases %d : décalage %d, attendu %d (un demi-pas)",
							r, col-prev, pas)
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
// donc les deux bords HORIZONTAUX. Deux défauts ont été corrigés ici, et le
// second expliquait le premier :
//
//   - elles étaient tracées à la largeur du BLOC entier (+ n-1 colonnes) ;
//   - puis à la largeur pleine d'une ligne de cases — encore un cran de trop.
//     Un losange n'a pas de bord horizontal : ses quatre côtés sont penchés,
//     donc aucun ne peut couvrir toute la largeur d'une ligne. Chaque bord fait
//     un cran de MOINS que sa ligne et se rogne du côté extérieur — le Nord
//     s'arrête un cran avant la fin, le Sud commence un cran après le début.
//
// C'est le tracé du damier compact à bordures de HexWiki, et le contrôle est
// fait sur le RENDU réel : on compare chaque bord à la ligne de cases qu'il
// coiffe, colonne par colonne.
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
		nom     string
		bord    int
		lignes  int // index de la ligne de cases coiffée
		rognage int // de combien le bord commence APRÈS le début de sa ligne
	}{
		{"Nord", 0, 0, 0},
		{"Sud", len(lines) - 1, n - 1, 1},
	}
	for _, c := range cas {
		l := lines[c.bord]
		got := strings.Count(l, "■")
		bordDebut, bordFin := etendue(l)
		rowDebut, rowFin := etendue(lines[borderLines(styleBordered)+c.lignes])
		rowLargeur := rowFin - rowDebut

		if got != rowLargeur-1 || bordFin-bordDebut != rowLargeur-1 {
			t.Errorf("bord %s : %d colonnes de bordure (%d ■), la ligne %d en fait %d — attendu un cran de moins",
				c.nom, bordFin-bordDebut, got, c.lignes+1, rowLargeur)
		}
		if want := rowDebut + c.rognage; bordDebut != want {
			t.Errorf("bord %s : commence en colonne %d, attendu %d (sa ligne commence en %d)",
				c.nom, bordDebut, want, rowDebut)
		}
		if bordDebut < rowDebut || bordFin > rowFin {
			t.Errorf("bord %s : %d..%d sort de sa ligne %d..%d — il dépasserait le losange",
				c.nom, bordDebut, bordFin, rowDebut, rowFin)
		}
	}
}

// TestEnteteAligneSurLesCases vérifie que les lettres de colonnes tombent sur
// le CONTENU de leurs cases, dans les trois damiers.
//
// Le défaut trouvé : l'en-tête était calculé avec la seule géométrie du losange
// (une colonne par case), quel que soit le damier. Sur le nid d'abeilles, dont
// l'alvéole fait quatre colonnes, les lettres se tassaient donc deux fois trop
// serré ; sur le damier à bordures, elles étaient décalées de deux colonnes
// vers la gauche, à cause des « \ » qui précèdent les cases.
//
// Le contrôle part du RENDU : on repère les cases sur la première ligne de
// cases, et on exige que chaque lettre soit à l'aplomb de la sienne.
func TestEnteteAligneSurLesCases(t *testing.T) {
	for style, nom := range boardStyles {
		t.Run(nom, func(t *testing.T) {
			m := damierTest(style, true)
			lines := m.boardLines()
			entete := []rune(lines[0])
			premiere := []rune(lines[1+borderLines(style)])

			// Colonnes du CONTENU des cases, relevées sur le dessin.
			var cases []int
			if style == styleHoney {
				// L'intérieur d'une alvéole commence juste après son « / ».
				for i, ch := range premiere {
					if ch == '/' {
						cases = append(cases, i+1)
					}
				}
			} else {
				// Une case de losange porte un marqueur : · ou une pierre.
				for i, ch := range premiere {
					if ch == '·' || ch == '⬢' || ch == '⬡' {
						cases = append(cases, i)
					}
				}
			}
			if len(cases) != m.board.N {
				t.Fatalf("%d cases repérées sur la première ligne, attendu %d",
					len(cases), m.board.N)
			}

			colonne := func(lettre rune) int {
				for i, ch := range entete {
					if ch == lettre {
						return i
					}
				}
				return -1
			}
			for q, want := range cases {
				lettre := rune('A' + q)
				got := colonne(lettre)
				if got < 0 {
					t.Fatalf("lettre %q absente de l'en-tête", string(lettre))
				}
				if got != want {
					t.Errorf("lettre %q en colonne %d, mais sa case est en %d",
						string(lettre), got, want)
				}
			}
		})
	}
}

// TestNidDabeillesFermeLeHaut vérifie que la première rangée du nid d'abeilles
// est bien fermée par sa ligne « __ » — celle que la référence HexWiki dessine.
//
// Les rangées suivantes sont fermées par le bas de celle du dessus : le « \/ »
// et le « /\ » des alvéoles voisines forment le sommet de celles d'en dessous.
// La PREMIÈRE n'a rien au-dessus d'elle, donc sans cette ligne le plateau
// s'ouvre par le haut.
//
// Le contrôle est fait sur le RENDU : les tirets doivent tomber exactement sur
// les deux colonnes INTÉRIEURES de chaque alvéole, là où passe leur bord
// supérieur — et sur rien d'autre.
func TestNidDabeillesFermeLeHaut(t *testing.T) {
	for _, coords := range []bool{false, true} {
		t.Run(fmt.Sprintf("coords=%v", coords), func(t *testing.T) {
			m := damierTest(styleHoney, coords)
			lines := m.boardLines()
			off := 0
			if coords {
				off = 1 // l'en-tête des lettres
			}
			fermeture := []rune(lines[off])
			premiere := []rune(lines[off+1])

			// Les deux colonnes intérieures de chaque alvéole, relevées sur
			// la première ligne de cases : elles suivent chaque « / ».
			var attendu []int
			for i, ch := range premiere {
				if ch == '/' {
					attendu = append(attendu, i+1, i+2)
				}
			}
			var tirets []int
			for i, ch := range fermeture {
				if ch == '_' {
					tirets = append(tirets, i)
				}
			}
			if len(tirets) != len(attendu) {
				t.Fatalf("%d tirets de fermeture, attendu %d (deux par alvéole)",
					len(tirets), len(attendu))
			}
			for i := range tirets {
				if tirets[i] != attendu[i] {
					t.Errorf("tiret %d en colonne %d, attendu %d", i, tirets[i], attendu[i])
				}
			}
		})
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
