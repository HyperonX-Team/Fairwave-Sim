<h1 align="center">Fairwave - un opérateur communautaire dans une boîte à pizza</h1>

<p align="center">
  <strong>LTE privé open source : branchez-le sur Ethernet, émettez en 4G ou 5G, accueillez vos voisins.</strong>
</p>

<p align="center">
  <img alt="CI" src="https://img.shields.io/badge/CI-GitHub_Actions-blue?logo=githubactions">
  <img alt="License: Apache-2.0" src="https://img.shields.io/badge/License-Apache_2.0-green.svg">
  <img alt="Release: v0.1.0 (lab)" src="https://img.shields.io/badge/release-v0.1.0-orange">
  <img alt="RF default" src="https://img.shields.io/badge/RF_TX-OFF_by_default-critical">
  <img alt="Stack" src="https://img.shields.io/badge/stack-Open5GS_%2B_srsRAN_%2B_Go-informational">
</p>

> [!IMPORTANT]
> **AVERTISSEMENT LÉGAL ET SPECTRE.** Fairwave fonctionne par défaut en mode **laboratoire / sans RF** (boucle FI zéro uniquement).
> Émettre sur des bandes cellulaires sans autorisation appropriée est **illégal dans la plupart des juridictions**.
> Vous êtes seul responsable des licences, des concessions SAS, des restrictions indoor et de l'homologation de type.
> HyperonX et les contributeurs fournissent le logiciel **tel quel** uniquement pour les réseaux privés légaux, la recherche
> et les régimes de spectre partagé. Voir [docs/spectrum-and-law/](docs/spectrum-and-law/index.md).

**À lire en :** [English](README.md) · [العربية](README.ar.md) · [中文](README.zh.md) · [Español](README.es.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [हिन्दी](README.hi.md) · [Italiano](README.it.md) · [Português](README.pt.md) · [Русский](README.ru.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Türkçe](README.tr.md) · [Polski](README.pl.md) · [Nederlands](README.nl.md) · [Українська](README.uk.md) · [Svenska](README.sv.md) · [Bahasa Indonesia](README.id.md) · [Tiếng Việt](README.vi.md)

---

## Le système cassé

La connectivité mobile est un cartel à un seul membre : une SIM, un opérateur, un contrat, un jardin clos.
Les cartes de couverture sont des brochures marketing ; les rues rurales en tombent ; les appartements perdent le signal ;
et vous - la personne qui paie - ne possédez aucune de l'infrastructure qui vous sert. Si le prix de votre opérateur
national augmente ou si sa tour se tait, votre seule option est… un autre monopole avec les mêmes tours et les mêmes conditions.

Le modem dans votre poche peut parler à une station de base à 20 mètres. La seule raison pour laquelle il ne parle pas
à *votre* station de base, c'est qu'on n'a jamais autorisé cette station à vous appartenir.

## La solution HyperonX

**Fairwave est l'opérateur communautaire : une small cell complète, open source, qui tient dans une boîte à pizza
et se branche sur un Ethernet ordinaire.**

Un café, une coopérative d'habitation, une salle des fêtes, un quartier - n'importe qui peut en exploiter une :

1. Démarrez l'image Fairwave sur un mini-PC (x86 ou ARM) avec un SDR connecté.
2. Lancez `fairwave node init`, répondez au questionnaire réglementaire (la TX reste coupée jusqu'à preuve d'autorisation).
3. Émettez des SIM Fairwave pour votre communauté - des cartes que vous possédez, des identifiants que vous contrôlez.
4. Les téléphones se connectent. Le trafic reste local quand c'est possible ; plusieurs boîtes à pizza se maillent ;
   l'accès Internet passe par un tunnel WireGuard sécurisé quand vous le voulez.

Construit sur une infrastructure ouverte éprouvée - **Open5GS** (EPC) et **srsRAN** (eNB/gNB) - avec un plan de
contrôle en Go, un tableau de bord opérateur, un approvisionnement SIM hors ligne et un mode laboratoire qui exécute
tout l'opérateur dans Docker avec zéro RF.

## Ce que vous obtenez

- **Une boîte à pizza qui est un opérateur** : identité de nœud, enrôlement, cycle de vie
  (`provision → register → on-air → peer → breakout`) géré par `fairwave-control`.
- **Attachement LTE réel** : Open5GS EPC + srsRAN eNB, PLMN configurable, zones de suivi,
  APN `internet` + `ims`, rupture locale au bord.
- **Opérations SIM Fairwave** : provisionneur offline-first ; génère Ki/OPc, lots CSV/JSON pour
  les bureaux de cartes, écrit dans HSS/UDM, contrôles de révocation et de remplacement. Lab et production strictement séparés.
- **eSIM de laboratoire (SM-DP+)** : votre propre serveur de profils façon SGP.22 et une eUICC logicielle -
  profils liés chiffrés, codes d'activation QR (`LPA:1$...`), cycle de téléchargement complet vérifié en CI
  sans matériel. Lab uniquement par conception ; la conformité GSMA est suivie comme éléments ouverts.
- **Maillage de voisinage** : découverte mDNS, contrôle mTLS, plan de données WireGuard, échange de routes.
- **Portail opérateur** : tableau de bord local-first - UEs (respectueux de la vie privée), backhaul, pairs, mode laboratoire.
- **Mode laboratoire complet** : tout le réseau sur IP pure (zmq) avec srsUE - sans radio, sans licence, compatible CI.
- **Portes réglementaires dans le code** : armer la TX exige code pays, reconnaissance de licence
  et liste blanche de fréquences. Refus compilé autrement.

## Pourquoi cela menace les opérateurs en place

Le verrouillage des opérateurs devient *optionnel pour la couverture locale*. Quand le café, la coopérative et le
village peuvent chacun faire tourner une cellule et les mailler, la SIM nationale n'est qu'une option d'itinérance,
pas le gardien. Le délestage, l'accès neutral host et les forfaits communautaires cessent d'être théoriques.
Fairwave ne reproduit pas le cartel des télécoms - il rend le monopole local du cartel contestable.

## Pourquoi c'est faisable maintenant

- **Open5GS** et **srsRAN** sont matures, proches de la production et activement développés.
- **Les SDR** (USRP, LimeSDR, BladeRF) couvrent les bandes LTE à puissance de small cell pour des centaines de dollars.
- **Le spectre partagé est réel** : CBRS aux États-Unis, licences locales au Royaume-Uni/UE, bandes LTE privées.
- **LTE privé + appels Wi-Fi** donnent des modèles légaux à copier au lieu de faire semblant que la réglementation n'existe pas.
- Le matériel classe mini-PC (et Raspberry Pi CM4/5 + HATs pour le développement) est bon marché et suffisant.

---

## Démarrage rapide - premier attachement UE en <30 minutes (sans RF, sans licence)

> Prérequis : Docker Engine 24+, 8 Go de RAM. Tout tourne dans des conteneurs ; rien n'émet.
> Pour le chemin de données complet (IP UE + ping) utilisez un Linux natif ; Docker Desktop sous
> Windows/macOS passe toutes les vérifications d'attachement côté EPC (voir [docs](docs/tutorials/lab-attach.md)).

```bash
./scripts/bootstrap.sh      # vérifie/installe les toolchains (Go, Docker, pre-commit)
make lab-up                 # construit les images, démarre EPC + eNB + srsUE, exécute les vérifications d'attachement
make status                 # santé en un coup d'œil : mme, sgwu/upf, enb, ue1
```

`make lab-up` vérifie (et affiche) tout ce qui suit :

1. Open5GS MME + HSS en cours d'exécution
2. eNB S1-MME connecté au MME
3. Connexion RRC de l'UE + accès aléatoire sur le PLMN de laboratoire
4. Authentification NAS de l'UE + mode de sécurité (milenage contre HSS)
5. Le MME crée le bearer EPS par défaut et envoie Attach Accept (IP UE allouée)

Puis regardez à l'intérieur :

```bash
fairwave node status                     # vue plan de contrôle de ce cluster
fairwave sim issue --count 2 --profile lab  # émet deux SIM de laboratoire
fairwave spectrum check --country US --band n48 --indoor  # démo de porte spectrale
make lab-down                            # arrête tout
```

(Sur les hôtes Linux avec un timing ZMQ stable, `docker exec -it ue1 ping -c3 10.45.0.1`
exerce le chemin de données complet sur le tunnel `tun_srsue`.)

Tout ce qui précède est **silencieux en RF** : la radio est émulée avec les dispositifs virtuels `srsran/zmq`.
Pour du vrai matériel, suivez [docs/tutorials/cafe-pilot.md](docs/tutorials/cafe-pilot.md) -
qui ne vous laissera pas activer la TX sans compléter le checklist légal.

## Documentation

Site complet : **lancez `make docs-serve`** - ou lisez-le dans l'arborescence sous [`docs/`](docs/index.md).

| Commencez ici | Ensuite |
|---|---|
| [Vision](docs/vision.md) | [Architecture](docs/architecture/overview.md) |
| [Démarrage rapide (30 min, sans RF)](docs/tutorials/quickstart-no-rf.md) | [Pilote café (2 h, avec checklist légal)](docs/tutorials/cafe-pilot.md) |
| [Spectre et loi](docs/spectrum-and-law/index.md) | [Modèle de menaces](design/threat-model.md) |
| [Cycle de vie SIM](docs/sim-lifecycle/index.md) | [Tissu de peering](docs/peering/index.md) |
| [Référence API](docs/api/index.md) | [ADRs](docs/adr/0000-index.md) |

## Statut

**Version laboratoire `v0.1.0`** : l'attachement EPC + zmq RAN + srsUE fonctionne de bout en bout ; plan de contrôle,
CLI, provisionneur SIM, eSIM de laboratoire (SM-DP+) et le site de documentation sont fonctionnels. Un cœur 5G SA free5GC avec comptage d'usage basé sur les CDR CHF est fourni en option (`core: free5gc`), avec des profils de laboratoire gNB/UE sur ZMQ et un test d'attachement en CI. Les chemins
RF réels sont validés sur du matériel de développement mais **désactivés par défaut**. Voir la [feuille de route](design/roadmap.md).

## Ce que Fairwave n'est PAS

- **Pas un capteur d'IMSI** - aucune interrogation passive ; les UEs doivent s'authentifier avec des identifiants que vous avez provisionnés.
- **Pas un Far West spectral** - la TX est verrouillée, et nous refusons les fonctions de contournement réglementaire.
- **Pas un opérateur national gratuit** - c'est de la couverture locale avec rupture optionnelle.
- **Pas un remplacement des appels d'urgence** - planifiez le comportement 911/112 dans chaque déploiement ([docs/ops/incident-response.md](docs/ops/incident-response.md)).

## Contribuer et gouvernance

Nous accueillons les contributeurs. Lisez [CONTRIBUTING.md](CONTRIBUTING.md) (style de code, DCO, tests),
[GOVERNANCE.md](GOVERNANCE.md) (comment les décisions sont prises), [SECURITY.md](SECURITY.md)
(divulgation des vulnérabilités ; modèle de menaces) et [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
Licence : **Apache-2.0** ([LICENSE](LICENSE)) ; attribution des tiers dans [NOTICE](NOTICE).

---

<p align="center">
  <sub>Construit par l'équipe HyperonX et la communauté Fairwave. L'air appartient à tous.</sub>
</p>
