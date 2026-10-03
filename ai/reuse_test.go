package ai

import (
	"testing"

	"hex-tui/hex"
)

// Comment ces tests prouvent-ils la réutilisation de l'arbre, sans instrumenter
// le moteur ? Par la CHAÎNE D'ASCENDANCE des nœuds.
//
// Le moteur retient le nœud du dernier coup joué. Si, au calcul suivant, il
// REPREND son arbre, alors le nouveau nœud retenu descend de l'ancien — on
// remonte de parent en parent jusqu'à le retrouver. Si, au contraire, il
// reconstruit tout, l'ancien nœud est injoignable : c'est un arbre neuf.
//
// C'est une preuve structurelle, pas une supposition.

const reuseBudget = 2000

// jouer fait jouer le moteur puis pose sa pierre sur le plateau.
func jouer(t *testing.T, e *AI, b *hex.Board, p hex.Player) (int, int) {
	t.Helper()
	r, c, ok := e.Move(*b, p)
	if !ok {
		t.Fatalf("le moteur doit jouer (tour %v)", p)
	}
	if !b.Set(r, c, p) {
		t.Fatalf("coup illégal du moteur : (%d,%d) déjà occupée", r, c)
	}
	return r, c
}

// descendDe indique si le nœud nd a l'ancêtre voulu dans sa chaîne de parents.
func descendDe(nd, ancetre *mctsNode) bool {
	for x := nd; x != nil; x = x.parent {
		if x == ancetre {
			return true
		}
	}
	return false
}

func TestTreeIsReusedBetweenMoves(t *testing.T) {
	const n = 9
	e := newAI(reuseBudget, 7, options{reuse: true})
	b := hex.NewBoard(n)

	// 1. Ouverture : le moteur joue le centre sans recherche, donc sans arbre.
	jouer(t, e, &b, hex.NordSud)

	// 2. Premier vrai calcul : l'arbre naît ici.
	jouer(t, e, &b, hex.EstOuest)

	if e.root == nil {
		t.Fatal("l'arbre doit être conservé après une recherche")
	}
	if e.root.move < 0 {
		t.Fatal("la racine conservée doit être le nœud du coup joué")
	}
	if len(e.root.children) == 0 {
		t.Fatal("l'arbre conservé doit avoir des enfants développés")
	}

	// On garde le nœud retenu, et on fait jouer à l'adversaire une réponse que
	// l'arbre a DÉJÀ envisagée — donc un de ses enfants.
	retenu := e.root
	reponse := retenu.children[0].move
	rr, rc := int(reponse)/n, int(reponse)%n
	if !b.Set(rr, rc, hex.NordSud) {
		t.Fatalf("la case (%d,%d) devrait être libre", rr, rc)
	}

	// 3. Nouveau calcul : l'arbre doit être REPRIS.
	e.Move(b, hex.EstOuest)

	if e.root == nil {
		t.Fatal("l'arbre doit rester conservé")
	}
	if !descendDe(e.root, retenu) {
		t.Fatal("l'arbre a été reconstruit : le nouveau nœud ne descend pas " +
			"de celui retenu au coup précédent")
	}
}

func TestTreeIsDiscardedOnDivergence(t *testing.T) {
	const n = 9
	e := newAI(reuseBudget, 11, options{reuse: true})
	b := hex.NewBoard(n)

	jouer(t, e, &b, hex.NordSud)
	jouer(t, e, &b, hex.EstOuest)
	if e.root == nil {
		t.Fatal("l'arbre doit être conservé après une recherche")
	}
	retenu := e.root

	// Le plateau revient à une seule pierre : c'est une ANNULATION, pas une
	// suite de la position. L'arbre ne doit plus être valable.
	arriere := hex.NewBoard(n)
	arriere.Set(4, 4, hex.NordSud)
	jouer(t, e, &arriere, hex.EstOuest)

	if e.root == nil {
		t.Fatal("un nouvel arbre doit être conservé après le calcul")
	}
	if descendDe(e.root, retenu) {
		t.Fatal("l'arbre aurait dû être jeté : le nouveau nœud descend " +
			"encore de celui d'une position abandonnée")
	}
}

// TestReuseSurvivesAWholeGame vérifie qu'une partie entière se déroule sans
// coup illégal ni blocage, arbre réutilisé d'un bout à l'autre.
func TestReuseSurvivesAWholeGame(t *testing.T) {
	const n = 9
	e := newAI(600, 5, options{reuse: true})
	b := hex.NewBoard(n)

	turn := hex.NordSud
	for mv := 0; mv < n*n && b.Winner() == hex.None && !b.IsFull(); mv++ {
		jouer(t, e, &b, turn)
		turn = turn.Other()
	}

	if w := b.Winner(); w == hex.None {
		t.Fatal("une partie complète doit avoir un vainqueur (théorème du Hex)")
	}
	if e.root == nil {
		t.Fatal("l'arbre doit être conservé jusqu'au bout")
	}
}

// TestReuseIsDeterministic vérifie que la réutilisation ne fait pas perdre la
// reproductibilité : deux moteurs de même graine, nourris des mêmes positions,
// jouent exactement les mêmes coups.
func TestReuseIsDeterministic(t *testing.T) {
	const n = 9
	moteurA := newAI(900, 42, options{reuse: true})
	moteurB := newAI(900, 42, options{reuse: true})
	ba, bb := hex.NewBoard(n), hex.NewBoard(n)

	for mv := 0; mv < 12; mv++ {
		p := hex.NordSud
		if mv%2 == 1 {
			p = hex.EstOuest
		}
		ra, ca, oka := moteurA.Move(ba, p)
		rb, cb, okb := moteurB.Move(bb, p)
		if !oka || !okb {
			break
		}
		if ra != rb || ca != cb {
			t.Fatalf("coups divergents au tour %d : (%d,%d) contre (%d,%d)",
				mv, ra, ca, rb, cb)
		}
		ba.Set(ra, ca, p)
		bb.Set(rb, cb, p)
	}
}
