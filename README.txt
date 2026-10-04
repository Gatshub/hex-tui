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

  Un troisième les a rejoints : la POLITIQUE DE SIMULATION. Les parties
  aléatoires peuvent préférer les cases qui touchent déjà une pierre au
  lieu de tirer uniformément (options{usePolicy: true}). Mesure sur
  200 parties à 9 x 9, budget 4 000, écart type 7,1 :

      politique locale 91  -  tirage uniforme 109

  1,27 σ, et du côté défavorable. Ce n'est pas une preuve que la
  politique NUIT, c'est l'absence de preuve qu'elle aide — le seuil du
  projet est 2 σ, et on en est loin. Désactivée par défaut elle aussi.

  Ce que ces trois résultats dessinent, et c'est plus utile que chacun
  pris séparément : ce moteur convertit le TEMPS DE CALCUL en force de
  façon très fiable (quadrupler le budget donne 46-2), et la finesse
  ajoutée pas du tout. Le levier qui marche est le budget ; les
  « mécanismes savants » essayés jusqu'ici ne payent pas. Une piste
  sérieuse est donc d'aller plus vite — rendre la simulation moins chère
  revient à augmenter le budget.

  Ce qui n'est PAS fait (voir « À FAIRE / IDÉES » plus bas, et la note
  STRATEGIE.md pour l'ordre recommandé) :
    - la détection de connexions ne connaît qu'UN template (celui de
      2e ligne de bord) : son déclenchement monte à 0,355 %, encore trop
      rare pour peser ;
    - pas d'élagage (région mustplay, cellules capturées/dominées) ;
    - le moteur reste perfectible (politique de simulation guidée) ;
    - pas de règle du gâteau (échange de camps) pour équilibrer
      l'avantage du premier joueur.

  Prochaine étape naturelle : les templates (STRATEGIE.md, levier A) —
  ils prolongent du code déjà écrit et déjà testé. Puis la politique de
  simulation, qui est le vrai levier de force.

--------------------------------------------------------------------
LIEUX
--------------------------------------------------------------------

  Sources .......... ~/Work/hex-tui
  Binaire installé .. ~/.local/bin/hex-tui         (dans le PATH)
  Compilation ....... ~/Work/hex-tui/bin/hex-tui   (ignoré par Git)
  Dépôt Git ......... ~/Work/hex-tui/.git          (branche master)
  Dépôt distant ..... https://github.com/Gatshub/hex-tui      (privé)
  Données de jeu .... aucune (parties en mémoire seulement)
  Note de moteur .... ~/Work/hex-tui/STRATEGIE.md  (extraction de la
                     feuille de route HexWiki et des leviers de force :
                     quoi tenter, dans quel ordre, et avec quel risque)
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

      Facile        25 sim./case    11×11 :   3 025 sim.   ~35 ms
      Moyen        100 sim./case    11×11 :  12 100 sim.   ~125 ms
      Difficile   2000 sim./case    11×11 : 242 000 sim.   ~7,9 s (pire coup)
                                    13×13 : 338 000 sim.   ~11 s (estimé)

  POURQUOI DIFFICILE VAUT 2000 (il valait 750). Parce que c'est le seul levier
  dont l'effet soit mesuré, et que la mesure est concluante. Duel sur 7×7,
  mêmes graines des deux côtés :

      200 parties :  2000 par case  112  -  750 par case   88     (1,7 σ)
      400 parties :  2000 par case  225  -  750 par case  175     (2,5 σ)

  Soit 56 % de victoires pour le nouveau budget, au-dessus du seuil de 2 σ que
  ce projet s'impose. C'est le PREMIER mécanisme du projet dont la mesure
  établisse qu'il rend le moteur plus fort : les trois autres (connexions
  virtuelles, réutilisation d'arbre, politique de simulation) ont tous été
  mesurés sans apport et désactivés.

  POURQUOI PAS DAVANTAGE. La courbe s'aplatit, et on peut le montrer : un
  rapport de 1,5 donnait 75-25 quand les budgets étaient faibles (4 000 contre
  6 000 simulations), alors qu'un rapport de 2,67 n'en donne plus que 56 %
  maintenant que les budgets sont élevés. Chaque doublement coûte de plus en
  plus cher en temps pour de moins en moins de force. 2000 est le point où le
  jeu reste jouable.

  LE PRIX, ET IL EST RÉEL. Le projet s'était fixé de rester sous environ une
  seconde par coup ; Difficile la dépasse largement (7,9 s sur le coup le plus
  lent d'un 11×11, mesuré par TestBenchSpeed). C'est un arbitrage ASSUMÉ, pas
  un oubli : le temps d'attente est le prix de la seule force que l'on sache
  acheter ici. Facile et Moyen sont INCHANGÉS — ils restent rapides.

  Attention au 13×13 : 338 000 simulations par coup, une dizaine de secondes
  sur les coups les plus lents. Qui veut jouer vite choisit Moyen.

  Et c'est ce qui désigne la suite. Le coût d'une simulation a été mesuré à
  ~28 µs, contre ~10 µs dans les chiffres historiques du projet : les temps
  annoncés autrefois étaient optimistes. Aller plus vite rendrait le budget
  moins cher, donc le moteur plus fort à temps égal — la vitesse est le
  levier derrière le levier.

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
  raccourci ne se déclenche que sur une fraction infime des nœuds
  explorés. Trop rare pour peser. À cela s'ajoute que la détection se
  trompe dans quelques cas : elle apporte donc à peu près autant de bruit
  que d'information. Chez MoHex, les connexions virtuelles servent surtout
  à GUIDER la simulation et à élaguer les coups inférieurs — un travail
  bien plus lourd que le raccourci pris ici.

  TEMPLATE DE BORD (ajouté depuis). La fermeture ne reliait au bord que
  les pierres POSÉES sur la ligne de bord. Elle connaît maintenant le
  « 2nd row edge template » de HexWiki : une pierre sur la DEUXIÈME ligne
  d'un bord, dont les deux voisines de la ligne de bord sont libres, est
  connectée à ce bord — l'adversaire en occupe une, on prend l'autre.

  Mesures, à positions IDENTIQUES (1 788 positions, recherche exhaustive) :

                          sans le template   avec
    connexions annoncées       229           516
    erreurs                      3             3
    taux d'erreur             1,31 %        0,58 %
    déclenchement             0,10 %        0,355 %

  Autrement dit : deux fois plus de connexions reconnues, sans une seule
  erreur de plus — le taux d'erreur est divisé par deux, et ce sont
  TOUJOURS LES MÊMES 3 erreurs (le cas connu du double pont menacé). Le
  déclenchement au moins triple. C'est une amélioration réelle du
  mécanisme, mais elle reste rare : un seul template ne suffira pas à
  faire payer les connexions virtuelles, il en faudra plusieurs. Le
  ziggurat (3e ligne) est le suivant sur la liste.

  Limite connue et mesurée (`TestVirtualConnectionIsSound`) : les 3
  erreurs restantes viennent d'un seul cas — un seul coup adverse qui
  menace deux ponts à la fois. Un test verrouille ce taux : s'il se
  dégradait, la suite échouerait.

  Deux pistes pour aller plus loin, par ordre de rendement :
    - une POLITIQUE de simulation guidée (répondre aux menaces locales au
      lieu de jouer au hasard) : c'est le principal levier des moteurs Hex ;
    - la vraie recherche H, qui calcule des connexions virtuelles exactes
      (et non l'approximation par ponts utilisée ici).

--------------------------------------------------------------------
TOUCHES
--------------------------------------------------------------------

    ← →   ou  h l        ouest / est
    ↑     ou  k  ou  y   diagonale nord-ouest
    ↓     ou  j  ou  n   diagonale sud-est
    Maj+↑ ou  u          diagonale nord-est
    Maj+↓ ou  b          diagonale sud-ouest
    Entrée / Espace      poser une pierre
    Retour arrière       annuler le dernier coup
    r                    réinitialiser le plateau
    ?                    afficher / fermer l'aide
    s   ou   Échap       RÉGLAGES — sans quitter la partie
    q                    quitter
    souris               1er clic : viser la case, 2e clic : poser

  À la FIN d'une partie, l'écran change de touches :

    Entrée / Espace      rejouer, à la taille choisie
    s                    RÉGLAGES — le même panneau qu'en cours de partie
    Échap                revenir au menu principal
    q                    quitter

  L'écran de fin ANNONCE ces quatre touches, et les gère toutes les quatre.
  Ce n'était pas le cas de « s », qui était affiché sans rien faire.

  j et k sont les pendants de ↓ et ↑, comme au menu et comme dans vi.
  Attention : ce n'est PAS le hjkl complet de vi — sur un réseau hexagonal
  il y a six directions, et h j k l n'en couvrent que quatre. Les deux
  autres (nord-est, sud-ouest) n'ont pas d'équivalent au clavier : il faut
  Maj+↑ / Maj+↓, ou u et b.

  (j et k avaient été ANNONCÉS dans l'aide sans être implémentés : ils ne
  faisaient rien. Puis, dans la foulée, MÊME écart pour « s » : l'écran de
  FIN DE PARTIE affichait « s réglages » alors que la touche ne faisait rien.
  Le motif n'est donc pas anecdotique — « l'aide promet, le code ne tient
  pas » — et il méritait mieux qu'un second correctif ponctuel.)

  Deux tests, deux rôles :

    - TestTouchesDeDeplacement énumère les touches de déplacement et exige
      que chacune bouge le curseur de la case attendue. Sa liste est ÉCRITE
      DANS LE TEST : c'est un filet contre la régression de ces six
      directions-là, pas contre l'ajout d'une touche annoncée ailleurs.

    - TestTouchesAnnonceesALaFin LIT la liste que l'écran affiche
      (annoncesFin, ui/view.go) et exige que chaque touche annoncée fasse
      quelque chose — et qu'une attente soit déclarée pour elle. Ajouter une
      annonce sans la câbler fait donc échouer le test. C'est ce contrôle-là
      qui manquait, et c'est celui que la première version de ce paragraphe
      croyait déjà avoir.)

  Note : depuis le passage au losange, les six directions sont les MÊMES
  sur toutes les lignes. Il n'y a plus à se demander si l'on est sur une
  ligne paire ou impaire avant de savoir où mène une diagonale — c'était
  le piège du tracé en quinconce, et il a disparu.

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
      Damier       effet immédiat (les quatre dessinent le même plateau)
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

  - Plateau en LOSANGE, chaque ligne décalée d'un demi-pas (jamais en
    quinconce : le décalage s'accumule, il ne revient pas en arrière).
  - Quatre damiers au choix, et coordonnées affichables (voir plus bas).
  - Curseur : fond bleu.
  - ⬢ Nord–Sud en cyan, ⬡ Est–Ouest en ambre.
  - À la victoire, la chaîne gagnante est mise en évidence en vert.
  - Bandeau du bas : à qui le tour, rappel des deux camps, touches.

  GRILLE : un caractère de terminal est ~2x plus haut que large. La case
  fait UNE colonne et le pas 2, donc le DEMI-PAS vaut 1 colonne. Chaque
  ligne de cases est décalée d'un demi-pas vers la droite par rapport à
  celle du dessus, et le décalage S'ACCUMULE (0, 1, 2, 3…) : c'est ce qui
  dessine le losange.

  Pourquoi si serré : sur un réseau hexagonal, deux cases diagonales sont
  à 60°, pas à 45°. Avec un pas de 4 colonnes, la diagonale tombait à
  45° — l'œil ne voyait plus six voisins mais quatre, les deux cases
  diagonales semblant trop loin. Il faut donc qu'une diagonale avance
  d'UNE colonne pour UNE ligne (l'écran compensant le reste par la
  forme allongée du caractère). Resserrer n'est pas qu'une question de
  place gagnée : c'est ce qui rend l'adjacence visible.

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
    1. Losange à bordures  les QUATRE bords matérialisés ( \ et ■ )
    2. Nid d'abeilles      deux lignes par case, l'hexagone est dessiné
    3. Losange de hexagones  l'hexagone, mais une DIAGONALE par ligne

  Les trois premiers montrent le plateau d'aplomb — une RANGÉE par ligne. Le
  quatrième le montre TOURNÉ : chaque ligne du dessin est une diagonale du
  plateau (r-q constant), et une diagonale compte n-|d| cases. D'où les lignes
  de 1, 2, … n, … 2, 1 cases, et la silhouette en losange allongé. C'est le
  damier n° 2 de HexWiki (Unicode_and_ASCII_boards), demandé par le joueur.

  Le plateau est le MÊME, et c'est vérifié : la somme des cases des diagonales
  vaut n² (TestLosangeDeHexagones). Seule la vue change — mêmes cases, mêmes
  voisins, mêmes règles.

  Dans ce damier, deux cases voisines DANS une ligne sont séparées par un
  « __ » : dans cette orientation, leur côté partagé est horizontal, alors
  qu'il est vertical dans le nid d'abeilles. Le pas d'une alvéole est donc de
  6 colonnes (4 + 2), et la ligne suivante est décalée de la moitié, 3.

  ON POSE LES CARACTÈRES SUR UN CANEVAS, et c'est le cœur du tracé. Une ligne
  du dessin porte DEUX moitiés de cases : celles de la diagonale qui monte
  (leur « /x \ ») et celles de la diagonale du dessus (leur « \__/ »). Les deux
  se recouvrent partiellement, et là où elles se recouvrent elles posent le
  MÊME caractère — le « \ » d'une case EST le « \ » de sa voisine.

  Deux bords sont posés à part : la ligne 0, le bord SUPÉRIEUR de la première
  diagonale — sans elle le sommet du losange reste béant —, et le « __ » à
  gauche de la première case et à droite de la dernière. Aux deux POINTES du
  losange, celui de gauche n'a pas la place : il est omis, exactement comme
  dans la référence.

  TROIS PIÈGES, tous payés, et tous désormais tenus par un test :

    - grouper les cases par r-q au lieu de r+q. Deux cases voisines dans la
      liste diffèrent alors de (1,+1), qui n'est PAS une direction du réseau
      (hex.DR/hex.DC) : elles sont à DEUX cases l'une de l'autre, le « __ »
      entre elles ne sépare rien, et le pourtour s'ouvre. Le test exige
      maintenant que deux cases rangées côte à côte soient VOISINES.
    - poser les glyphes ⬢ et ⬡ dans le canevas. Ils font deux colonnes pour UN
      rune : le canevas, indexé par colonne, partait de travers. Il ne porte
      plus qu'un repère d'une colonne, et c'est le style qui occupe les deux à
      l'émission.
    - glisser les séquences ANSI dans le canevas : chaque échappement y devient
      un caractère. Géométrie et style sont séparés — le canevas est nu, le
      style est appliqué en sortie.

  Ce n'est PAS le même problème que sur le nid d'abeilles, où la place du bord
  supérieur est réellement prise par les « \ » et « / » du bas de l'alvéole :
  là-bas il faudrait une troisième ligne, ici il fallait juste poser les deux
  moitiés ensemble.

    ATTENTION aux coordonnées : le losange de hexagones N'EN MONTRE PAS. Ses
    lignes sont des diagonales, donc ni « la ligne 5 » ni « la colonne C » ne
    désignent une ligne du dessin. Les poser le long des bords est possible
    mais n'est pas fait ; en attendant, mieux vaut ne rien montrer qu'une
    marge qui ment. Le réglage « Coordonnées » est donc sans effet dans ce
    damier-là, et c'est dit.

    ATTENTION aux FLÈCHES : dans ce damier, gauche et droite suivent l'ÉCRAN,
    pas le plateau. La voisine par « est » (r+1, c+1) s'y dessine en effet à
    GAUCHE — suivre les directions du plateau inverserait les deux flèches,
    et c'est exactement le défaut que le joueur a vu. C'est horizDir() qui
    porte la règle. Le haut et le bas, eux, gardent les directions du plateau :
    elles y sont déjà « en haut à gauche » et « en bas à droite », la meilleure
    approximation de la verticale que la grille permette.

    LES QUATRE BORDS SONT MARQUÉS à leur milieu — N et O sur la ligne du haut,
    E et S sur celle du bas, aux couleurs des deux camps. Sans ces marques, on
    ne sait pas quel camp relie quoi : le plateau est montré TOURNÉ, et les
    coordonnées ne sont pas affichées. La référence marque de même, aux mêmes
    endroits (ses X et 0).

    Où sont les bords d'une diagonale : ses PREMIÈRES cases sont la ligne r=0
    (Nord) puis la colonne c=n-1 (Est) ; ses DERNIÈRES, la colonne c=0 (Ouest)
    puis la ligne r=n-1 (Sud). Deux lignes suffisent donc à porter les quatre
    marques. Chaque marque est posée JUSTE AVANT le premier caractère dessiné
    de sa ligne, et JUSTE APRÈS le dernier : une colonne calculée écrasait le
    « \ » de la diagonale du dessus dans la moitié basse, qui déborde d'un
    caractère de plus que la haute.

  Dans le damier n° 1, les deux bords HORIZONTAUX (haut et bas) sont les
  bords NORD et SUD — ce sont ceux que relie le camp NordSud (r=0 et
  r=n-1) — et ils sont tracés en lignes pleines. Les deux bords penchés
  sont EST et OUEST, tracés avec des « \ ».

  ATTENTION à la longueur de ces lignes pleines : un losange n'a pas de
  bord horizontal, ses QUATRE côtés sont penchés. Aucun bord ne peut donc
  couvrir toute la largeur d'une ligne de cases. Chacun fait UN CRAN DE
  MOINS que sa ligne, et se rogne du côté EXTÉRIEUR :

      Nord   commence avec sa ligne, s'arrête un cran avant la fin
      Sud    commence un cran après le début, finit avec sa ligne

  C'est le tracé du damier compact à bordures de HexWiki, relevé colonne
  par colonne (sa ligne 1 va de 10 à 34, sa bordure Nord de 10 à 33).
  TestBorduresNordSud verrouille la longueur ET la position, mesurées sur
  le rendu réel : deux corrections successives sur ce seul détail ont
  montré que l'œil ne suffisait pas à trancher.

  Le nid d'abeilles est le seul où l'on VOIT les six voisins, puisqu'ils
  partagent un côté. C'est la réponse à la question « cette case
  touche-t-elle celle-là ? », qui restait ouverte avec les deux autres :
  y répondre demandait de savoir si l'on était sur une ligne paire ou
  impaire. C'est aussi le plus encombrant (deux lignes par case).

  SA GÉOMÉTRIE EST À PART, et c'est le piège de ce fichier. Une alvéole
  fait QUATRE colonnes, contre une seule pour une case de losange :

      d'une rangée à la suivante : 2 colonnes, la MOITIÉ d'une alvéole
      — et non 1, sinon les alvéoles se chevauchent de trois colonnes
      sur quatre et le dessin n'est plus un nid d'abeilles ;
      les deux lignes d'une rangée partent de la MÊME colonne.

  Ce pas de 1 était celui du losange, appliqué par mégarde au nid
  d'abeilles : les rangées s'empilaient, et les numéros de ligne
  dérivaient vers la gauche au fil des rangées. D'où rowStep() et
  rowIndent(), qui répondent selon le damier.

  LA SOURIS SUIT LA MÊME GÉOMÉTRIE (cellGeom), et c'est pour la même
  raison : elle était restée sur le pas du losange, donc un clic sur une
  alvéole tombait une case trop loin — et le dernier de chaque rangée
  n'était même plus atteignable. TestCellAtRoundTrip part maintenant des
  cases TELLES QU'ELLES SONT DESSINÉES, dans les trois damiers d'aplomb (le
  losange de hexagones, dont les lignes sont des diagonales, a son propre
  test).

  Les COORDONNÉES s'affichent à la demande : les lettres (A B C…) en haut,
  alignées sur la première ligne de cases, et les numéros de ligne (1 2 3…)
  À GAUCHE SEULEMENT — à droite ils faisaient doublon et encombraient la
  marge du losange.

  L'en-tête des lettres dépend LUI AUSSI du damier. L'ancien calcul, écrit
  pour le seul losange, tassait les lettres deux fois trop serré sur le
  nid d'abeilles (deux colonnes d'écart pour des alvéoles de quatre) et
  les décalait de deux colonnes vers la gauche sur le damier à bordures,
  à cause des « \ » qui précèdent les cases. TestEnteteAligneSurLesCases
  mesure l'aplomb lettre par lettre, sur le rendu, pour les trois damiers
  d'aplomb.

  Chaque numéro suit le décalage de SA ligne : il est donc juste à gauche
  de ses propres cases et non perdu dans une marge lointaine, comme sur
  les diagrammes de HexWiki. Le damier « à bordures » ne numérote pas ses
  deux lignes de bords : ce ne sont pas des lignes de cases.

  Pour voir les huit combinaisons sans lancer le jeu :

      HEX_PREVIEW=1 go test ./ui/ -run TestApercuDamiers -v

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
                    taux d'erreur de la fermeture par ponts (0,58 % —
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

  VUE D'ENSEMBLE, ET ORDRE RECOMMANDÉ : voir STRATEGIE.md.

  C'est l'extraction de la feuille de route de HexWiki (Strategy_roadmap)
  et des pages qu'elle pointe (ziggurat, Captured cell, Mustplay region).
  Elle range les concepts en deux familles — les RÈGLES SÛRES, qui
  éliminent un coup sans jamais se tromper, et les HEURISTIQUES, qui
  guident sans garantir — et signale le piège principal : la région
  mustplay est la plus forte des idées, mais elle n'est sûre QUE si les
  connexions virtuelles sont exactes. Or les nôtres se trompent dans
  0,58 % des cas, ce qui suffirait à élaguer un coup gagnant en silence.

  Les idées ci-dessous sont listées par ancienneté, pas par priorité.

  [ ] Moteur : templates dans la fermeture (ziggurat, template de bord II)
      But : notre détection ne connaît que les ponts. Les templates de
      bord (niveau 100 de la feuille de route, très fréquents là où les
      parties se décident) la rendraient bien plus souvent utile : elle
      ne se déclenche aujourd'hui que sur 0,23 % des nœuds.

      Piste : ajouter les carriers de edge template II et du ziggurat
      (III1-a) à la fermeture de ai/vc.go. Risque de JUSTESSE à
      surveiller : TestVirtualConnectionIsSound (exhaustif 4×4, seuil
      3 %) doit passer avant ET après.

  [ ] Moteur : élaguer les cellules CAPTURÉES / DOMINÉES
      But : « il n'est jamais avantageux de jouer dans une zone capturée
      par l'adversaire ». Règle LOCALE et sûre, plus simple que la région
      mustplay. Exemple type : la 2e ligne de bord — si l'adversaire joue
      l'une des deux cases, je joue l'autre et sa pierre meurt.

      Piste : détection locale autour des pierres de bord, sans fermeture
      globale. C'est un élagage SÛR : il ne peut pas affaiblir le moteur.

  [ ] Moteur : région MUSTPLAY (élagage exact)
      But : tout coup hors de l'intersection des porteurs des menaces de
      l'adversaire est PERDANT — théorème, pas heuristique. C'est ce qui
      a permis de résoudre le 7×7 en 2005, et le plus fort levier de
      cette liste.

      PRÉREQUIS : une détection conservatrice, qui n'affirme une
      connexion que si elle est démontrée. Manquer une menace rend la
      région plus GRANDE (élagage plus timide) : on perd de la vitesse,
      jamais de la justesse. C'est le bon sens de l'erreur.

      À ne tenter qu'après les templates, et avec un test exhaustif
      (4×4 puis 5×5) qui vérifie qu'aucun coup gagnant n'est élagué.

  [ ] Moteur : guider la POLITIQUE de simulation — MESURÉ, SANS APPORT
      200 parties à 9 x 9, budget 4 000 : politique locale 91 contre
      tirage uniforme 109, soit 1,27 σ du côté défavorable. Désactivée
      par défaut ; le code reste (options{usePolicy: true}) et le banc
      d'essai aussi : HEX_BENCH_POLICY=1 go test ./ai/ -run TestBenchPolicy

      Idée non essayée : une politique plus fine, qui répond aux menaces
      LOCALES DE LA DERNIÈRE PIERRE plutôt que « près de n'importe quelle
      pierre ». C'est la vraie intuition de la feuille de route (niveau
      300), et elle n'a pas été testée — seulement la version grossière.

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
      But : remplacer l'approximation par ponts (0,58 % d'erreur) par un
      calcul exact, puis s'en servir pour GUIDER la simulation plutôt que
      pour couper l'arbre — c'est là qu'elles paient chez MoHex.

  [ ] Moteur : réutilisation de l'arbre, à réévaluer
      Mesurée à 95-105 sur 200 parties (neutre à légèrement négative), donc
      désactivée. Mais la mesure a été faite sur un 9×9 à 4 000 simulations.
      Sur un 13×13 à 84 500 simulations, l'arbre est bien plus profond et le
      calcul bien mieux renseigné : il n'est pas exclu que la conclusion
      s'inverse. À refaire avant de l'enterrer définitivement.