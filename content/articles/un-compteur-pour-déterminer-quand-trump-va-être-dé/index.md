---
title: "Un compteur pour déterminer quand Trump va être dégagé (ou pas)"
date: 2025-01-26
tags: ["Tech", "Photos", "Coding", "Politics", "Bluesky"]
aliases: ["/2025/01/26/un-compteur-pour-dterminer-quand.html"]
---

# Documentation du Script N8N

## Description

Ce script JavaScript est conçu pour N8N afin de calculer et de publier quotidiennement des décomptes avant des événements politiques majeurs aux États-Unis, à savoir :

- Les **Midterms** (élections de mi-mandat).
- L'**élection présidentielle**.
- Le **jour de l'investiture**.

Il génère également une barre de progression graphique pour chaque événement sous forme de texte.

## Fonctionnalités

### 1. Définition des dates clés

Le script définit les dates des événements cibles :

- **Midterms** : 3 novembre 2026.
- **Élection présidentielle** : 7 novembre 2028.
- **Jour de l'investiture** : 20 janvier 2029.

### 2. Calcul des jours restants

Le script calcule :

- Le nombre total de jours entre aujourd'hui et chaque événement.
- Le pourcentage de progression en fonction des jours écoulés.

### 3. Génération de barres de progression

Une fonction génère une barre de progression graphique, composée de blocs pleins (`█`) et de blocs vides (`▒`), représentant visuellement l'avancée jusqu'à l'événement.

### 4. Génération et publication du message

Le script produit un message comprenant :

- Le décompte des jours restants pour chaque événement.
- Les barres de progression associées.

## Exemple de Résultat

Voici un exemple du message généré :


```
There are 646 days until the Midterms, 1381 days until the next Presidential Election, and 1490 days until the next Inauguration Day.
Midterms Progress: █▒▒▒▒▒▒▒▒▒ 9%
Presidential Election Progress: █▒▒▒▒▒▒▒▒▒ 4% 
Until Inauguration: ▒▒▒▒▒▒▒▒▒▒ 0%
```


## Code

```
// JavaScript code for N8N to calculate and post daily countdowns with a graphical loading bar

// Define the reference start date (Trump's 2025 inauguration)
const startOfMandate = new Date('2025-01-20T00:00:00Z');

// Define the target dates
const midtermsDate = new Date('2026-11-03T00:00:00Z'); // Next USA Midterms
const presidentialElectionDate = new Date('2028-11-07T00:00:00Z'); // Next Presidential Election
const inaugurationDate = new Date('2029-01-20T00:00:00Z'); // Next Inauguration Day

// Get the current date
const currentDate = new Date();

// Function to calculate correct progress
function calculateProgress(targetDate) {
  const totalDays = Math.ceil((targetDate - startOfMandate) / (1000 * 60 * 60 * 24));
  const elapsedDays = Math.ceil((currentDate - startOfMandate) / (1000 * 60 * 60 * 24));
  return Math.min(100, Math.max(0, Math.floor((elapsedDays / totalDays) * 100)));
}

// Calculate days remaining
const daysUntilMidterms = Math.ceil((midtermsDate - currentDate) / (1000 * 60 * 60 * 24));
const daysUntilPresidentialElection = Math.ceil((presidentialElectionDate - currentDate) / (1000 * 60 * 60 * 24));
const daysUntilInauguration = Math.ceil((inaugurationDate - currentDate) / (1000 * 60 * 60 * 24));

// Calculate accurate progress percentages
const midtermsProgress = calculateProgress(midtermsDate);
const presidentialProgress = calculateProgress(presidentialElectionDate);
const inaugurationProgress = calculateProgress(inaugurationDate);

// Create a graphical loading bar function
function createLoadingBar(percentage) {
  const totalBars = 10; // Length of the loading bar
  const filledBars = Math.floor((percentage / 100) * totalBars);
  const emptyBars = totalBars - filledBars;
  return `${'█'.repeat(filledBars)}${'▒'.repeat(emptyBars)} ${percentage}%`;
}

// Generate the loading bars
const midtermsLoadingBar = createLoadingBar(midtermsProgress);
const presidentialLoadingBar = createLoadingBar(presidentialProgress);
const inaugurationLoadingBar = createLoadingBar(inaugurationProgress);

// Generate the message
const message = `There are ${daysUntilMidterms} days until the Midterms, ${daysUntilPresidentialElection} days until the next Presidential Election, and ${daysUntilInauguration} days until the next Inauguration Day.\n\n` +
  `Midterms Progress: \n${midtermsLoadingBar}\n\n` +
  `Presidential Election Progress: \n${presidentialLoadingBar}\n\n` +
  `Inauguration Progress: \n${inaugurationLoadingBar}`;

// Output the message
return [{
  json: {
    message,
  },
}];

```


##  Où il publie 

Le script retourne le message sous forme d'un objet JSON, prêt à être utilisé dans un flux N8N pour une publication quotidienne via un nœud horaire configuré.

## Configuration Recommandée

- **Fuseau horaire du serveur** : CET (heure allemande).
- **Heure de publication** : 14h00 CET, correspondant à 8h00 ET (heure de la côte Est des États-Unis).

## Utilisation

1. Intégrez le script dans un nœud de fonction JavaScript dans N8N.

2. Ajoutez un nœud horaire configuré pour exécuter le flux quotidiennement.3. Reliez le nœud de fonction à un nœud de sortie ou à un service tiers pour publier le message (par exemple Bluesky.).

## Résultat 

[Trumpwatch](https://bsky.app/profile/trumpwatch.skyfleet.blue/post/3lkqzr6bg5n27)
