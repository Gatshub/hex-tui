package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hex-tui/ai"
	"hex-tui/hex"
)

// Géométrie du plateau : une case fait UNE colonne, séparée de la suivante par
// une colonne vide. Le pas vaut donc 2 colonnes, et le DEMI-PAS 1.
//
// Pourquoi si serré : un caractère de terminal est environ deux fois plus haut
// que large. Pour qu'une diagonale du réseau hexagonal apparaisse à 60° — sa
// vraie direction — il faut qu'elle avance d'une colonne pour une ligne. Avec
// un pas de 4 colonnes, les diagonales tombaient à 45° et l'œil ne voyait plus
// six voisins mais quatre : les deux cases diagonales semblaient trop loin.
// Resserrer n'est donc pas qu'une question de place gagnée.
//
// Chaque ligne est décalée d'un demi-pas vers la droite PAR RAPPORT À CELLE DU
// DESSUS : le décalage s'accumule (0, 1, 2, 3…) et ne revient jamais en
// arrière. C'est ce qui dessine le losange.
//
// Le piège à ne pas refaire : décaler UNE LIGNE SUR DEUX (r%2). On obtient
// alors un rectangle à bords en dents de scie, où les diagonales ne se touchent
// plus — l'image ne correspond plus aux vraies cases voisines.
const (
	cellW   = 1
	cellGap = 1
	rowOff  = 1 // demi-pas en colonnes = (cellW+cellGap)/2
)

// indent renvoie le décalage horizontal, en colonnes, de la ligne r.
// Le plateau, les bordures et les coordonnées s'appuient tous dessus : c'est la
// seule définition du « demi-pas » dans l'interface.
func indent(r int) int { return rowOff * r }

// ---------------------------------------------------------------- plateau --

// renderCell existe pour les tests et les aperçus : c'est une case du damier
// « losange », rendue dans une largeur fixe. Le rendu du plateau lui-même est
// dans board.go (trois damiers au choix).
func (m Model) renderCell(r, q int) string {
	mark, st := m.cellMark(r, q)
	return st.Width(cellW).Align(lipgloss.Center).Render(mark)
}

// ---------------------------------------------------------------- écrans --

func (m Model) View() string {
	if !m.ready {
		return ""
	}
	if m.showHelp {
		return m.assemble(m.helpLines())
	}
	if m.showSettings {
		return m.viewSettings()
	}
	if m.screen != scrMenu &&
		(m.width < m.boardWidth()+2 || m.height < len(m.boardLines())+8) {
		return m.viewTooSmall()
	}
	switch m.screen {
	case scrMenu:
		return m.viewMenu()
	case scrWin:
		return m.viewWin()
	default:
		return m.viewPlay()
	}
}

func (m Model) viewTooSmall() string {
	msg := styleWarn.Render(fmt.Sprintf("Fenêtre trop petite : %d×%d", m.width, m.height)) +
		"\n\n" + styleMuted.Render(fmt.Sprintf("Plateau %d×%d : au moins %d colonnes",
		m.board.N, m.board.N, m.boardWidth()+2)) +
		"\n" + styleMuted.Render(fmt.Sprintf("et %d lignes.", len(m.boardLines())+8)) +
		"\n\n" + styleMuted.Render("Échap : menu   ·   q : quitter")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().Align(lipgloss.Center).Render(msg))
}

// cfgRow rend une ligne « libellé ◀ valeur ▶ » d'un panneau de réglages.
func (m Model) cfgRow(sel bool, label, v string) string {
	mk := "  "
	lbl := styleMuted.Render(fmt.Sprintf("%-11s", label))
	val := styleFG.Render("◀  " + v + "  ▶")
	if sel {
		hi := lipgloss.NewStyle().Foreground(colAccent).Bold(true)
		mk = hi.Render("▶ ")
		lbl = lipgloss.NewStyle().Foreground(colFG).Bold(true).
			Render(fmt.Sprintf("%-11s", label))
		val = hi.Render("◀  " + v + "  ▶")
	}
	return mk + lbl + val
}

// cfgDimRow rend une ligne désactivée.
func (m Model) cfgDimRow(label, v string) string {
	return "  " + styleMuted.Render(fmt.Sprintf("%-11s", label)) + styleMuted.Render(v)
}

// cfgActRow rend une ligne d'action (pas de valeur à faire défiler).
func (m Model) cfgActRow(sel bool, label string) string {
	if sel {
		return lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render("▶ " + label)
	}
	return "  " + styleFG.Render(label)
}

// cfgLevelRow rend la ligne du niveau, grisée à deux joueurs.
func (m Model) cfgLevelRow(sel bool) string {
	if m.opponentIdx != 1 {
		return m.cfgDimRow("Niveau", "—  (sans objet à deux joueurs)")
	}
	return m.cfgRow(sel, "Niveau", ai.Levels[m.levelIdx].Name)
}

// cfgSize retourne la taille sélectionnée, formatée.
func (m Model) cfgSize() string {
	n := Sizes[m.sizeIdx]
	return fmt.Sprintf("%d × %d", n, n)
}

func (m Model) viewMenu() string {
	items := []string{
		m.cfgRow(m.menuRow == 0, "Taille", m.cfgSize()),
		m.cfgRow(m.menuRow == 1, "Adversaire", opponents[m.opponentIdx]),
		m.cfgLevelRow(m.menuRow == 2),
		m.cfgRow(m.menuRow == 3, "Damier", boardStyles[m.styleIdx]),
		m.cfgRow(m.menuRow == 4, "Coordonnées", ouiNon(m.showCoords)),
		"",
		m.cfgActRow(m.menuRow == lastMenuRow, "Commencer"),
	}
	lines := []string{
		styleTitle.Render("H E X"),
		styleMuted.Render("Le jeu de John Nash — relier ses deux bords"),
		"",
		styleBox.Render(strings.Join(items, "\n")),
		"",
		m.menuFooter(),
	}
	return m.assemble(lines)
}

// viewSettings est le panneau de réglages accessible EN COURS DE PARTIE. Il
// laisse le plateau intact : on peut changer d'adversaire ou de niveau sans
// avoir à ressortir du jeu, ce qui était le défaut signalé.
func (m Model) viewSettings() string {
	items := []string{
		m.cfgRow(m.settingsRow == 0, "Taille", m.cfgSize()),
		m.cfgRow(m.settingsRow == 1, "Adversaire", opponents[m.opponentIdx]),
		m.cfgLevelRow(m.settingsRow == 2),
		m.cfgRow(m.settingsRow == 3, "Damier", boardStyles[m.styleIdx]),
		m.cfgRow(m.settingsRow == 4, "Coordonnées", ouiNon(m.showCoords)),
		"",
		m.cfgActRow(m.settingsRow == rowNewGame, "Nouvelle partie"),
		m.cfgActRow(m.settingsRow == rowMainMenu, "Menu principal"),
	}
	lines := []string{
		styleTitle.Render("Réglages"),
		styleSubtitle.Render(fmt.Sprintf("partie %d × %d en cours — conservée",
			m.board.N, m.board.N)),
		"",
		styleBox.Render(strings.Join(items, "\n")),
		"",
		styleGood.Render("Niveau, adversaire, damier et coordonnées s'appliquent aussitôt."),
		styleMuted.Render("La taille n'agit qu'avec « Nouvelle partie »."),
		"",
		m.settingsFooter(),
	}
	return m.assemble(lines)
}

// ouiNon rend un booléen en français, pour les lignes de réglages.
func ouiNon(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}

// settingsFooter adapte le rappel des touches à la largeur disponible.
func (m Model) settingsFooter() string {
	for _, s := range []string{
		"↑ ↓ choisir   ·   ← → modifier   ·   Entrée valider   ·   Échap reprendre la partie",
		"↑↓ choisir  ·  ←→ modifier  ·  Entrée  ·  Échap reprendre",
		"↑↓ ←→ Entrée  ·  Échap reprendre",
	} {
		if lipgloss.Width(s) <= m.width {
			return styleMuted.Render(s)
		}
	}
	return styleMuted.Render("Échap reprendre")
}

// menuFooter adapte le rappel des touches à la largeur disponible.
func (m Model) menuFooter() string {
	for _, s := range []string{
		"↑ ↓ choisir   ·   ← → modifier   ·   Entrée lancer   ·   q quitter",
		"↑↓ choisir  ·  ←→ modifier  ·  Entrée lancer  ·  q quitter",
		"↑↓ ←→ Entrée  ·  q quitter",
	} {
		if lipgloss.Width(s) <= m.width {
			return styleMuted.Render(s)
		}
	}
	return styleMuted.Render("q quitter")
}

func (m Model) viewPlay() string {
	return m.assemble(m.buildPlay())
}

func (m Model) buildPlay() []string {
	lines := []string{
		styleTitle.Render("H E X"),
		styleSubtitle.Render(m.subtitle()),
	}
	lines = append(lines, m.boardLines()...)
	lines = append(lines, "")
	lines = append(lines, m.statusStyle.Render(m.status))
	lines = append(lines, m.legendLine())
	lines = append(lines, m.hintsLine())
	return lines
}

func (m Model) viewWin() string {
	lines := []string{
		styleTitle.Render("H E X"),
		styleSubtitle.Render(fmt.Sprintf("Partie terminée — %d × %d", m.board.N, m.board.N)),
	}
	lines = append(lines, m.boardLines()...)
	lines = append(lines, "")
	lines = append(lines, playerStyle(m.winner).Bold(true).
		Render("★  "+m.winner.String()+" relie ses deux bords — victoire !  ★"))
	lines = append(lines, styleMuted.Render(fmt.Sprintf("en %d coups", m.board.Moves)))
	lines = append(lines, "")
	lines = append(lines, styleMuted.Render(bandeauFin()))
	return m.assemble(lines)
}

// annonce : une touche affichée à l'écran de fin, avec le libellé qui l'annonce.
type annonce struct {
	key   string // la touche, telle que Bubble Tea la transmet
	label string // ce que l'écran écrit
}

// annoncesFin liste les touches de l'écran de fin. Le bandeau affiché est
// construit À PARTIR de cette liste, et le test en tire les touches qu'il
// exerce : l'écran ne peut donc plus annoncer une touche que winKey ne gère
// pas, ni l'inverse.
//
// C'est l'erreur commise deux fois dans ce projet — l'aide annonçait « j » et
// « k » sans qu'ils déplacent le curseur, puis « s réglages » à la fin d'une
// partie sans que « s » fasse quoi que ce soit. Une aide qui ment est pire
// qu'une aide absente : on croit le programme cassé.
var annoncesFin = []annonce{
	{"enter", "Entrée nouvelle partie"},
	{"s", "s réglages"},
	{"esc", "Échap menu"},
	{"q", "q quitter"},
}

// bandeauFin rend le bandeau d'aide de l'écran de fin à partir de annoncesFin.
func bandeauFin() string {
	labels := make([]string, 0, len(annoncesFin))
	for _, a := range annoncesFin {
		labels = append(labels, a.label)
	}
	return strings.Join(labels, "   ·   ")
}

// ------------------------------------------------------------- fragments --

func (m Model) subtitle() string {
	if m.winner != hex.None {
		return "victoire de " + m.winner.String()
	}
	s := fmt.Sprintf("au tour de %s", m.turn.String())
	if m.vsAI {
		s += fmt.Sprintf("   ·   ordinateur (%s)", ai.Levels[m.levelIdx].Name)
	}
	return s
}

func (m Model) legendLine() string {
	full := styleMuted.Render("haut ↕ bas : ") + styleNordSud.Render("⬢ Nord–Sud") +
		styleMuted.Render("   ·   gauche ↔ droite : ") + styleEstOuest.Render("⬡ Est–Ouest")
	if lipgloss.Width(full) <= m.width {
		return full
	}
	short := styleNordSud.Render("⬢ Nord–Sud") +
		styleMuted.Render("  ·  ") + styleEstOuest.Render("⬡ Est–Ouest")
	if lipgloss.Width(short) <= m.width {
		return short
	}
	return styleNordSud.Render("⬢") + styleMuted.Render("  ·  ") + styleEstOuest.Render("⬡")
}

func (m Model) hintsLine() string {
	full := [][2]string{
		{"h j k l/←→↑↓", "déplacer"}, {"y u b n", "les 4 diagonales"}, {"Entrée", "poser"},
		{"⌫", "annuler"}, {"s", "réglages"}, {"?", "aide"}, {"q", "quitter"},
	}
	short := [][2]string{
		{"h j k l", "déplacer"}, {"y u b n", "diagonales"}, {"Entrée", "poser"},
		{"s", "réglages"}, {"?", "aide"}, {"q", "quitter"},
	}
	mini := [][2]string{{"s", "réglages"}, {"q", "quitter"}}
	build := func(items [][2]string) string {
		parts := make([]string, 0, len(items))
		for _, it := range items {
			parts = append(parts, styleHelpKey.Render(it[0])+styleMuted.Render(" "+it[1]))
		}
		return strings.Join(parts, styleMuted.Render("  ·  "))
	}
	for _, items := range [][][2]string{full, short, mini} {
		if l := build(items); lipgloss.Width(l) <= m.width {
			return l
		}
	}
	return build(mini)
}

func (m Model) helpLines() []string {
	rows := [][2]string{
		{"← →  ou  h l", "se déplacer d'est en ouest"},
		{"↑  ou  k  ou  y", "diagonale nord-ouest"},
		{"↓  ou  j  ou  n", "diagonale sud-est"},
		{"Maj+↑  ou  u", "diagonale nord-est"},
		{"Maj+↓  ou  b", "diagonale sud-ouest"},
		{"Entrée / Espace", "poser une pierre"},
		{"Retour arrière", "annuler le dernier coup"},
		{"r", "réinitialiser le plateau"},
		{"s  ou  Échap", "réglages — SANS quitter la partie"},
		{"souris", "1er clic : viser   ·   2e clic : poser"},
		{"?", "fermer cette aide"},
		{"q", "quitter"},
	}
	width := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > width {
			width = w
		}
	}
	var b strings.Builder
	b.WriteString(styleTitle.Render("Aide — Hex") + "\n\n")
	for _, r := range rows {
		pad := strings.Repeat(" ", width-lipgloss.Width(r[0])+3)
		b.WriteString(styleHelpKey.Render(r[0]) + pad + styleMuted.Render(r[1]) + "\n")
	}
	b.WriteString("\n" + styleMuted.Render("⬢ Nord–Sud relie le haut au bas ; ⬡ Est–Ouest la gauche à la droite."))
	b.WriteString("\n" + styleMuted.Render("Pas de match nul : un des deux camps gagne toujours."))
	return strings.Split(styleBox.Render(strings.TrimRight(b.String(), "\n")), "\n")
}

// ------------------------------------------------------------- assemblage --

func (m Model) assemble(lines []string) string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, l)
	}
	top := 0
	if gap := (m.height - len(out)) / 2; gap > 0 {
		top = gap
	}
	return strings.Repeat("\n", top) + strings.Join(out, "\n")
}

// -------------------------------------------------------------- souris --

// cellAt convertit des coordonnées terminal en (ligne, colonne) du plateau. Il
// doit suivre exactement le damier choisi : nombre de lignes par case, décalage
// en demi-pas, et marge des coordonnées s'ils sont affichés.
func (m Model) cellAt(x, y int) (int, int, bool) {
	lines := m.buildPlay()
	top := 0
	if gap := (m.height - len(lines)) / 2; gap > 0 {
		top = gap
	}
	gridY := top + 2 + borderLines(m.styleIdx) // titre + sous-titre + bordure haute
	if m.showCoords {
		gridY++ // décalage : l'en-tête des lettres fait partie du bloc
	}
	gridX := (m.width - m.boardWidth()) / 2
	if m.showCoords {
		gridX += 3 // « 12 » et une espace, à gauche
	}

	// Une ligne de cases peut occuper plusieurs lignes de texte (nid d'abeilles).
	per := linesPerCell(m.styleIdx)
	r := (y - gridY) / per
	if y < gridY || r < 0 || r >= m.board.N {
		return 0, 0, false
	}
	lx := x - gridX - indent(r)
	if lx < 0 {
		return 0, 0, false
	}
	q := lx / (cellW + cellGap)
	if q < 0 || q >= m.board.N {
		return 0, 0, false
	}
	if lx%(cellW+cellGap) >= cellW {
		return 0, 0, false // entre deux cases
	}
	return r, q, true
}
