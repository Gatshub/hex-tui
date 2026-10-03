package ai

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
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

// engine construit un moteur avec les réglages par défaut (arbre réutilisé).
func engine(rollouts int, seed int64) mover {
	return engineOpts(rollouts, seed, options{reuse: true})
}

// engineOpts permet de comparer deux variantes du moteur toutes choses égales
// par ailleurs. C'est la seule façon honnête de savoir si un mécanisme apporte
// quelque chose.
func engineOpts(rollouts int, seed int64, o options) mover {
	e := newAI(rollouts, seed, o)
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
				if scratch.playout(order, opp, rng, false) == me {
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
			start := time.Now()
			const trials = 3
			for k := 0; k < trials; k++ {
				e := New(budget, 12345) // moteur neuf : arbre à construire
				e.Move(b, hex.NordSud)
			}
			per := time.Since(start) / trials
			fmt.Printf("   %6d sim. → %8s", budget, per.Round(time.Millisecond))
		}
		fmt.Println()
	}

	// Coût réel d'un coup EN COURS DE PARTIE : comme l'arbre est réutilisé, il
	// grossit à chaque coup. C'est ce chiffre-là qui décide de la fluidité.
	fmt.Println("\n===== coup le plus lent d'une partie (11x11) =====")
	for i, lv := range Levels {
		budget := Rollouts(i, 11)
		b := hex.NewBoard(11)
		e := New(budget, 999)
		worst := time.Duration(0)
		for mv := 0; mv < 60 && !b.IsFull(); mv++ {
			p := hex.NordSud
			if mv%2 == 1 {
				p = hex.EstOuest
			}
			t0 := time.Now()
			r, c, ok := e.Move(b, p)
			d := time.Since(t0)
			if !ok {
				break
			}
			if d > worst {
				worst = d
			}
			b.Set(r, c, p)
			if b.Winner() != hex.None {
				break
			}
		}
		fmt.Printf("  %-10s → %8s (arbre accumulé)\n",
			lv.Name, worst.Round(time.Millisecond))
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

	c, d := duel(n, games,
		engineOpts(budget, 33, options{useVC: true, reuse: true}),
		engineOpts(budget, 44, options{useVC: false, reuse: true}))
	fmt.Printf("  connexions virtuelles : avec %2d - sans %2d\n", c, d)

	// LA comparaison qui compte pour la force : réutiliser l'arbre de
	// recherche d'un coup à l'autre, ou le reconstruire à chaque fois ?
	g, h := duel(n, games,
		engineOpts(budget, 77, options{reuse: true}),
		engineOpts(budget, 88, options{reuse: false}))
	fmt.Printf("  arbre réutilisé : avec %2d - sans %2d\n", g, h)

	e, f := duel(n, games, engine(budget, 55), engine(budget/4, 66))
	fmt.Printf("  contrôle — 4x plus de simulations : %2d - %2d\n", e, f)
}

// TestBenchReuse tranche la question de la réutilisation de l'arbre, avec
// assez de parties pour que le résultat ait un sens. À 48 parties l'écart type
// vaut 3,5 : un effet de quelques parties y est invisible.
//
//	HEX_BENCH_REUSE=1 [HEX_BENCH_GAMES=200] go test ./ai/ -run TestBenchReuse -v -count=1
func TestBenchReuse(t *testing.T) {
	if os.Getenv("HEX_BENCH_REUSE") == "" {
		t.Skip("définir HEX_BENCH_REUSE=1 pour mesurer la réutilisation")
	}
	const n = 9
	const budget = 4000
	games := 200
	if v := os.Getenv("HEX_BENCH_GAMES"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			games = k
		}
	}
	fmt.Printf("\n===== réutilisation de l'arbre — %d parties, plateau %d x %d =====\n",
		games, n, n)
	fmt.Printf("  (écart type : %.1f partie ; en dessous, c'est du bruit)\n\n",
		math.Sqrt(float64(games))/2)

	a, b := duel(n, games,
		engineOpts(budget, 101, options{reuse: true}),
		engineOpts(budget, 202, options{reuse: false}))
	fmt.Printf("  arbre réutilisé %2d - reconstruit %2d\n", a, b)
}

func TestBench(t *testing.T) {
	if os.Getenv("HEX_BENCH") == "" {
		t.Skip("définir HEX_BENCH=1 pour lancer le banc d'essai")
	}

	for _, n := range []int{7, 9} {
		cells := n * n
		fmt.Printf("\n===== plateau %d x %d =====\n", n, n)

		// A. Le moteur contre un joueur au hasard.
		for i, lv := range Levels {
			const games = 14
			wins := 0
			for g := 0; g < games; g++ {
				e := engine(Rollouts(i, n), int64(5000+g))
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
				lv.Name, Rollouts(i, n), wins, games)
		}

		// B. Les niveaux sont-ils ordonnés ?
		sw, ww := duel(n, 14, engine(Rollouts(2, n), 777), engine(Rollouts(0, n), 888))
		fmt.Printf("  Difficile (%6d) contre Facile (%6d) : %2d - %2d\n",
			Rollouts(2, n), Rollouts(0, n), sw, ww)

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
		c, d := duel(n, 14,
			engineOpts(2000, 555, options{useVC: true, reuse: true}),
			engineOpts(2000, 666, options{useVC: false, reuse: true}))
		fmt.Printf("  avec connexions virtuelles contre sans : %2d - %2d\n", c, d)

		// F. Réutiliser l'arbre de recherche d'un coup à l'autre.
		g, h := duel(n, 14,
			engineOpts(2000, 777, options{reuse: true}),
			engineOpts(2000, 888, options{reuse: false}))
		fmt.Printf("  arbre réutilisé contre reconstruit : %2d - %2d\n", g, h)
	}
}

// TestBenchPolicy tranche la question de la POLITIQUE de simulation : les
// parties aléatoires tirent-elles mieux en préférant les cases qui touchent
// déjà une pierre, au lieu de jouer uniformément ?
//
// C'est le levier le plus prometteur de la feuille de route HexWiki (niveau
// 300), et le seul qui touche CHAQUE simulation — contrairement aux connexions
// virtuelles, qui ne se déclenchent que sur 0,3 % des nœuds.
//
//	HEX_BENCH_POLICY=1 [HEX_BENCH_GAMES=200] go test ./ai/ -run TestBenchPolicy -v -count=1 -timeout 580s
func TestBenchPolicy(t *testing.T) {
	if os.Getenv("HEX_BENCH_POLICY") == "" {
		t.Skip("définir HEX_BENCH_POLICY=1 pour mesurer la politique")
	}
	const n = 9
	const budget = 4000
	games := 200
	if v := os.Getenv("HEX_BENCH_GAMES"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			games = k
		}
	}
	fmt.Printf("\n===== politique de simulation — %d parties, plateau %d x %d =====\n",
		games, n, n)
	fmt.Printf("  (écart type : %.1f partie ; en dessous, c'est du bruit)\n\n",
		math.Sqrt(float64(games))/2)

	a, b := duel(n, games,
		engineOpts(budget, 303, options{usePolicy: true}),
		engineOpts(budget, 404, options{usePolicy: false}))
	fmt.Printf("  politique locale %2d - tirage uniforme %2d\n", a, b)
}

// TestBenchBudget chiffre enfin ce que coûte ET ce que rapporte le seul levier
// dont l'effet soit établi. Le projet savait qu'un QUADRUplement donne 46-2,
// mais pas ce qu'un facteur modeste donne — or c'est le facteur modeste qui est
// réellement jouable (voir les temps dans TestBenchSpeed).
//
// Deux précautions, apprises à la première tentative :
//
//   - le duel se joue à 4 000 et 6 000 simulations absolues, pas aux budgets
//     réels des niveaux (40 000 à 60 000) : à ces budgets, 200 parties
//     dépassent le délai d'un test. Le RAPPORT testé est le même (1,5), ce qui
//     est la grandeur qui compte ;
//
//   - mais un rapport mesuré à budget faible surestime le gain à budget élevé,
//     où la courbe s'aplatit. La mesure est donc un MAJORANT du bénéfice réel
//     du passage à 750 par case. Si elle ne conclut déjà pas ici, elle ne
//     conclura pas davantage aux budgets réels.
//
//     HEX_BENCH_BUDGET=1 [HEX_BENCH_GAMES=100] go test ./ai/ -run TestBenchBudget -v -count=1 -timeout 580s
func TestBenchBudget(t *testing.T) {
	if os.Getenv("HEX_BENCH_BUDGET") == "" {
		t.Skip("définir HEX_BENCH_BUDGET=1 pour mesurer l'effet du budget")
	}
	const n = 9
	games := 100
	if v := os.Getenv("HEX_BENCH_GAMES"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			games = k
		}
	}
	// Le rapport testé est celui du passage envisagé : 500 → 750 par case.
	const gros = 6000
	const petit = 4000

	fmt.Printf("\n===== effet du budget (rapport 1,5) — %d parties, plateau %d x %d =====\n",
		games, n, n)
	fmt.Printf("  (écart type : %.1f partie ; en dessous, c'est du bruit)\n", math.Sqrt(float64(games))/2)
	fmt.Printf("  budget faible = MAJORANT du gain réel (la courbe s'aplatit en haut)\n")

	a, b := duel(n, games, engine(gros, 505), engine(petit, 606))
	fmt.Printf("  6000 simulations %2d - 4000 simulations %2d\n", a, b)
}
