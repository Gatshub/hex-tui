package ai

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"

	"hex-tui/hex"
)

// Banc d'essai du moteur, lancé à la demande (il joue de vraies parties) :
//
//	HEX_BENCH=1 go test ./ai/ -count=1 -timeout 580s
//
// La comparaison décisive est la section « UCT contre Monte-Carlo plat à
// travail égal » : mêmes simulations rapides, même nombre de parties simulées
// au total, seule la façon de les dépenser change.

type mover func(hex.Board, hex.Player) (int, int)

func engine(rollouts int, seed int64) mover {
	e := New(rollouts, seed)
	return func(b hex.Board, p hex.Player) (int, int) {
		r, c, ok := e.Move(b, p)
		if !ok {
			return -1, -1
		}
		return r, c
	}
}

// engineVC permet de comparer le moteur AVEC et SANS connexions virtuelles,
// toutes choses égales par ailleurs.
func engineVC(rollouts int, seed int64, useVC bool) mover {
	e := newAI(rollouts, seed, useVC)
	return func(b hex.Board, p hex.Player) (int, int) {
		r, c, ok := e.Move(b, p)
		if !ok {
			return -1, -1
		}
		return r, c
	}
}

// flatMover reproduit l'ANCIEN moteur : Monte-Carlo plat, chaque case libre
// recevant le même nombre de simulations. Il tourne sur les mêmes simulations
// rapides que le moteur UCT, pour que la comparaison porte sur l'algorithme et
// non sur la vitesse d'exécution.
func flatMover(perCandidate int, seed int64) mover {
	rng := rand.New(rand.NewSource(seed))
	return func(b hex.Board, p hex.Player) (int, int) {
		n := b.N
		fb := newFastBoard(n)
		fb.load(b)
		me, opp := toStone(p), toStone(p.Other())
		free := fb.freeCells(nil)
		if len(free) == 0 {
			return -1, -1
		}
		scratch := newFastBoard(n)
		order := make([]int32, 0, n*n)

		// gagner tout de suite, sinon bloquer
		for _, idx := range free {
			r, c := int(idx)/n, int(idx)%n
			scratch.copyFrom(fb)
			scratch.place(r, c, me)
			if scratch.connected(me) {
				return r, c
			}
		}
		for _, idx := range free {
			r, c := int(idx)/n, int(idx)%n
			scratch.copyFrom(fb)
			scratch.place(r, c, opp)
			if scratch.connected(opp) {
				return r, c
			}
		}

		best, bestScore := -1, -1
		for _, idx := range free {
			r, c := int(idx)/n, int(idx)%n
			score := 0
			for k := 0; k < perCandidate; k++ {
				scratch.copyFrom(fb)
				scratch.place(r, c, me)
				order = scratch.freeCells(order)
				if scratch.playout(order, opp, rng) == me {
					score++
				}
			}
			if score > bestScore {
				bestScore, best = score, int(idx)
			}
		}
		return best / n, best % n
	}
}

func randomMover(seed int64) mover {
	rng := rand.New(rand.NewSource(seed))
	return func(b hex.Board, p hex.Player) (int, int) {
		var free [][2]int
		for r := 0; r < b.N; r++ {
			for c := 0; c < b.N; c++ {
				if b.At(r, c) == hex.None {
					free = append(free, [2]int{r, c})
				}
			}
		}
		if len(free) == 0 {
			return -1, -1
		}
		pick := free[rng.Intn(len(free))]
		return pick[0], pick[1]
	}
}

func playGame(n int, north, south mover) hex.Player {
	b := hex.NewBoard(n)
	turn := hex.NordSud
	for {
		if w := b.Winner(); w != hex.None {
			return w
		}
		if b.IsFull() {
			return hex.None
		}
		var r, c int
		if turn == hex.NordSud {
			r, c = north(b, turn)
		} else {
			r, c = south(b, turn)
		}
		if r < 0 {
			return hex.None
		}
		b.Set(r, c, turn)
		turn = turn.Other()
	}
}

// duel fait jouer deux adversaires en échangeant les camps à chaque partie,
// pour annuler l'avantage du premier coup (qui, au Hex, est réel).
func duel(n, games int, a, b mover) (aWins, bWins int) {
	for g := 0; g < games; g++ {
		if g%2 == 0 {
			if playGame(n, a, b) == hex.NordSud {
				aWins++
			} else {
				bWins++
			}
		} else {
			if playGame(n, b, a) == hex.EstOuest {
				aWins++
			} else {
				bWins++
			}
		}
	}
	return
}

func TestBenchSpeed(t *testing.T) {
	if os.Getenv("HEX_BENCH") == "" {
		t.Skip("définir HEX_BENCH=1 pour mesurer la vitesse")
	}
	fmt.Println("\n===== coût d'un coup (recherche UCT) =====")
	for _, n := range []int{7, 9, 11, 13} {
		b := hex.NewBoard(n)
		for r := 0; r < n/2; r++ {
			b.Set(r, r, hex.NordSud)
			b.Set(r, r+1, hex.EstOuest)
		}
		fmt.Printf("  plateau %2d x %2d :", n, n)
		for _, budget := range []int{1000, 5000, 20000} {
			e := New(budget, 12345)
			start := time.Now()
			const trials = 3
			for k := 0; k < trials; k++ {
				e.Move(b, hex.NordSud)
			}
			per := time.Since(start) / trials
			fmt.Printf("   %6d sim. → %8s", budget, per.Round(time.Millisecond))
		}
		fmt.Println()
	}
}

// TestBenchLong tranche les deux comparaisons qui comptent, avec assez de
// parties pour que le résultat ait un sens statistique (à 14 parties, l'écart
// type est d'environ 2,5 parties : tout écart plus petit n'est que du bruit) :
//
//	HEX_BENCH_LONG=1 go test ./ai/ -run TestBenchLong -v -count=1 -timeout 580s
func TestBenchLong(t *testing.T) {
	if os.Getenv("HEX_BENCH_LONG") == "" {
		t.Skip("définir HEX_BENCH_LONG=1 pour la comparaison longue")
	}
	const n = 9
	const games = 48
	const budget = 4000
	per := budget / (n * n)

	fmt.Printf("\n===== comparaison longue — plateau %d x %d, %d parties =====\n", n, n, games)
	fmt.Printf("  (à %d parties, l'écart type est de %.1f partie)\n\n", games, math.Sqrt(float64(games))/2)

	a, b := duel(n, games, engine(budget, 11), flatMover(per, 22))
	fmt.Printf("  UCT (%d sim.) contre Monte-Carlo plat (%d/case) : %2d - %2d\n",
		budget, per, a, b)

	c, d := duel(n, games, engineVC(budget, 33, true), engineVC(budget, 44, false))
	fmt.Printf("  connexions virtuelles : avec %2d - sans %2d\n", c, d)

	e, f := duel(n, games, engine(budget, 55), engine(budget/4, 66))
	fmt.Printf("  contrôle — 4x plus de simulations : %2d - %2d\n", e, f)
}

func TestBench(t *testing.T) {
	if os.Getenv("HEX_BENCH") == "" {
		t.Skip("définir HEX_BENCH=1 pour lancer le banc d'essai")
	}

	for _, n := range []int{7, 9} {
		cells := n * n
		fmt.Printf("\n===== plateau %d x %d =====\n", n, n)

		// A. Le moteur contre un joueur au hasard.
		for _, lv := range Levels {
			const games = 14
			wins := 0
			for g := 0; g < games; g++ {
				e := engine(lv.Rollouts, int64(5000+g))
				r := randomMover(int64(9000 + g))
				var w hex.Player
				if g%2 == 0 {
					w = playGame(n, e, r)
				} else {
					w = playGame(n, r, e)
				}
				if (g%2 == 0 && w == hex.NordSud) || (g%2 == 1 && w == hex.EstOuest) {
					wins++
				}
			}
			fmt.Printf("  %-10s (%6d sim.) vs hasard : %2d/%d\n",
				lv.Name, lv.Rollouts, wins, games)
		}

		// B. Les niveaux sont-ils ordonnés ?
		sw, ww := duel(n, 14, engine(Levels[2].Rollouts, 777), engine(Levels[0].Rollouts, 888))
		fmt.Printf("  Difficile (%6d) contre Facile (%6d) : %2d - %2d\n",
			Levels[2].Rollouts, Levels[0].Rollouts, sw, ww)

		// C. La comparaison qui compte : UCT contre Monte-Carlo plat, à
		//    travail égal (même nombre de parties simulées au total).
		const total = 6000
		per := total / cells
		u, f := duel(n, 14, engine(total, 111), flatMover(per, 222))
		fmt.Printf("  UCT (%d sim.) contre plat (%d/case, même total) : %2d - %2d\n",
			total, per, u, f)

		// D. Contrôle : le moteur contre lui-même doit être équilibré.
		x, y := duel(n, 14, engine(2000, 333), engine(2000, 444))
		fmt.Printf("  contrôle (même force) : %2d - %2d\n", x, y)

		// E. Les connexions virtuelles apportent-elles quelque chose ?
		//    Même moteur, même budget, seule la détection de ponts change.
		c, d := duel(n, 14, engineVC(2000, 555, true), engineVC(2000, 666, false))
		fmt.Printf("  avec connexions virtuelles contre sans : %2d - %2d\n", c, d)
	}
}
