---
title: "Déployer PostgreSQL avec des Volumes Externes : Un Parcours Semé d'Embûches"
date: 2025-01-07
tags: ["Tech"]
aliases: ["/2025/01/07/dployer-postgresql-avec-des-volumes.html"]
---

Après une journée à tenter de déployer un serveur PostgreSQL via Docker Compose, avec un volume de données situé sur un disque secondaire dans sa VM , je me suis retrouvé face à une série de problèmes de permissions.

Ce défi, partagé par de nombreux personnes et sans résolution claire m'a conduit à explorer des alternatives telles que Kubernetes avec des PersistentVolumeClaims (PVC) ou l'utilisation d'Ansible pour une installation native.

Contexte : Migrer un déploiement Gitlab sur un cluster Openshift avec sa base de données PostgreSQL déployée à part sur une VM (RHEL 9 VMware) en natif. 

## Docker Compose et Volumes Externes : Une Relation Compliquée

L'utilisation de Docker Compose pour déployer PostgreSQL est courante. Cependant, dès que l'on souhaite stocker les données sur un volume externe, les choses se compliquent. Les problèmes de permissions surviennent fréquemment, surtout lorsque le volume est monté à partir d'un disque secondaire ou d'un système de fichiers monté.

Exemple de fichier docker-compose.yml :

```console
services:
  db:
    image: postgres:latest
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: mydb
    volumes:
      - db-data:/var/lib/postgresql/data

volumes:
  db-data:
    driver: local
    driver_opts:
      type: none
      device: /chemin/vers/disque/externe
      o: bind
```


Dans cet exemple, le volume db-data est monté depuis un chemin spécifique sur le disque externe. Cependant, des problèmes de permissions peuvent survenir si les UID/GID entre l'hôte et le conteneur ne correspondent pas. Des solutions comme l'ajustement des permissions ou l'utilisation de l'option userns-remap de Docker peuvent être envisagées, mais elles ajoutent une complexité supplémentaire et surtout, elle n'ont pas été concluante. 

J'ai encore la possibilité de builder mes propres images Docker avec mon propre Dockerfile, mais je voulais justement éviter cet approche. 

Kubernetes et PersistentVolumeClaims : Une Gestion Plus Fine des Volumes

Face aux limitations de Docker Compose, Kubernetes offre une gestion plus robuste des volumes persistants grâce aux PersistentVolumes (PV) et PersistentVolumeClaims (PVC). Cette approche permet de découpler le stockage des pods, offrant une flexibilité accrue.

Exemple de configuration d'un PersistentVolume et d'un PersistentVolumeClaim :

```console
apiVersion: v1
kind: PersistentVolume
metadata:
  name: postgres-pv
spec:
  storageClassName: manual
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/chemin/vers/disque/externe"

---

apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
spec:
  storageClassName: manual
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

En utilisant un PVC, le pod PostgreSQL peut accéder au stockage persistant sans se soucier des détails de l'infrastructure sous-jacente. Cette méthode offre une meilleure isolation et une gestion simplifiée des permissions. 

Ansible : Automatiser le Déploiement de PostgreSQL

Une autre alternative consiste à déployer PostgreSQL directement sur la machine hôte en utilisant Ansible. Cette approche permet un contrôle total sur l'installation et la configuration, tout en évitant les complications liées aux conteneurs.

Exemple de playbook Ansible pour installer PostgreSQL :

```console
- name: Installer PostgreSQL
  hosts: db_servers
  become: yes
  vars:
    postgresql_version: 14
    postgresql_data_directory: /chemin/vers/disque/externe
  tasks:
    - name: Installer les paquets PostgreSQL
      apt:
        name:
          - postgresql-{{ postgresql_version }}
          - postgresql-contrib
        state: present
        update_cache: yes

    - name: Configurer le répertoire des données
      lineinfile:
        path: /etc/postgresql/{{ postgresql_version }}/main/postgresql.conf
        regexp: '^data_directory ='
        line: "data_directory = '{{ postgresql_data_directory }}'"

    - name: Initialiser la base de données
      command: "pg_ctlcluster {{ postgresql_version }} main initdb"
      args:
        creates: "{{ postgresql_data_directory }}/PG_VERSION"

    - name: Démarrer et activer PostgreSQL
      service:
        name: postgresql
        state: started
        enabled: yes
```


Ce playbook installe PostgreSQL, configure le répertoire des données sur le disque externe, initialise la base de données et démarre le service. L'utilisation d'Ansible garantit une configuration cohérente et reproductible sur différents environnements. 

Conclusion

Le déploiement de PostgreSQL avec des volumes de données sur des disques externes présente des défis, notamment en matière de permissions et de complexité de configuration. Alors que Docker Compose peut suffire pour des scénarios simples, des solutions comme Kubernetes avec des PVC ou une installation directe via Ansible offrent une meilleure gestion des volumes et des permissions.

Dans un prochain article, je partagerai la solution que j'aurai finalement adoptée pour cette migration, en détaillant les étapes et les leçons apprises au cours du processus.

