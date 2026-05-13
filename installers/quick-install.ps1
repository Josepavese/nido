# Nido Quick Installer - Windows Edition
# Downloads only the binary. No repo cloning. Lightning fast.

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "  Nido Quick Install" -ForegroundColor Cyan
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
    Write-Host "[ERROR] No pre-built release artifact for Windows/$arch." -ForegroundColor Red
    Write-Host "   Use the source installer from installers/build-from-source.sh on a supported shell." -ForegroundColor Gray
    exit 1
}

# Fetch latest release
Write-Host "[INFO] Fetching latest release..." -ForegroundColor Cyan
$version = $env:NIDO_VERSION
if (-not $version) {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/Josepavese/nido/releases/latest"
        $version = $release.tag_name
        Write-Host "[OK] Latest version: $version" -ForegroundColor Green
    } catch {
        Write-Host "[ERROR] Failed to fetch latest release" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "[OK] Requested version: $version" -ForegroundColor Green
}

# Build download URL
$archiveName = "nido-windows-$arch.zip"
$downloadUrl = "https://github.com/Josepavese/nido/releases/download/$version/$archiveName"
$checksumUrl = "https://github.com/Josepavese/nido/releases/download/$version/SHA256SUMS"

Write-Host "[DOWNLOAD] Downloading $archiveName..." -ForegroundColor Cyan

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
        Write-Host "[WARN] SHA256SUMS not available for $version; skipping archive checksum verification." -ForegroundColor Yellow
        return
    }

    $line = Get-Content -LiteralPath $ChecksumPath | Where-Object { $_ -match "\s\*?$([Regex]::Escape($ArchiveName))$" } | Select-Object -First 1
    if (-not $line) {
        Write-Host "[WARN] $ArchiveName not listed in SHA256SUMS; skipping archive checksum verification." -ForegroundColor Yellow
        return
    }
    $expected = ($line -split '\s+')[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $ArchivePath).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "Archive checksum mismatch for $ArchiveName"
    }
    Write-Host "[OK] Archive checksum verified" -ForegroundColor Green
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
    Write-Host "[OK] Binary installed to $targetPath" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Download failed" -ForegroundColor Red
    exit 1
} finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}

# Download themes
$themesUrl = "https://raw.githubusercontent.com/Josepavese/nido/main/resources/themes.json"
$themesPath = "$nidoHome\themes.json"
Write-Host "[DESKTOP] Fetching visual themes..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $themesUrl -OutFile $themesPath
    Write-Host "[OK] Themes installed to $themesPath" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Failed to download themes (skipped)" -ForegroundColor Yellow
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
    Write-Host "[OK] Default config created" -ForegroundColor Green
}

# Add to PATH
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$binDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$binDir", "User")
    Write-Host "[OK] Added to PATH (restart terminal to apply)" -ForegroundColor Green
}
$env:Path = "$env:Path;$binDir"

# Desktop Integration
Write-Host "[DESKTOP] Setting up Desktop Integration..." -ForegroundColor Cyan
$iconUrl = "https://raw.githubusercontent.com/Josepavese/nido/main/resources/nido.png"
$iconPath = "$nidoHome\nido.png"
try {
    Invoke-WebRequest -Uri $iconUrl -OutFile $iconPath
} catch {
    Write-Host "[WARN] Generic icon will be used (download failed)" -ForegroundColor Yellow
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
Write-Host "[OK] Start Menu shortcut created" -ForegroundColor Green

# --- Dependency Check & Proactive Install ---
Write-Host "[INFO] Checking flight readiness (dependencies)..." -ForegroundColor Cyan

function Test-IsAdmin {
    try {
        $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
        $principal = New-Object Security.Principal.WindowsPrincipal($identity)
        return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    } catch {
        return $false
    }
}

function Refresh-CurrentPath {
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $windowsApps = Join-Path $env:LOCALAPPDATA "Microsoft\WindowsApps"
    $qemuPath = Join-Path $env:ProgramFiles "qemu"
    $programFilesX86 = ${env:ProgramFiles(x86)}
    $extraPaths = @($windowsApps, $binDir)
    if (Test-Path $qemuPath) { $extraPaths += $qemuPath }
    if ($programFilesX86) {
        $qemuPathX86 = Join-Path $programFilesX86 "qemu"
        if ($env:ProgramFiles -ne $programFilesX86 -and (Test-Path $qemuPathX86)) { $extraPaths += $qemuPathX86 }
    }
    $env:Path = @($machinePath, $userPath, ($extraPaths -join ";")) -join ";"
}

function Test-CommandAvailable {
    param([string]$Name)
    return [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

function Ensure-Winget {
    if (Test-CommandAvailable "winget") {
        Write-Host "[OK] winget is already present." -ForegroundColor Green
        return $true
    }

    Write-Host "[WARN]  winget is missing. Nido uses it to install Windows VM dependencies." -ForegroundColor Yellow
    Write-Host "[INFO] Checking whether App Installer is present but not registered..." -ForegroundColor Cyan
    try {
        Add-AppxPackage -RegisterByFamilyName -MainPackage Microsoft.DesktopAppInstaller_8wekyb3d8bbwe -ErrorAction Stop
        Refresh-CurrentPath
        Start-Sleep -Seconds 2
        if (Test-CommandAvailable "winget") {
            Write-Host "[OK] winget registered." -ForegroundColor Green
            return $true
        }
    } catch {
        Write-Host "   App Installer registration was not available: $($_.Exception.Message)" -ForegroundColor Gray
    }

    Write-Host "[INSTALL] Installing Microsoft App Installer (winget)..." -ForegroundColor Cyan

    $wingetTemp = Join-Path ([System.IO.Path]::GetTempPath()) ("nido-winget-" + [System.Guid]::NewGuid().ToString("N"))
    $wingetBundle = Join-Path $wingetTemp "Microsoft.DesktopAppInstaller.msixbundle"
    try {
        New-Item -ItemType Directory -Force -Path $wingetTemp | Out-Null
        Invoke-WebRequest -Uri "https://aka.ms/getwinget" -OutFile $wingetBundle
        Add-AppxPackage -Path $wingetBundle
        Refresh-CurrentPath
        Start-Sleep -Seconds 2
        if (Test-CommandAvailable "winget") {
            Write-Host "[OK] winget installed." -ForegroundColor Green
            return $true
        }
        Write-Host "[WARN]  App Installer completed, but winget is not visible in this session yet." -ForegroundColor Yellow
        Write-Host "   Restart PowerShell and rerun the installer if dependency installation does not continue." -ForegroundColor Gray
        return $false
    } catch {
        Write-Host "[WARN]  Failed to install winget automatically: $($_.Exception.Message)" -ForegroundColor Yellow
        Write-Host "   Install App Installer from Microsoft, then rerun this installer." -ForegroundColor Gray
        return $false
    } finally {
        Remove-Item -LiteralPath $wingetTemp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Get-HypervisorPlatformState {
    try {
        $feature = Get-WindowsOptionalFeature -Online -FeatureName HypervisorPlatform -ErrorAction Stop
        return [string]$feature.State
    } catch {
        return "Unknown"
    }
}

function Enable-HypervisorPlatform {
    $enableCommand = "Enable-WindowsOptionalFeature -Online -FeatureName HypervisorPlatform -All -NoRestart; bcdedit /set hypervisorlaunchtype auto"
    try {
        if (Test-IsAdmin) {
            Enable-WindowsOptionalFeature -Online -FeatureName HypervisorPlatform -All -NoRestart | Out-Null
            bcdedit /set hypervisorlaunchtype auto | Out-Null
        } else {
            Write-Host "[ADMIN] Requesting administrator approval to enable Windows Hypervisor Platform..." -ForegroundColor Cyan
            $encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($enableCommand))
            $proc = Start-Process -FilePath "powershell.exe" -Verb RunAs -Wait -PassThru -ArgumentList "-NoProfile -ExecutionPolicy Bypass -EncodedCommand $encoded"
            if ($proc.ExitCode -ne 0) {
                throw "elevated PowerShell exited with code $($proc.ExitCode)"
            }
        }
        Write-Host "[OK] Windows Hypervisor Platform enabled." -ForegroundColor Green
        Write-Host "[RESTART] A Windows restart is required before WHPX acceleration is available." -ForegroundColor Yellow
        return $true
    } catch {
        Write-Host "[WARN]  Could not enable Windows Hypervisor Platform automatically: $($_.Exception.Message)" -ForegroundColor Yellow
        Write-Host "   Manual command: Enable-WindowsOptionalFeature -Online -FeatureName HypervisorPlatform -All" -ForegroundColor Gray
        return $false
    }
}

Refresh-CurrentPath

$qemuRuntimeInstalled = $false
$qemuImgInstalled = $false
try {
    if (Get-Command "qemu-system-x86_64" -ErrorAction SilentlyContinue) { $qemuRuntimeInstalled = $true }
    elseif (Get-Command "qemu-system-aarch64" -ErrorAction SilentlyContinue) { $qemuRuntimeInstalled = $true }
    elseif (Get-Command "qemu-system" -ErrorAction SilentlyContinue) { $qemuRuntimeInstalled = $true }
    if (Get-Command "qemu-img" -ErrorAction SilentlyContinue) { $qemuImgInstalled = $true }
} catch {}
$qemuInstalled = $qemuRuntimeInstalled -and $qemuImgInstalled

if (-not $qemuInstalled) {
    Write-Host "[WARN]  QEMU dependencies are incomplete. Nido needs qemu-system and qemu-img to hatch VMs." -ForegroundColor Yellow
    if (-not $qemuRuntimeInstalled) { Write-Host "   - qemu-system: Missing" -ForegroundColor Yellow }
    if (-not $qemuImgInstalled) { Write-Host "   - qemu-img: Missing" -ForegroundColor Yellow }
    $response = Read-Host "[INSTALL] Would you like to install QEMU dependencies automatically via winget? (y/N)"
    if ($response -eq "y") {
        if (Ensure-Winget) {
            Write-Host "[INSTALL]  Installing QEMU via winget..." -ForegroundColor Cyan
            winget install --id SoftwareFreedomConservancy.QEMU -e --source winget --scope machine --accept-package-agreements --accept-source-agreements
            if ($LASTEXITCODE -eq 0) {
                Refresh-CurrentPath
                Write-Host "[TIP] Note: You might need to restart your terminal for QEMU to be in your PATH." -ForegroundColor Yellow
                $qemuInstalled = $true
            } else {
                Write-Host "[WARN]  QEMU installation failed via winget (exit $LASTEXITCODE)." -ForegroundColor Yellow
                Write-Host "   Try manually: winget install --id SoftwareFreedomConservancy.QEMU -e --source winget" -ForegroundColor Gray
            }
        } else {
            Write-Host "[WARN]  Cannot install QEMU automatically because winget is unavailable." -ForegroundColor Yellow
        }
    } else {
        Write-Host "[TIP] Skipping automatic installation. You'll need to install it manually." -ForegroundColor Gray
        Write-Host "   QEMU Windows options: https://www.qemu.org/download/#windows" -ForegroundColor Gray
    }
} else {
    Write-Host "[OK] QEMU is already present." -ForegroundColor Green
}

$hypervisorState = Get-HypervisorPlatformState
$hypervisorNeedsRestart = $false
$hypervisorEnabled = ($hypervisorState -eq "Enabled")
if ($hypervisorEnabled) {
    Write-Host "[OK] Windows Hypervisor Platform is enabled." -ForegroundColor Green
} elseif ($hypervisorState -match "EnablePending|EnabledPending") {
    $hypervisorNeedsRestart = $true
    Write-Host "[RESTART] Windows Hypervisor Platform enablement is pending. Restart Windows to activate WHPX." -ForegroundColor Yellow
} else {
    if ($hypervisorState -eq "Unknown") {
        Write-Host "[WARN]  Could not determine Windows Hypervisor Platform state." -ForegroundColor Yellow
    } else {
        Write-Host "[WARN]  Windows Hypervisor Platform is $hypervisorState." -ForegroundColor Yellow
    }
    Write-Host "   Nido can run with TCG fallback, but WHPX is much faster." -ForegroundColor Gray
    $response = Read-Host "[WHPX] Enable Windows Hypervisor Platform now? This requires administrator approval and a reboot. (y/N)"
    if ($response -eq "y") {
        if (Enable-HypervisorPlatform) {
            $hypervisorNeedsRestart = $true
        }
    } else {
        Write-Host "[TIP] Skipping WHPX enablement. Nido will use TCG fallback when WHPX is unavailable." -ForegroundColor Gray
    }
}

Write-Host "[OK] Seed ISO support is built into Nido; no external ISO tool is required." -ForegroundColor Green

Write-Host ""
Write-Host "[DONE] Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor White
Write-Host "  1. Restart your terminal" -ForegroundColor Cyan
Write-Host "  2. Verify install: " -NoNewline; Write-Host "nido version" -ForegroundColor Cyan
Write-Host "  3. Check system: " -NoNewline; Write-Host "nido doctor" -ForegroundColor Cyan
Write-Host ""

if ($qemuInstalled) {
    Write-Host "[OK] All systems go! You are ready to fly!" -ForegroundColor Green
} else {
    Write-Host "[TIP] Note: Missing dependencies may limit functionality." -ForegroundColor Yellow
}

if ($hypervisorNeedsRestart) {
    Write-Host "[RESTART] Restart Windows to finish WHPX enablement." -ForegroundColor Magenta
} elseif (-not $hypervisorEnabled) {
    Write-Host "[TIP] Pro Tip: Enable 'Windows Hypervisor Platform' in Windows Features for max speed." -ForegroundColor Magenta
}
Write-Host ""
Write-Host '"It''s not a VM, it''s a lifestyle."' -ForegroundColor White
