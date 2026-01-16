---
title: "Comment installer oGraph Bluesky Poster en phase de test ?"
date: 2023-09-09
tags: ["Tech", "Photos", "Bluesky"]
aliases: ["/2023/09/09/comment-installer-bluesky.html"]
---

## Chrome/ Vivaldi/ Edge
Tu veux tester mon extension avant qu'elle soit mise dans l'extension store ?

Télécharge le fichier zip [Bluesky-Poster](https://sources.rmendes.net/Bluesky-Poster.zip)
- va dans les extensions de ton navigateur : 
tu peux taper ceci dans la barre d'adresse/url pour y arriver au choix en fonction de ton navigateur :
chrome://extensions 
vivaldi://extensions
edge://extensions


# image 1
Va dans les extensions via le menu

<img src="uploads/2023/f3103b7d63.png" width="600" height="566" alt="capture d'écran montrant comment aller dans le Menu des extensions sur Chrome">

# image 1.2
Active le mode développeur 

<img src="uploads/2023/2023-09-09-20-20.png" width="600" height="185" alt="Capture d'écran montrant comment activer le mode dev et charger un extension hors store">

Et charge l'extension préalablement dézippée dans tes Documents ou ailleurs sur ton ordinateur, met là à un endroit fixe qui ne bougera plus. 

# image 1.3 
Épingle l'extension pour qu'elle soit à porter de main 

<img src="uploads/2023/4051954698.png" width="600" height="208" alt="">

# image 1.4
l'extension est installée, il ne te reste plus qu'à te connecter !

<img src="uploads/2023/e2f412f793.png" width="600" height="208" alt="capture d'écran montrant bluesky poster prêt à se connecter">

## Code source
[Source](https://github.com/rmdes/oGraph-Bluesky-Poster) 

# Firefox

[Lien](https://sources.rmendes.net/firefox_extension.zip) vers le téléchargement pour Firefox

## Instruction

- Ouvrez Firefox et accédez à **about:debugging** dans la bar d'adresse.
- Cliquez sur "This Firefox" dans la barre latérale gauche.
- Cliquez sur "Charger le module complémentaire temporaire" et sélectionnez le fichier ZIP nouvellement téléchargé.
- L'extension doit maintenant être installée pour les tests, et l'icône doit apparaître correctement dans l'interface utilisateur


## bug
pas de bug recensé, autre que le site web sur lequel vous êtes, doit être en conformité avec les meta opengraph pour que la carte intégrée sur Bluesky s'affiche avec une image, à part cela, même si le site distant n'a pas cette balise meta ou n'a pas d'image définie, le post va fonctionner avec ou sans apport de texte de votre part, avec ou sans images. 
