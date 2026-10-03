package ai

import (
	"math/rand"

	"hex-tui/hex"
)

// Représentation rapide d'un plateau, pour les simulations.
//
// Deux idées font la vitesse :
//
//  1. La détection de victoire est INCRÉMENTALE. Un union-find relie entre
//     elles les pierres d'une même couleur ; quatre nœuds virtuels représentent
//     les bords (nord/sud d'un côté, ouest/est de l'autre). Dès qu'un camp
//     relie ses deux bords, on le sait en temps quasi constant — au lieu de
//     remplir tout le plateau puis de refaire un parcours.
//
//  2. Chaque pierre posée est reliée à ses voisines de même couleur, et au bord
//     si elle y touche. Il n'y a donc jamais de recalcul global.
//
// Une simulation s'arrête ainsi typiquement bien avant le plateau plein, ce qui
// est le principal gain : le coût d'une partie simulée devient proportionnel au
// nombre de coups réellement joués, pas à la taille du plateau.

const (
	noStone uint8 = iota // case vide
	nsStone              // pierre Nord–Sud
	eoStone              // pierre Est–Ouest
)

// L'adjacence est celle du paquet hex (hex.DR / hex.DC) : une seule source de
// vérité pour les règles, le moteur et l'affichage. Auparavant elle était
// recopiée ici, et la version recopiée avait divergé.

// fastBoard est un plateau de jeu optimisé pour les simulations.
type fastBoard struct {
	n      int
	cells  []uint8
	parent []int32
	rank   []int32

	north, south, west, east int32 // nœuds virtuels des quatre bords
}

func newFastBoard(n int) *fastBoard {
	fb := &fastBoard{
		n:      n,
		cells:  make([]uint8, n*n),
		parent: make([]int32, n*n+4),
		rank:   make([]int32, n*n+4),
	}
	fb.north = int32(n * n)
	fb.south = fb.north + 1
	fb.west = fb.north + 2
	fb.east = fb.north + 3
	fb.clear()
	return fb
}

// clear remet le plateau à vide.
func (fb *fastBoard) clear() {
	for i := range fb.cells {
		fb.cells[i] = noStone
	}
	for i := range fb.parent {
		fb.parent[i] = int32(i)
		fb.rank[i] = 0
	}
}

// load installe une position à partir du plateau public.
func (fb *fastBoard) load(b hex.Board) {
	fb.clear()
	for r := 0; r < b.N; r++ {
		for c := 0; c < b.N; c++ {
			switch b.At(r, c) {
			case hex.NordSud:
				fb.place(r, c, nsStone)
			case hex.EstOuest:
				fb.place(r, c, eoStone)
			}
		}
	}
}

// copyFrom recopie une position, union-find compris. C'est l'opération faite
// avant chaque simulation : un simple recopiage de tableaux, sans allocation.
func (fb *fastBoard) copyFrom(src *fastBoard) {
	copy(fb.cells, src.cells)
	copy(fb.parent, src.parent)
	copy(fb.rank, src.rank)
}

func (fb *fastBoard) find(x int32) int32 {
	for fb.parent[x] != x {
		fb.parent[x] = fb.parent[fb.parent[x]] // compression de chemin
		x = fb.parent[x]
	}
	return x
}

func (fb *fastBoard) union(a, b int32) {
	ra, rb := fb.find(a), fb.find(b)
	if ra == rb {
		return
	}
	if fb.rank[ra] < fb.rank[rb] {
		ra, rb = rb, ra
	}
	fb.parent[rb] = ra
	if fb.rank[ra] == fb.rank[rb] {
		fb.rank[ra]++
	}
}

// place pose une pierre et met le union-find à jour.
func (fb *fastBoard) place(r, c int, p uint8) {
	i := int32(r*fb.n + c)
	fb.cells[i] = p

	for k := 0; k < 6; k++ {
		nr, nc := r+hex.DR[k], c+hex.DC[k]
		if nr < 0 || nr >= fb.n || nc < 0 || nc >= fb.n {
			continue
		}
		j := int32(nr*fb.n + nc)
		if fb.cells[j] == p {
			fb.union(i, j)
		}
	}

	// Raccordement au bord du camp.
	if p == nsStone {
		if r == 0 {
			fb.union(i, fb.north)
		}
		if r == fb.n-1 {
			fb.union(i, fb.south)
		}
	} else {
		if c == 0 {
			fb.union(i, fb.west)
		}
		if c == fb.n-1 {
			fb.union(i, fb.east)
		}
	}
}

// connected indique si le camp p a relié ses deux bords.
func (fb *fastBoard) connected(p uint8) bool {
	if p == nsStone {
		return fb.find(fb.north) == fb.find(fb.south)
	}
	return fb.find(fb.west) == fb.find(fb.east)
}

// freeCells liste les cases vides (les indices de cases, pas les coordonnées).
func (fb *fastBoard) freeCells(dst []int32) []int32 {
	dst = dst[:0]
	for i, v := range fb.cells {
		if v == noStone {
			dst = append(dst, int32(i))
		}
	}
	return dst
}

// touches dit si la case idx touche au moins une pierre déjà posée. Sert à la
// politique de simulation : une case isolée est presque toujours un coup perdu.
func (fb *fastBoard) touches(idx int32) bool {
	r, c := int(idx)/fb.n, int(idx)%fb.n
	for k := 0; k < 6; k++ {
		nr, nc := r+hex.DR[k], c+hex.DC[k]
		if nr >= 0 && nr < fb.n && nc >= 0 && nc < fb.n &&
			fb.cells[nr*fb.n+nc] != noStone {
			return true
		}
	}
	return false
}

// playout remplit les cases libres en alternant les coups, et renvoie le
// vainqueur. S'arrête dès qu'un camp relie ses bords.
//
// Deux politiques de tirage :
//
//	sans politique — les cases libres sont mélangées et jouées dans cet ordre.
//	  Chaque case a la même chance, y compris celles qui sont loin de toute
//	  pierre et qui ne peuvent rien décider.
//
//	avec politique — les cases qui TOUCHENT déjà une pierre sont séparées des
//	  autres, et tirées deux fois sur trois. L'idée vient du niveau 300 de la
//	  feuille de route HexWiki (« reconnaître ce qui est connecté et ce qui ne
//	  l'est pas, pour éviter de défendre l'indéfendable ») : un coup au milieu
//	  d'une région vide ne défend ni n'attaque rien.
//
// La partition se fait EN PLACE dans `order` (aucune allocation : c'est la
// boucle la plus chaude du moteur), et les cases proches sont évaluées une
// seule fois, au début de la simulation. C'est une approximation assumée.
func (fb *fastBoard) playout(order []int32, turn uint8, rng *rand.Rand, policy bool) uint8 {
	shuffle := func(s []int32) {
		rng.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
	}

	if !policy {
		shuffle(order)
		for _, idx := range order {
			r, c := int(idx)/fb.n, int(idx)%fb.n
			fb.place(r, c, turn)
			if fb.connected(turn) {
				return turn
			}
			turn = other(turn)
		}
		return noStone
	}

	// Partition en place : les cases proches d'une pierre au début, les
	// isolées ensuite.
	k := 0
	for i := range order {
		if fb.touches(order[i]) {
			order[i], order[k] = order[k], order[i]
			k++
		}
	}
	near, far := order[:k], order[k:]
	shuffle(near)
	shuffle(far)

	for len(near)+len(far) > 0 {
		var idx int32
		// Deux fois sur trois dans les cases proches, et toujours s'il n'en
		// reste plus que là.
		if len(near) > 0 && (len(far) == 0 || rng.Intn(3) != 0) {
			j := rng.Intn(len(near))
			idx = near[j]
			near[j] = near[len(near)-1]
			near = near[:len(near)-1]
		} else {
			j := rng.Intn(len(far))
			idx = far[j]
			far[j] = far[len(far)-1]
			far = far[:len(far)-1]
		}
		r, c := int(idx)/fb.n, int(idx)%fb.n
		fb.place(r, c, turn)
		if fb.connected(turn) {
			return turn
		}
		turn = other(turn)
	}
	return noStone
}

// other renvoie le camp adverse.
func other(p uint8) uint8 { return 3 - p } // 1 <-> 2
