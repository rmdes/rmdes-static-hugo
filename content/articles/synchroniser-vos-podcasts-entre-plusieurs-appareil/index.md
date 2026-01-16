---
title: "Synchroniser vos podcasts entre plusieurs appareils avec AntennaPod et Nextcloud"
date: 2025-05-04
tags: ["Tech", "Audio"]
aliases: ["/2025/05/04/synchroniser-vos-podcasts-entre-plusieurs.html"]
---


Vous écoutez vos podcasts avec Spotify ou autre platforme et vous en avez assez d’être enfermé dans un écosystème fermé comme Google Podcasts ? Vous voulez pouvoir changer de téléphone ou utiliser plusieurs appareils sans perdre vos abonnements ni votre historique d’écoute ? Ce tutoriel est pour vous.

Grâce à **Nextcloud** et à son application **GPodder Sync**, vous pouvez synchroniser automatiquement vos abonnements, la progression de lecture, et même les épisodes écoutés entre plusieurs instances d’AntennaPod — ou avec d’autres applications compatibles avec le protocole *gpodder.net*.

## Pourquoi utiliser Nextcloud + GPodderSync ?

- **Synchronisation complète** entre plusieurs téléphones ou tablettes Android utilisant AntennaPod.
- **Pas de perte de données** : si vous réinstallez votre téléphone, il suffit de reconnecter AntennaPod à votre Nextcloud pour tout récupérer (abonnements, position de lecture, épisodes lus).
- **Indépendance** : vous ne dépendez d’aucun service tiers (comme Google, Spotify ou Apple).
- **Interopérabilité** : compatible avec d'autres clients de podcast utilisant le protocole gpodder.net.

---

## Prérequis

- Un **serveur Nextcloud** accessible via une URL ou une adresse IP (auto-hébergé ou chez un fournisseur).
- L’application **GPodder Sync** installée sur votre instance Nextcloud (c’est une app Nextcloud libre).
- L’application **AntennaPod** installée sur votre téléphone Android.

---

## Installation de GPodder Sync sur Nextcloud

1. Connectez-vous à l’interface web de votre Nextcloud en tant qu’administrateur.
2. Allez dans le **Catalogue d’applications**.
3. Recherchez l’app **GPodder Sync** et installez-la.
4. Une fois installée, l’app sera disponible à l’adresse :  
   `https://votre-nextcloud.tld/index.php/apps/gpoddersync/`

Projet officiel : [https://apps.nextcloud.com/apps/gpoddersync](https://apps.nextcloud.com/apps/gpoddersync)

---

## Connexion d’AntennaPod à Nextcloud

1. Ouvrez **AntennaPod**.
2. Allez dans **Paramètres > Synchronisation**.
3. Appuyez sur **Choisir le fournisseur de synchronisation**.
4. Sélectionnez **Nextcloud**.
5. Entrez l’adresse de votre serveur Nextcloud, par exemple :  
   `https://votre-nextcloud.tld`
6. Appuyez sur **Continuer**.
7. Une fenêtre de connexion s’ouvre dans le navigateur : entrez vos identifiants Nextcloud.
8. Autorisez AntennaPod à accéder à votre compte.
9. Une fois connecté, la synchronisation se fera automatiquement.

---

## Synchronisation entre plusieurs appareils

- Répétez les étapes ci-dessus sur chaque appareil.
- AntennaPod synchronisera automatiquement les abonnements, les épisodes lus, et la position de lecture.
- Vous pouvez forcer une synchronisation manuellement via l’option **Forcer la synchronisation**.

---

## Bonnes pratiques

- Faites une première **sauvegarde OPML** au cas où, avant de basculer vers la synchronisation via Nextcloud.
- Activez la synchronisation seulement **sur des appareils que vous utilisez réellement**, pour éviter les conflits.
- Si vous avez déjà commencé à écouter des épisodes avant de lier l’appareil, pensez à appuyer sur **Forcer la synchronisation** après l’activation, afin d’envoyer les états d’écoute existants.

---

## Alternative : utiliser le serveur public gpodder.net

Si vous ne voulez pas héberger votre propre Nextcloud :
- Inscrivez-vous sur [https://www.gpodder.net](https://www.gpodder.net)
- Créez un ou plusieurs “devices” (appareils) dans l’interface web
- Connectez AntennaPod au serveur en suivant les mêmes étapes que ci-dessus, en choisissant cette fois le fournisseur “gPodder”

**Attention** : le serveur public est souvent saturé. L’expérience peut être instable.

---

## Conclusion

Avec AntennaPod et Nextcloud, vous gardez le contrôle total sur vos podcasts, sans dépendance aux services fermés. C’est une solution idéale pour les utilisateurs soucieux de leur vie privée, ou tout simplement pour ceux qui veulent une synchronisation robuste entre plusieurs appareils.
