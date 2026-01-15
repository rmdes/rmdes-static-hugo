---
title: "How to use SDKMAN with MobaXterm on Win11"
date: 2025-07-11
excerpt: "A guide to installing and configuring SDKMAN on Windows 11 with MobaXterm for Java development."
headerAlt: "SDKMAN and MobaXterm"
---

## Prerequisites

Install required packages via Cygwin's apt-cyg:

```bash
apt-cyg install curl zip unzip sed
```

## Installing SDKMAN!

```bash
curl -s "https://get.sdkman.io" | bash
```

## Bashrc Configuration

Add this compatibility shim to ensure SDKMAN works properly with MobaXterm:

```bash
# --- SDKMAN & MobaXterm/Cygwin compatibility shim -----------------

if [[ -d "$HOME/.sdkman" ]]; then
  PLATFORM_FILE="$HOME/.sdkman/var/platform"
  TARGET="windowsx64"
  [[ ! -f "$PLATFORM_FILE" || "$(cat "$PLATFORM_FILE")" != "$TARGET" ]] &&
      printf '%s\n' "$TARGET" > "$PLATFORM_FILE"
fi
```

## Key Features

The installation script provides:

- Automatic prerequisite checking
- Intelligent vendor selection (Zulu prioritized)
- Interactive menu for alternative distributions
- Default Java version selection
- Persistent `JAVA_HOME` configuration

This automates installation of Maven and multiple JDK versions (7, 8, 11, 17, 21) with Zulu distribution preference and fallback options.
