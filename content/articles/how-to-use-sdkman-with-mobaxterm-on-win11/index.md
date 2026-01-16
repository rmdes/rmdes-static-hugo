---
title: "How to use SDKMAN with MobaXterm on Win11"
date: 2025-07-11
tags: ["Tech", "Coding", "Fun"]
aliases: ["/2025/07/11/how-to-use-sdkman-with.html"]
---

# prerequisites – Moba’s built-in Cygwin has apt-cyg; otherwise use the GUI package manager
```sh
apt-cyg install curl zip unzip sed
```


# install SDKMAN!

```sh
curl -s "https://get.sdkman.io" | bash
```

# Adapt your .Bashrc

```sh
# --- SDKMAN & MobaXterm/Cygwin compatibility shim -----------------

if [[ -d "$HOME/.sdkman" ]]; then
  PLATFORM_FILE="$HOME/.sdkman/var/platform"
  TARGET="windowsx64"
  # refresh every shell start in case sdk selfupdate reset it
  [[ ! -f "$PLATFORM_FILE" || "$(cat "$PLATFORM_FILE")" != "$TARGET" ]] &&
      printf '%s\n' "$TARGET" > "$PLATFORM_FILE"
fi
# ------------------------------------------------------------------
```

# Script à copier dans votre /home/mobaxterm
```sh
#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# Installe Maven + les JDK majeurs (7 / 8 / 11 / 17 / 21) via SDKMAN!
# Préférence Zulu, menu alternatif si absent.
# À la fin, propose de choisir la version Java par défaut.
# ---------------------------------------------------------------------------
set -eo pipefail               # -u volontairement omis pour la compatibilité SDKMAN!

### 0. Prérequis ---------------------------------------------------------------
for cmd in curl awk grep tr; do
  command -v "$cmd" >/dev/null || { echo "$cmd is required" ; exit 1 ; }
done

### 1. SDKMAN! -----------------------------------------------------------------
if [[ ! -s "$HOME/.sdkman/bin/sdkman-init.sh" ]]; then
  echo "Installing SDKMAN!…"
  curl -s https://get.sdkman.io | bash
fi
# shellcheck source=/dev/null
source "$HOME/.sdkman/bin/sdkman-init.sh"

### 2. Patch platform pour MobaXterm/Cygwin ------------------------------------
[[ "$(uname -s)" == CYGWIN* ]] && echo windowsx64 >"$HOME/.sdkman/var/platform"

### 3. Fonctions utilitaires ----------------------------------------------------
preferred_vendors=(zulu tem kona librca graalce ms oracle)

have() { [[ -d "$HOME/.sdkman/candidates/java/$1" ]]; }

get_available_ids() {           # $1=major
  local major=$1 vendor
  for vendor in "${preferred_vendors[@]}"; do
    sdk list java | awk -F'|' \
      -v maj="${major}\\." -v vend="$vendor" -v dir="$HOME/.sdkman" '
      {
        for(i=1;i<=NF;i++) gsub(/^[ \t]+|[ \t]+$/, "", $i)
        version=$3; dist=$4; status=$5; id=$6
        if(dist==vend && version ~ "^"maj && status!="installed"){
          cmd="test -d \""dir"/candidates/java/"id"\""
          if(system(cmd)!=0) print id
        }
      }' 
  done | awk '!seen[$0]++'
}

install_build() {               # $1=identifier
  yes | sdk install java "$1"
}

choose_id() {                   # $1=major, reste=ids
  local major=$1; shift; local ids=("$@") choice
  echo -e "\n--- Java $major : pas de build Zulu trouvé ---"
  echo "Sélectionnez une distribution (0 = ignorer) :"
  select choice in "${ids[@]}"; do
    [[ $REPLY =~ ^[0-9]+$ ]] || { echo "Entrez un nombre." ; continue ; }
    [[ $REPLY == 0 ]] && echo "" && return
    [[ -n $choice ]] && echo "$choice" && return
  done
}

### 4. Boucle d’installation ----------------------------------------------------
declare -a installed_ids        # pour proposer le choix final
majors=(7 8 11 17 21)

for major in "${majors[@]}"; do
  # 4a. déjà présent ?
  installed=$(sdk list java | awk -F'|' -v maj="${major}\\." '
    $0 ~ /^[[:space:]]/ {
      for(i=1;i<=NF;i++) gsub(/^[ \t]+|[ \t]+$/, "", $i)
      if($3 ~ "^"maj && $5=="installed"){ print $6 ; exit }
    }')
  if [[ -n $installed ]] || have "*-${major}" ; then
    echo "✓  Java $major déjà installé ($installed) – ignoré."
    installed_ids+=("$installed")
    continue
  fi

  # 4b. builds disponibles
  mapfile -t ids < <(get_available_ids "$major")
  if [[ ${#ids[@]} -eq 0 ]]; then
    echo "⚠  Aucun build disponible pour Java $major."
    continue
  fi

  # 4c. priorité Zulu, sinon menu
  zulu=""
  for id in "${ids[@]}"; do [[ $id == *"-zulu" ]] && { zulu=$id; break; }; done
  if [[ -n $zulu ]]; then
    echo "➡  Installation du dernier Zulu Java $major ($zulu)…"
    install_build "$zulu"
    installed_ids+=("$zulu")
  else
    selected=$(choose_id "$major" "${ids[@]}")
    if [[ -n $selected ]]; then
      echo "➡  Installation de $selected…"
      install_build "$selected"
      installed_ids+=("$selected")
    else
      echo "⏭  Java $major ignoré."
    fi
  fi
done

### 5. Maven --------------------------------------------------------------------
if ! sdk current maven >/dev/null 2>&1; then
  echo -e "\n➡  Installation de Maven…"
  yes | sdk install maven
else
  echo -e "\n✓  Maven déjà installé."
fi

### 6. Choix de la version Java par défaut -------------------------------------
echo -e "\n--- Sélection de la version Java par défaut ------------------------"
PS3="Numéro (0 = ne rien changer) : "
select def in "${installed_ids[@]}"; do
  [[ $REPLY =~ ^[0-9]+$ ]] || { echo "Entrez un nombre." ; continue ; }
  if [[ $REPLY == 0 ]]; then
    echo "⏭  Aucun changement de version courante."
  elif [[ -n $def ]]; then
    sdk default java "$def" || true   # code 1 inoffensif
    echo "✓  $def est maintenant la version par défaut."
  fi
  break
done

# Exporte JAVA_HOME pour la session actuelle
export JAVA_HOME="$HOME/.sdkman/candidates/java/current"
export PATH="$JAVA_HOME/bin:$PATH"

### 7. Persistance JAVA_HOME ----------------------------------------------------
if ! grep -q 'export JAVA_HOME=.*\.sdkman' "$HOME/.bashrc" 2>/dev/null; then
cat >>"$HOME/.bashrc" <<'EOF'

# --- SDKMAN! Java/Maven (auto-ajout) ------------------------------------------
export JAVA_HOME="$HOME/.sdkman/candidates/java/current"
export PATH="\$JAVA_HOME/bin:\$PATH"
# -----------------------------------------------------------------------------#
EOF
  echo "ℹ  JAVA_HOME ajouté à ~/.bashrc"
fi

### 8. Résumé ------------------------------------------------------------------
echo -e "\nÉtat actuel :"
sdk current
echo -e "\n✅  Installation terminée."
``````shvi install_java_stack.sh
chmod +x install_java_stack.sh
```# Installer les Java versions
```sh
./install_java_stack.sh
```
```sh
  11/07/2025   16:09.24   /home/mobaxterm  ./install_java_stack.sh
✓  Java 7 déjà installé (7.0.352-zulu) – ignoré.
✓  Java 8 déjà installé (8.0.452.fx-zulu) – ignoré.
✓  Java 11 déjà installé (11.0.27.fx-zulu) – ignoré.
✓  Java 17 déjà installé (17.0.15.fx-zulu) – ignoré.
✓  Java 21 déjà installé (21.0.7.fx-zulu) – ignoré.

✓  Maven déjà installé.

--- Sélection de la version Java par défaut ------------------------
1) 7.0.352-zulu     3) 11.0.27.fx-zulu  5) 21.0.7.fx-zulu
2) 8.0.452.fx-zulu  4) 17.0.15.fx-zulu
Numéro (0 = ne rien changer) : 0
⏭  Aucun changement de version courante.

État actuel :

Using:

java: 17.0.15.fx-zulu
maven: 3.9.10

✅  Installation terminée.
```
