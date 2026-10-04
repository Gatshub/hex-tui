package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hex-tui/hex"
)

// Trois damiers au choix du joueur. Ils dessinent tous le MÊME plateau (mêmes
// cases, mêmes voisins, mêmes règles) : seule la façon de le montrer change.
//
//  0. Losange            — une ligne par case, nos glyphes ⬢ ⬡ ·
//  1. Losange à bordures — idem, plus les bords Est–Ouest matérialisés
//  2. Nid d'abeilles     — deux lignes par case, l'hexagone est dessiné
//
// Le nid d'abeilles est le seul où l'on VOIT les six voisins, puisqu'ils
// partagent un côté. C'est aussi le plus encombrant.
var boardStyles = []string{"Losange", "Losange à bordures", "Nid d'abeilles"}

const (
	styleRhombus  = 0
	styleBordered = 1
	styleHoney    = 2
)

// coordsMargin : « 12 » et une espace, à gauche seulement. Les numéros à droite
// faisaient doublon et encombraient la marge du losange.
const coordsMargin = 3

// boardRow est une ligne du dessin. row vaut -1 pour une ligne qui n'est PAS une
// ligne de cases (les bordures Est–Ouest du damier n° 1) : ces lignes ne doivent
// pas recevoir de numéro.
type boardRow struct {
	text string
	row  int
}

// linesPerCell donne le nombre de lignes de texte occupées par une ligne de
// cases, selon le damier.
func linesPerCell(style int) int {
	if style == styleHoney {
		return 2
	}
	return 1
}

// borderLines donne le nombre de lignes de décoration ajoutées AVANT la première
// ligne de cases. Sert au calcul de la position du curseur pour la souris.
func borderLines(style int) int {
	if style == styleBordered {
		return 1
	}
	return 0
}

// cellMark renvoie le glyphe et le style d'une case : pierre du camp, case vide,
// ou curseur.
func (m Model) cellMark(r, q int) (string, lipgloss.Style) {
	mark, st := "·", styleMuted
	switch m.board.At(r, q) {
	case hex.NordSud:
		mark, st = "⬢", styleNordSud
	case hex.EstOuest:
		mark, st = "⬡", styleEstOuest
	}
	if m.onPath(r, q) {
		st = styleGood
	}
	if m.screen != scrWin && r == m.cursorR && q == m.cursorC {
		st = lipgloss.NewStyle().
			Foreground(colCursorFG).
			Background(colCursorBG).
			Bold(true)
	}
	return mark, st
}

// boardRawWidth renvoie la largeur du plateau seul, sans les coordonnées.
func (m Model) boardRawWidth() int {
	n := m.board.N
	switch m.styleIdx {
	case styleBordered:
		// « \ » + espace + cases + espace + « \ »
		return n*cellW + (n-1)*cellGap + indent(n-1) + 4
	case styleHoney:
		// chaque alvéole occupe 4 colonnes (le côté est partagé)
		return n*4 + indent(n-1)
	default:
		return n*cellW + (n-1)*cellGap + indent(n-1)
	}
}

// boardWidth renvoie la largeur totale du bloc, coordonnées comprises.
func (m Model) boardWidth() int {
	w := m.boardRawWidth()
	if m.showCoords {
		w += coordsMargin
	}
	return w
}

// pad complète une ligne à la largeur du bloc.
func (m Model) pad(line string) string {
	if d := m.boardWidth() - lipgloss.Width(line); d > 0 {
		return line + strings.Repeat(" ", d)
	}
	return line
}

// boardLines rend le plateau dans le damier choisi, coordonnées comprises.
//
// Les numéros de ligne ne sont écrits QU'À GAUCHE, et chacun suit le décalage de
// SA ligne : il est donc juste à gauche de ses propres cases, et non dans une
// marge lointaine. C'est ce que font les diagrammes de HexWiki, et c'est ce qui
// rend le numéro utile — on lit « la ligne 5 », pas « un numéro aligné quelque
// part à gauche ».
func (m Model) boardLines() []string {
	rows := m.boardRows()
	out := make([]string, 0, len(rows)+1)
	if m.showCoords {
		out = append(out, m.pad(strings.Repeat(" ", coordsMargin)+m.coordHeader()))
	}
	for _, b := range rows {
		line := b.text
		if m.showCoords {
			if b.row < 0 {
				// Ligne de décoration (bordure) : pas de numéro, mais on
				// réserve la même marge pour que tout reste aligné.
				line = strings.Repeat(" ", coordsMargin) + line
			} else {
				creux := strings.Repeat(" ", indent(b.row))
				line = creux + fmt.Sprintf("%2d ", b.row+1) +
					strings.TrimPrefix(b.text, creux)
			}
		}
		out = append(out, m.pad(line))
	}
	return out
}

// boardRows produit les lignes du damier, sans coordonnées et non complétées.
func (m Model) boardRows() []boardRow {
	switch m.styleIdx {
	case styleBordered:
		return m.rowsBordered()
	case styleHoney:
		return m.rowsHoneycomb()
	default:
		return m.rowsRhombus()
	}
}

// rowsRhombus : le damier d'origine, une ligne par case.
func (m Model) rowsRhombus() []boardRow {
	n := m.board.N
	rows := make([]boardRow, 0, n)
	for r := 0; r < n; r++ {
		var sb strings.Builder
		sb.WriteString(strings.Repeat(" ", indent(r)))
		for q := 0; q < n; q++ {
			if q > 0 {
				sb.WriteString(strings.Repeat(" ", cellGap))
			}
			mark, st := m.cellMark(r, q)
			sb.WriteString(st.Width(cellW).Align(lipgloss.Center).Render(mark))
		}
		rows = append(rows, boardRow{text: sb.String(), row: r})
	}
	return rows
}

// rowsBordered : comme le losange, mais les quatre bords sont matérialisés.
//
// Les bords NORD et SUD sont les deux bords HORIZONTAUX, en haut et en bas du
// losange : ce sont eux que relie le camp NordSud (r=0 et r=n-1). Ils sont donc
// tracés en lignes pleines.
//
// LES QUATRE BORDS SONT PENCHÉS. C'est le point qu'on rate : un losange n'a
// pas un bord horizontal plus long que les autres, ses quatre côtés suivent la
// même inclinaison d'un demi-pas. Aucun bord ne peut donc faire toute la
// largeur d'une ligne de cases : il en dépasse du côté vers lequel le losange
// s'incline. Chaque bord fait UN CRAN DE MOINS que sa ligne, et il est rogné du
// côté EXTÉRIEUR :
//
//   - le bord Nord commence avec sa ligne et s'arrête un cran avant la fin ;
//   - le bord Sud commence un cran après le début de sa ligne et finit avec.
//
// C'est exactement le tracé du damier « compact » à bordures de HexWiki
// (Unicode_and_ASCII_boards), vérifié colonne par colonne : la ligne 1 y porte
// ses « \ » en 10 et 34 (25 colonnes) et la bordure Nord court de 10 à 33.
//
// Les bords EST et OUEST sont les deux bords PENCHÉS (c=0 et c=n-1), tracés
// avec des « \ » au début et à la fin de chaque ligne.
func (m Model) rowsBordered() []boardRow {
	n := m.board.N
	rows := make([]boardRow, 0, n+2)

	// Un cran de moins que la largeur d'une ligne de cases.
	w := m.borderedRowWidth() - 1
	bord := func(creux int) boardRow {
		return boardRow{
			text: strings.Repeat(" ", creux) + styleBorder.Render(strings.Repeat("■", w)),
			row:  -1,
		}
	}
	rows = append(rows, bord(indent(0)))

	for r := 0; r < n; r++ {
		var sb strings.Builder
		sb.WriteString(strings.Repeat(" ", indent(r)))
		sb.WriteString(styleBorder.Render("\\ "))
		for q := 0; q < n; q++ {
			if q > 0 {
				sb.WriteString(strings.Repeat(" ", cellGap))
			}
			mark, st := m.cellMark(r, q)
			sb.WriteString(st.Width(cellW).Align(lipgloss.Center).Render(mark))
		}
		sb.WriteString(styleBorder.Render(" \\"))
		rows = append(rows, boardRow{text: sb.String(), row: r})
	}

	rows = append(rows, bord(indent(n-1)+1))
	return rows
}

// borderedRowWidth : largeur d'une ligne de cases du damier à bordures, ses
// décorations « \ » comprises. Les bords Nord et Sud font un cran de moins.
func (m Model) borderedRowWidth() int {
	n := m.board.N
	return n*cellW + (n-1)*cellGap + 4 // « \ » + espace + cases + espace + « \ »
}

// rowsHoneycomb : le nid d'abeilles. Chaque case est un hexagone de deux lignes
// larges de quatre colonnes. Deux cases voisines PARTAGENT un côté : c'est ce
// qui rend l'adjacence visible.
func (m Model) rowsHoneycomb() []boardRow {
	n := m.board.N
	rows := make([]boardRow, 0, 2*n)
	for r := 0; r < n; r++ {
		var top, bot strings.Builder
		top.WriteString(strings.Repeat(" ", indent(r)))
		bot.WriteString(strings.Repeat(" ", indent(r)))
		for q := 0; q < n; q++ {
			mark, st := m.cellMark(r, q)
			// L'intérieur de l'alvéole fait deux colonnes. Sur une case vide on
			// n'écrit rien, mais on colore tout l'intérieur — sinon le curseur
			// serait invisible.
			if mark == "·" {
				mark = " "
			}
			top.WriteString("/" + st.Width(2).Align(lipgloss.Center).Render(mark) + "\\")
			bot.WriteString("\\__/")
		}
		// Les deux lignes portent le même numéro : elles forment une seule ligne
		// de cases.
		rows = append(rows, boardRow{text: top.String(), row: r})
		rows = append(rows, boardRow{text: bot.String(), row: r})
	}
	return rows
}

// coordHeader : les lettres de colonnes, alignées sur la PREMIÈRE ligne de
// cases — comme sur les diagrammes de HexWiki.
func (m Model) coordHeader() string {
	n := m.board.N
	var sb strings.Builder
	sb.WriteString(strings.Repeat(" ", indent(0)))
	for q := 0; q < n; q++ {
		if q > 0 {
			sb.WriteString(strings.Repeat(" ", cellGap))
		}
		lettre := string(rune('A' + q))
		sb.WriteString(styleCoord.Width(cellW).Align(lipgloss.Center).Render(lettre))
	}
	return sb.String()
}
