// Package ai fournit l'adversaire du jeu de Hex.
//
// Le moteur s'appuie sur trois piliers :
//
//  1. Une recherche UCT (Monte-Carlo arborescente) sur des simulations
//     rapides, où la victoire est détectée en cours de remplissage grâce à un
//     union-find incrémental — la partie simulée s'arrête dès qu'un camp relie
//     ses bords, au lieu de remplir tout le plateau.
//
//  2. Un arbre de recherche CONSERVÉ d'un coup à l'autre : la réflexion déjà
//     faite n'est pas jetée. C'est le gain de force le plus important, et il
//     ne coûte pas une simulation de plus (voir adopt / retain).
//
//  3. Deux vérifications exactes et peu coûteuses, faites avant la recherche :
//     gagner tout de suite si c'est possible, sinon empêcher la victoire
//     immédiate de l'adversaire.
//
// Modèle : la recherche UCT est le squelette de MoHex, l'état de l'art du Hex
// (UCT + calculs de connexions virtuelles), qui a gagné les tournois 11×11 et
// 13×13 de 2015. Ici les ponts sont détectés (voir vc.go) mais servent de
// guide, pas de preuve — l'approximation est documentée et mesurée.
//
// Rappel utile : sur un plateau PLEIN, le théorème du Hex garantit qu'un camp a
// relié ses bords. Une simulation rend donc toujours un verdict, sans « match
// nul » à gérer.
package ai

import (
	"math/rand"

	"hex-tui/hex"
)

// Level décrit un niveau de difficulté.
//
// Le budget est exprimé PAR CASE, et non en simulations absolues. La raison est
// mesurable : un plateau 13×13 compte 169 cases, un 7×7 en compte 49. À budget
// constant, le grand plateau recevait donc presque quatre fois MOINS
// d'attention par case, et se trouvait bien plus mal joué à niveau affiché égal.
// Le budget suit désormais la surface.
type Level struct {
	Name    string
	PerCell int
}

// Levels liste les niveaux proposés, du plus faible au plus fort.
//
// Les valeurs sont des simulations UCT PAR CASE. Sur un 11×11 (121 cases), cela
// fait 3 025, 12 100 et 60 500 simulations, soit environ 30 ms, 120 ms et
// 600 ms par coup. Sur un 13×13 : 4 225, 16 900 et 84 500, soit environ
// 55 ms, 220 ms et 1,1 s.
//
// À ne pas confondre avec l'ancien Monte-Carlo plat, qui comptait ses
// simulations par case CANDIDATE — le premier piège rencontré.
var Levels = []Level{
	{Name: "Facile", PerCell: 25},
	{Name: "Moyen", PerCell: 100},
	{Name: "Difficile", PerCell: 500},
}

// Rollouts renvoie le budget de simulations d'un niveau sur un plateau de côté
// n (soit n×n cases). Les mesures de bench_test.go servent de garde-fou sur le
// temps de réponse : au-delà d'environ une seconde par coup, augmenter le
// budget n'est plus acceptable pour un jeu au clavier.
func Rollouts(levelIdx, n int) int {
	if levelIdx < 0 || levelIdx >= len(Levels) {
		levelIdx = len(Levels) - 1
	}
	return Levels[levelIdx].PerCell * n * n
}

// options regroupe les réglages internes du moteur. Ils existent pour que le
// banc d'essai puisse comparer deux variantes à budget égal — c'est la seule
// façon honnête de savoir si un mécanisme apporte quelque chose.
type options struct {
	useVC bool // raccourcis par connexions virtuelles (mesurés : sans effet)
	reuse bool // conserver l'arbre de recherche d'un coup à l'autre
}

// AI est le moteur de jeu.
//
// Il est STATEFUL : il conserve son arbre de recherche d'un appel à l'autre.
// Un moteur neuf doit donc être créé pour chaque nouvelle partie (ou laisser
// la détection de divergence faire son travail, voir adopt).
type AI struct {
	rollouts int
	rng      *rand.Rand
	useVC    bool
	reuse    bool

	// Arbre conservé entre deux coups. root est le nœud de la position
	// courante, rootFB sa description (cases + union-find). Les deux valent
	// nil tant qu'aucune recherche n'a eu lieu ou que la position a divergé.
	root   *mctsNode
	rootFB *fastBoard

	// VCFires compte les fois où une connexion virtuelle a effectivement
	// déclenché un raccourci de recherche. Instrumentation : sert à expliquer
	// pourquoi les connexions virtuelles n'apportent rien (elles ne se
	// déclenchent presque jamais dans un arbre de recherche aussi court).
	VCFires int
}

// New crée un moteur. La graine rend le comportement reproductible (utile pour
// les tests) ; en jeu on passe une graine variable.
//
// Deux mécanismes sont implémentés mais DÉSACTIVÉS par défaut, parce qu'ils ont
// été mesurés sans apport :
//
//   - les CONNEXIONS VIRTUELLES (24-24 puis 32-16 à 48 parties, jamais au-delà
//     de 2,3 σ) : elles coûtent ~25 % de temps ;
//   - la RÉUTILISATION DE L'ARBRE (95-105 sur 200 parties, soit moins d'un
//     écart type) : elle coûte un peu de temps et de complexité.
//
// Le code reste en place, testé, avec le banc d'essai pour refaire la mesure :
// c'est ce qui distingue « écarté après vérification » de « oublié ».
// Le levier qui, lui, est solidement établi, c'est le BUDGET de simulations
// (un quadruplement donne 46-2) — voir Rollouts.
func New(rollouts int, seed int64) *AI {
	return newAI(rollouts, seed, options{})
}

// newAI permet de régler les mécanismes internes. Sert à mesurer leur apport
// réel : c'est la seule raison d'être de ce paramètre.
func newAI(rollouts int, seed int64, o options) *AI {
	if rollouts < 1 {
		rollouts = 1
	}
	return &AI{
		rollouts: rollouts,
		rng:      rand.New(rand.NewSource(seed)),
		useVC:    o.useVC,
		reuse:    o.reuse,
	}
}

// SetRollouts change le budget de recherche sans perdre l'arbre déjà construit.
// C'est ce qui permet de changer de niveau EN COURS DE PARTIE sans repartir de
// zéro : la réflexion déjà faite reste valable.
func (a *AI) SetRollouts(n int) {
	if n >= 1 {
		a.rollouts = n
	}
}

func toStone(p hex.Player) uint8 {
	if p == hex.NordSud {
		return nsStone
	}
	return eoStone
}

// oneAddedStone compare deux positions et dit si la seconde s'obtient à partir
// de la première en ajoutant AU PLUS une pierre.
//
//   - renvoie (-1, true)  si les positions sont identiques ;
//   - renvoie (i, true)   si la seule différence est l'ajout de la case i ;
//   - renvoie (-1, false) dans tous les autres cas (retrait, changement de
//     camp, plusieurs ajouts) : c'est une divergence.
//
// C'est le test qui rend la réutilisation sûre : on ne réutilise l'arbre que si
// la position descend bien de celle qu'il décrit.
func oneAddedStone(from, to *fastBoard) (int, bool) {
	added := -1
	for i := range to.cells {
		if to.cells[i] == from.cells[i] {
			continue
		}
		if from.cells[i] != noStone || to.cells[i] == noStone {
			return -1, false
		}
		if added >= 0 {
			return -1, false
		}
		added = i
	}
	return added, true
}

// adopt cherche dans l'arbre conservé le nœud correspondant à la position cur.
// Renvoie le nœud, la description de sa position, et true si trouvé.
//
// Deux cas mènent à une réutilisation :
//   - la position est IDENTIQUE à celle de la racine (on fouille encore la même
//     position, les statistiques s'accumulent) ;
//   - un seul coup a été joué, et c'est l'un des enfants déjà développés.
//
// Tout le reste (annulation, plateau réinitialisé, changement de taille, coup
// adverse jamais envisagé) est une divergence : on repart d'un arbre neuf.
func (a *AI) adopt(cur *fastBoard) (*mctsNode, *fastBoard, bool) {
	if !a.reuse || a.root == nil || a.rootFB == nil || a.rootFB.n != cur.n {
		return nil, nil, false
	}
	added, ok := oneAddedStone(a.rootFB, cur)
	if !ok {
		return nil, nil, false
	}
	if added < 0 {
		return a.root, a.rootFB, true
	}
	if ch := a.root.child(int32(added)); ch != nil {
		return ch, cur, true
	}
	return nil, nil, false
}

// retain conserve le sous-arbre du coup choisi, pour que le prochain appel
// reprenne la réflexion au lieu de la recommencer.
func (a *AI) retain(root *mctsNode, rootFB *fastBoard, move int32) {
	a.root, a.rootFB = nil, nil
	if !a.reuse {
		return
	}
	ch := root.child(move)
	if ch == nil {
		return
	}
	nb := newFastBoard(rootFB.n)
	nb.copyFrom(rootFB)
	nb.place(int(move)/rootFB.n, int(move)%rootFB.n, ch.player)
	a.root, a.rootFB = ch, nb
}

// Move choisit un coup pour le joueur p. Renvoie ok=false si le plateau est
// plein (plus rien à jouer).
func (a *AI) Move(b hex.Board, p hex.Player) (int, int, bool) {
	if b.IsFull() {
		return 0, 0, false
	}
	n := b.N
	me, opp := toStone(p), toStone(p.Other())

	cur := newFastBoard(n)
	cur.load(b)
	free := cur.freeCells(nil)
	if len(free) == 0 {
		return 0, 0, false
	}
	at := func(idx int32) (int, int) { return int(idx) / n, int(idx) % n }
	scratch := newFastBoard(n)

	// 1. Gagner tout de suite si c'est possible.
	for _, idx := range free {
		r, c := at(idx)
		scratch.copyFrom(cur)
		scratch.place(r, c, me)
		if scratch.connected(me) {
			return r, c, true
		}
	}

	// 2. Sinon, empêcher la victoire immédiate de l'adversaire.
	for _, idx := range free {
		r, c := at(idx)
		scratch.copyFrom(cur)
		scratch.place(r, c, opp)
		if scratch.connected(opp) {
			return r, c, true
		}
	}

	// 3. Ouverture : la recherche est trop bruitée sur un plateau vide pour
	//    trancher utilement ; on joue le centre, l'ouverture de référence.
	if b.Moves == 0 {
		m := n / 2
		return m, m, true
	}

	// 4. Recherche UCT, en reprenant l'arbre précédent quand la position
	//    descend de lui.
	root, rootFB, ok := a.adopt(cur)
	if !ok {
		root, rootFB = newRootNode(cur, me), cur
	}
	idx := a.search(root, rootFB, a.rollouts)
	if idx < 0 {
		return 0, 0, false
	}
	a.retain(root, rootFB, idx)

	r, c := at(idx)
	return r, c, true
}
