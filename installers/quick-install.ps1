# 🪺 Nido Quick Installer - Windows Edition
# Downloads only the binary. No repo cloning. Lightning fast.

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "  🪺 Nido Quick Install" -ForegroundColor Cyan
Write-Host "  Lightning-fast VM management" -ForegroundColor Cyan
Write-Host ""

# Detect architecture
$processorArch = $env:PROCESSOR_ARCHITECTURE
if ($env:PROCESSOR_ARCHITEW6432) {
    $processorArch = $env:PROCESSOR_ARCHITEW6432
}
$arch = switch -Regex ($processorArch) {
    "^(AMD64|x86_64)$" { "amd64"; break }
    "^ARM64$" { "arm64"; break }
    default { "386" }
}
if ($arch -ne "amd64") {
    Write-Host "❌ No pre-built release artifact for Windows/$arch." -ForegroundColor Red
    Write-Host "   Use the source installer from installers/build-from-source.sh on a supported shell." -ForegroundColor Gray
    exit 1
}

# Fetch latest release
Write-Host "🔍 Fetching latest release..." -ForegroundColor Cyan
$version = $env:NIDO_VERSION
if (-not $version) {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/Josepavese/nido/releases/latest"
        $version = $release.tag_name
        Write-Host "✅ Latest version: $version" -ForegroundColor Green
    } catch {
        Write-Host "❌ Failed to fetch latest release" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "✅ Requested version: $version" -ForegroundColor Green
}

# Build download URL
$archiveName = "nido-windows-$arch.zip"
$downloadUrl = "https://github.com/Josepavese/nido/releases/download/$version/$archiveName"
$checksumUrl = "https://github.com/Josepavese/nido/releases/download/$version/SHA256SUMS"

Write-Host "📥 Downloading $archiveName..." -ForegroundColor Cyan

$nidoHome = "$env:USERPROFILE\.nido"
$binDir = "$nidoHome\bin"
$targetPath = "$binDir\nido.exe"
$validatorPath = "$binDir\nido-validator.exe"
$registryDir = "$nidoHome\registry"

# Create directories
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\vms" | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\run" | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\images" | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\backups" | Out-Null
New-Item -ItemType Directory -Force -Path $registryDir | Out-Null

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nido-install-" + [System.Guid]::NewGuid().ToString("N"))
$archivePath = Join-Path $tempRoot $archiveName
$checksumPath = Join-Path $tempRoot "SHA256SUMS"
$extractDir = Join-Path $tempRoot "extract"
New-Item -ItemType Directory -Force -Path $tempRoot, $extractDir | Out-Null

function Verify-ReleaseChecksum {
    param(
        [string]$ChecksumUrl,
        [string]$ChecksumPath,
        [string]$ArchivePath,
        [string]$ArchiveName
    )
    try {
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile $ChecksumPath
    } catch {
        Write-Host "⚠️ SHA256SUMS not available for $version; skipping archive checksum verification." -ForegroundColor Yellow
        return
    }

    $line = Get-Content -LiteralPath $ChecksumPath | Where-Object { $_ -match "\s\*?$([Regex]::Escape($ArchiveName))$" } | Select-Object -First 1
    if (-not $line) {
        Write-Host "⚠️ $ArchiveName not listed in SHA256SUMS; skipping archive checksum verification." -ForegroundColor Yellow
        return
    }
    $expected = ($line -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $ArchivePath).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "Archive checksum mismatch for $ArchiveName"
    }
    Write-Host "✅ Archive checksum verified" -ForegroundColor Green
}

# Download and extract release archive
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath
    Verify-ReleaseChecksum -ChecksumUrl $checksumUrl -ChecksumPath $checksumPath -ArchivePath $archivePath -ArchiveName $archiveName
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force
    $packageDir = Join-Path $extractDir "nido-windows-$arch"
    $packageBinary = Join-Path $packageDir "nido.exe"
    if (-not (Test-Path $packageBinary)) {
        throw "Release archive does not contain nido.exe"
    }
    Copy-Item -LiteralPath $packageBinary -Destination $targetPath -Force
    $packageValidator = Join-Path $packageDir "nido-validator.exe"
    if (Test-Path $packageValidator) {
        Copy-Item -LiteralPath $packageValidator -Destination $validatorPath -Force
    }
    $packageRegistry = Join-Path $packageDir "registry"
    if (Test-Path $packageRegistry) {
        Copy-Item -Path (Join-Path $packageRegistry "*") -Destination $registryDir -Recurse -Force
    }
    Write-Host "✅ Binary installed to $targetPath" -ForegroundColor Green
} catch {
    Write-Host "❌ Download failed" -ForegroundColor Red
    exit 1
} finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}

# Download themes
$themesUrl = "https://raw.githubusercontent.com/Josepavese/nido/main/resources/themes.json"
$themesPath = "$nidoHome\themes.json"
Write-Host "🎨 Fetching visual themes..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $themesUrl -OutFile $themesPath
    Write-Host "✅ Themes installed to $themesPath" -ForegroundColor Green
} catch {
    Write-Host "⚠️ Failed to download themes (skipped)" -ForegroundColor Yellow
}

# Create default config if missing
$configPath = "$nidoHome\config.env"
if (-not (Test-Path $configPath)) {
    @"
# Nido Configuration
BACKUP_DIR=$env:USERPROFILE\.nido\backups
TEMPLATE_DEFAULT=template-headless
SSH_USER=vmuser
"@ | Out-File -FilePath $configPath -Encoding UTF8
    Write-Host "✅ Default config created" -ForegroundColor Green
}

# Add to PATH
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$binDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$binDir", "User")
    Write-Host "✅ Added to PATH (restart terminal to apply)" -ForegroundColor Green
}
$env:Path = "$env:Path;$binDir"

# Desktop Integration
Write-Host "🎨 Setting up Desktop Integration..." -ForegroundColor Cyan
$iconUrl = "https://raw.githubusercontent.com/Josepavese/nido/main/resources/nido.png"
$iconPath = "$nidoHome\nido.png"
try {
    Invoke-WebRequest -Uri $iconUrl -OutFile $iconPath
} catch {
    Write-Host "⚠️ Generic icon will be used (download failed)" -ForegroundColor Yellow
}

$shell = New-Object -ComObject WScript.Shell
$startMenu = [Environment]::GetFolderPath("Programs")
$shortcutPath = Join-Path $startMenu "Nido.lnk"
$shortcut = $shell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = "$binDir\nido.exe"
$shortcut.Arguments = "gui"
$shortcut.WorkingDirectory = "$nidoHome"
$shortcut.Description = "The Universal VM Nest"
if (Test-Path $iconPath) {
    $shortcut.IconLocation = "$iconPath"
}
$shortcut.Save()
Write-Host "✅ Start Menu shortcut created" -ForegroundColor Green

# --- Dependency Check & Proactive Install ---
Write-Host "🔍 Checking flight readiness (dependencies)..." -ForegroundColor Cyan
$qemuInstalled = $false
try {
    if (Get-Command "qemu-system-x86_64" -ErrorAction SilentlyContinue) { $qemuInstalled = $true }
    elseif (Get-Command "qemu-system-aarch64" -ErrorAction SilentlyContinue) { $qemuInstalled = $true }
    elseif (Get-Command "qemu-system" -ErrorAction SilentlyContinue) { $qemuInstalled = $true }
} catch {}

$isoToolInstalled = $false
try {
    if (Get-Command "mkisofs" -ErrorAction SilentlyContinue) { $isoToolInstalled = $true }
    elseif (Get-Command "genisoimage" -ErrorAction SilentlyContinue) { $isoToolInstalled = $true }
    elseif (Get-Command "xorriso" -ErrorAction SilentlyContinue) { $isoToolInstalled = $true }
} catch {}

if (-not $qemuInstalled) {
    Write-Host "⚠️  QEMU is missing. Nido needs it to hatch VMs." -ForegroundColor Yellow
    $response = Read-Host "📦 Would you like to install QEMU dependencies automatically via winget? (y/N)"
    if ($response -eq "y") {
        Write-Host "🛠️  Installing QEMU via winget..." -ForegroundColor Cyan
        winget install --id SoftwareFreedomConservancy.QEMU -e --scope machine --accept-package-agreements --accept-source-agreements
        Write-Host "💡 Note: You might need to restart your terminal for QEMU to be in your PATH." -ForegroundColor Yellow
        $qemuInstalled = $true
    } else {
        Write-Host "💡 Skipping automatic installation. You'll need to install it manually." -ForegroundColor Gray
        Write-Host "   QEMU Windows options: https://www.qemu.org/download/#windows" -ForegroundColor Gray
    }
} else {
    Write-Host "✅ QEMU is already present." -ForegroundColor Green
}

if (-not $isoToolInstalled) {
    Write-Host "⚠️  ISO creation tool missing (mkisofs/genisoimage)." -ForegroundColor Yellow
    Write-Host "   Cloud-init seed generation will fail without it." -ForegroundColor Yellow
    
    if (Get-Command "choco" -ErrorAction SilentlyContinue) {
        $response = Read-Host "📦 Would you like to install cdrtools via Chocolatey? (y/N)"
        if ($response -eq "y") {
            Write-Host "🛠️  Installing cdrtools via Chocolatey..." -ForegroundColor Cyan
            choco install cdrtools -y
            Write-Host "✅ cdrtools installed." -ForegroundColor Green
            $isoToolInstalled = $true
        } else {
             Write-Host "💡 Skipping automatic installation." -ForegroundColor Gray
        }
    } else {
        Write-Host "   Recommended: Install 'cdrtools' via Chocolatey or Scoop manually." -ForegroundColor Gray
    }
} else {
    Write-Host "✅ ISO creation tools are present." -ForegroundColor Green
}

Write-Host ""
Write-Host "🎉 Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor White
Write-Host "  1. Restart your terminal" -ForegroundColor Cyan
Write-Host "  2. Verify install: " -NoNewline; Write-Host "nido version" -ForegroundColor Cyan
Write-Host "  3. Check system: " -NoNewline; Write-Host "nido doctor" -ForegroundColor Cyan
Write-Host ""

if ($qemuInstalled -and $isoToolInstalled) {
    Write-Host "✨ All systems go! You are ready to fly!" -ForegroundColor Green
} else {
    Write-Host "💡 Note: Missing dependencies may limit functionality." -ForegroundColor Yellow
}

Write-Host "💡 Pro Tip: Ensure 'Windows Hypervisor Platform' is enabled in Windows Features for max speed!" -ForegroundColor Magenta
Write-Host ""
Write-Host '"It''s not a VM, it''s a lifestyle." 🪺' -ForegroundColor White
