HEX-TUI — le jeu de Hex pour le terminal
========================================

Écrit en Go avec les bibliothèques Charm (Bubble Tea + Lip Gloss),
en plein écran.

Méthode de travail de ce dossier : voir  ~/Work/METHODE.md

--------------------------------------------------------------------
ÉTAT ACTUEL   (mis à jour le samedi 3 octobre 2026)
--------------------------------------------------------------------

  Fini et utilisable. Rien en chantier, rien de cassé.

    - Règles complètes du Hex sur plateau losange (7 à 13), vérifiées
      par un test de propriété : sur un plateau plein, il y a TOUJOURS
      exactement un vainqueur (théorème du pas-de-match-nul).
    - Deux joueurs au clavier, OU adversaire ordinateur (3 niveaux).
    - Curseur 6 directions, souris, annuler, aide, écran de victoire
      avec la chaîne gagnante mise en évidence.
    - Panneau de réglages EN COURS DE PARTIE (touche « s » ou Échap) :
      on change de niveau ou d'adversaire sans perdre le plateau.
    - `go test ./...` passe : règles (hex/), moteur (ai/), interface (ui/).
    - Binaire installé et fonctionnel (~/.local/bin/hex-tui).

  Le moteur, et ce qu'il vaut (mesuré, pas supposé) : recherche UCT
  avec coups décisifs vérifiés exactement. Il bat largement l'ancien
  algorithme (39-9 à travail égal). Les connexions virtuelles ont été
  implémentées puis MESURÉES : 24-24, aucun apport — elles sont donc
  désactivées par défaut, code et mesure conservés.

  Ce qui n'est PAS fait (voir « À FAIRE / IDÉES » plus bas) :
    - le moteur reste perfectible (politique de simulation guidée) ;
    - pas de règle du gâteau (échange de camps) pour équilibrer
      l'avantage du premier joueur.

  Prochaine étape naturelle : guider la simulation du moteur, ou la
  règle du gâteau.

--------------------------------------------------------------------
LIEUX
--------------------------------------------------------------------

  Sources .......... ~/Work/hex-tui
  Binaire installé .. ~/.local/bin/hex-tui         (dans le PATH)
  Compilation ....... ~/Work/hex-tui/bin/hex-tui   (ignoré par Git)
  Dépôt Git ......... ~/Work/hex-tui/.git          (branche master)
  Dépôt distant ..... https://github.com/Gatshub/hex-tui      (privé)
  Données de jeu .... aucune (parties en mémoire seulement)
  Documentation ..... ~/Work/hex-tui/README.txt (ce fichier)
                      ~/Work/METHODE.md (méthode commune)

--------------------------------------------------------------------
LE JEU
--------------------------------------------------------------------

Le Hex a été inventé par Piet Hein (1942) puis redécouvert par John Nash.
Deux joueurs posent chacun une pierre sur une case libre du losange.
Le premier à relier ses deux bords gagne :

    ⬢  Nord–Sud   relie le bord du HAUT au bord du BAS
    ⬡  Est–Ouest  relie le bord de GAUCHE au bord de DROITE

Règle remarquable : le match nul est IMPOSSIBLE. Un des deux camps
gagne toujours (théorème démontré par Nash). C'est vérifié par un test
automatique sur des milliers de plateaux pleins.

--------------------------------------------------------------------
LANCER
--------------------------------------------------------------------

    hex-tui

Le menu permet de régler trois choses (↑ ↓ pour choisir la ligne,
← → pour changer la valeur) :

    Taille       7 × 7, 9 × 9, 11 × 11 ou 13 × 13
    Adversaire   2 joueurs  ou  Ordinateur
    Niveau       Facile / Moyen / Difficile (si Ordinateur)

--------------------------------------------------------------------
ADVERSAIRE
--------------------------------------------------------------------

  « 2 joueurs » : les deux camps se jouent au clavier, chacun son tour.
  « Ordinateur » : l'ordinateur joue Est–Ouest, vous gardez Nord–Sud —
  et donc l'avantage du premier coup (au Hex, le premier joueur gagne
  théoriquement). Le coup du moteur se calcule sans figer l'affichage,
  avec un message « L'ordinateur réfléchit… ».

  Le moteur procède par recherche UCT (Monte-Carlo arborescente) : au lieu
  de dépenser le même budget sur chaque case, il approfondit celles qui
  promettent. Deux coups décisifs sont vérifiés exactement avant, car ils
  sont peu coûteux :

    1. gagner tout de suite si c'est possible ;
    2. sinon, bloquer la victoire immédiate de l'adversaire.

  Les trois niveaux ne changent que le nombre de simulations par coup
  (1 200 / 6 000 / 25 000), soit ~35 ms, ~180 ms et ~750 ms sur un 11×11.
  La simulation est toujours concluante : sur un plateau plein, le théorème
  du Hex garantit qu'un camp a gagné, donc chaque tirage rend un verdict —
  jamais de « match nul » à gérer.

    ATTENTION à ne pas confondre les deux budgétisations. L'ancien moteur
    comptait des simulations PAR CASE CANDIDATE ; l'UCT compte des
    simulations AU TOTAL. Un même nombre veut donc dire 100 fois moins de
    travail. C'est le premier piège rencontré lors de la réécriture.

--------------------------------------------------------------------
LE MOTEUR : CE QU'IL VAUT, ET COMMENT C'EST MESURÉ
--------------------------------------------------------------------

  L'ancien moteur était un Monte-Carlo PLAT (même budget sur chaque case)
  sur des simulations lentes. Il était faible, et ce n'était pas une
  impression : mesuré, il perdait déjà 5-15 et 4-16 contre son propre
  niveau supérieur.

  Le moteur actuel repose sur deux changements :

    1. des simulations RAPIDES — union-find incrémental, la partie simulée
       s'arrête dès qu'un camp relie ses bords au lieu de remplir tout le
       plateau (ai/board.go) ;
    2. la recherche UCT à la place du Monte-Carlo plat (ai/mcts.go).

  Résultats mesurés (`HEX_BENCH=1`, et `HEX_BENCH_LONG=1` pour la version
  longue à 48 parties, où l'écart type est de 3,5 parties — en dessous,
  ce n'est que du bruit) :

    UCT contre Monte-Carlo plat, à travail égal ....... 39 -  9
    Contrôle : quatre fois plus de simulations ........ 46 -  2
    Connexions virtuelles activées / désactivées ...... 24 - 24

  Lecture honnête : l'UCT est un gain massif et établi. Les CONNEXIONS
  VIRTUELLES, en revanche, n'apportent RIEN dans l'usage qui en est fait
  ici — c'est mesuré, pas supposé. Elles sont donc désactivées par défaut
  (elles coûtent ~25 % de temps pour un gain nul) et le code reste en
  place, avec `newAI(..., true)` pour les rallumer et le banc pour refaire
  la mesure.

  Pourquoi elles ne paient pas — et c'est chiffré, pas supposé : le
  raccourci ne se déclenche que sur 0,23 % des nœuds explorés (382 fois
  sur 164 000 dans TestVCFiringRate). Trop rare pour peser. À cela s'ajoute
  que la détection se trompe dans ~0,44 % des cas : elle apporte donc à peu
  près autant de bruit que d'information. Chez MoHex, les connexions
  virtuelles servent surtout à GUIDER la simulation et à élaguer les coups
  inférieurs — un travail bien plus lourd que le raccourci pris ici.

  Limite connue et mesurée (`TestVirtualConnectionIsSound`) : la fermeture
  par ponts se trompe dans environ 0,44 % des cas. Le cas fautif est
  identifié — un seul coup adverse qui menace deux ponts à la fois. Un test
  verrouille ce taux : s'il se dégradait, la suite échouerait.

  Deux pistes pour aller plus loin, par ordre de rendement :
    - une POLITIQUE de simulation guidée (répondre aux menaces locales au
      lieu de jouer au hasard) : c'est le principal levier des moteurs Hex ;
    - la vraie recherche H, qui calcule des connexions virtuelles exactes
      (et non l'approximation par ponts utilisée ici).

--------------------------------------------------------------------
TOUCHES
--------------------------------------------------------------------

    ← →   ou  h l        ouest / est
    ↑     ou  y          diagonale nord-ouest
    Maj+↑ ou  u          diagonale nord-est
    ↓     ou  n          diagonale sud-est
    Maj+↓ ou  b          diagonale sud-ouest
    Entrée / Espace      poser une pierre
    Retour arrière       annuler le dernier coup
    r                    réinitialiser le plateau
    ?                    afficher / fermer l'aide
    s   ou   Échap       RÉGLAGES — sans quitter la partie
    q                    quitter
    souris               1er clic : viser la case, 2e clic : poser

  Note : sur un plateau en quinconce, les voisins diagonaux changent de
  côté selon la parité de la ligne. Les six directions sont donc bien
  six touches distinctes — c'est le piège classique du Hex.

--------------------------------------------------------------------
CHANGER DE NIVEAU SANS QUITTER LA PARTIE
--------------------------------------------------------------------

  En cours de partie, « s » (ou Échap) ouvre un panneau de réglages.
  Le plateau est CONSERVÉ : on ajuste et on reprend exactement où on en
  était. C'est le seul écran qui permet de changer d'avis sans rejouer.

      ↑ ↓        choisir la ligne
      ← →        changer la valeur
      Entrée     valider (referme et reprend la partie)
      Échap / s  refermer et reprendre

  Ce qu'on peut y changer :

      Taille       agit seulement avec « Nouvelle partie » (un plateau
                   ne peut pas changer de taille en cours de route)
      Adversaire   effet immédiat
      Niveau       effet immédiat — pris en compte dès le prochain
                   coup de l'ordinateur

  Le panneau contient aussi « Nouvelle partie » (nouveau plateau, mêmes
  réglages) et « Menu principal » (quitter la partie — la seule sortie
  explicite).

  Pourquoi Échap n'abandonne plus la partie : avant, il renvoyait au
  menu et faisait perdre le plateau ; une fausse manœuvre coûtait la
  partie. Il ouvre désormais les réglages, et il faut passer par
  « Menu principal » pour sortir volontairement.

--------------------------------------------------------------------
CE QUI EST AFFICHÉ
--------------------------------------------------------------------

  - Plateau en losange, cases en quinconce (lignes impaires décalées).
  - Curseur : fond bleu.
  - ⬢ Nord–Sud en cyan, ⬡ Est–Ouest en ambre.
  - À la victoire, la chaîne gagnante est mise en évidence en vert.
  - Bandeau du bas : à qui le tour, rappel des deux camps, touches.

  GRILLE : un caractère de terminal est ~2x plus haut que large. Le
  plateau utilise donc des cases de 3 colonnes espacées de 4, décalées
  de 2 colonnes une ligne sur deux : c'est ce qui donne le quinconce.

--------------------------------------------------------------------
FONCTIONNEMENT INTERNE
--------------------------------------------------------------------

  hex/hex.go        Logique pure, sans UI :
                      - Board n x n, cases (ligne, colonne)
                      - Neighbors() : les 6 voisins, quinconce standard
                      - HasWon() / Winner() : parcours en largeur depuis
                        le bord de départ du joueur
                      - WinningCells() : la chaîne gagnante, pour l'affichage
  hex/hex_test.go   Tests : symétrie des voisins, victoires connues, chaîne
                    trouée, double placement, et le THÉORÈME « pas de match
                    nul » vérifié sur des plateaux pleins aléatoires
  ai/board.go       Plateau rapide pour les simulations : union-find
                    incrémental, victoire détectée dès qu'un camp relie
                    ses bords (sans remplir tout le plateau)
  ai/mcts.go        Recherche UCT (sélection UCB1, expansion, simulation,
                    remontée) ; emplacement des connexions virtuelles
  ai/vc.go          Connexions virtuelles : détection des PONTS et
                    fermeture de proche en proche. Approximation mesurée
  ai/ai.go          Interface publique : gagner/bloquer, ouverture au
                    centre, puis recherche UCT
  ai/ai_test.go     Tests : victoire immédiate saisie, menace adverse
                    bloquée, reproductibilité par graine, plateau plein,
                    et 8 parties gagnées sur 8 contre un joueur aléatoire
  ai/vc_test.go     Test des ponts, et test qui MESURE et verrouille le
                    taux d'erreur de la fermeture par ponts (~0,44 %),
                    en la comparant à la valeur exacte du jeu par
                    recherche exhaustive sur un petit plateau
  ai/bench_test.go  Banc d'essai : vitesse, niveaux, UCT contre plat,
                    avec/sans connexions virtuelles, et contrôles
  ui/styles.go      Palette (Tokyo Night, adaptative clair/sombre)
  ui/model.go       État, clavier 6 directions, souris, annuler, menu,
                    enchaînement des coups et calcul du moteur à part
  ui/view.go        Rendu du plateau, menu, aide, victoire
  ui/view_test.go   Tests : géométrie du plateau, clic → case, pas de
                    débordement, déplacement diagonal (parité), victoire
  ui/ai_test.go     Tests : le moteur répond et ne fige pas l'interface,
                    silence du moteur à deux joueurs, navigation du menu
  ui/settings_test.go  Tests : changer le niveau en cours de partie garde
                    le plateau intact, Échap n'abandonne plus la partie,
                    le passage à l'ordinateur en cours de jeu le fait jouer,
                    « Nouvelle partie » et « Menu principal »

--------------------------------------------------------------------
HISTORIQUE DE CONCEPTION (pour mémoire)
--------------------------------------------------------------------

  Le premier rendu essayé était le « nid d'abeilles » ASCII classique
  (alvéoles dessinées avec / \ _). Il est très joli, MAIS son adjacence
  n'est pas celle du Hex : dans ce pavage, deux cases d'une même ligne
  ne se touchent pas (une alvéole s'intercale), et le graphe contient des
  triangles — impossible dans un réseau hexagonal.

  Ce n'est pas une impression : le test du théorème « pas de match nul »
  échoue avec ce pavage, et les trois façons d'apparier les bords aussi.

  D'où le choix d'un treillis de pierres en quinconce : moins « nid
  d'abeilles » visuellement, mais l'adjacence est exactement celle du
  vrai Hex, et le théorème est vérifié.

--------------------------------------------------------------------
REBUILD / TESTS
--------------------------------------------------------------------

  Go est fourni par mise (go 1.27.1). Si « go » n'est pas trouvé :

      mise use -g go@1.27.1

  Ensuite :

      cd ~/Work/hex-tui
      make build      # compile -> bin/hex-tui
      make test       # lance tous les tests
      make run        # compile et lance
      make install    # installe dans ~/.local/bin
      make clean

  Aperçu textuel des écrans, sans terminal interactif :

      HEX_PREVIEW=1 go test ./ui/ -run TestPreview -v

--------------------------------------------------------------------
DÉSINSTALLER
--------------------------------------------------------------------

      rm ~/.local/bin/hex-tui
      rm -rf ~/Work/hex-tui           # supprime aussi les sources

--------------------------------------------------------------------
À FAIRE / IDÉES
--------------------------------------------------------------------

  [ ] Moteur : guider la POLITIQUE de simulation
      But : le moteur joue au hasard pendant les simulations. Un vrai
      moteur Hex répond aux menaces locales — c'est le principal levier
      de force restant (bien plus que les connexions virtuelles).

      État actuel : UCT + coups décisifs exacts. Bat l'ancien moteur
      39-9 à travail égal. Les connexions virtuelles sont implémentées
      mais désactivées (mesurées à 24-24, aucun apport).

      Piste : dans ai/board.go, playout() remplit au hasard. Essayer une
      heuristique de réponse locale (si l'adversaire menace un pont, y
      répondre), puis MESURER au banc d'essai contre la version actuelle.

      Mesure à refaire : HEX_BENCH=1 go test ./ai/ -count=1

  [ ] Règle du gâteau (pie rule) pour équilibrer la partie
      But : au Hex, le premier joueur gagne avec un jeu correct. Le
      second peut « échanger » les camps après le premier coup.

      Piste : après le premier coup, demander au joueur Est–Ouest s'il
      veut échanger les camps. À activer au menu, comme option.

  [ ] Connexions virtuelles : vraie recherche H
      But : remplacer l'approximation par ponts (0,44 % d'erreur) par un
      calcul exact, puis s'en servir pour GUIDER la simulation plutôt que
      pour couper l'arbre — c'est là qu'elles paient chez MoHex.