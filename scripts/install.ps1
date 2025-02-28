# Configuration
$repo = "benoitpetit/prompt-my-project"
$binaryName = "pmp"
$installDir = "$env:LOCALAPPDATA\Programs\pmp"

# Vérifier TLS 1.2
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Vérifier l'accès Internet
Write-Host "Vérification de la connexion à GitHub..."
try {
    $response = Invoke-WebRequest -Uri "https://github.com" -UseBasicParsing -Method Head -TimeoutSec 10
    if ($response.StatusCode -ne 200) {
        throw "GitHub inaccessible"
    }
} catch {
    Write-Host "Erreur: Pas de connexion Internet ou GitHub inaccessible"
    exit 1
}

# Vérifier les permissions d'écriture
try {
    New-Item -ItemType Directory -Force -Path "$env:TEMP\pmp_test" -ErrorAction Stop | Remove-Item -Force
} catch {
    Write-Host "Erreur: Permissions insuffisantes"
    exit 1
}

# Détection de l'architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Création du dossier d'installation
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}

# Obtenir la dernière version depuis GitHub
Write-Host "Récupération de la dernière version..."
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -TimeoutSec 10
    $version = $release.tag_name

    if (-not $version) {
        throw "Version introuvable"
    }
} catch {
    Write-Host "Erreur: Impossible de récupérer la dernière version"
    exit 1
}

# Construire l'URL de téléchargement
$downloadUrl = "https://github.com/$repo/releases/download/$version/${binaryName}_${version}_windows_${arch}.zip"
Write-Host "Version détectée: $version"
Write-Host "Architecture: windows/$arch"

# Télécharger et extraire l'archive
$zipPath = Join-Path $env:TEMP "pmp.zip"
Write-Host "Téléchargement depuis $downloadUrl..."
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -TimeoutSec 60
} catch {
    Write-Host "Erreur lors du téléchargement: $_"
    exit 1
}

try {
    Expand-Archive -Path $zipPath -DestinationPath $installDir -Force
} catch {
    Write-Host "Erreur lors de l'extraction de l'archive: $_"
    exit 1
}

# Ajouter au PATH si nécessaire
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    try {
        [Environment]::SetEnvironmentVariable(
            "Path",
            "$userPath;$installDir",
            "User"
        )
        Write-Host "Dossier d'installation ajouté au PATH utilisateur"
    } catch {
        Write-Host "Avertissement: Impossible de modifier la variable PATH: $_"
    }
}

# Nettoyage
Remove-Item $zipPath -Force -ErrorAction SilentlyContinue

Write-Host "✅ Installation terminée ! Veuillez redémarrer votre terminal pour utiliser 'pmp'"
Write-Host "Utilisez 'pmp --help' pour voir les options disponibles."
