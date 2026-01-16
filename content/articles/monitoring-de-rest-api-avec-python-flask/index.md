---
title: "Monitoring de Rest-Api avec Python / Flask "
date: 2025-01-03
aliases: ["/2025/01/03/monitoring-de-restapi-avec-python.html"]
---

Aujourd'hui au taf j'ai j'ai mis au point un chouette petit projet avec #python et #flask en interface, c'est une simple petite application Web qui veille sur l'état de disponibilité de quelques API, en mode true/false et puis en cas de false, si l'api indique que le web service est indisponible, ne répond pas, l'interface met à la disposition de l'employé un bouton pour relancer le Web service via son API (start/stop), l'UI est tout simple, avec la liste des services et une boule verte ou rouge en fonction de l'état du Web services.. Si le service est opérationnel les boutons de redémarrage ne s'affichent pas, si pas le voyant devient rouge et le bouton s'affiche.. Je voudrais encore voir comment mettre des logs a disposition pour que les employés évitent de redémarrer sans aller voir la cause de l'arrêt du Web service.. Dans un premier temps ça devrait ouvrir le dossier réseaux dans lequel les logs sont stocké en dur mais je parie qu'avec un peu plus de code je devrais pouvoir afficher quelques lignes de logs juste avant que le service plante..

Bref affaire à suivre.. 
