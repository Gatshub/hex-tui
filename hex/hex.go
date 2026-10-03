// Package hex implémente les règles du jeu de Hex sur un plateau en LOSANGE
// n x n. Logique pure, sans dépendance UI.
//
// Coordonnées : (ligne, colonne), 0 en haut à gauche. La case (r,c) est le
// r-ième losange de la ligne r. Chaque ligne est décalée d'un DEMI-PAS vers la
// droite par rapport à celle du dessus : le décalage s'accumule, il ne revient
// jamais en arrière. C'est ce qui donne le losange de l'échiquier de Hex — le
// décalage en quinconce (une ligne sur deux), lui, donnerait un rectangle à
// bords dentelés.
//
// Conséquence sur l'adjacence : les six voisins sont les MÊMES pour toutes les
// cases, sans dépendre de la parité de la ligne. Voir DR/DC.
//
// Les deux joueurs relient deux bords opposés :
//   - NordSud  relie la première ligne (r=0) à la dernière (r=n-1)
//   - EstOuest relie la première colonne (c=0) à la dernière (c=n-1)
package hex

// Les six directions, en ORDRE CYCLIQUE : est, sud-est, sud-ouest, ouest,
// nord-ouest, nord-est.
//
// L'ordre n'est pas cosmétique : deux directions séparées d'un cran dans cet
// ordre sont à deux cases l'une de l'autre et partagent un voisin commun. Le
// moteur s'en sert pour détecter les ponts (voir ai/vc.go), et l'interface pour
// les six déplacements du curseur. Une seule source de vérité pour les trois.
var (
	DR = [6]int{0, 1, 1, 0, -1, -1}
	DC = [6]int{1, 0, -1, -1, 0, 1}
)

// Player identifie un camp.
type Player int8

const (
	None Player = iota
	NordSud
	EstOuest
)

// String renvoie le nom affichable du joueur.
func (p Player) String() string {
	switch p {
	case NordSud:
		return "Nord–Sud"
	case EstOuest:
		return "Est–Ouest"
	}
	return "aucun"
}

// Other renvoie le joueur adverse.
func (p Player) Other() Player {
	if p == NordSud {
		return EstOuest
	}
	return NordSud
}

// Board est un plateau n x n. La case (r,c) est stockée à l'indice r*n+c.
type Board struct {
	N     int
	Cells []Player
	Moves int // nombre de coups joués
}

// NewBoard crée un plateau vide de côté n.
func NewBoard(n int) Board {
	return Board{N: n, Cells: make([]Player, n*n)}
}

// InBounds indique si (r,c) est sur le plateau.
func (b Board) InBounds(r, c int) bool {
	return r >= 0 && r < b.N && c >= 0 && c < b.N
}

// At renvoie l'occupant de la case (None si vide ou hors plateau).
func (b Board) At(r, c int) Player {
	if !b.InBounds(r, c) {
		return None
	}
	return b.Cells[r*b.N+c]
}

// Set place un pion. Renvoie false si la case est hors plateau ou occupée.
func (b *Board) Set(r, c int, p Player) bool {
	if !b.InBounds(r, c) || b.At(r, c) != None {
		return false
	}
	b.Cells[r*b.N+c] = p
	b.Moves++
	return true
}

// Clear vide une case (utilisé par les tests).
func (b *Board) Clear(r, c int) {
	if b.InBounds(r, c) && b.At(r, c) != None {
		b.Cells[r*b.N+c] = None
		b.Moves--
	}
}

// Neighbors renvoie les cases voisines existantes de (r,c).
//
// Les six directions sont les MÊMES pour toutes les cases : est/ouest sur la
// même ligne, plus quatre diagonales. Aucune n'utilise la parité de la ligne —
// c'est la conséquence directe du losange, où chaque ligne est décalée d'un
// demi-pas dans le MÊME sens.
func (b Board) Neighbors(r, c int) [][2]int {
	out := make([][2]int, 0, 6)
	for k := 0; k < 6; k++ {
		nr, nc := r+DR[k], c+DC[k]
		if b.InBounds(nr, nc) {
			out = append(out, [2]int{nr, nc})
		}
	}
	return out
}

// IsFull indique si toutes les cases sont occupées.
func (b Board) IsFull() bool { return b.Moves == b.N*b.N }

// Winner renvoie le joueur qui a relié ses deux bords, ou None.
func (b Board) Winner() Player {
	if b.HasWon(NordSud) {
		return NordSud
	}
	if b.HasWon(EstOuest) {
		return EstOuest
	}
	return None
}

// HasWon indique si p a relié ses deux bords. Parcours en largeur depuis le
// bord de départ du joueur, à travers ses propres pions.
func (b Board) HasWon(p Player) bool {
	startR, startC, goalR, goalC := 0, 0, 0, 0
	switch p {
	case NordSud:
		// départ : ligne 0 ; arrivée : ligne n-1
		startR, goalR = 0, b.N-1
	default:
		// départ : colonne 0 ; arrivée : colonne n-1
		startC, goalC = 0, b.N-1
	}
	seen := make([]bool, b.N*b.N)
	var queue [][2]int
	push := func(r, c int) {
		if !b.InBounds(r, c) || seen[r*b.N+c] || b.At(r, c) != p {
			return
		}
		seen[r*b.N+c] = true
		queue = append(queue, [2]int{r, c})
	}
	if p == NordSud {
		for c := 0; c < b.N; c++ {
			push(startR, c)
		}
	} else {
		for r := 0; r < b.N; r++ {
			push(r, startC)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		r, c := cur[0], cur[1]
		if (p == NordSud && r == goalR) || (p == EstOuest && c == goalC) {
			return true
		}
		for _, nb := range b.Neighbors(r, c) {
			push(nb[0], nb[1])
		}
	}
	return false
}

// WinningCells renvoie les cases formant la chaîne gagnante de p (de son bord
// de départ à son bord d'arrivée), ou nil si p n'a pas relié ses bords.
func (b Board) WinningCells(p Player) [][2]int {
	startR, startC, goalR, goalC := 0, 0, 0, 0
	switch p {
	case NordSud:
		startR, goalR = 0, b.N-1
	default:
		startC, goalC = 0, b.N-1
	}
	parent := make([]int, b.N*b.N)
	for i := range parent {
		parent[i] = -2 // -2 = jamais vu, -1 = racine
	}
	var queue [][2]int
	push := func(r, c, from int) {
		if !b.InBounds(r, c) {
			return
		}
		i := r*b.N + c
		if parent[i] != -2 || b.At(r, c) != p {
			return
		}
		parent[i] = from
		queue = append(queue, [2]int{r, c})
	}
	if p == NordSud {
		for c := 0; c < b.N; c++ {
			push(startR, c, -1)
		}
	} else {
		for r := 0; r < b.N; r++ {
			push(r, startC, -1)
		}
	}
	goal := -1
	for len(queue) > 0 && goal < 0 {
		cur := queue[0]
		queue = queue[1:]
		r, c := cur[0], cur[1]
		if (p == NordSud && r == goalR) || (p == EstOuest && c == goalC) {
			goal = r*b.N + c
			break
		}
		for _, nb := range b.Neighbors(r, c) {
			push(nb[0], nb[1], r*b.N+c)
		}
	}
	if goal < 0 {
		return nil
	}
	var path [][2]int
	for i := goal; i >= 0; i = parent[i] {
		path = append(path, [2]int{i / b.N, i % b.N})
	}
	return path
}

// String rend le plateau en texte, pour les diagnostics et les tests.
// X : Nord–Sud, O : Est–Ouest, . : case libre.
//
// Chaque ligne est décalée d'un demi-pas (un caractère, la case en faisant
// deux) vers la droite : le losange apparaît, et deux cases voisines se
// touchent réellement à l'écran.
func (b Board) String() string {
	out := make([]rune, 0, b.N*(3*b.N+2))
	for r := 0; r < b.N; r++ {
		for i := 0; i < r; i++ {
			out = append(out, ' ')
		}
		for c := 0; c < b.N; c++ {
			if c > 0 {
				out = append(out, ' ')
			}
			switch b.At(r, c) {
			case NordSud:
				out = append(out, 'X', ' ')
			case EstOuest:
				out = append(out, 'O', ' ')
			default:
				out = append(out, '.', ' ')
			}
		}
		out = append(out, '\n')
	}
	return string(out)
}

// Clone renvoie une copie indépendante du plateau.
func (b Board) Clone() Board {
	out := Board{N: b.N, Cells: make([]Player, len(b.Cells)), Moves: b.Moves}
	copy(out.Cells, b.Cells)
	return out
}
