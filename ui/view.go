package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"hex-tui/ai"
	"hex-tui/hex"
)

// Géométrie du plateau : chaque case occupe 3 colonnes, espacées de 4. Une case
// fait donc 4 colonnes de pas, et le DEMI-PAS vaut 2 colonnes.
//
// Chaque ligne est décalée d'un demi-pas vers la droite PAR RAPPORT À CELLE DU
// DESSUS : le décalage s'accumule (0, 2, 4, 6…) et ne revient jamais en
// arrière. C'est ce qui dessine le losange, et ce qui fait que les quatre cases
// diagonales voisines se touchent réellement à l'écran.
//
// Le piège à ne pas refaire : décaler UNE LIGNE SUR DEUX (r%2). On obtient
// alors un rectangle à bords en dents de scie, où les diagonales ne se touchent
// plus — l'image ne correspond plus aux vraies cases voisines.
const (
	cellW   = 3
	cellGap = 1
	rowOff  = 2 // demi-pas en colonnes = (cellW+cellGap)/2
)

// indent renvoie le décalage horizontal, en colonnes, de la ligne r.
func indent(r int) int { return rowOff * r }

// boardWidth renvoie la largeur en colonnes du plateau courant, décalage de la
// dernière ligne compris.
func (m Model) boardWidth() int {
	n := m.board.N
	return n*cellW + (n-1)*cellGap + indent(n-1)
}

// ---------------------------------------------------------------- plateau --

func (m Model) renderCell(r, q int) string {
	p := m.board.At(r, q)
	mark, st := "·", styleMuted
	switch p {
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
	return st.Width(cellW).Align(lipgloss.Center).Render(mark)
}

func (m Model) renderBoard() []string {
	n := m.board.N
	lines := make([]string, 0, n)
	for r := 0; r < n; r++ {
		var sb strings.Builder
		sb.WriteString(strings.Repeat(" ", indent(r)))
		for q := 0; q < n; q++ {
			if q > 0 {
				sb.WriteString(strings.Repeat(" ", cellGap))
			}
			sb.WriteString(m.renderCell(r, q))
		}
		// On complète à la largeur maximale : sinon les lignes basses, plus
		// larges à cause du décalage, seraient centrées différemment et le
		// losange apparaîtrait en escalier.
		line := sb.String()
		if pad := m.boardWidth() - lipgloss.Width(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		lines = append(lines, line)
	}
	return lines
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
		(m.width < m.boardWidth()+2 || m.height < m.board.N+8) {
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
		"\n" + styleMuted.Render(fmt.Sprintf("et %d lignes.", m.board.N+8)) +
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
		"",
		m.cfgActRow(m.menuRow == 3, "Commencer"),
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
		"",
		m.cfgActRow(m.settingsRow == 3, "Nouvelle partie"),
		m.cfgActRow(m.settingsRow == 4, "Menu principal"),
	}
	lines := []string{
		styleTitle.Render("Réglages"),
		styleSubtitle.Render(fmt.Sprintf("partie %d × %d en cours — conservée",
			m.board.N, m.board.N)),
		"",
		styleBox.Render(strings.Join(items, "\n")),
		"",
		styleGood.Render("Le niveau et l'adversaire s'appliquent aussitôt."),
		styleMuted.Render("La taille n'agit qu'avec « Nouvelle partie »."),
		"",
		m.settingsFooter(),
	}
	return m.assemble(lines)
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
	lines = append(lines, m.renderBoard()...)
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
	lines = append(lines, m.renderBoard()...)
	lines = append(lines, "")
	lines = append(lines, playerStyle(m.winner).Bold(true).
		Render("★  "+m.winner.String()+" relie ses deux bords — victoire !  ★"))
	lines = append(lines, styleMuted.Render(fmt.Sprintf("en %d coups", m.board.Moves)))
	lines = append(lines, "")
	lines = append(lines, styleMuted.Render("Entrée nouvelle partie   ·   s réglages   ·   q quitter"))
	return m.assemble(lines)
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
		{"hjkl/←→", "déplacer"}, {"yu bn", "diagonales"}, {"Entrée", "poser"},
		{"⌫", "annuler"}, {"s", "réglages"}, {"?", "aide"}, {"q", "quitter"},
	}
	short := [][2]string{
		{"hjkl", "déplacer"}, {"Entrée", "poser"}, {"s", "réglages"},
		{"?", "aide"}, {"q", "quitter"},
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
		{"↑  ou  y", "diagonale nord-ouest"},
		{"Maj+↑  ou  u", "diagonale nord-est"},
		{"↓  ou  n", "diagonale sud-est"},
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

// cellAt convertit des coordonnées terminal en (ligne, colonne) du plateau.
func (m Model) cellAt(x, y int) (int, int, bool) {
	lines := m.buildPlay()
	top := 0
	if gap := (m.height - len(lines)) / 2; gap > 0 {
		top = gap
	}
	gridY := top + 2 // titre + sous-titre
	gridX := (m.width - m.boardWidth()) / 2

	r := y - gridY
	if r < 0 || r >= m.board.N {
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
