# Configuration
$repo = "benoitpetit/prompt-my-project"
$binaryName = "pmp"
$installDir = "$env:LOCALAPPDATA\Programs\pmp"

# Check TLS 1.2
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Check Internet access
Write-Host "Checking GitHub connection..."
try {
    $response = Invoke-WebRequest -Uri "https://github.com" -UseBasicParsing -Method Head -TimeoutSec 10
    if ($response.StatusCode -ne 200) {
        throw "GitHub unreachable"
    }
} catch {
    Write-Host "Error: No Internet connection or GitHub is unreachable"
    exit 1
}

# Check write permissions
try {
    New-Item -ItemType Directory -Force -Path "$env:TEMP\pmp_test" -ErrorAction Stop | Remove-Item -Force
} catch {
    Write-Host "Error: Insufficient permissions"
    exit 1
}

# Architecture detection
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Create installation directory
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}

# Get the latest version from GitHub
Write-Host "Retrieving the latest version..."
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -TimeoutSec 10
    
    # Display debug information
    Write-Host "Version information received:"
    Write-Host "Response type: $($release.GetType().Name)"
    
    $version = $release.tag_name

    if (-not $version) {
        # Try an alternative approach if tag_name property is not found
        if ($release -is [string]) {
            # If the response is a string, try to parse it manually
            if ($release -match '"tag_name":\s*"([^"]+)"') {
                $version = $matches[1]
            }
        }
        
        if (-not $version) {
            throw "Version not found"
        }
    }
    
    # Display version for debugging
    Write-Host "Extracted version: $version"
    
    # Make sure the version is in the correct format for the download URL
    $versionWithoutV = $version -replace '^v', ''  # Remove 'v' at the beginning if present

    # Build download URLs
    $downloadUrl = "https://github.com/$repo/releases/download/$version/${binaryName}_${version}_windows_${arch}.zip"
    $altDownloadUrl = "https://github.com/$repo/releases/download/$version/${binaryName}_${versionWithoutV}_windows_${arch}.zip"
    Write-Host "Main URL: $downloadUrl"
    Write-Host "Alternative URL: $altDownloadUrl"

    # Check if the download URL exists
    $urlFound = $false
    $finalUrl = ""

    try {
        $response = Invoke-WebRequest -Uri $downloadUrl -Method Head -UseBasicParsing -TimeoutSec 10
        $finalUrl = $downloadUrl
        $urlFound = $true
        Write-Host "URL found: $downloadUrl"
    } catch {
        Write-Host "Main URL not found, trying alternative URL..."
        try {
            $response = Invoke-WebRequest -Uri $altDownloadUrl -Method Head -UseBasicParsing -TimeoutSec 10
            $finalUrl = $altDownloadUrl
            $urlFound = $true
            Write-Host "Alternative URL found: $altDownloadUrl"
        } catch {
            Write-Host "Error: The binary for your system (windows/$arch) doesn't exist in version $version"
            Write-Host "URLs not found:"
            Write-Host "- $downloadUrl"
            Write-Host "- $altDownloadUrl"
            exit 1
        }
    }
    
} catch {
    Write-Host "Error: Unable to retrieve the latest version"
    Write-Host "Error details: $_"
    exit 1
}

# Download and extract the archive
$zipPath = Join-Path $env:TEMP "pmp.zip"
Write-Host "Downloading from $finalUrl..."
try {
    Invoke-WebRequest -Uri $finalUrl -OutFile $zipPath -TimeoutSec 60
} catch {
    Write-Host "Error during download: $_"
    exit 1
}

# Create a temporary directory for extraction before copying to the final directory
$extractDir = Join-Path $env:TEMP "pmp_extract"
if (Test-Path $extractDir) {
    Remove-Item -Path $extractDir -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $extractDir | Out-Null

Write-Host "Extracting archive..."
try {
    Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force
} catch {
    Write-Host "Error extracting the archive: $_"
    exit 1
}

# Search for the binary in the extraction directory
Write-Host "Searching for binary..."
$binaryPath = Get-ChildItem -Path $extractDir -Recurse -Filter "$binaryName.exe" | Select-Object -First 1 -ExpandProperty FullName

if (-not $binaryPath) {
    Write-Host "Error: Binary $binaryName.exe not found after extraction"
    Write-Host "Extraction directory contents:"
    Get-ChildItem -Path $extractDir -Recurse | ForEach-Object { Write-Host "  $($_.FullName)" }
    exit 1
}

Write-Host "Binary found: $binaryPath"

# Copy the binary and associated files to the installation directory
Write-Host "Installing to $installDir..."
try {
    # Make sure installation directory is empty
    if (Test-Path $installDir) {
        Get-ChildItem -Path $installDir -Recurse | Remove-Item -Force -Recurse
    } else {
        New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    }
    
    # Copy binary
    Copy-Item -Path $binaryPath -Destination (Join-Path $installDir "$binaryName.exe") -Force
    
    # Copy other useful files (README, LICENSE, etc.) if they exist
    $docFiles = @("README.md", "LICENSE")
    foreach ($file in $docFiles) {
        $docPath = Get-ChildItem -Path $extractDir -Recurse -Filter $file | Select-Object -First 1 -ExpandProperty FullName
        if ($docPath) {
            Copy-Item -Path $docPath -Destination $installDir -Force
        }
    }
} catch {
    Write-Host "Error during installation: $_"
    exit 1
}

# Add to PATH if necessary
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    try {
        [Environment]::SetEnvironmentVariable(
            "Path",
            "$userPath;$installDir",
            "User"
        )
        Write-Host "Installation directory added to user PATH"
    } catch {
        Write-Host "Warning: Unable to modify PATH variable: $_"
    }
}

# Cleanup
Remove-Item $zipPath -Force -ErrorAction SilentlyContinue

Write-Host "✅ Installation complete! Please restart your terminal to use 'pmp'"
Write-Host "Use 'pmp --help' to see available options."
