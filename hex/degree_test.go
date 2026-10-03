package hex

import (
	"fmt"
	"sort"
	"testing"
)

// TestNeighborDegrees vérifie et DOCUMENTE le nombre de voisins de chaque case.
//
// C'est la réponse à une question légitime : « chaque case n'est-elle reliée
// qu'à 4 cases ? ». Non : les cases INTÉRIEURES en ont 6. Seules les cases de
// bord en ont moins, et c'est inévitable — un bord n'a pas de voisin au-delà.
//
// Ce test documente aussi la FORME du plateau : un losange. Deux signatures
// mesurables la distinguent du tracé en quinconce (rectangle à bords dentelés)
// qui était utilisé avant :
//
//   - les quatre coins sont symétriques deux à deux (2, 3, 3, 2) ;
//   - toutes les cases de bord, hors coins, ont exactement 4 voisins.
//
// Si quelqu'un remet une géométrie en quinconce, ces deux contrôles échouent.
func TestNeighborDegrees(t *testing.T) {
	const n = 9
	b := NewBoard(n)

	counts := map[int]int{}
	degs := map[[2]int]int{}
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			d := len(b.Neighbors(r, c))
			counts[d]++
			degs[[2]int{r, c}] = d
		}
	}

	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	fmt.Printf("répartition des degrés sur un %d x %d :\n", n, n)
	for _, k := range keys {
		fmt.Printf("  %d voisins : %2d cases\n", k, counts[k])
	}

	// Toute case intérieure (hors bord) a exactement 6 voisins : le réseau est
	// bien hexagonal, ce qui est le point important pour la correction du jeu.
	inter := 0
	for r := 1; r < n-1; r++ {
		for c := 1; c < n-1; c++ {
			if d := degs[[2]int{r, c}]; d != 6 {
				t.Errorf("case intérieure (%d,%d) : %d voisins, attendu 6", r, c, d)
			}
			inter++
		}
	}

	// Aucune case n'a plus de 6 voisins : l'adjacence ne fabrique jamais de
	// triangle (un triangle est impossible sur un réseau hexagonal, et sa
	// présence signalerait une adjacence fausse).
	for rc, d := range degs {
		if d > 6 {
			t.Errorf("case (%d,%d) : %d voisins, un réseau hexagonal n'en permet que 6",
				rc[0], rc[1], d)
		}
	}

	// Les quatre coins d'un LOSANGE forment deux paires symétriques : 2, 3, 3, 2.
	// C'est la signature de la forme, et ce qui la distingue du tracé en
	// quinconce (qui donnait 2, 3, 2, 3 : pas de symétrie, bord dentelé).
	coins := map[[2]int]int{
		{0, 0}: 2, {0, n - 1}: 3, {n - 1, 0}: 3, {n - 1, n - 1}: 2,
	}
	for rc, want := range coins {
		if got := degs[rc]; got != want {
			t.Errorf("coin (%d,%d) : %d voisins, attendu %d (losange)",
				rc[0], rc[1], got, want)
		}
	}

	// Toute case de bord (hors coin) a exactement 4 voisins : c'est la marque
	// d'un bord DROIT. Un bord en dents de scie en produirait aussi des 3 et
	// des 5, comme le tracé en quinconce.
	edges := map[int]int{}
	for r := 0; r < n; r++ {
		for c := 0; c < n; c++ {
			if r == 0 || r == n-1 || c == 0 || c == n-1 {
				if (r == 0 || r == n-1) && (c == 0 || c == n-1) {
					continue // coins, déjà vérifiés
				}
				edges[degs[[2]int{r, c}]]++
			}
		}
	}
	for d, count := range edges {
		if d != 4 {
			t.Errorf("%d cases de bord ont %d voisins au lieu de 4 "+
				"(bord non droit)", count, d)
		}
	}

	sum := 0
	for _, d := range degs {
		sum += d
	}
	fmt.Printf("  cases intérieures à 6 voisins : %d sur %d\n", inter, n*n)
	fmt.Printf("  cases de bord à 4 voisins : %d\n", edges[4])
	fmt.Printf("  somme des degrés : %d, soit %d arêtes\n", sum, sum/2)
	fmt.Printf("  coins : 2,3,3,2 — symétriques, donc losange\n")
}
