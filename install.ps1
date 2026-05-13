# install.ps1 — fetch the latest remote-shell-mcp release for this OS/arch,
# place the two binaries on PATH, and optionally wire them into detected
# MCP clients (Claude Code, Claude Desktop, Codex CLI).
#
# Usage (one-liner):
#   irm https://raw.githubusercontent.com/jaenster/remote-shell-mcp/main/install.ps1 | iex
#
# Or with flags:
#   iwr -useb https://.../install.ps1 -OutFile install.ps1
#   .\install.ps1 -NoSetup
#   .\install.ps1 -Version v0.1.6
#   .\install.ps1 -InstallDir 'C:\Tools\bin'
#   .\install.ps1 -Yes

[CmdletBinding()]
param(
    [string]$Version = 'latest',
    [string]$InstallDir = '',
    # BaseUrl overrides the GitHub release URL prefix; the asset is fetched
    # from "$BaseUrl/$asset" instead of the github.com release page. CI uses
    # this to point at locally-built snapshot artifacts; end users don't need it.
    [string]$BaseUrl = '',
    [switch]$NoSetup,
    [switch]$Yes
)

$ErrorActionPreference = 'Stop'
$Repo = 'jaenster/remote-shell-mcp'

# --- detect arch -----------------------------------------------------------
# PROCESSOR_ARCHITECTURE reflects the running shell, not the OS — under WoW64
# (32-bit PowerShell on 64-bit Windows) it would say AMD64 wrongly for an x86
# shell. Use [Runtime.InteropServices.RuntimeInformation] which reports the
# OS architecture regardless of the host process.
$osArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
switch ($osArch) {
    'X64'   { $arch = 'amd64' }
    'Arm64' { $arch = 'arm64' }
    default {
        Write-Error "Unsupported architecture: $osArch"
        exit 1
    }
}

# --- pick install dir ------------------------------------------------------
if (-not $InstallDir) {
    # %LOCALAPPDATA%\Programs\remote-shell-mcp is the Windows convention for
    # user-scope CLI tools that aren't MSI-installed (gh, uv, rustup all
    # follow variations of this). No admin needed.
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\remote-shell-mcp'
}
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
Write-Host "Install dir: $InstallDir"

# --- resolve version -------------------------------------------------------
# Only resolve `latest` against GitHub when -BaseUrl isn't overridden.
if (-not $BaseUrl -and $Version -eq 'latest') {
    Write-Host 'Resolving latest release...'
    # The /releases/latest URL 302s to /releases/tag/<tag>. Follow manually so
    # we get the tag without depending on `gh` being installed.
    $resp = Invoke-WebRequest -Uri "https://github.com/$Repo/releases/latest" -MaximumRedirection 0 -ErrorAction SilentlyContinue
    if (-not $resp -or -not $resp.Headers.Location) {
        Write-Error 'Could not resolve latest release tag (no Location header on /releases/latest).'
        exit 1
    }
    $loc = [string]$resp.Headers.Location
    $Version = $loc.Split('/')[-1]
}
Write-Host "Version: $Version"

# --- download + verify -----------------------------------------------------
$asset = "remote-shell-mcp_$($Version.TrimStart('v'))_windows_$arch.zip"
if ($BaseUrl) {
    $prefix = $BaseUrl.TrimEnd('/')
} else {
    $prefix = "https://github.com/$Repo/releases/download/$Version"
}
$url = "$prefix/$asset"
$tmp   = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "rsm-install-$(Get-Random)") -Force
try {
    $pkg = Join-Path $tmp.FullName $asset
    Write-Host "Downloading $url"
    try {
        Invoke-WebRequest -Uri $url -OutFile $pkg -UseBasicParsing
    } catch {
        Write-Error "Download failed: $url`n(verify the release exists at https://github.com/$Repo/releases)"
        exit 1
    }

    # Optional checksum verification — matches install.sh's behavior.
    $sumsUrl = "$prefix/checksums.txt"
    try {
        $sums = Invoke-WebRequest -Uri $sumsUrl -UseBasicParsing
        $expected = ($sums.Content -split "`n" |
            Where-Object { $_ -match "  $([Regex]::Escape($asset))$" } |
            ForEach-Object { ($_ -split '\s+')[0] } |
            Select-Object -First 1)
        if ($expected) {
            $got = (Get-FileHash -Algorithm SHA256 -Path $pkg).Hash.ToLowerInvariant()
            if ($got -ne $expected) {
                Write-Error "Checksum mismatch! got=$got expected=$expected"
                exit 1
            }
            Write-Host 'Checksum OK'
        }
    } catch {
        # Checksums optional — same as install.sh.
    }

    # --- extract ----------------------------------------------------------
    Expand-Archive -Path $pkg -DestinationPath $tmp.FullName -Force
    foreach ($exe in @('remote-shell-mcpd.exe', 'remote-shell-mcp.exe')) {
        $src = Join-Path $tmp.FullName $exe
        if (-not (Test-Path $src)) {
            Write-Error "Archive missing $exe"
            exit 1
        }
        $dst = Join-Path $InstallDir $exe
        # If a previous version is running (daemon), Move-Item will fail. Tell
        # the user instead of silently leaving a half-installed state.
        try {
            Move-Item -Path $src -Destination $dst -Force
        } catch {
            Write-Error "Failed to write $dst — is the daemon currently running? Stop it and re-run.`n  $_"
            exit 1
        }
        Write-Host "Installed $dst"
    }
} finally {
    Remove-Item -Path $tmp.FullName -Recurse -Force -ErrorAction SilentlyContinue
}

# --- PATH ------------------------------------------------------------------
# Add to the User PATH (no admin), idempotent. Process env gets updated too so
# the rest of this script (and any subshell the user launches in the same
# session) sees the binary.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$pathParts = if ($userPath) { $userPath.Split(';') } else { @() }
if ($pathParts -notcontains $InstallDir) {
    $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "Added $InstallDir to user PATH (open a new terminal to pick it up)."
}
if (($env:Path -split ';') -notcontains $InstallDir) {
    $env:Path = "$env:Path;$InstallDir"
}

# --- register with MCP clients --------------------------------------------
if (-not $NoSetup) {
    Write-Host ''
    Write-Host 'Registering with detected MCP clients...'
    $setupArgs = @('setup')
    if ($Yes) { $setupArgs += '--yes' }
    & (Join-Path $InstallDir 'remote-shell-mcp.exe') @setupArgs
}

Write-Host ''
Write-Host "Done. Try:  remote-shell-mcp version"
