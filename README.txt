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

  Le moteur, et ce qu'il vaut (mesuré, pas supposé) : recherche UCT, coups
  décisifs vérifiés exactement, et budget de simulations proportionnel à la
  surface du plateau. Il bat largement l'ancien algorithme (38-10 à travail
  égal). Le budget est le levier dominant — quatre fois plus de simulations
  donnent 46-2 — c'est donc lui qu'on a augmenté : environ 2 à 3,5 fois plus
  de simulations par coup, selon le niveau et la taille du plateau.

  Deux mécanismes ont été implémentés PUIS mesurés sans apport : les
  connexions virtuelles (24-24, puis 32-16 — non concluant) et la
  réutilisation de l'arbre de recherche (95-105 sur 200 parties). Tous deux
  sont désactivés par défaut, code et mesure conservés.

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

  Les trois niveaux changent le nombre de simulations par coup. Ce budget est
  proportionnel à la SURFACE du plateau, et non fixe : un 13×13 compte 169
  cases contre 49 pour un 7×7, et un budget constant donnait donc presque
  quatre fois MOINS d'attention par case au grand plateau — un jeu nettement
  plus faible à niveau affiché égal.

      Facile        25 sim./case    11×11 :  3 025 sim.   ~35 ms
      Moyen        100 sim./case    11×11 : 12 100 sim.   ~125 ms
      Difficile    500 sim./case    11×11 : 60 500 sim.   ~650 ms
                                    13×13 : 84 500 sim.   ~1,1 s

  La simulation est toujours concluante : sur un plateau plein, le théorème
  du Hex garantit qu'un camp a gagné, donc chaque tirage rend un verdict —
  jamais de « match nul » à gérer.

    ATTENTION à ne pas confondre les budgétisations. L'ancien moteur comptait
    ses simulations PAR CASE CANDIDATE ; l'UCT compte des simulations AU
    TOTAL, et ici le budget est exprimé par case DU PLATEAU. Un même nombre
    veut donc dire tout autre chose. C'est le premier piège rencontré lors de
    la réécriture.

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

  Résultats mesurés (banc d'essai, plateau 9×9, 4 000 simulations par coup) :

    Comparaison                                    parties    résultat
    UCT contre Monte-Carlo plat, à travail égal        48      38 - 10
    Contrôle : quatre fois plus de simulations         48      46 -  2
    Connexions virtuelles : avec / sans                48      24 - 24
    Connexions virtuelles : avec / sans (arbre repris) 48      32 - 16
    Arbre réutilisé / reconstruit                     200      95 - 105

  L'écart type vaut 3,5 partie à 48 parties et 7 à 200 : en dessous, ce n'est
  que du bruit. Lecture honnête : l'UCT est un gain massif et établi ; le
  BUDGET est le levier dominant (46-2), et c'est donc sur lui qu'on agit ; les
  CONNEXIONS VIRTUELLES ne montrent rien de concluant (24-24, puis 32-16, soit
  2,3 σ) ; la RÉUTILISATION DE L'ARBRE est neutre à légèrement négative
  (95-105, moins d'un écart type).

  Ces deux mécanismes sont donc DÉSACTIVÉS par défaut — c'est mesuré, pas
  supposé — mais ils restent dans le code, testés, avec le banc d'essai pour
  refaire la mesure (`newAI(..., options{useVC: true})` et
  `options{reuse: true}`). C'est la différence entre « écarté après
  vérification » et « oublié ».

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
      Damier       effet immédiat (les trois dessinent le même plateau)
      Coordonnées  effet immédiat (A B C… en haut, 1 2 3… sur les côtés)

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

  GRILLE : un caractère de terminal est ~2x plus haut que large. La case
  fait 3 colonnes et le pas 4, donc le DEMI-PAS vaut 2 colonnes. Chaque
  ligne de cases est décalée d'un demi-pas vers la droite par rapport à
  celle du dessus, et le décalage S'ACCUMULE (0, 2, 4, 6…) : c'est ce qui
  dessine le losange, et ce qui fait que les quatre voisins diagonaux se
  touchent vraiment à l'écran.

  Le piège à ne pas refaire : décaler UNE LIGNE SUR DEUX. On obtenait un
  rectangle à bords en dents de scie, où les diagonales ne se touchaient
  plus — l'image ne correspondait plus aux vraies cases voisines, et les
  quatre coins du plateau n'étaient même pas symétriques (2, 3, 2, 3 au
  lieu de 2, 3, 3, 2). TestNeighborDegrees verrouille les deux.

--------------------------------------------------------------------
TROIS DAMIERS, ET LES COORDONNÉES
--------------------------------------------------------------------

  Le damier se choisit dans le menu ET dans le panneau de réglages en
  cours de partie (il change donc sans perdre la partie). Les trois
  dessinent le MÊME plateau : mêmes cases, mêmes voisins, mêmes règles.

    0. Losange             une ligne par case, glyphes ⬢ ⬡ ·
    1. Losange à bordures  bords Est–Ouest matérialisés ( \ et ■ )
    2. Nid d'abeilles      deux lignes par case, l'hexagone est dessiné

  Le nid d'abeilles est le seul où l'on VOIT les six voisins, puisqu'ils
  partagent un côté. C'est la réponse à la question « cette case
  touche-t-elle celle-là ? », qui restait ouverte avec les deux autres :
  y répondre demandait de savoir si l'on était sur une ligne paire ou
  impaire. C'est aussi le plus encombrant (deux lignes par case).

  Les COORDONNÉES (A B C… en haut, 1 2 3… sur les côtés) s'affichent à la
  demande. Les lettres sont alignées sur la première ligne de cases, comme
  sur les diagrammes de HexWiki. Le damier « à bordures » laisse la marge
  vide en face de ses deux lignes de bords : ce ne sont pas des lignes de
  cases, elles n'ont donc pas de numéro.

  Pour voir les six combinaisons sans lancer le jeu :

      HEX_PREVIEW=1 go test ./ui/ -run TestApercuTroisDamiers -v

--------------------------------------------------------------------
FONCTIONNEMENT INTERNE
--------------------------------------------------------------------

  hex/hex.go        Logique pure, sans UI :
                      - Board n x n, cases (ligne, colonne)
                      - les six directions DR/DC, en ordre cyclique, SEULE
                        définition des voisins pour tout le programme
                      - Neighbors() : les 6 voisins du losange
                      - HasWon() / Winner() : parcours en largeur depuis
                        le bord de départ du joueur
                      - WinningCells() : la chaîne gagnante, pour l'affichage
  hex/hex_test.go   Tests : symétrie des voisins, victoires connues, chaîne
                    trouée, double placement, et le THÉORÈME « pas de match
                    nul » vérifié sur des plateaux pleins aléatoires
  hex/degree_test.go  Test qui DOCUMENTE la forme du plateau : les cases
                    intérieures ont 6 voisins, les bords exactement 4, et
                    les quatre coins 2,3,3,2. La symétrie des coins est la
                    signature du losange ; le test échoue si on remet un
                    tracé en quinconce.
  ui/board.go       LES TROIS DAMIERS et les coordonnées. Une seule
                    définition du demi-pas (indent), utilisée par le dessin,
                    les bordures, les coordonnées et le clic de souris.
  ai/board.go       Plateau rapide pour les simulations : union-find
                    incrémental, victoire détectée dès qu'un camp relie
                    ses bords (sans remplir tout le plateau)
  ai/mcts.go        Recherche UCT (sélection UCB1, expansion, simulation,
                    remontée) ; emplacement des connexions virtuelles
  ai/vc.go          Connexions virtuelles : détection des PONTS et
                    fermeture de proche en proche. Approximation mesurée
  ai/ai.go          Interface publique et niveaux : gagner/bloquer,
                    ouverture au centre, budget par case (Rollouts),
                    puis recherche UCT
  ai/ai_test.go     Tests : victoire immédiate saisie, menace adverse
                    bloquée, reproductibilité par graine, plateau plein,
                    et 8 parties gagnées sur 8 contre un joueur aléatoire
  ai/reuse_test.go  Tests de la réutilisation de l'arbre : elle a bien lieu
                    quand la position descend du nœud retenu, et l'arbre est
                    bien jeté en cas de divergence (annulation, nouvelle
                    partie, coup adverse jamais envisagé)
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

  Trois détours, dont deux erreurs. Les noter évite de les refaire.

  1. Le tracé en QUINCONCE (une ligne sur deux décalée) avait été adopté
     parce qu'il « ressemblait » à un réseau hexagonal. Il donnait en fait
     un rectangle à bords dentelés, pas le losange de Hex, et faisait
     dépendre les directions de la parité de la ligne — « nord-ouest » ne
     menait pas au même endroit selon la ligne. Remplacé par le décalage
     PROGRESSIF d'un demi-pas, avec UNE SEULE table de directions
     partagée par les règles, le moteur et l'interface. Il y en avait
     trois, et elles avaient divergé.

  2. Un premier dessin « nid d'abeilles » avait été écarté comme invalide
     (le théorème du pas-de-match-nul échouait). La conclusion était trop
     forte : c'est ce dessin-là qui était faux, pas le nid d'abeilles —
     HexWiki s'en sert pour de vrais diagrammes. Le nid d'abeilles est
     désormais proposé comme damier n° 2, et c'est le seul où l'on voit
     les six voisins, puisqu'ils partagent un côté.

  3. Un dessin de losange décalant d'UNE CASE ENTIÈRE par ligne au lieu
     d'un demi-pas : les diagonales ne se touchaient plus, l'image
     mentait sur les voisins. TestStylesRespectentLaGeometrie mesure le
     décalage sur le rendu réel, pour chaque damier, et interdit que ça
     revienne sans qu'on s'en aperçoive.

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
      But : pendant les simulations, le moteur joue au hasard. Un vrai
      moteur Hex répond aux menaces locales — c'est le principal levier
      de force restant, maintenant que le budget a été augmenté.

      État actuel : UCT + coups décisifs exacts + budget proportionnel à
      la surface. Bat l'ancien moteur 38-10 à travail égal.

      Piste : dans ai/board.go, playout() remplit au hasard. Essayer une
      heuristique de réponse locale (si l'adversaire menace un pont, y
      répondre), puis MESURER au banc d'essai contre la version actuelle.
      La mesure tranche, pas l'intuition : jusqu'ici, deux mécanismes
      « savants » se sont révélés sans apport.

      Mesure : HEX_BENCH=1 go test ./ai/ -run TestBench -count=1

  [ ] Moteur : RAVE (statistiques AMAF)
      But : combiner les statistiques de l'arbre avec celles des
      simulations. C'est LE mécanisme qui fait progresser l'UCT quand
      chaque coup est peu visité — donc précisément aux niveaux Facile et
      Moyen, et sur les grands plateaux.

      Piste : à la racine, compter pour chaque coup les victoires du camp
      gagnant quand ce coup a été joué pendant la simulation (AMAF), puis
      choisir en mélangeant taux réel et taux AMAF.

  [ ] Moteur : restreindre les coups d'ouverture au voisinage
      But : au tout début, une case loin de toute pierre est presque
      toujours un coup perdu. L'écarter (rayon 2 le temps que le plateau
      se garnisse) concentrerait le budget sur quelques cases utiles au
      lieu de le diluer sur 121.

      Piste : filtrer les cases libres de la racine par distance à la
      plus proche pierre, avec repli sur toutes les cases si l'ensemble
      est vide. À MESURER : c'est une heuristique, pas une évidence.

  [ ] Règle du gâteau (pie rule) pour équilibrer la partie
      But : au Hex, le premier joueur gagne avec un jeu correct. Le
      second peut « échanger » les camps après le premier coup.

      Piste : après le premier coup, demander au joueur Est–Ouest s'il
      veut échanger les camps. À activer au menu, comme option.

  [ ] Connexions virtuelles : vraie recherche H
      But : remplacer l'approximation par ponts (0,44 % d'erreur) par un
      calcul exact, puis s'en servir pour GUIDER la simulation plutôt que
      pour couper l'arbre — c'est là qu'elles paient chez MoHex.

  [ ] Moteur : réutilisation de l'arbre, à réévaluer
      Mesurée à 95-105 sur 200 parties (neutre à légèrement négative), donc
      désactivée. Mais la mesure a été faite sur un 9×9 à 4 000 simulations.
      Sur un 13×13 à 84 500 simulations, l'arbre est bien plus profond et le
      calcul bien mieux renseigné : il n'est pas exclu que la conclusion
      s'inverse. À refaire avant de l'enterrer définitivement.