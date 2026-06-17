$ErrorActionPreference = "Stop"

# Determine Architecture
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64") { "amd64" } else { "386" }
$Version = "v1.1.2"
$BinaryName = "blsqui-cli-windows-$Arch.exe"
$DownloadUrl = "https://github.com/blsqui/blsqui-cli/releases/download/$Version/$BinaryName"

Write-Host "📥 Downloading Blsqui CLI $Version for Windows ($Arch)..." -ForegroundColor Cyan

# Define a clean installation path in User's AppData (Avoids strict Admin Privilege blocks)
$InstallDir = "$env:USERPROFILE\.blsqui"
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

$ExePath = Join-Path $InstallDir "blsqui.exe"

# Download the binary asset
Invoke-WebRequest -Uri $DownloadUrl -OutFile $ExePath

# Permanently append to User PATH environment variable if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*\.blsqui*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "⚙️ Added Blsqui to your User PATH environment variable." -ForegroundColor Yellow
}

Write-Host "✅ Blsqui CLI installed successfully!" -ForegroundColor Green
Write-Host "💡 Please restart your PowerShell window and run 'blsqui' to begin." -ForegroundColor Green