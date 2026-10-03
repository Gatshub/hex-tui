// Package ai fournit l'adversaire du jeu de Hex.
//
// Le moteur s'appuie sur deux piliers :
//
//  1. Une recherche UCT (Monte-Carlo arborescente) sur des simulations
//     rapides, où la victoire est détectée en cours de remplissage grâce à un
//     union-find incrémental — la partie simulée s'arrête dès qu'un camp relie
//     ses bords, au lieu de remplir tout le plateau.
//
//  2. Deux vérifications exactes et peu coûteuses, faites avant la recherche :
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

// Level décrit la force du moteur : le nombre de simulations par coup.
type Level struct {
	Name     string
	Rollouts int
}

// Levels liste les niveaux proposés, du plus faible au plus fort.
//
// Les valeurs sont des SIMULATIONS UCT, pas des « simulations par case » comme
// dans l'ancien Monte-Carlo plat : la budgétisation n'est pas la même, et
// oublier ce détail a été le premier piège rencontré. Elles sont calibrées sur
// les mesures de vitesse de bench_test.go (recherche UCT avec connexions
// virtuelles) : environ 35 ms, 180 ms et 750 ms par coup sur un plateau 11×11.
var Levels = []Level{
	{Name: "Facile", Rollouts: 1200},
	{Name: "Moyen", Rollouts: 6000},
	{Name: "Difficile", Rollouts: 25000},
}

// AI est le moteur de jeu.
type AI struct {
	rollouts int
	rng      *rand.Rand
	useVC    bool

	// VCFires compte les fois où une connexion virtuelle a effectivement
	// déclenché un raccourci de recherche. Instrumentation : sert à expliquer
	// pourquoi les connexions virtuelles n'apportent rien (elles ne se
	// déclenchent presque jamais dans un arbre de recherche aussi court).
	VCFires int
}

// New crée un moteur. La graine rend le comportement reproductible (utile pour
// les tests) ; en jeu on passe une graine variable.
//
// Les connexions virtuelles sont DÉSACTIVÉES par défaut : mesuré à 48 parties,
// elles n'apportent rien (24-24) tout en coûtant ~25 % de temps. La raison est
// chiffrée dans TestVCFiringRate — le raccourci ne se déclenche que sur 0,23 %
// des nœuds explorés. Passer par newAI(..., true) pour les rallumer.
func New(rollouts int, seed int64) *AI { return newAI(rollouts, seed, false) }

// newAI permet de désactiver les connexions virtuelles. Sert à mesurer leur
// apport réel : c'est la seule raison d'être de ce paramètre.
func newAI(rollouts int, seed int64, useVC bool) *AI {
	if rollouts < 1 {
		rollouts = 1
	}
	return &AI{
		rollouts: rollouts,
		rng:      rand.New(rand.NewSource(seed)),
		useVC:    useVC,
	}
}

func toStone(p hex.Player) uint8 {
	if p == hex.NordSud {
		return nsStone
	}
	return eoStone
}

// Move choisit un coup pour le joueur p. Renvoie ok=false si le plateau est
// plein (plus rien à jouer).
func (a *AI) Move(b hex.Board, p hex.Player) (int, int, bool) {
	if b.IsFull() {
		return 0, 0, false
	}
	n := b.N
	me, opp := toStone(p), toStone(p.Other())

	fb := newFastBoard(n)
	fb.load(b)
	free := fb.freeCells(nil)
	if len(free) == 0 {
		return 0, 0, false
	}
	at := func(idx int32) (int, int) { return int(idx) / n, int(idx) % n }
	scratch := newFastBoard(n)

	// 1. Gagner tout de suite si c'est possible.
	for _, idx := range free {
		r, c := at(idx)
		scratch.copyFrom(fb)
		scratch.place(r, c, me)
		if scratch.connected(me) {
			return r, c, true
		}
	}

	// 2. Sinon, empêcher la victoire immédiate de l'adversaire.
	for _, idx := range free {
		r, c := at(idx)
		scratch.copyFrom(fb)
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

	// 4. Recherche UCT.
	idx := a.search(fb, me, a.rollouts)
	if idx < 0 {
		return 0, 0, false
	}
	r, c := at(idx)
	return r, c, true
}
