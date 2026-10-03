package ai

import (
	"fmt"
	"math/rand"
	"testing"

	"hex-tui/hex"
)

// ------------------------------------------------------------------ ponts --

// Un pont doit relier deux pierres séparées de deux cases quand les deux cases
// intermédiaires sont libres, et cesser de tenir dès que l'une est occupée.
func TestBridgeBasics(t *testing.T) {
	const n = 7
	// (4,4) et (5,5) sont à deux pas ; leurs cases communes sont (4,5) et (5,4).
	b := hex.NewBoard(n)
	b.Set(4, 4, hex.NordSud)
	b.Set(5, 5, hex.NordSud)

	v := newVirtBoard(n)
	fb := newFastBoard(n)

	linked := func(want bool, what string) {
		t.Helper()
		fb.load(b)
		v.build(fb, nsStone)
		if got := v.sameGroup(4, 4, 5, 5); got != want {
			t.Fatalf("%s : (4,4) et (5,5) reliées=%v, attendu %v", what, got, want)
		}
	}

	linked(true, "les deux cases du pont sont libres")

	// L'adversaire prend une case : le pont ne tient plus « tout seul ».
	b.Set(4, 5, hex.EstOuest)
	linked(false, "une case du pont occupée")

	// Les deux : toujours pas.
	b.Set(5, 4, hex.EstOuest)
	linked(false, "les deux cases du pont occupées")

	// Une case du pont reprise par le camp : le pont tient à nouveau.
	b.Clear(4, 5)
	b.Set(4, 5, hex.NordSud)
	linked(true, "une case du pont prise par le camp lui-même")
}

// Deux pierres adjacentes sont reliées, trivialement.
func TestAdjacencyIsConnection(t *testing.T) {
	const n = 6
	b := hex.NewBoard(n)
	b.Set(2, 2, hex.NordSud)
	fb := newFastBoard(n)
	v := newVirtBoard(n)

	fb.load(b)
	v.build(fb, nsStone)
	if v.connects(nsStone) {
		t.Fatal("une pierre seule ne relie pas les bords")
	}

	// relier la colonne entière, de haut en bas, par adjacence (Nord–Sud)
	for r := 0; r < n; r++ {
		b.Set(r, 0, hex.NordSud)
	}
	fb.load(b)
	v.build(fb, nsStone)
	if !v.connects(nsStone) {
		t.Fatal("une colonne pleine doit relier nord et sud")
	}
}

// ------------------------------------------- la fermeture est-elle fiable ? --

// solver : valeur exacte d'une position, par recherche exhaustive. Sert de
// juge : si la fermeture par ponts annonce une connexion là où le camp perd
// avec un jeu parfait, alors elle se trompe et on ne peut pas s'en servir comme
// d'une preuve.
type solver struct {
	n    int
	memo map[uint64]uint8
}

func newSolver(n int) *solver { return &solver{n: n, memo: make(map[uint64]uint8)} }

func (s *solver) key(ns, eo uint16, turn uint8) uint64 {
	return uint64(ns) | uint64(eo)<<16 | uint64(turn)<<32
}

func (s *solver) solve(ns, eo uint16, turn uint8) uint8 {
	k := s.key(ns, eo, turn)
	if v, ok := s.memo[k]; ok {
		return v
	}
	fb := newFastBoard(s.n)
	for i := 0; i < s.n*s.n; i++ {
		r, c := i/s.n, i%s.n
		if ns&(1<<uint(i)) != 0 {
			fb.place(r, c, nsStone)
		} else if eo&(1<<uint(i)) != 0 {
			fb.place(r, c, eoStone)
		}
	}
	if fb.connected(nsStone) {
		s.memo[k] = nsStone
		return nsStone
	}
	if fb.connected(eoStone) {
		s.memo[k] = eoStone
		return eoStone
	}
	// Au Hex, aucun match nul : soit le camp au trait gagne, soit il perd.
	res := other(turn)
	for i := 0; i < s.n*s.n; i++ {
		bit := uint16(1) << uint(i)
		if ns&bit != 0 || eo&bit != 0 {
			continue
		}
		nns, neo := ns, eo
		if turn == nsStone {
			nns |= bit
		} else {
			neo |= bit
		}
		if s.solve(nns, neo, other(turn)) == turn {
			res = turn
			break
		}
	}
	s.memo[k] = res
	return res
}

// TestVCFiringRate mesure à quelle fréquence les connexions virtuelles se
// déclenchent réellement PENDANT une recherche. C'est l'explication du résultat
// du banc d'essai : si elles ne se déclenchent presque jamais, elles ne peuvent
// rien apporter — quelle que soit leur justesse.
func TestVCFiringRate(t *testing.T) {
	const n = 9
	const budget = 4000
	e := newAI(budget, 12345, true)
	b := hex.NewBoard(n)
	turn := hex.NordSud
	moves := 0
	for b.Winner() == hex.None && !b.IsFull() {
		r, c, ok := e.Move(b, turn)
		if !ok {
			break
		}
		b.Set(r, c, turn)
		turn = turn.Other()
		moves++
	}
	total := (moves + 1) * budget
	t.Logf("partie de %d coups à %d simulations : %d déclenchements de connexion "+
		"virtuelle sur ~%d nœuds explorés (%.4f %%)",
		moves, budget, e.VCFires, total, 100*float64(e.VCFires)/float64(total))
}

// TestVirtualConnectionIsSound confronte la fermeture par ponts à la valeur
// exacte du jeu, sur des milliers de positions tirées au hasard sur un petit
// plateau.
//
// Résultat mesuré : elle se trompe RAREMENT, mais elle se trompe. Le cas fautif
// est connu — un seul coup adverse menace deux ponts à la fois (il joue une
// case qui est le « trou » de deux ponts de la chaîne, et on ne peut en
// réparer qu'un). C'est précisément ce que résout la vraie recherche H.
//
// Ce test ne cherche donc pas l'absence d'erreur : il FIXE LA LIMITE. Si le
// taux se dégradait (régression), le test échouerait.
func TestVirtualConnectionIsSound(t *testing.T) {
	const n = 4
	const samples = 2000
	const maxErrorRate = 0.03 // 3 % : marge au-dessus du ~0,4 % mesuré
	rng := rand.New(rand.NewSource(20260101))
	s := newSolver(n)
	fb := newFastBoard(n)
	v := newVirtBoard(n)
	board := hex.NewBoard(n)

	announced, wrong, checkable := 0, 0, 0
	var worst string

	for i := 0; i < samples; i++ {
		// Position aléatoire : assez de pierres pour que la recherche exacte
		// reste faisable, mais assez peu pour qu'il reste de la place.
		board = hex.NewBoard(n)
		ns, eo := uint16(0), uint16(0)
		stones := 8 + rng.Intn(4) // 8 à 11 pierres sur 16 cases
		perm := rng.Perm(n * n)
		for k := 0; k < stones; k++ {
			cell := perm[k]
			p := hex.NordSud
			if k%2 == 1 {
				p = hex.EstOuest
			}
			board.Set(cell/n, cell%n, p)
			if p == hex.NordSud {
				ns |= 1 << uint(cell)
			} else {
				eo |= 1 << uint(cell)
			}
		}
		// On ne teste que les positions non encore terminées.
		fb.load(board)
		if fb.connected(nsStone) || fb.connected(eoStone) {
			continue
		}
		checkable++

		turn := nsStone
		if rng.Intn(2) == 0 {
			turn = eoStone
		}
		truth := s.solve(ns, eo, turn)

		for _, p := range []uint8{nsStone, eoStone} {
			v.build(fb, p)
			if !v.connects(p) {
				continue
			}
			announced++
			if truth != p {
				wrong++
				worst = fmt.Sprintf(
					"la fermeture annonce une connexion pour %v, mais la valeur "+
						"exacte donne la victoire à %v (au trait : %v)\n%s",
					p, truth, turn, board.String())
			}
		}
	}

	rate := float64(wrong) / float64(announced)
	t.Logf("positions examinées : %d · connexions annoncées : %d · erreurs : %d (%.2f %%)",
		checkable, announced, wrong, rate*100)

	if announced < 100 {
		t.Fatalf("trop peu de connexions annoncées (%d) pour conclure", announced)
	}
	if rate > maxErrorRate {
		t.Fatalf("taux d'erreur %.2f %% au-dessus de la limite de %.0f %% : %s",
			rate*100, maxErrorRate*100, worst)
	}
	if wrong > 0 {
		t.Logf("exemple d'erreur connue (double pont menacé) :\n%s", worst)
	}
}
