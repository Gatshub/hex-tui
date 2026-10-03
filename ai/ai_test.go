package ai

import (
	"math/rand"
	"testing"

	"hex-tui/hex"
)

// Le moteur doit saisir une victoire immédiate.
func TestTakesImmediateWin(t *testing.T) {
	b := hex.NewBoard(7)
	for r := 0; r < 6; r++ {
		b.Set(r, 0, hex.NordSud)
	}
	m := New(10, 1)
	r, c, ok := m.Move(b, hex.NordSud)
	if !ok {
		t.Fatal("aucun coup proposé")
	}
	if r != 6 || c != 0 {
		t.Fatalf("coup joué (%d,%d), attendu (6,0) pour gagner", r, c)
	}
}

// Le moteur doit empêcher une victoire immédiate de l'adversaire.
// Est–Ouest relie la colonne 0 à la colonne 6 : avec la ligne 0 occupée de la
// colonne 0 à la colonne 5, il gagnerait en jouant (0,6).
func TestBlocksImmediateLoss(t *testing.T) {
	b := hex.NewBoard(7)
	for c := 0; c < 6; c++ {
		b.Set(0, c, hex.EstOuest)
	}
	if !b.Clone().HasWon(hex.EstOuest) {
		// vérification du décor : (0,6) doit bien être un coup gagnant
		bb := b.Clone()
		bb.Set(0, 6, hex.EstOuest)
		if !bb.HasWon(hex.EstOuest) {
			t.Fatal("décor invalide : (0,6) n'est pas un coup gagnant")
		}
	}
	m := New(10, 1)
	r, c, ok := m.Move(b, hex.NordSud)
	if !ok {
		t.Fatal("aucun coup proposé")
	}
	if r != 0 || c != 6 {
		t.Fatalf("coup joué (%d,%d), attendu (0,6) pour bloquer", r, c)
	}
}

// Même graine, mêmes coups : indispensable pour des tests fiables.
func TestDeterministicWithSeed(t *testing.T) {
	b := hex.NewBoard(7)
	b.Set(3, 3, hex.NordSud)
	b.Set(3, 4, hex.EstOuest)

	r1, c1, _ := New(40, 42).Move(b, hex.NordSud)
	r2, c2, _ := New(40, 42).Move(b, hex.NordSud)
	if r1 != r2 || c1 != c2 {
		t.Fatalf("coups différents selon la graine : (%d,%d) puis (%d,%d)", r1, c1, r2, c2)
	}
}

// Un plateau plein ne propose plus de coup.
func TestNoMoveOnFullBoard(t *testing.T) {
	b := hex.NewBoard(3)
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			p := hex.NordSud
			if (r+c)%2 == 1 {
				p = hex.EstOuest
			}
			b.Set(r, c, p)
		}
	}
	if _, _, ok := New(10, 1).Move(b, hex.NordSud); ok {
		t.Fatal("un coup a été proposé sur un plateau plein")
	}
}

// Le moteur doit dominer nettement un joueur purement aléatoire.
func TestBeatsRandom(t *testing.T) {
	const games = 8
	wins := 0
	for g := 0; g < games; g++ {
		rng := rand.New(rand.NewSource(int64(100 + g)))
		b := hex.NewBoard(5)
		engine := New(1500, int64(1000+g)) // budget UCT, pas « par candidat »
		turn := hex.NordSud
		for {
			if b.Winner() != hex.None || b.IsFull() {
				break
			}
			if turn == hex.NordSud {
				r, c, ok := engine.Move(b, turn)
				if !ok {
					break
				}
				b.Set(r, c, turn)
			} else {
				var free [][2]int
				for r := 0; r < 5; r++ {
					for c := 0; c < 5; c++ {
						if b.At(r, c) == hex.None {
							free = append(free, [2]int{r, c})
						}
					}
				}
				if len(free) == 0 {
					break
				}
				pick := free[rng.Intn(len(free))]
				b.Set(pick[0], pick[1], turn)
			}
			turn = turn.Other()
		}
		if b.Winner() == hex.NordSud {
			wins++
		}
	}
	if wins < games-1 {
		t.Fatalf("le moteur n'a gagné que %d parties sur %d contre un joueur aléatoire",
			wins, games)
	}
	t.Logf("moteur : %d victoires sur %d contre un joueur aléatoire", wins, games)
}
