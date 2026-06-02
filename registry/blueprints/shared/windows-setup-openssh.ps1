$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$logPath = "C:\Windows\Temp\nido-openssh-setup.log"
Start-Transcript -Path $logPath -Append | Out-Null

function Install-OpenSSHCapability {
    try {
        $capability = Get-WindowsCapability -Online -Name "OpenSSH.Server*" | Select-Object -First 1
        if (-not $capability) {
            Write-Warning "OpenSSH Server capability was not found."
            return $false
        }

        if ($capability.State -eq "Installed") {
            return $true
        }

        $capabilityName = $capability.Name
        Add-WindowsCapability -Online -Name $capabilityName -LimitAccess | Out-Null
        return $true
    } catch {
        Write-Warning "OpenSSH capability installation failed: $($_.Exception.Message)"
        return $false
    }
}

function Install-OpenSSHArchive {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

    $url = "https://github.com/PowerShell/Win32-OpenSSH/releases/latest/download/OpenSSH-Win64.zip"
    $archivePath = Join-Path $env:TEMP "OpenSSH-Win64.zip"
    $extractPath = Join-Path $env:TEMP "OpenSSH-Win64"
    $installPath = Join-Path $env:ProgramFiles "OpenSSH"

    Remove-Item -LiteralPath $archivePath -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $extractPath -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force -Path $extractPath | Out-Null
    New-Item -ItemType Directory -Force -Path $installPath | Out-Null

    Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $archivePath
    Expand-Archive -Path $archivePath -DestinationPath $extractPath -Force

    $source = Get-ChildItem -Path $extractPath -Directory |
        Where-Object { $_.Name -like "OpenSSH-Win64*" } |
        Select-Object -First 1
    if (-not $source) {
        throw "OpenSSH archive did not contain OpenSSH-Win64."
    }

    Copy-Item -Path (Join-Path $source.FullName "*") -Destination $installPath -Recurse -Force
    & (Join-Path $installPath "install-sshd.ps1")
}

try {
    if (-not (Get-Service -Name sshd -ErrorAction SilentlyContinue)) {
        if (-not (Install-OpenSSHCapability)) {
            Install-OpenSSHArchive
        }
    }

    if (-not (Get-Service -Name sshd -ErrorAction SilentlyContinue)) {
        throw "sshd service was not installed."
    }

    Set-Service -Name sshd -StartupType Automatic
    & sc.exe config sshd start= auto | Out-Null
    Start-Service -Name sshd

    if (-not (Get-NetFirewallRule -Name sshd -ErrorAction SilentlyContinue)) {
        New-NetFirewallRule `
            -Name sshd `
            -DisplayName "OpenSSH Server (sshd)" `
            -Enabled True `
            -Direction Inbound `
            -Protocol TCP `
            -Action Allow `
            -LocalPort 22 | Out-Null
    }
} finally {
    Stop-Transcript | Out-Null
}
