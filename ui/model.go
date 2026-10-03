package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hex-tui/ai"
	"hex-tui/hex"
)

type screen int

const (
	scrMenu screen = iota
	scrPlay
	scrWin
)

// Sizes liste les tailles de plateau proposées.
var Sizes = []int{7, 9, 11, 13}

// Adversaires proposés au menu.
var opponents = []string{"2 joueurs", "Ordinateur"}

// L'ordinateur joue toujours Est–Ouest : l'humain garde Nord–Sud, et donc
// l'avantage du premier coup (au Hex, le premier joueur gagne théoriquement).
const aiPlayer = hex.EstOuest

// Model est l'état complet du programme.
type Model struct {
	width, height int
	ready         bool
	screen        screen
	showHelp      bool
	showSettings  bool

	// Menu.
	menuRow     int // 0 taille, 1 adversaire, 2 niveau, 3 commencer
	sizeIdx     int
	opponentIdx int
	levelIdx    int

	// Panneau de réglages en cours de partie.
	settingsRow int // 0 taille, 1 adversaire, 2 niveau, 3 nouvelle partie, 4 menu

	// Partie.
	board    hex.Board
	turn     hex.Player
	winner   hex.Player
	winPath  [][2]int
	cursorR  int
	cursorC  int
	history  []hex.Board
	vsAI     bool
	thinking bool

	// Moteur de jeu, conservé pendant TOUTE la partie : il garde son arbre de
	// recherche d'un coup à l'autre, ce qui le rend nettement plus fort à
	// budget égal (la réflexion déjà faite n'est plus jetée). Recréé à chaque
	// nouvelle partie.
	engine *ai.AI

	status      string
	statusStyle lipgloss.Style
}

// New construit le modèle initial (écran de réglages).
func New() Model {
	return Model{
		screen:      scrMenu,
		menuRow:     3,
		sizeIdx:     2, // 11x11
		opponentIdx: 1, // ordinateur
		levelIdx:    1, // moyen
		turn:        hex.NordSud,
		statusStyle: styleMuted,
	}
}

func (m Model) Init() tea.Cmd { return nil }

// aiMoveMsg porte le coup calculé par le moteur.
type aiMoveMsg struct {
	r, c int
	ok   bool
}

// Update traite les messages Bubble Tea.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		return m, nil
	case aiMoveMsg:
		m.thinking = false
		if msg.ok && m.winner == hex.None && m.screen == scrPlay {
			m.placeAt(msg.r, msg.c)
		}
		return m, nil
	case tea.MouseMsg:
		m.handleMouse(msg)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// ----------------------------------------------------------------- clavier --

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}
	if m.showHelp {
		switch key {
		case "?", "esc", "q", "enter", " ":
			m.showHelp = false
		}
		return m, nil
	}
	if m.showSettings {
		return m.settingsKey(key)
	}
	switch m.screen {
	case scrMenu:
		return m.menuKey(key)
	case scrPlay:
		return m.playKey(key)
	default:
		return m.winKey(key)
	}
}

func (m Model) menuKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.menuRow > 0 {
			m.menuRow--
		}
	case "down", "j":
		if m.menuRow < 3 {
			m.menuRow++
		}
	case "left", "h":
		m.menuChange(-1)
	case "right", "l":
		m.menuChange(1)
	case "enter", " ":
		return m.start(Sizes[m.sizeIdx]), nil
	}
	return m, nil
}

// menuChange modifie la valeur de la ligne sélectionnée au menu.
func (m *Model) menuChange(d int) { m.changeValue(m.menuRow, d) }

// changeValue modifie la valeur de la ligne `row` (0 taille, 1 adversaire,
// 2 niveau). La ligne est passée en paramètre car le menu et le panneau de
// réglages en cours de partie partagent ces valeurs mais pas leur sélection.
// Le nombre de joueurs est répercuté tout de suite, pour que le réglage en
// cours de partie prenne effet sans attendre.
func (m *Model) changeValue(row, d int) {
	cycle := func(v, n int) int { return ((v+d)%n + n) % n }
	switch row {
	case 0:
		m.sizeIdx = cycle(m.sizeIdx, len(Sizes))
	case 1:
		m.opponentIdx = cycle(m.opponentIdx, len(opponents))
		m.vsAI = m.opponentIdx == 1
	case 2:
		m.levelIdx = cycle(m.levelIdx, len(ai.Levels))
	}
}

func (m Model) playKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "esc":
		m.openSettings()
		return m, nil
	case "s":
		m.openSettings()
		return m, nil
	case "?":
		m.showHelp = true
		return m, nil
	}
	if m.thinking {
		return m, nil // le moteur réfléchit : on ne bouscule pas la partie
	}
	switch key {
	case "r":
		m.restart()
		return m, nil
	case "backspace", "delete":
		m.undo()
		return m, nil
	case "enter", " ":
		return m, m.place()
	case "h", "left":
		m.move("W")
	case "l", "right":
		m.move("E")
	case "y", "up":
		m.move("NW")
	case "u", "shift+up":
		m.move("NE")
	case "b", "shift+down":
		m.move("SW")
	case "n", "down":
		m.move("SE")
	}
	return m, nil
}

func (m Model) winKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = scrMenu
	case "enter", " ":
		return m.start(Sizes[m.sizeIdx]), nil
	}
	return m, nil
}

// -------------------------------------------------------- réglages en jeu --

// openSettings ouvre le panneau de réglages sans toucher à la partie : c'est
// tout l'intérêt — on peut changer d'adversaire ou de niveau à tout moment,
// sans avoir à quitter la partie en cours.
func (m *Model) openSettings() {
	m.showSettings = true
	m.settingsRow = 2 // on arrive sur le niveau, la demande la plus fréquente
}

// settingsKey gère le panneau de réglages.
func (m Model) settingsKey(key string) (tea.Model, tea.Cmd) {
	const lastRow = 4
	switch key {
	case "q":
		return m, tea.Quit
	case "s", "esc", "?":
		return m, m.closeSettings()
	case "up", "k":
		if m.settingsRow > 0 {
			m.settingsRow--
		}
	case "down", "j":
		if m.settingsRow < lastRow {
			m.settingsRow++
		}
	case "left", "h":
		m.changeValue(m.settingsRow, -1)
	case "right", "l":
		m.changeValue(m.settingsRow, 1)
	case "enter", " ":
		switch m.settingsRow {
		case 3: // nouvelle partie, avec la taille choisie
			return m.start(Sizes[m.sizeIdx]), nil
		case 4: // retour au menu principal
			m.showSettings = false
			m.thinking = false
			m.screen = scrMenu
			return m, nil
		default:
			return m, m.closeSettings()
		}
	}
	return m, nil
}

// closeSettings referme le panneau et reprend la partie là où elle en était.
// Si le passage à l'ordinateur vient de mettre la main au moteur, celui-ci se
// remet au travail immédiatement.
func (m *Model) closeSettings() tea.Cmd {
	m.showSettings = false
	if m.vsAI && m.winner == hex.None && m.turn == aiPlayer && !m.thinking {
		m.thinking = true
		m.setStatus(styleEstOuest, "L'ordinateur réfléchit…")
		return m.aiThinking()
	}
	return nil
}

// ------------------------------------------------------------------ souris --

func (m *Model) handleMouse(msg tea.MouseMsg) {
	if m.screen != scrPlay || m.showHelp || m.thinking {
		return
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return
	}
	r, c, ok := m.cellAt(msg.X, msg.Y)
	if !ok {
		return
	}
	if m.cursorR == r && m.cursorC == c {
		m.place() // second clic sur la même case : on joue
		return
	}
	m.cursorR, m.cursorC = r, c
}

// ------------------------------------------------------------------- jeu --

func (m *Model) start(n int) Model {
	m.board = hex.NewBoard(n)
	m.turn = hex.NordSud
	m.winner = hex.None
	m.winPath = nil
	m.history = nil
	m.showHelp = false
	m.showSettings = false
	m.thinking = false
	m.vsAI = m.opponentIdx == 1
	// Moteur neuf pour chaque partie : nouveau tirage aléatoire, et arbre de
	// recherche vide (celui de la partie précédente ne vaut plus rien).
	m.engine = ai.New(ai.Rollouts(m.levelIdx, n), time.Now().UnixNano())
	m.cursorR, m.cursorC = n-1, n-1
	m.screen = scrPlay
	m.setStatus(styleNordSud, "Nord–Sud commence")
	return *m
}

// dirIndex relie les six directions du clavier aux six directions du paquet hex.
// L'ordre est celui, cyclique, de hex.DR / hex.DC : est, sud-est, sud-ouest,
// ouest, nord-ouest, nord-est.
//
// Le curseur n'a donc plus sa propre table de directions : il n'existe plus
// qu'une seule définition des voisins dans tout le programme. Auparavant il y
// en avait trois (règles, moteur, interface), et elles avaient divergé.
var dirIndex = map[string]int{
	"E": 0, "SE": 1, "SW": 2, "W": 3, "NW": 4, "NE": 5,
}

// step traduit une direction en déplacement (dr, dc).
func (m Model) step(dir string) (int, int) {
	k, ok := dirIndex[dir]
	if !ok {
		return 0, 0
	}
	return hex.DR[k], hex.DC[k]
}

func (m *Model) move(dir string) {
	dr, dc := m.step(dir)
	if nr, nc := m.cursorR+dr, m.cursorC+dc; m.board.InBounds(nr, nc) {
		m.cursorR, m.cursorC = nr, nc
	}
}

// place pose une pierre sur la case du curseur.
func (m *Model) place() tea.Cmd {
	if m.winner != hex.None {
		return nil
	}
	if m.board.At(m.cursorR, m.cursorC) != hex.None {
		m.setStatus(styleWarn, "Case déjà occupée")
		return nil
	}
	return m.placeAt(m.cursorR, m.cursorC)
}

// placeAt pose une pierre pour le joueur courant, puis passe la main — ou
// lance la réflexion du moteur si c'est à lui de jouer.
func (m *Model) placeAt(r, c int) tea.Cmd {
	m.history = append(m.history, m.board.Clone())
	m.board.Set(r, c, m.turn)

	if w := m.board.Winner(); w != hex.None {
		m.winner = w
		m.winPath = m.board.WinningCells(w)
		m.screen = scrWin
		return nil
	}
	m.turn = m.turn.Other()

	if m.vsAI && m.turn == aiPlayer {
		m.thinking = true
		m.setStatus(styleEstOuest, "L'ordinateur réfléchit…")
		return m.aiThinking()
	}
	m.setStatus(playerStyle(m.turn), "Au tour de "+m.turn.String())
	return nil
}

// aiThinking calcule le coup du moteur hors du fil d'affichage : l'interface
// reste fluide même sur un grand plateau au niveau difficile.
func (m *Model) aiThinking() tea.Cmd {
	if m.engine == nil {
		m.engine = ai.New(ai.Rollouts(m.levelIdx, m.board.N), time.Now().UnixNano())
	}
	// Un changement de niveau en cours de partie ne jette pas l'arbre déjà
	// construit : seul le budget de la prochaine recherche change.
	m.engine.SetRollouts(ai.Rollouts(m.levelIdx, m.board.N))

	engine := m.engine
	board := m.board.Clone()
	p := m.turn
	return func() tea.Msg {
		r, c, ok := engine.Move(board, p)
		return aiMoveMsg{r: r, c: c, ok: ok}
	}
}

func (m *Model) undo() {
	if len(m.history) == 0 {
		m.setStatus(styleWarn, "Rien à annuler")
		return
	}
	m.board = m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.turn = m.turn.Other()
	m.setStatus(playerStyle(m.turn), "Coup annulé — au tour de "+m.turn.String())
}

func (m *Model) restart() {
	n := m.board.N
	m.board = hex.NewBoard(n)
	m.turn = hex.NordSud
	m.winner = hex.None
	m.winPath = nil
	m.history = nil
	m.thinking = false
	m.engine = nil // le moteur reprendra un arbre neuf, avec une nouvelle graine
	m.cursorR, m.cursorC = n-1, n-1
	m.setStatus(styleMuted, "Plateau réinitialisé")
}

func (m *Model) setStatus(st lipgloss.Style, text string) {
	m.status, m.statusStyle = text, st
}

// onPath indique si la case fait partie de la chaîne gagnante.
func (m Model) onPath(r, c int) bool {
	for _, p := range m.winPath {
		if p[0] == r && p[1] == c {
			return true
		}
	}
	return false
}
