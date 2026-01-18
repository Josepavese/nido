# 🪺 Nido Quick Installer - Windows Edition
# Downloads only the binary. No repo cloning. Lightning fast.

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "  🪺 Nido Quick Install" -ForegroundColor Cyan
Write-Host "  Lightning-fast VM management" -ForegroundColor Cyan
Write-Host ""

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Fetch latest release
Write-Host "🔍 Fetching latest release..." -ForegroundColor Cyan
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/Josepavese/nido/releases/latest"
    $version = $release.tag_name
    Write-Host "✅ Latest version: $version" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to fetch latest release" -ForegroundColor Red
    exit 1
}

# Build download URL
$binaryName = "nido-windows-$arch.exe"
$downloadUrl = "https://github.com/Josepavese/nido/releases/download/$version/$binaryName"

Write-Host "📥 Downloading $binaryName..." -ForegroundColor Cyan

$nidoHome = "$env:USERPROFILE\.nido"
$binDir = "$nidoHome\bin"
$targetPath = "$binDir\nido.exe"

# Create directories
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\vms" | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\run" | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\images" | Out-Null
New-Item -ItemType Directory -Force -Path "$nidoHome\backups" | Out-Null

# Download binary
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $targetPath
    Write-Host "✅ Binary installed to $targetPath" -ForegroundColor Green
} catch {
    Write-Host "❌ Download failed" -ForegroundColor Red
    exit 1
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

Write-Host ""
Write-Host "🎉 Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor White
Write-Host "  1. Restart your terminal" -ForegroundColor Cyan
Write-Host "  2. Verify install: " -NoNewline; Write-Host "nido version" -ForegroundColor Cyan
Write-Host "  3. Check system: " -NoNewline; Write-Host "nido doctor" -ForegroundColor Cyan
Write-Host ""
Write-Host "💡 Note: You'll need QEMU installed to run VMs" -ForegroundColor Yellow
Write-Host "   Install via: " -NoNewline; Write-Host "choco install qemu" -ForegroundColor Cyan
Write-Host ""
Write-Host "💡 Tip: PowerShell completion is not yet automatic. Run " -NoNewline; Write-Host "nido completion powershell" -ForegroundColor Cyan -NoNewline; Write-Host " to check status."
Write-Host ""
Write-Host '"It''s not a VM, it''s a lifestyle." 🪺' -ForegroundColor White
