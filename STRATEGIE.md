STRATÉGIE : CE QU'ON PEUT TIRER DE HEXWIKI
====================================================================

Note de travail, 3 octobre 2026.
Source principale : https://www.hexwiki.net/index.php/Strategy_roadmap
Complétée par : Captured cell, Mustplay region, Ziggurat.

Cette note existe parce que la question « comment rendre le moteur plus
fort ? » a une réponse documentée, et qu'il vaut mieux la lire que
l'inventer. Elle distingue ce que la feuille de route apporte vraiment de
ce qu'elle n'apporte pas.

--------------------------------------------------------------------
1. CE QU'EST CETTE PAGE, ET CE QU'ELLE N'EST PAS
--------------------------------------------------------------------

Strategy_roadmap est une progression PÉDAGOGIQUE : elle liste les
concepts de Hex dans l'ordre où un HUMAIN doit les apprendre, par niveau
Elo (0-100 = débutant, 900 = meilleur joueur mondial).

  100  templates (pont, ziggurat), blocage
  200  échelles (ladders) et évasions, templates de bord
  300  pivot/climb, intrusions, reconnaître ce qui est connecté,
       quand échanger
  400  switchback, foldback, Tom's move, foiling, minimaxing (murs),
       coups de coin en ouverture
  500  minimaxing systématique, tidiness, région mustplay, josekis
  600  ouvertures, flancs/templates de pivot, capture et domination

Deux avertissements avant de s'en servir :

  - L'ORDRE EST HUMAIN, pas informatique. Le ziggurat est « niveau 100 »
    parce qu'un débutant humain peut le retenir ; pour un moteur c'est
    un ajout de template parmi d'autres. À l'inverse, « minimaxing »
    (niveau 400) est ce que fait déjà, très bien, n'importe quel moteur
    Monte-Carlo. Ne pas prendre cet ordre comme un ordre de travail.

  - La page ne dit RIEN de neuf sur UCT, les simulations ou le budget.
    Elle est utile pour la CONNAISSANCE DU JEU, pas pour la recherche.
    Le moteur n'y trouvera pas « comment mieux chercher », mais
    « ce qu'il ne sait pas encore reconnaître ».

--------------------------------------------------------------------
2. LA DISTINCTION QUI COMPTE : SÛR OU HEURISTIQUE
--------------------------------------------------------------------

C'est le tri le plus utile de cette lecture. Les concepts se rangent en
deux familles qui ne se traitent pas du tout de la même façon.

  RÈGLES SÛRES (elles ÉLIMINENT des coups, sans se tromper)

    - Région mustplay : l'intersection des porteurs de toutes les
      menaces de l'adversaire. Propriété démontrée : tout coup HORS de
      cette région est PERDANT. Ce n'est pas une heuristique, c'est un
      théorème. Effet : réduit massivement le facteur de branchement.
    - Cellules capturées : une zone capturée par l'adversaire n'a
      jamais d'intérêt pour moi → on ne la joue pas.
    - Cellules dominées : parmi deux cases, l'une domine l'autre si
      tout ce que l'on peut faire par l'une on peut le faire par
      l'autre → on ne joue que la dominante.

    Ce sont les seules qui donnent de la force SANS rien coûter en
    justesse. Un moteur qui élague avec elles ne peut pas s'affaiblir.

  CONNAISSANCES HEURISTIQUES (elles GUIDENT, sans garantir)

    - tidiness, minimaxing, « quand répondre localement », foiling,
      coups de coin, josekis.
    - Utiles pour une POLITIQUE DE SIMULATION (choisir les coups des
      parties aléatoires) ou un livre d'ouvertures. Elles ne se
      démontrent pas ; elles se mesurent.

--------------------------------------------------------------------
3. ÉTAT DE NOTRE MOTEUR FACE À CETTE FEUILLE DE ROUTE
--------------------------------------------------------------------

Ce qu'on a déjà, et qui correspond à quel niveau de la feuille :

  pont (bridge) ................. implémenté (ai/vc.go), MESURÉ SANS GAIN
                                   (VC activées 24-24 contre 32-16, seuil
                                   de déclenchement 0,23 % des nœuds)
  adjacence = connexion ......... oui (testé)
  coups décisifs exacts ......... oui (victoire immédiate, blocage)
  ziggurat ...................... NON (le suivant sur la liste)
  templates de bord ............. OUI, mais UN SEUL (2e ligne)
  reconnaître le connecté ....... partiellement (fermeture par ponts et
                                   template de bord, taux d'erreur mesuré
                                   0,58 % — 3/516)
  minimaxing (murs) ............. implicitement, par les simulations
  région mustplay ............... NON
  cellules capturées/dominées ... NON
  livre d'ouvertures ............ NON

--------------------------------------------------------------------
4. LE PIÈGE, ET IL EST SÉRIEUX
--------------------------------------------------------------------

La région mustplay est la plus belle idée de la page, et c'est celle
qu'il ne faut PAS brancher telle quelle.

Raison : sa sûreté repose entièrement sur l'EXACTITUDE des connexions
virtuelles qu'on lui donne. « Tout coup hors de cette région est
perdant » n'est vrai que si les menaces sont de VRAIES menaces.

Or notre fermeture par ponts se trompe dans 0,58 % des cas : elle
déclare connecté ce qui ne l'est pas. Utilisée pour ÉLAGUER, une erreur
de ce genre ne coûte pas 0,4 % de finesse — elle coûte LE COUP GAGNANT,
silencieusement, précisément dans les positions tendues où le moteur
doit être bon.

C'est un changement de nature, pas de degré :

  « la VC se trompe de temps en temps »  →  gênant, 0,23 % d'usage
  « la VC se trompe et je l'utilise pour élaguer »  →  je peux perdre
                                                        une partie gagnée

Donc : AVANT d'élaguer, rendre la détection CONSERVATRICE — n'affirmer
une connexion que si elle est démontrée, quitte à en manquer. Manquer
une menace rend la région mustplay plus GRANDE, donc l'élagage plus
timide : on perd un peu de vitesse, jamais de justesse. C'est le bon
sens de l'erreur.

--------------------------------------------------------------------
5. LES CINQ LEVIERS, PAR RAPPORT GAIN / TRAVAIL / RISQUE
--------------------------------------------------------------------

A. TEMPLATES DANS LA FERMETURE (ziggurat, template de bord II)

   Feuille de route : niveau 100 (« apprendre les templates ») — et
   c'est justement ce que notre détection ne connaît pas.

   Pourquoi ça compte : nos simulations coupent la partie dès qu'une
   connexion virtuelle apparaît. Si la détection ne connaît que les
   ponts, elle ne coupe presque jamais (0,23 %). Les templates de bord
   sont TRÈS fréquents au bord, là où les parties se décident.

   Gain attendu : le taux de déclenchement monte, ce qui rend
   utilisable le mécanisme déjà écrit et déjà testé.
   Travail : moyen (carriers petits, mais il faut les tables).
   Risque : de justesse — à vérifier avec TestVirtualConnectionIsSound
   (exhaustif sur 4×4, seuil 3 %) avant et après.

B. ÉLAGAGE PAR RÉGION MUSTPLAY

   Feuille de route : niveau 500. Gain potentiellement le plus fort —
   c'est ce qui a permis de résoudre 7×7 en 2005.
   Travail : élevé. Risque : ÉLEVÉ, et conditionné au point 4.
   À ne tenter qu'APRÈS A, et avec un test exhaustif sur 4×4 puis 5×5
   qui vérifie que le coup gagnant n'est jamais élagué.

C. ÉLAGAGE DES CELLULES CAPTURÉES / DOMINÉES

   Feuille de route : niveau 600. Plus simple que B, plus étroit.
   La règle de la 2e ligne de bord (« si l'adversaire joue l'une, je
   joue l'autre et sa pierre meurt ») se détecte localement, sans
   fermeture globale. Risque plus faible que B.

D. LIVRE D'OUVERTURES (coups de coin, josekis)

   Feuille de route : niveaux 400 à 600. Facile, mesurable, plafond
   bas : n'aide que les tout premiers coups, que l'UCT trouve déjà
   correctement avec un budget décent.

E. POLITIQUE DE SIMULATION GUIDÉE

   Feuille de route : niveau 300, « reconnaître ce qui est connecté et
   ce qui ne l'est pas, pour éviter de défendre l'indéfendable ». C'est
   la phrase qui vise le plus juste pour un moteur Monte-Carlo.

   Nos parties aléatoires jouent uniformément au hasard. La
   littérature est constante : c'est LÀ que se gagne la force d'un
   moteur de Hex. Gain le plus élevé des cinq ; travail le plus élevé ;
   risque de justesse nul (une politique ne fait qu'orienter, elle
   n'élimine rien).

--------------------------------------------------------------------
6. ORDRE RECOMMANDÉ
--------------------------------------------------------------------

  1. A (templates) — prolonge du code déjà écrit et déjà testé, risque
     de justesse maîtrisé par le test exhaustif existant.
  2. C (capturé/dominé) — règles locales, sûres, peu de code.
  3. E (politique) — le vrai levier de force, à faire quand on a du
     temps pour mesurer sérieusement.
  4. B (mustplay) — seulement une fois la détection conservatrice
     prouvée. Le plus fort, mais le seul qui puisse perdre une partie
     gagnée en silence.
  5. D (ouvertures) — confort, pas force.

--------------------------------------------------------------------
7. MÉTHODE DE MESURE (à ne pas oublier)
--------------------------------------------------------------------

  Les deux mécanismes déjà en place ont été mesurés PUIS désactivés
  (VC 24-24 puis 32-16 ; réutilisation d'arbre 95-105, soit 0,71 σ).
  Leçon : un mécanisme qui ne paie pas doit être désactivé, pas
  conservé « au cas où ».

  Et : la puissance statistique de nos mesures est faible. Un écart de
  5 parties sur 100 n'est pas un résultat. Les bancs existants
  (HEX_BENCH_LONG, HEX_BENCH_GAMES) savent faire 200 parties ; pour
  juger un levier comme E, il faudra ce volume-là, pas moins.

  Corollaire honnête sur l'état actuel : les mesures de force (39-10,
  95-105) ont été faites sur l'ANCIEN plateau en quinconce. Depuis le
  passage au losange, le moteur joue une autre région du réseau. Ces
  chiffres restent indicatifs ; aucun n'a été refait.

--------------------------------------------------------------------
8. CE QUE LA PAGE N'APPORTE PAS
--------------------------------------------------------------------

  - Rien sur l'UCT, le budget, l'élagage des arbres : ce sont des
    questions de recherche, pas de stratégie.
  - Rien sur la règle du gâteau (swap) : elle concerne le jeu à deux
    joueurs humains, pas notre configuration.
  - Aucun moyen de PROUVER qu'un ajout rend le moteur plus fort. La
    page donne des idées, pas des garanties. Seuls les bancs d'essai
    tranchent.
