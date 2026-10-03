package hex

import (
	"math/rand"
	"testing"
)

// Les voisins doivent être symétriques et au nombre de six à l'intérieur.
func TestNeighborsSymmetric(t *testing.T) {
	for n := 3; n <= 7; n++ {
		b := NewBoard(n)
		for r := 0; r < n; r++ {
			for c := 0; c < n; c++ {
				nb := b.Neighbors(r, c)
				if len(nb) > 6 {
					t.Fatalf("n=%d (%d,%d) : %d voisins", n, r, c, len(nb))
				}
				for _, v := range nb {
					found := false
					for _, back := range b.Neighbors(v[0], v[1]) {
						if back[0] == r && back[1] == c {
							found = true
						}
					}
					if !found {
						t.Fatalf("n=%d : (%d,%d) -> (%d,%d) non symétrique",
							n, r, c, v[0], v[1])
					}
				}
			}
		}
	}
	// Cas de référence, plateau 5x5, case intérieure (2,2) : 6 voisins.
	b := NewBoard(5)
	if got := len(b.Neighbors(2, 2)); got != 6 {
		t.Fatalf("(2,2) a %d voisins, attendu 6", got)
	}
	// Une case de coin (0,0) du losange n'a que 2 voisins sur le plateau.
	if got := len(b.Neighbors(0, 0)); got != 2 {
		t.Fatalf("coin (0,0) a %d voisins, attendu 2", got)
	}
}

// Nord-Sud gagne en reliant la ligne 0 à la ligne n-1.
func TestNordSudWins(t *testing.T) {
	b := NewBoard(5)
	for r := 0; r < 5; r++ {
		if !b.Set(r, 0, NordSud) {
			t.Fatal("placement refusé")
		}
	}
	if b.Winner() != NordSud {
		t.Fatalf("gagnant = %v, attendu Nord–Sud", b.Winner())
	}
	if b.HasWon(EstOuest) {
		t.Fatal("Est–Ouest ne doit pas avoir gagné")
	}
}

// Est-Ouest gagne en reliant la colonne 0 à la colonne n-1.
func TestEstOuestWins(t *testing.T) {
	b := NewBoard(5)
	for c := 0; c < 5; c++ {
		b.Set(0, c, EstOuest)
	}
	if b.Winner() != EstOuest {
		t.Fatalf("gagnant = %v, attendu Est–Ouest", b.Winner())
	}
}

// Une chaîne interrompue ne gagne pas.
func TestGapBreaksChain(t *testing.T) {
	b := NewBoard(5)
	for r := 0; r < 5; r++ {
		if r == 2 {
			continue // trou au milieu
		}
		b.Set(r, 0, NordSud)
	}
	if b.HasWon(NordSud) {
		t.Fatal("une chaîne trouée ne doit pas gagner")
	}
}

// Un pion ne peut pas être posé deux fois au même endroit.
func TestNoDoublePlacement(t *testing.T) {
	b := NewBoard(5)
	if !b.Set(1, 1, NordSud) {
		t.Fatal("premier placement refusé")
	}
	if b.Set(1, 1, EstOuest) {
		t.Fatal("un second placement sur une case occupée doit être refusé")
	}
	if b.Moves != 1 {
		t.Fatalf("Moves = %d, attendu 1", b.Moves)
	}
}

// THÉORÈME FONDAMENTAL DU HEX : pas de match nul. Sur un plateau plein, un
// joueur (et un seul) a nécessairement relié ses deux bords.
func TestNoDrawOnFullBoard(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, n := range []int{3, 4, 5, 6, 7} {
		for iter := 0; iter < 40; iter++ {
			b := NewBoard(n)
			// Remplissage aléatoire de toutes les cases.
			cells := make([][2]int, 0, n*n)
			for r := 0; r < n; r++ {
				for c := 0; c < n; c++ {
					cells = append(cells, [2]int{r, c})
				}
			}
			rng.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
			for i, cell := range cells {
				p := NordSud
				if i%2 == 1 {
					p = EstOuest
				}
				b.Set(cell[0], cell[1], p)
			}
			if !b.IsFull() {
				t.Fatalf("n=%d : le plateau devrait être plein", n)
			}
			w := b.Winner()
			if w == None {
				t.Fatalf("n=%d iter=%d : plateau plein sans gagnant (match nul impossible)",
					n, iter)
			}
			// Exactement un gagnant.
			both := b.HasWon(NordSud) && b.HasWon(EstOuest)
			if both {
				t.Fatalf("n=%d iter=%d : les deux joueurs gagnent", n, iter)
			}
		}
	}
}

func TestCloneIndependent(t *testing.T) {
	b := NewBoard(4)
	b.Set(0, 0, NordSud)
	c := b.Clone()
	c.Set(1, 1, EstOuest)
	if b.At(1, 1) != None {
		t.Fatal("le clone partage la mémoire avec l'original")
	}
}
