# Configuration
$repo = "benoitpetit/prompt-my-project"
$binaryName = "pmp"
$installDir = "$env:LOCALAPPDATA\Programs\pmp"

# Vérifier TLS 1.2
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Vérifier l'accès Internet
try {
    Invoke-RestMethod -Uri "https://api.github.com" -Method Head
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
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Obtenir la dernière version depuis GitHub
Write-Host "Récupération de la dernière version..."
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
$version = $release.tag_name

if (-not $version) {
    Write-Host "Erreur: Impossible de récupérer la dernière version"
    exit 1
}

# Construire l'URL de téléchargement
$downloadUrl = "https://github.com/$repo/releases/download/$version/${binaryName}_${version}_windows_${arch}.zip"

# Télécharger et extraire l'archive
$zipPath = Join-Path $env:TEMP "pmp.zip"
Write-Host "Téléchargement de $downloadUrl..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath
Expand-Archive -Path $zipPath -DestinationPath $installDir -Force

# Ajouter au PATH si nécessaire
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$userPath;$installDir",
        "User"
    )
}

# Nettoyage
Remove-Item $zipPath

Write-Host "Installation terminée ! Veuillez redémarrer votre terminal pour utiliser 'pmp'"
Write-Host "Utilisez 'pmp --help' pour voir les options disponibles."
