#Requires -Version 5.1
<#
.SYNOPSIS
    OctoJ installer for Windows.
.DESCRIPTION
    Downloads and installs the latest OctoJ release from GitHub.
    Does NOT require administrator privileges.
.EXAMPLE
    iwr https://raw.githubusercontent.com/OctavoBit/octoj/main/scripts/install.ps1 | iex
.EXAMPLE
    # Install specific version
    $env:OCTOJ_VERSION = "v0.1.0"
    iwr https://raw.githubusercontent.com/OctavoBit/octoj/main/scripts/install.ps1 | iex
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$REPO = "OctavoBit/octoj"
$BINARY = "octoj.exe"
$INSTALL_DIR = "$env:USERPROFILE\.octoj\bin"

function Write-Banner {
    Write-Host ""
    Write-Host "  ___  ___ _        _ " -ForegroundColor Cyan
    Write-Host " / _ \/ __| |_ ___ | |" -ForegroundColor Cyan
    Write-Host "| (_) \__ \  _/ _ \| |" -ForegroundColor Cyan
    Write-Host " \___/|___/\__\___/|_|  by OctavoBit" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Java JDK Version Manager" -ForegroundColor White
    Write-Host ""
}

function Write-Step {
    param([string]$Message)
    Write-Host "`n==> $Message" -ForegroundColor Blue -NoNewline
    Write-Host ""
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/$REPO/releases/latest"
    try {
        $response = Invoke-RestMethod -Uri $apiUrl -Headers @{ "User-Agent" = "octoj-installer/1.0" }
        return $response.tag_name
    }
    catch {
        throw "Failed to get latest version: $_"
    }
}

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-FileHash256 {
    param([string]$FilePath)
    $hash = Get-FileHash -Path $FilePath -Algorithm SHA256
    return $hash.Hash.ToLower()
}

function Add-ToUserPath {
    param([string]$Directory)

    $currentPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
    $dirs = $currentPath -split ";"

    if ($dirs -notcontains $Directory) {
        $newPath = "$Directory;$currentPath"
        [System.Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$Directory;$env:PATH"
        Write-Success "Added $Directory to user PATH"
        return $true
    }
    else {
        Write-Host "  $Directory already in PATH" -ForegroundColor Gray
        return $false
    }
}

function Main {
    Write-Banner

    # Determine version
    $version = $env:OCTOJ_VERSION
    if (-not $version) {
        Write-Step "Getting latest version"
        $version = Get-LatestVersion
        Write-Success "Latest version: $version"
    }
    else {
        Write-Success "Using specified version: $version"
    }

    # Detect architecture
    Write-Step "Detecting platform"
    $arch = Get-Architecture
    Write-Success "Architecture: $arch"

    $assetName = "octoj-windows-$arch.exe"
    $downloadUrl = "https://github.com/$REPO/releases/download/$version/$assetName"
    $checksumUrl = "$downloadUrl.sha256"

    Write-Step "Downloading OctoJ $version"
    Write-Host "  URL: $downloadUrl" -ForegroundColor Gray

    # Create install directory
    if (-not (Test-Path $INSTALL_DIR)) {
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
    }

    $tmpFile = [System.IO.Path]::GetTempFileName() + ".exe"

    try {
        # Download binary
        $progressPreference = 'Continue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tmpFile -UseBasicParsing

        Write-Success "Downloaded to temporary location"

        # Verify checksum
        Write-Step "Verifying checksum"
        try {
            $checksumResponse = Invoke-WebRequest -Uri $checksumUrl -UseBasicParsing
            $expectedChecksum = ($checksumResponse.Content -split '\s+')[0].Trim().ToLower()
            $actualChecksum = Get-FileHash256 -FilePath $tmpFile

            if ($expectedChecksum -eq $actualChecksum) {
                Write-Success "Checksum verified"
            }
            else {
                throw "Checksum mismatch! Expected: $expectedChecksum, Got: $actualChecksum"
            }
        }
        catch {
            if ($_.Exception.Message -like "*Checksum mismatch*") {
                throw
            }
            Write-Warn "Could not verify checksum (continuing anyway): $_"
        }

        # Install binary
        Write-Step "Installing OctoJ"
        $installPath = Join-Path $INSTALL_DIR $BINARY
        Move-Item -Path $tmpFile -Destination $installPath -Force
        Write-Success "Installed to: $installPath"

        # Add to PATH
        Write-Step "Configuring PATH"
        Add-ToUserPath -Directory $INSTALL_DIR | Out-Null

        # Run octoj init
        Write-Step "Configuring environment"
        try {
            & $installPath init --apply
        }
        catch {
            Write-Warn "Automatic environment setup failed. Run 'octoj init --apply' manually."
        }

        # Broadcast environment change
        $code = @"
using System;
using System.Runtime.InteropServices;
public class Win32 {
    [DllImport("user32.dll", SetLastError=true, CharSet=CharSet.Auto)]
    public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
}
"@
        Add-Type -TypeDefinition $code -ErrorAction SilentlyContinue
        $result = [UIntPtr]::Zero
        try {
            [Win32]::SendMessageTimeout([IntPtr]0xffff, 0x001A, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
        }
        catch { }

        Write-Host ""
        Write-Host "OctoJ $version installed successfully!" -ForegroundColor Green
        Write-Host ""
        Write-Host "IMPORTANT: Restart your terminal for PATH changes to take effect." -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Get started:"
        Write-Host "  octoj search 21          # search for JDK 21"
        Write-Host "  octoj install 21         # install Temurin JDK 21"
        Write-Host "  octoj use temurin@21     # activate JDK 21"
        Write-Host "  octoj doctor             # check installation"
        Write-Host ""
        Write-Host "Documentation: https://github.com/$REPO#readme"
        Write-Host ""
    }
    catch {
        if (Test-Path $tmpFile) {
            Remove-Item $tmpFile -Force -ErrorAction SilentlyContinue
        }
        throw
    }
}

Main
