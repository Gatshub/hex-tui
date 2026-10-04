package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hex-tui/hex"
)

// Quatre damiers au choix du joueur. Ils dessinent tous le MÊME plateau (mêmes
// cases, mêmes voisins, mêmes règles) : seule la façon de le montrer change.
//
//  0. Losange              — une ligne par case, nos glyphes ⬢ ⬡ ·
//  1. Losange à bordures   — idem, plus les quatre bords matérialisés
//  2. Nid d'abeilles       — deux lignes par case, l'hexagone est dessiné
//  3. Losange de hexagones — l'hexagone, mais une DIAGONALE par ligne
//
// Le nid d'abeilles et le losange de hexagones sont les deux où l'on VOIT les
// six voisins, puisqu'ils partagent un côté. Ce sont aussi les plus encombrants.
//
// Les trois premiers montrent le plateau d'aplomb (une RANGÉE par ligne) ; le
// quatrième le montre tourné (une DIAGONALE par ligne), ce qui lui donne cette
// silhouette de losange allongé. C'est le damier n° 2 de HexWiki
// (Unicode_and_ASCII_boards).
var boardStyles = []string{"Losange", "Losange à bordures", "Nid d'abeilles", "Losange de hexagones"}

const (
	styleRhombus  = 0
	styleBordered = 1
	styleHoney    = 2
	styleDiamond  = 3
)

// coordsMargin : « 12 » et une espace, à gauche seulement. Les numéros à droite
// faisaient doublon et encombraient la marge du losange.
const coordsMargin = 3

// honeyCell : largeur d'une alvéole, en colonnes. Le côté est partagé avec
// l'alvéole voisine, donc quatre colonnes suffisent : « / », deux colonnes
// intérieures, « \ ».
const honeyCell = 4

// honeyStep : décalage d'une rangée d'alvéoles sur la précédente, soit la
// MOITIÉ d'une alvéole. C'est ce demi-pas qui fait EMBOÎTER les rangées. Avec
// le décalage du losange (1 — juste, car sa case ne fait qu'une colonne), les
// alvéoles se chevauchent de trois colonnes sur quatre et le dessin n'est plus
// un nid d'abeilles.
const honeyStep = honeyCell / 2

// Le losange de hexagones dessine une DIAGONALE par ligne, avec les mêmes
// alvéoles que le nid d'abeilles : « / », deux colonnes intérieures, « \ ».
// Deux cases voisines DANS la ligne sont donc ici séparées par un « __ », qui
// est leur côté partagé — celui que le nid d'abeilles dessine en fin de ligne.
//
// Une ligne fait donc six colonnes par alvéole, et la ligne suivante est
// décalée de la MOITIÉ : diamondStep. C'est le même emboîtement que le nid
// d'abeilles, à un autre pas.
const (
	diamondCell   = 4
	diamondSep    = 2
	diamondPeriod = diamondCell + diamondSep
	diamondStep   = diamondPeriod / 2
)

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
//
// Le damier à bordures ouvre par sa bordure Nord ; le nid d'abeilles, par la
// ligne « __ » qui ferme le haut de sa première rangée — celle que la référence
// HexWiki dessine et qui nous manquait.
func borderLines(style int) int {
	if style == styleBordered || style == styleHoney {
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
		// chaque alvéole occupe honeyCell colonnes (le côté est partagé)
		return n*honeyCell + honeyStep*(n-1)
	case styleDiamond:
		// La ligne la plus longue est la diagonale du milieu : n alvéoles
		// (« / », deux colonnes, « \ ») séparées par n-1 « __ ».
		return n*diamondPeriod - diamondSep
	default:
		return n*cellW + (n-1)*cellGap + indent(n-1)
	}
}

// coordsVisible dit si les coordonnées ont un sens dans le damier courant.
//
// Elles n'en ont pas dans le losange de hexagones : ses lignes sont des
// DIAGONALES, donc ni « la ligne 5 » ni « la colonne C » ne désignent une ligne
// du dessin. Les afficher demanderait de les poser le long des bords, ce qui
// n'est pas fait — mieux vaut ne rien montrer qu'une marge qui ment.
func (m Model) coordsVisible() bool {
	return m.showCoords && m.styleIdx != styleDiamond
}

// boardWidth renvoie la largeur totale du bloc, coordonnées comprises.
func (m Model) boardWidth() int {
	w := m.boardRawWidth()
	if m.coordsVisible() {
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

// rowStep : décalage d'une ligne de cases à la suivante, en colonnes, dans le
// damier courant. Un demi-pas pour le losange (sa case fait une colonne), une
// demi-alvéole pour le nid d'abeilles (la sienne en fait quatre).
func (m Model) rowStep() int {
	if m.styleIdx == styleHoney {
		return honeyStep
	}
	return rowOff
}

// rowIndent : décalage horizontal, en colonnes, de la ligne de cases r dans le
// damier courant.
//
// Il dépend du damier, et c'est le piège de ce fichier : le losange (et le
// damier à bordures, qui est le même) décale d'un demi-pas de 1 colonne, tandis
// que le nid d'abeilles décale de honeyStep, parce que son alvéole fait quatre
// colonnes. Confondre les deux fait dériver les numéros de ligne par rapport à
// leurs cases.
func (m Model) rowIndent(r int) int {
	return m.rowStep() * r
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
	if m.coordsVisible() {
		out = append(out, m.pad(strings.Repeat(" ", coordsMargin)+m.coordHeader()))
	}
	for _, b := range rows {
		line := b.text
		if m.coordsVisible() {
			if b.row < 0 {
				// Ligne de décoration (bordure) : pas de numéro, mais on
				// réserve la même marge pour que tout reste aligné.
				line = strings.Repeat(" ", coordsMargin) + line
			} else {
				creux := strings.Repeat(" ", m.rowIndent(b.row))
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
	case styleDiamond:
		return m.rowsDiamond()
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
// larges de honeyCell colonnes. Deux cases voisines PARTAGENT un côté : c'est
// ce qui rend l'adjacence visible.
//
// Les deux lignes d'une rangée partent de la MÊME colonne — une alvéole a un
// sommet plat en haut comme en bas — et la rangée suivante est décalée d'un
// DEMI-PAS : honeyStep, la moitié d'une alvéole. C'est ce décalage, et non
// celui du losange, qui fait emboîter les rangées au lieu de les empiler.
func (m Model) rowsHoneycomb() []boardRow {
	n := m.board.N
	rows := make([]boardRow, 0, 2*n+1)

	// Le haut de la PREMIÈRE rangée, et lui seul, n'est fermé par rien : les
	// rangées suivantes le sont par le bas de celle du dessus. On trace donc
	// la ligne « __ » de la référence, dont les tirets tombent sur les deux
	// colonnes intérieures de chaque alvéole — c'est-à-dire exactement là où
	// passe leur bord supérieur.
	rows = append(rows, boardRow{
		text: strings.Repeat(" ", m.rowIndent(0)) + strings.Repeat(" __ ", n),
		row:  -1,
	})

	for r := 0; r < n; r++ {
		creux := strings.Repeat(" ", m.rowIndent(r))
		var top, bot strings.Builder
		top.WriteString(creux)
		bot.WriteString(creux)
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

// diamondLine : les cases de la diagonale r+q = s, dans l'ordre des r croissants.
//
// C'EST LA BONNE DIAGONALE, et c'est mesurable. Deux cases qui se suivent dans
// cette liste diffèrent de (1, -1) — la direction SUD-OUEST, donc deux cases
// VOISINES : elles partagent un côté, et c'est ce côté que le dessin figure par
// le « __ » qui les sépare.
//
// L'AUTRE diagonale (r-q constant) ne marche pas, et le premier jet de ce
// damier s'y était trompé. Deux cases qui s'y suivent diffèrent de (1, +1) :
// ce n'est PAS une direction du réseau (voir hex.DR/hex.DC), elles sont à deux
// cases l'une de l'autre. Le « __ » entre elles ne séparait donc rien, les
// vraies cases voisines tombaient ailleurs, et le pourtour s'ouvrait — sommet
// béant, bords inférieurs décousus, tirets orphelins aux deux pointes.
//
// Les deux diagonales donnent pourtant la même silhouette : une diagonale du
// plateau compte n - |d| cases, soit 1, 2, … n, … 2, 1, et la somme vaut n².
// Ce qui les distingue n'est pas la forme, c'est de savoir si les cases
// qu'on range côte à côte se touchent vraiment.
func diamondLine(n, s int) [][2]int {
	lo, hi := 0, n-1
	if s >= n {
		lo = s - (n - 1)
	}
	if s <= n-1 {
		hi = s
	}
	cells := make([][2]int, 0, hi-lo+1)
	for r := lo; r <= hi; r++ {
		cells = append(cells, [2]int{r, s - r})
	}
	return cells
}

// diamondIndent : décalage de la ligne portant la diagonale d. Il grandit avec
// la distance au milieu, d'un demi-période par cran — c'est l'emboîtement.
func diamondIndent(d int) int {
	if d < 0 {
		d = -d
	}
	return diamondStep * d
}

// rowsDiamond : le losange de hexagones — le damier n° 2 de HexWiki.
//
// C'est le MÊME plateau que les autres, montré TOURNÉ : une ligne du dessin est
// une DIAGONALE du plateau (r+q constant), et non une rangée. D'où la
// silhouette en losange allongé.
//
// ON POSE LES CARACTÈRES SUR UN CANEVAS, et c'est le cœur du tracé. Une ligne
// du dessin porte DEUX moitiés de cases : celles de la diagonale qui monte
// (leur « /x \ ») et celles de la diagonale du dessus (leur « \__/ »). Les deux
// se recouvrent partiellement, et là où elles se recouvrent elles doivent poser
// le MÊME caractère : le « \ » d'une case EST le « \ » de sa voisine. Les poser
// naïvement bout à bout, comme un premier jet l'a fait, ne dessine qu'une moitié
// de chaque case — et le pourtour s'ouvre.
//
// Deux bords sont posés à part, parce que personne d'autre ne les pose :
//
//   - la ligne 0, le bord SUPÉRIEUR de la première diagonale, sans quoi le
//     sommet du losange reste béant ;
//   - le « __ » à gauche de la première case et à droite de la dernière. Aux
//     deux POINTES du losange, celui de gauche n'a pas la place : il est omis,
//     exactement comme le fait la référence — et c'est celui que le joueur a vu
//     « ne servir à rien » quand je l'avais décalé pour le faire tenir.
func (m Model) rowsDiamond() []boardRow {
	n := m.board.N
	larg := m.boardRawWidth()

	// ind : décalage de la diagonale s, dont le « / » de la première case est
	// la première colonne.
	ind := func(s int) int { return diamondIndent(s - (n - 1)) }

	canevas := func() []rune {
		l := make([]rune, larg)
		for i := range l {
			l[i] = ' '
		}
		return l
	}
	pose := func(l []rune, col int, txt string) {
		for i, ch := range txt {
			if c := col + i; c >= 0 && c < len(l) {
				l[c] = ch
			}
		}
	}

	rows := make([]boardRow, 0, 2*n+1)

	// Ligne 0 : le bord supérieur de la première diagonale.
	haut := canevas()
	for i := range diamondLine(n, 0) {
		pose(haut, ind(0)+1+6*i, "__") // les deux colonnes INTÉRIEURES de la case
	}
	rows = append(rows, boardRow{text: strings.TrimRight(string(haut), " "), row: -1})

	for k := 1; k <= 2*n; k++ {
		s := k - 1 // la diagonale dont les cases font le HAUT de cette ligne
		l := canevas()
		// Le canevas ne porte que la GÉOMÉTRIE, en caractères nus. Le style des
		// cases est mis de côté et appliqué à l'émission : glisser les
		// séquences ANSI dans le canevas y ferait autant de « caractères », et
		// tout le tracé partirait de travers.
		stylé := map[int]string{}

		// Les cases de la diagonale s : leur haut et leur contenu.
		if s < 2*n-1 {
			for i, c := range diamondLine(n, s) {
				mark, st := m.cellMark(c[0], c[1])
				col := ind(s) + 6*i
				// Les glyphes ⬢ et ⬡ font DEUX colonnes pour UN rune : le
				// canevas est indexé par colonne, donc on n'y met qu'un repère
				// d'une colonne et c'est le style, à l'émission, qui occupe les
				// deux. Poser le glyphe ici décalerait toute la ligne.
				pose(l, col, "/ ")
				pose(l, col+3, `\`)
				stylé[col+1] = st.Width(2).Align(lipgloss.Center).Render(mark)
			}
		}
		// Les cases de la diagonale du DESSUS : leur bas, qui ferme aussi les
		// cases ci-dessus sur les côtés.
		if s-1 >= 0 {
			for i := range diamondLine(n, s-1) {
				pose(l, ind(s-1)+6*i, `\__/`)
			}
		}
		// Les deux bords que personne n'a posés : le « __ » à gauche de la
		// première case et à droite de la dernière. C'est la PORTÉE du dessin qui
		// les règle, et elle suffit :
		//
		//   - moitié haute, la diagonale du dessus est plus à DROITE et ne
		//     couvre pas ces colonnes : on les pose, elles ferment le bord ;
		//   - moitié basse, elle est plus à GAUCHE et son « \__/ » occupe DÉJÀ
		//     ces deux colonnes : les reposer ne changerait pas un caractère ;
		//   - aux deux POINTES, la colonne tombe hors du dessin : rien n'est
		//     posé, et c'est exactement ce que fait la référence.
		//
		// Une condition « suis-je dans la moitié haute ? » a été essayée puis
		// retirée : la mutation qui la neutralise ne change pas un caractère du
		// rendu. Elle ne servait à rien, et un test ne peut pas la défendre.
		if s < 2*n-1 {
			nb := len(diamondLine(n, s))
			if g := ind(s) - diamondSep; g >= 0 {
				pose(l, g, strings.Repeat("_", diamondSep))
			}
			if d := ind(s) + 6*(nb-1) + diamondCell; d < larg {
				pose(l, d, strings.Repeat("_", diamondSep))
			}
		}

		var sb strings.Builder
		for col := 0; col < larg; {
			if txt, ok := stylé[col]; ok {
				sb.WriteString(txt)
				col += 2
				continue
			}
			sb.WriteRune(l[col])
			col++
		}
		rows = append(rows, boardRow{text: strings.TrimRight(sb.String(), " "), row: -1})
	}
	return rows
}

// coordCol : colonne de la LETTRE de la colonne q dans l'en-tête du damier
// courant. Elle doit tomber sur le CONTENU de la case, jamais à côté, et le
// calcul dépend donc du damier : une case de losange fait une colonne, une
// alvéole en fait quatre, et le damier à bordures décale ses cases de ses deux
// caractères « \ ».
func (m Model) coordCol(q int) int {
	switch m.styleIdx {
	case styleBordered:
		return indent(0) + 2 + q*(cellW+cellGap) // « \ » puis les cases
	case styleHoney:
		return q*honeyCell + 1 // première des deux colonnes intérieures
	default:
		return indent(0) + q*(cellW+cellGap)
	}
}

// coordHeader : les lettres de colonnes, alignées sur la PREMIÈRE ligne de
// cases — comme sur les diagrammes de HexWiki.
func (m Model) coordHeader() string {
	n := m.board.N
	var sb strings.Builder
	col := 0
	for q := 0; q < n; q++ {
		c := m.coordCol(q)
		if c > col {
			sb.WriteString(strings.Repeat(" ", c-col))
		}
		sb.WriteString(styleCoord.Render(string(rune('A' + q))))
		col = c + 1
	}
	return sb.String()
}
