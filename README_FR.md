# Prompt My Project (PMP)

[README IN ENGLISH](/README.md) 


<p align="center">
    <img src="./logo.png" alt="Prompt My Project" width="800">
    <p align="center">Outil en ligne de commande pour générer des prompts structurés à partir de votre code source, optimisés pour les assistants IA</p>
</p>

<div align="center">
    <a href="https://github.com/benoitpetit/prompt-my-project/blob/main/LICENSE">
        <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT">
    </a>
    <a href="https://github.com/benoitpetit/prompt-my-project/releases">
        <img src="https://img.shields.io/github/v/release/benoitpetit/prompt-my-project" alt="Latest Release">
    </a>
    <a href="https://opensource.org">
        <img src="https://img.shields.io/badge/Open%20Source-%E2%9D%A4-brightgreen" alt="Open Source Love">
    </a>
    <a href="https://github.com/benoitpetit/prompt-my-project/stargazers">
        <img src="https://img.shields.io/github/stars/benoitpetit/prompt-my-project" alt="GitHub Stars">
    </a>
    <a href="https://golang.org/dl/">
        <img src="https://img.shields.io/badge/Go-%3E%3D%201.21-blue.svg" alt="Go Version">
    </a>
</div>

## ✨ Fonctionnalités

<div align="center">
    <table>
        <tr>
            <td align="center">📂</td>
            <td><strong>Navigation Intelligente</strong><br/>Analyse récursivement la structure de votre projet</td>
            <td align="center">🔍</td>
            <td><strong>Détection des Binaires</strong><br/>Identifie et évite intelligemment les fichiers binaires</td>
        </tr>
        <tr>
            <td align="center">🎯</td>
            <td><strong>Filtrage par Motifs</strong><br/>Prend en charge les motifs avancés d'inclusion/exclusion</td>
            <td align="center">⚡</td>
            <td><strong>Compatible Git</strong><br/>Respecte les règles .gitignore de votre projet</td>
        </tr>
        <tr>
            <td align="center">📊</td>
            <td><strong>Contrôle de Taille</strong><br/>Options flexibles de filtrage par taille de fichier</td>
            <td align="center">🚀</td>
            <td><strong>Haute Performance</strong><br/>Traitement concurrent avec pools de workers</td>
        </tr>
        <tr>
            <td align="center">💾</td>
            <td><strong>Mise en Cache Intelligente</strong><br/>Mise en cache optimisée du contenu des fichiers</td>
            <td align="center">📝</td>
            <td><strong>Statistiques Détaillées</strong><br/>Métriques complètes sur votre projet</td>
        </tr>
        <tr>
            <td align="center">🔬</td>
            <td><strong>Détection de Technologies</strong><br/>Identifie automatiquement les technologies utilisées</td>
            <td align="center">🎯</td>
            <td><strong>Analyse des Fichiers Clés</strong><br/>Identifie les fichiers les plus importants du projet</td>
        </tr>
        <tr>
            <td align="center">⚠️</td>
            <td><strong>Détection de Problèmes</strong><br/>Identifie les problèmes potentiels de qualité du code</td>
            <td align="center">📊</td>
            <td><strong>Métriques de Complexité</strong><br/>Analyse avancée de la complexité du code</td>
        </tr>
        <tr>
            <td align="center">🔄</td>
            <td><strong>Formats de Sortie Multiples</strong><br/>Export en TXT, JSON ou XML avec contenu complet</td>
            <td align="center">🔌</td>
            <td><strong>Intégration API</strong><br/>Formats structurés pour usage programmatique</td>
        </tr>
    </table>
</div>

## 🧠 Fonctionnalités Avancées

### Cache de Détection des Binaires

PMP maintient un cache persistant des résultats de détection de fichiers binaires pour améliorer les performances :
- Cache stocké dans `~/.pmp/cache/binary_cache.json`
- Utilise les métadonnées des fichiers (taille, heure de modification) comme clés de cache
- Gère automatiquement l'invalidation du cache
- Utilise un répertoire temporaire si le répertoire personnel n'est pas disponible

### Estimation Intelligente des Tokens

L'estimateur de tokens utilise un algorithme sophistiqué qui :
- Différencie le contenu code et texte (4 caractères/token pour le code, 5 pour le texte)
- Applique des poids spéciaux aux caractères de syntaxe (accolades, sauts de ligne, etc.)
- Traite les grands fichiers en streaming pour éviter les problèmes de mémoire
- Fournit des estimations précises pour les limites de contexte des modèles IA

### Détection Intelligente des Technologies

- Identifie automatiquement les langages de programmation et frameworks
- Utilise à la fois les extensions de fichiers et les fichiers de configuration spécifiques
- Détecte les outils courants et les configurations CI/CD
- Fournit des informations sur la stack technologique du projet

### Analyse de Complexité du Code

Effectue une analyse statique avancée :
- Distribution de la taille des fichiers et du nombre de lignes
- Analyse de la profondeur des répertoires (signale >5 niveaux de profondeur)
- Métriques de modularité du code (signale les fichiers >100KB)
- Identifie les problèmes potentiels de maintenance (fichiers >500 lignes)

### Gestion des Erreurs

- Système de réessai automatique pour les fichiers problématiques
- Stratégie de backoff intelligente (délai de 100ms entre les tentatives)
- Limites de réessai basées sur la taille (max 1MB pour les tentatives de réessai)
- Maximum 3 tentatives de réessai par fichier
- Dégradation gracieuse pour les gros fichiers

### Optimisations de Performance

- Traitement concurrent des fichiers avec pools de workers
- Streaming efficace en mémoire pour les gros fichiers
- Mise en cache intelligente du contenu des fichiers et de la détection des binaires
- Limites d'affichage adaptatives pour les grands répertoires
- Génération efficace de la structure arborescente

## 🔧 Configuration par Défaut

| Paramètre      | Valeur    | Description                  |
| ------------- | --------- | ---------------------------- |
| Taille Min    | 1KB       | Taille minimale des fichiers |
| Taille Max    | 100MB     | Taille maximale des fichiers |
| Dossier Sortie| ./prompts | Répertoire de sortie pour les prompts |
| GitIgnore     | true      | Respecter les règles .gitignore |
| Workers       | Cœurs CPU | Nombre de workers parallèles |
| Max Fichiers  | 500       | Nombre maximum de fichiers   |
| Taille Totale Max| 10MB   | Taille totale maximale du projet |
| Limite Réessai| 1MB       | Taille maximale pour les réessais |
| Barre Progrès | 40 chars  | Largeur de l'indicateur de progression |
| Format Sortie | txt       | Format de sortie (txt/json/xml) |

## 📂 Organisation de la Sortie

PMP génère un fichier de prompt bien structuré qui inclut :

- Informations et statistiques du projet
- Visualisation complète de la structure des fichiers
- Contenu formaté des fichiers
- Estimations du nombre de tokens et de caractères

Les prompts sont automatiquement sauvegardés dans :

- `./prompts/` (par défaut, automatiquement ajouté au .gitignore)
- Ou dans le dossier spécifié par `--output`

Les fichiers sont nommés selon un format horodaté : `prompt_YYYYMMDD_HHMMSS.txt`

## 🎯 Artefacts de Build

Format : `pmp_<version>_<os>_<arch>.<ext>`
Exemple : `pmp_v1.0.0_linux_amd64.tar.gz`

### Architectures Supportées

- amd64 (x86_64)
- arm64 (aarch64)

### Systèmes Supportés

- Linux (.tar.gz)
- macOS/Darwin (.tar.gz)
- Windows (.zip)

### Processus de Build

Le script de build :
- Détecte automatiquement la version à partir des tags git
- Génère des binaires pour toutes les plateformes supportées
- Crée des archives compressées (.tar.gz pour Unix, .zip pour Windows)
- Inclut README et LICENSE dans chaque archive
- Génère des checksums SHA-256 pour tous les artefacts

Les artefacts de build sont placés dans le répertoire `dist` :
```
dist/
├── pmp_v1.0.0_linux_amd64.tar.gz
├── pmp_v1.0.0_linux_arm64.tar.gz
├── pmp_v1.0.0_darwin_amd64.tar.gz
├── pmp_v1.0.0_darwin_arm64.tar.gz
├── pmp_v1.0.0_windows_amd64.zip
├── pmp_v1.0.0_windows_arm64.zip
└── checksums.txt
```

## 🚀 Installation

### macOS & Linux

```bash
curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/refs/heads/master/scripts/install.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/benoitpetit/prompt-my-project/refs/heads/master/scripts/install.ps1 | iex
```

## 🗑️ Désinstallation

### macOS & Linux

```bash
curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/refs/heads/master/scripts/remove.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/benoitpetit/prompt-my-project/refs/heads/master/scripts/remove.ps1 | iex
```

## 🛠️ Utilisation

### Syntaxe de Base

```bash
pmp [options] [chemin]
```

### Options Disponibles

| Option           | Alias | Description                          | Défaut |
| ---------------- | ----- | ------------------------------------ | ------- |
| `--include`      | `-i`  | Inclure uniquement les fichiers correspondant aux motifs | - |
| `--exclude`      | `-e`  | Exclure les fichiers correspondant aux motifs | - |
| `--min-size`     | -     | Taille minimale des fichiers         | 1KB |
| `--max-size`     | -     | Taille maximale des fichiers         | 100MB |
| `--no-gitignore` | -     | Ignorer le fichier .gitignore        | false |
| `--output`       | `-o`  | Dossier de sortie pour le fichier prompt | ./prompts |
| `--workers`      | -     | Nombre de workers parallèles         | Cœurs CPU |
| `--max-files`    | -     | Nombre maximum de fichiers (0 = illimité) | 500 |
| `--max-total-size` | -   | Taille totale maximale (0 = illimitée) | 10MB |
| `--format`       | `-f`  | Format de sortie (txt, json, ou xml) | txt |
| `--help`         | -     | Afficher l'aide                      | - |
| `--version`      | -     | Afficher la version                  | - |

### Exemples de Motifs

- `*.go` - Tous les fichiers Go
- `src/` - Tous les fichiers dans le répertoire src
- `test/*` - Tous les fichiers dans le répertoire test
- `*.{js,ts}` - Tous les fichiers JavaScript et TypeScript
- `!vendor/*` - Exclure tous les fichiers dans le répertoire vendor

### Exemples Rapides

```bash
# Analyser le répertoire courant
pmp .

# Analyser un chemin de projet spécifique
pmp /chemin/vers/votre/projet

# Générer une sortie JSON
pmp . --format json

# Générer une sortie XML
pmp . --format xml

# Filtrer par types de fichiers
pmp . -i "*.go" -i "*.md"

# Exclure les fichiers de test et le répertoire vendor
pmp . -e "test/*" -e "vendor/*"

# Personnaliser les limites de taille
pmp . --min-size 500B --max-size 1MB

# Contrôler la taille totale du projet
pmp . --max-total-size 50MB --max-files 1000

# Spécifier le répertoire de sortie
pmp . -o ~/prompts

# Ignorer les règles .gitignore
pmp . --no-gitignore

# Ajuster le nombre de workers
pmp . --workers 4

# Combiner plusieurs options
pmp . -i "*.{js,ts}" -e "node_modules/*" --max-size 500KB -o ./analysis
```

## 🚄 Performance

PMP utilise plusieurs stratégies d'optimisation pour maximiser les performances :

### Traitement Concurrent

- Pool de workers adaptatif basé sur les cœurs CPU disponibles
- Traitement parallèle des fichiers avec gestion efficace de la mémoire
- Suivi de progression en temps réel avec limitation du taux de rafraîchissement
- Gestion intelligente des erreurs avec mécanisme de réessai

### Gestion de la Mémoire

- Tampons réutilisables pour la lecture des fichiers
- Streaming des gros fichiers pour éviter la saturation de la mémoire
- Limites de taille configurables pour les fichiers individuels et le total
- Nettoyage automatique des fichiers temporaires

### Système de Cache

- Cache persistant pour la détection des fichiers binaires
- Mise en cache du contenu des fichiers pour éviter les lectures multiples
- Clés de cache basées sur les métadonnées des fichiers
- Repli automatique vers un cache temporaire si nécessaire

### Optimisations d'Affichage

- Limites adaptatives sur les fichiers affichés par répertoire
- Réduction progressive de la verbosité pour les grands répertoires
- Barre de progression optimisée avec mises à jour limitées
- Formatage intelligent des durées et des tailles

### Limites et Seuils

- Taille maximale des fichiers pour les réessais : 1MB
- Délai entre les réessais : 100ms
- Réessais maximum : 3
- Taille maximale pour l'analyse de complexité : 5MB
- Taille maximale pour le comptage des lignes : 10MB

## 🔧 Compilation depuis les sources

### Prérequis

- Go 1.21 ou supérieur
- Git

### Étapes de Compilation

```bash
# Cloner le dépôt
git clone https://github.com/benoitpetit/prompt-my-project.git
cd prompt-my-project

# Installer les dépendances
go mod tidy

# Compiler
./scripts/build.sh

# Exécuter
go run main.go [options] [chemin]
```

## 🛠️ Configuration Avancée

### Exemple d'Intégration CI/CD

```yaml
generate_ia_prompt:
  stage: analysis
  image: golang:1.21
  script:
    - curl -sSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/install.sh | bash
    - pmp --output ./artifacts/prompts
  artifacts:
    paths:
      - ./artifacts/prompts/
```

## ⚙️ Sous le Capot

### Architecture de Traitement Concurrent

- **Pool de Workers** : Utilise un pool de workers adaptatif basé sur les ressources système disponibles
- **Cache Intelligent** : Mise en cache du contenu des fichiers pour éviter les lectures multiples
- **Gestion de la Mémoire** : Utilisation de tampons réutilisables pour l'analyse des fichiers

### Détection des Fichiers Binaires

Combinaison de trois méthodes pour une identification précise :

1. Analyse des extensions (.png, .exe, ..)
2. Vérification du type MIME
3. Détection des caractères non-texte

### Fonctionnalités d'Analyse Avancées

- **Détection de Technologies** : Identifie automatiquement les langages de programmation, frameworks et outils utilisés dans votre projet en fonction des extensions de fichiers et des fichiers de configuration spécifiques.
- **Analyse des Fichiers Clés** : Identifie les fichiers les plus importants de votre projet en fonction de :
  - Noms de fichiers importants courants (main.go, index.js, etc.)
  - Emplacement dans la structure du projet (niveau racine, src/, etc.)
  - Type et objectif du fichier (configuration, documentation, etc.)
- **Analyse de la Qualité du Code** : Détecte les problèmes potentiels tels que :
  - Gros fichiers (>100KB) qui pourraient nécessiter une modularisation
  - Fichiers longs (>500 lignes) qui pourraient être difficiles à maintenir
  - Répertoires profondément imbriqués (>5 niveaux) suggérant une structure complexe
- **Métriques de Complexité** : Fournit une analyse avancée du code :
  - Total et moyenne des lignes de code
  - Distribution des tailles de fichiers et percentiles
  - Statistiques basées sur les extensions avec pourcentages
  - Répartition de l'utilisation des technologies

### Estimation des Tokens

PMP utilise un système sophistiqué d'estimation des tokens qui :
- Différencie le contenu code et texte
- Prend en compte les caractères spéciaux et la syntaxe
- Fournit des comptages précis de tokens pour les limites de contexte des modèles IA
- Utilise un streaming efficace pour les gros fichiers

## 📋 Exemple de Sortie de Prompt

PMP prend en charge trois formats de sortie :

### Format TXT

```text
INFORMATIONS DU PROJET
-----------------------------------------------------
• Nom du Projet : prompt-my-project
• Généré le : 2025-03-13 22:30:40
• Généré avec : Prompt My Project (PMP) v1.0.0
• Hôte : bigmaster
• OS : linux/amd64

TECHNOLOGIES DÉTECTÉES
-----------------------------------------------------
• Go
• Go Modules

FICHIERS CLÉS
-----------------------------------------------------
Ces fichiers sont probablement les plus importants pour comprendre le projet :
• LICENSE
• README.md
• go.sum
• main.go

POINTS D'INTÉRÊT
-----------------------------------------------------
Ces éléments peuvent mériter une attention particulière lors de l'analyse :
• 25.0% des fichiers (1) contiennent plus de 500 lignes, ce qui peut rendre le code difficile à maintenir

STATISTIQUES DES FICHIERS
-----------------------------------------------------
• Total des Fichiers : 4
• Taille Totale : 75 kB
• Taille Moyenne des Fichiers : 19 kB
• Total des Lignes de Code : 2495
• Moyenne de Lignes par Fichier : 623
• Médiane des Lignes par Fichier : 352
• 90% des fichiers ont moins de 2078 lignes
• Principales Extensions par Taille :
  - .go : 59 kB
  - .md : 11 kB
  - .sum : 4.0 kB
• Types de Fichiers :
  - <sans-extension> : 1 fichiers (25.0%)
  - .md : 1 fichiers (25.0%)
  - .sum : 1 fichiers (25.0%)
  - .go : 1 fichiers (25.0%)

SUGGESTIONS D'ANALYSE
-----------------------------------------------------
Lors de l'analyse de ce projet, considérez les approches suivantes :
• Pour un projet utilisant Go, Go Modules, examinez les modèles et pratiques typiques de ces technologies
• Commencez par analyser les fichiers clés identifiés, qui contiennent probablement la logique principale
• Portez une attention particulière aux points d'intérêt identifiés, qui peuvent révéler des problèmes ou des opportunités d'amélioration
• Le projet contient de gros fichiers. Recherchez des opportunités de modularisation et de séparation des responsabilités
• Recherchez les modèles de conception utilisés et évaluez s'ils sont implémentés efficacement
• Identifiez les zones potentielles de dette technique ou d'optimisation

STATISTIQUES DES TOKENS
-----------------------------------------------------
• Nombre Estimé de Tokens : 23992
• Nombre de Caractères : 75537

=====================================================

STRUCTURE DU PROJET :
-----------------------------------------------------

└── prompt-my-project (4 fichiers)
    ├── LICENSE
    ├── README.md
    ├── go.sum
    └── main.go

CONTENU DES FICHIERS :
-----------------------------------------------------

================================================
Fichier : LICENSE
================================================
MIT License

Copyright (c) 2025 [Benoît PETIT]

Permission is hereby granted, free of char....
```

### Format JSON
Sortie structurée au format JSON, parfaite pour le traitement programmatique :

```json
{
  "project_info": {
    "name": "my-project",
    "generated_at": "2025-03-13T22:30:40Z",
    "generator": "Prompt My Project (PMP) v1.0.0",
    "host": "hostname",
    "os": "linux/amd64"
  },
  "technologies": ["Go", "Go Modules"],
  "key_files": ["main.go", "README.md"],
  "issues": [
    "25.0% des fichiers (1) contiennent plus de 500 lignes, ce qui peut rendre le code difficile à maintenir"
  ],
  "statistics": {
    "file_count": 4,
    "total_size": 75000,
    "total_size_human": "75 kB",
    "avg_file_size": 18750,
    "token_count": 23992,
    "char_count": 75537,
    "files_per_second": 12.5
  },
  "file_types": [
    {
      "extension": ".go",
      "count": 1
    },
    {
      "extension": ".md",
      "count": 1
    }
  ],
  "files": [
    {
      "path": "main.go",
      "size": 25000,
      "content": "package main\n\nimport ...",
      "language": "Go"
    },
    {
      "path": "README.md",
      "size": 15000,
      "content": "# My Project\n\nDescription...",
      "language": "Markdown"
    }
  ]
}
```

### Format XML
Sortie structurée au format XML, adaptée à l'intégration avec des outils basés sur XML :

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project>
  <project_info>
    <n>my-project</n>
    <generated_at>2025-03-13T22:30:40Z</generated_at>
    <generator>Prompt My Project (PMP) v1.0.0</generator>
    <host>hostname</host>
    <os>linux/amd64</os>
  </project_info>
  <technologies>
    <technology>Go</technology>
    <technology>Go Modules</technology>
  </technologies>
  <key_files>
    <file>main.go</file>
    <file>README.md</file>
  </key_files>
  <issues>
    <issue>25.0% des fichiers (1) contiennent plus de 500 lignes, ce qui peut rendre le code difficile à maintenir</issue>
  </issues>
  <statistics>
    <file_count>4</file_count>
    <total_size>75000</total_size>
    <total_size_human>75 kB</total_size_human>
    <avg_file_size>18750</avg_file_size>
    <token_count>23992</token_count>
    <char_count>75537</char_count>
    <files_per_second>12.5</files_per_second>
  </statistics>
  <file_types>
    <type extension=".go">1</type>
    <type extension=".md">1</type>
  </file_types>
  <files>
    <file>
      <path>main.go</path>
      <size>25000</size>
      <content>package main

import ...</content>
      <language>Go</language>
    </file>
    <file>
      <path>README.md</path>
      <size>15000</size>
      <content># My Project

Description...</content>
      <language>Markdown</language>
    </file>
  </files>
</project>
```

## 🔌 Cas d'Utilisation des Formats

Chaque format de sortie sert des objectifs différents :

### Format Texte (TXT)
- **Idéal pour** : Utilisation directe avec des assistants IA comme ChatGPT, Claude ou Gemini
- **Avantages** : Lisible par l'homme, bien structuré, facile à copier-coller
- **À utiliser quand** : Vous voulez analyser rapidement un projet avec un assistant IA

### Format JSON
- **Idéal pour** : Traitement programmatique, extraction de données et intégration avec d'autres outils
- **Avantages** : Facile à analyser, données structurées, compatible avec la plupart des langages de programmation
- **À utiliser quand** : Construction d'outils d'automatisation, intégration avec des pipelines CI/CD ou création d'outils d'analyse personnalisés

### Format XML
- **Idéal pour** : Intégration avec des systèmes d'entreprise et des outils basés sur XML
- **Avantages** : Structure hiérarchique, compatible avec les outils de traitement XML
- **À utiliser quand** : Travail avec des systèmes qui attendent une entrée XML ou lors de l'utilisation de transformations XSLT

### Exemples d'Intégration

```bash
# Générer du JSON et traiter avec jq
pmp . --format json | jq '.statistics.token_count'

# Générer du XML et transformer avec XSLT
pmp . --format xml > project.xml && xsltproc transform.xslt project.xml > report.html

# Utiliser dans un pipeline CI/CD
pmp . --format json --output ./artifacts/analysis
```

## 📄 Licence

Ce projet est sous licence MIT - voir le fichier [LICENSE](LICENSE) pour plus de détails.
