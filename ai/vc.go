package ai

import "hex-tui/hex"

// Connexions virtuelles, la connaissance du jeu qui manquait au moteur.
//
// Un PONT relie deux pierres distantes de deux cases lorsque les deux cases
// intermédiaires sont LIBRES. L'idée : si l'adversaire en prend une, je prends
// l'autre, et la liaison est faite — il ne peut donc pas l'empêcher.
//
//      a . v          a et b : mes pierres
//       w . b          v et w : les deux cases du pont, libres
//
// Une chaîne de ponts d'un bord à l'autre est une connexion virtuelle : une
// position où, en répondant correctement, on finit par relier ses bords.
//
// ATTENTION — c'est une APPROXIMATION. La fermeture par ponts ignore le cas où
// un seul coup adverse menace deux ponts à la fois : l'adversaire joue une case
// qui est le « trou » de deux ponts, je ne peux en réparer qu'un. C'est
// précisément ce que traite la vraie recherche H (celle de MoHex). On s'en sert
// donc comme d'un GUIDE (ordonnancement, arrêt anticipé des simulations), et
// non comme d'une preuve — c'est mesuré et documenté dans les tests.

// L'ordre cyclique des six directions est celui de hex.DR / hex.DC : est,
// sud-est, sud-ouest, ouest, nord-ouest, nord-est.
//
// Il est INDISPENSABLE ici : deux directions séparées d'un cran dans cet ordre
// sont à deux cases l'une de l'autre et partagent un voisin commun. C'est
// exactement la définition d'un pont, et le losange rend cet ordre identique
// pour toutes les cases (le tracé en quinconce, lui, le faisait dépendre de la
// parité de la ligne).

// virtBoard entretient les liaison d'un camp : adjacence directe et ponts.
// Les quatre derniers nœuds représentent les bords du plateau.
type virtBoard struct {
	n      int
	parent []int32
	rank   []int32
}

func newVirtBoard(n int) *virtBoard {
	return &virtBoard{
		n:      n,
		parent: make([]int32, n*n+4),
		rank:   make([]int32, n*n+4),
	}
}

func (v *virtBoard) find(x int32) int32 {
	for v.parent[x] != x {
		v.parent[x] = v.parent[v.parent[x]]
		x = v.parent[x]
	}
	return x
}

func (v *virtBoard) union(a, b int32) {
	ra, rb := v.find(a), v.find(b)
	if ra == rb {
		return
	}
	if v.rank[ra] < v.rank[rb] {
		ra, rb = rb, ra
	}
	v.parent[rb] = ra
	if v.rank[ra] == v.rank[rb] {
		v.rank[ra]++
	}
}

// neigh renvoie l'indice d'un voisin en ordre cyclique, ou -1 hors plateau.
func (v *virtBoard) neigh(r, c int, dr, dc *[6]int, k int) int {
	nr, nc := r+dr[k], c+dc[k]
	if nr < 0 || nr >= v.n || nc < 0 || nc >= v.n {
		return -1
	}
	return nr*v.n + nc
}

// build constitue les liaisons du camp p présentes sur le plateau fb.
func (v *virtBoard) build(fb *fastBoard, p uint8) {
	n := fb.n
	for i := range v.parent {
		v.parent[i] = int32(i)
		v.rank[i] = 0
	}
	base := int32(n * n)

	// 1. Adjacence directe et raccordement aux bords.
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			if fb.cells[r*n+c] != p {
				continue
			}
			i := int32(r*n + c)
			for k := 0; k < 6; k++ {
				nr, nc := r+hex.DR[k], c+hex.DC[k]
				if nr < 0 || nr >= n || nc < 0 || nc >= n {
					continue
				}
				if fb.cells[nr*n+nc] == p {
					v.union(i, int32(nr*n+nc))
				}
			}
			if p == nsStone {
				if r == 0 {
					v.union(i, base) // nord
				}
				if r == n-1 {
					v.union(i, base+1) // sud
				}
			} else {
				if c == 0 {
					v.union(i, base+2) // ouest
				}
				if c == n-1 {
					v.union(i, base+3) // est
				}
			}
		}
	}

	// 2. Ponts : deux pierres du camp séparées d'un cran dans l'ordre
	//    cyclique, avec les DEUX cases intermédiaires libres.
	for r := 0; r < n; r++ {
		dr, dc := &hex.DR, &hex.DC
		for c := 0; c < n; c++ {
			if fb.cells[r*n+c] != noStone {
				continue // la case « v » du pont doit être libre
			}
			for k := 0; k < 6; k++ {
				a := v.neigh(r, c, dr, dc, k)
				w := v.neigh(r, c, dr, dc, (k+1)%6)
				b := v.neigh(r, c, dr, dc, (k+2)%6)
				if a < 0 || w < 0 || b < 0 {
					continue
				}
				if fb.cells[w] != noStone {
					continue // l'autre case du pont doit être libre
				}
				if fb.cells[a] == p && fb.cells[b] == p {
					v.union(int32(a), int32(b))
				}
			}
		}
	}
}

// connects indique si le camp p est virtuellement relié de bord à bord.
func (v *virtBoard) connects(p uint8) bool {
	base := int32(v.n * v.n)
	if p == nsStone {
		return v.find(base) == v.find(base+1)
	}
	return v.find(base+2) == v.find(base+3)
}

// sameGroup indique si deux cases sont dans la même liaison (adjacence ou
// chaîne de ponts). Sert surtout aux tests, pour vérifier les ponts eux-mêmes
// indépendamment des bords.
func (v *virtBoard) sameGroup(r1, c1, r2, c2 int) bool {
	return v.find(int32(r1*v.n+c1)) == v.find(int32(r2*v.n+c2))
}
