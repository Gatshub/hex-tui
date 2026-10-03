package ai

import (
	"math"
)

// UCT — « Upper Confidence bounds applied to Trees » — remplace le Monte-Carlo
// plat. Au lieu de répartir le même budget sur toutes les cases, on approfondit
// celles qui promettent, sans jamais abandonner complètement les autres
// (compromis exploration / exploitation). C'est le squelette de MoHex.

const ucbC = 1.4142135623730951 // sqrt(2)

type mctsNode struct {
	move   int32 // case jouée pour venir ici (-1 pour la racine)
	player uint8 // camp qui a joué ce coup
	turn   uint8 // camp au trait dans ce nœud
	parent *mctsNode

	children []*mctsNode
	moves    []int32 // cases libres dans cette position
	nextMove int     // prochaine case à essayer

	// theoreticalWin mémorise le camp qui, dans cette position, est relié de
	// bord à bord par une connexion VIRTUELLE (chaîne de ponts). C'est un
	// raccourci de recherche : inutile de simuler une position dont l'issue est
	// déjà connue. Calculé une fois, à la création du nœud.
	//
	// Attention : la fermeture par ponts est une APPROXIMATION (elle ignore le
	// cas où un seul coup adverse menace deux ponts à la fois). Voir vc.go et
	// le test qui fixe le taux d'erreur mesuré.
	theoreticalWin uint8

	visits int
	wins   float64 // victoires du camp « player »
}

// newRootNode crée une racine pour la position décrite par fb, où c'est à turn
// de jouer.
func newRootNode(fb *fastBoard, turn uint8) *mctsNode {
	return &mctsNode{
		move:   -1,
		player: other(turn),
		turn:   turn,
		moves:  fb.freeCells(nil),
	}
}

// child renvoie l'enfant correspondant à un coup, ou nil s'il n'a pas encore
// été développé.
func (nd *mctsNode) child(move int32) *mctsNode {
	for _, ch := range nd.children {
		if ch.move == move {
			return ch
		}
	}
	return nil
}

func (nd *mctsNode) expanded() bool { return nd.nextMove >= len(nd.moves) }

// selectChild applique la formule UCB1.
func (nd *mctsNode) selectChild() *mctsNode {
	logN := math.Log(float64(nd.visits))
	best := nd.children[0]
	bestVal := math.Inf(-1)
	for _, ch := range nd.children {
		// ch.wins compte les victoires du camp « nd.turn », celui qui choisit.
		val := ch.wins/float64(ch.visits) + ucbC*math.Sqrt(logN/float64(ch.visits))
		if val > bestVal {
			bestVal, best = val, ch
		}
	}
	return best
}

// bestMove renvoie le coup le plus visité parmi les enfants de la racine :
// c'est le choix standard, plus robuste qu'un simple meilleur taux.
func (nd *mctsNode) bestMove() int32 {
	best := nd.children[0]
	for _, ch := range nd.children {
		if ch.visits > best.visits {
			best = ch
		}
	}
	return best.move
}

// withoutMove copie la liste des cases libres en retirant celle qui vient
// d'être jouée.
func withoutMove(moves []int32, skip int32) []int32 {
	out := make([]int32, 0, len(moves)-1)
	for _, m := range moves {
		if m != skip {
			out = append(out, m)
		}
	}
	return out
}

// search mène « budget » simulations depuis le nœud root, dont la position est
// décrite par rootFB (c'est à root.turn de jouer), et renvoie l'indice de la
// case choisie (-1 s'il n'y a plus rien à jouer).
//
// L'arbre est MODIFIÉ en place : ses statistiques s'accumulent. C'est ce qui
// permet de le conserver d'un coup à l'autre (voir AI.retain) au lieu de
// repartir de zéro et de gaspiller tout le travail déjà fait.
func (a *AI) search(root *mctsNode, rootFB *fastBoard, budget int) int32 {
	n := rootFB.n
	if len(root.moves) == 0 {
		return -1
	}

	fb := newFastBoard(n)
	order := make([]int32, 0, n*n)
	vc := newVirtBoard(n)

	for i := 0; i < budget; i++ {
		fb.copyFrom(rootFB)
		nd := root
		winner := noStone

		// 1. Descente dans la partie déjà explorée.
		for nd.expanded() && len(nd.children) > 0 {
			nd = nd.selectChild()
			fb.place(int(nd.move)/n, int(nd.move)%n, nd.player)
			if fb.connected(nd.player) {
				winner = nd.player
				break
			}
			if nd.theoreticalWin != noStone {
				winner = nd.theoreticalWin
				break
			}
		}

		// 2. Essai d'un coup neuf.
		if winner == noStone && !nd.expanded() {
			m := nd.moves[nd.nextMove]
			nd.nextMove++
			child := &mctsNode{
				move:   m,
				player: nd.turn,
				turn:   other(nd.turn),
				parent: nd,
				moves:  withoutMove(nd.moves, m),
			}
			nd.children = append(nd.children, child)
			fb.place(int(m)/n, int(m)%n, child.player)
			nd = child

			if fb.connected(child.player) {
				winner = child.player
			} else if a.useVC {
				// Deux raccourcis par connexion virtuelle :
				//  - le camp qui vient de jouer se relie déjà de bord à bord,
				//    par une chaîne de ponts : la partie est gagnée ;
				//  - l'adversaire, qui doit jouer, possède déjà une telle
				//    chaîne : la position est perdue, inutile de la simuler.
				vc.build(fb, child.player)
				if vc.connects(child.player) {
					child.theoreticalWin = child.player
					winner = child.player
					a.VCFires++
				} else {
					vc.build(fb, child.turn)
					if vc.connects(child.turn) {
						child.theoreticalWin = child.turn
						winner = child.turn
						a.VCFires++
					}
				}
			}
		}

		// 3. Simulation aléatoire jusqu'à ce qu'un camp se relie.
		if winner == noStone {
			order = fb.freeCells(order)
			winner = fb.playout(order, nd.turn, a.rng, a.policy)
		}

		// 4. Remontée des résultats.
		for x := nd; x != nil; x = x.parent {
			x.visits++
			if x.player == winner {
				x.wins++
			}
		}
	}
	return root.bestMove()
}
