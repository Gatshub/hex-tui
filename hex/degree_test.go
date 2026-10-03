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
// Ce test documente aussi la conséquence du tracé en QUINCONCE (odd-r) retenu
// ici : la région dessinée est un rectangle à bords DENTELÉS, pas le losange de
// l'échiquier de Hex. Les degrés de bord s'en ressentent (voir la répartition
// affichée : ils devraient être réguliers le long d'un bord droit).
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

	// Les quatre coins : dans un tracé en quinconce, ils ne sont pas
	// symétriques deux à deux (deux en ont 2, deux en ont 3). C'est la
	// signature du bord dentelé. On le VERROUILLE pour que tout changement de
	// géométrie fasse échouer ce test et oblige à se poser la question.
	coins := map[[2]int]int{
		{0, 0}: 2, {0, n - 1}: 3, {n - 1, 0}: 2, {n - 1, n - 1}: 3,
	}
	for rc, want := range coins {
		if got := degs[rc]; got != want {
			t.Errorf("coin (%d,%d) : %d voisins, attendu %d (bord dentelé)",
				rc[0], rc[1], got, want)
		}
	}

	sum := 0
	for _, d := range degs {
		sum += d
	}
	fmt.Printf("  cases intérieures à 6 voisins : %d sur %d\n", inter, n*n)
	fmt.Printf("  somme des degrés : %d, soit %d arêtes\n", sum, sum/2)
	fmt.Printf("  coins : 2,3,2,3 — asymétrie = bord dentelé\n")
}
