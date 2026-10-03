# One-time per-machine setup for c1slim-sim development on Windows.
# Safe to re-run: every step is idempotent.
#
# Usage (from the repo root):
#   .\setup.ps1              # full setup: enable hook, add remote, check deps
#   .\setup.ps1 -CheckOnly   # read-only dependency self-check; non-zero exit if a
#                            #   blocking dependency is missing (CI / scripts)
#
# ASCII only: Windows PowerShell 5.1 reads a BOM-less file as ANSI, so a single
# non-ASCII byte can break quote pairing and make the whole script unparseable
# (same trap as build.ps1). The pre-commit hook checks every staged *.ps1 for
# non-ASCII bytes, which keeps this file honest.
#
# What it does:
#   1. Enables the repo pre-commit hook (core.hooksPath -> .githooks). That setting
#      lives in .git/config, so a fresh clone does NOT inherit it -- run this once
#      per machine. (skipped with -CheckOnly)
#   2. Adds a read-only remote for the firmware upstream (tracking only).
#      (skipped with -CheckOnly)
#   3. Reports whether build prerequisites are present: Go 1.26+, SDL3, and
#      Python + Pillow + numpy (the last is only needed for -Target check).

param([switch]$CheckOnly)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$script:blocking = 0

function Info { param([string]$m) Write-Host "       $m" -ForegroundColor DarkGray }
function Ok   { param([string]$m) Write-Host "[ OK ] $m" -ForegroundColor Green }
function Warn { param([string]$m) Write-Host "[WARN] $m" -ForegroundColor Yellow }
function Bad  { param([string]$m) Write-Host "[FAIL] $m" -ForegroundColor Red; $script:blocking++ }

if ($CheckOnly) {
    Write-Host '==> c1slim-sim dependency check (read-only)' -ForegroundColor Cyan
}
else {
    Write-Host '==> c1slim-sim machine setup' -ForegroundColor Cyan
}

# --- 1. git (+ hook/remote setup unless -CheckOnly) ---------------------------
$git = Get-Command git -ErrorAction SilentlyContinue
if (-not $git) {
    Bad 'git not found on PATH'
}
else {
    Ok ('git found (' + (& git --version) + ')')
    if ($CheckOnly) {
        Info 'check-only: not touching git config or remotes'
    }
    else {
        Push-Location $root
        try {
            & git config core.hooksPath .githooks
            if ($LASTEXITCODE -ne 0) { Bad 'failed to set core.hooksPath' }
            else { Ok 'pre-commit hook enabled (core.hooksPath = .githooks)' }

            $remotes = & git remote
            if ($remotes -contains 'upstream-fw') {
                Info 'upstream-fw remote already present'
            }
            else {
                & git remote add upstream-fw https://github.com/theBillLee/c1-slim.git
                if ($LASTEXITCODE -eq 0) { Ok 'added upstream-fw -> theBillLee/c1-slim' }
                else { Warn 'could not add upstream-fw remote (optional)' }
            }
        }
        finally { Pop-Location }
    }
}

# --- 2. Go 1.26+ --------------------------------------------------------------
$go = $null
if (Test-Path 'D:\DevProgramsSDK\go1.26\bin\go.exe') { $go = 'D:\DevProgramsSDK\go1.26\bin\go.exe' }
if (-not $go) {
    $g = Get-Command go -ErrorAction SilentlyContinue
    if ($g) { $go = $g.Source }
}
if (-not $go) {
    Bad 'Go not found (need 1.26+; install it or put it on PATH)'
}
else {
    $ver = (& $go version)
    $m = [regex]::Match($ver, 'go(\d+)\.(\d+)')
    $major = 0; $minor = 0
    if ($m.Success) { $major = [int]$m.Groups[1].Value; $minor = [int]$m.Groups[2].Value }
    if ($major -gt 1 -or ($major -eq 1 -and $minor -ge 26)) { Ok ('Go found: ' + $ver) }
    else { Bad ('Go too old: ' + $ver + ' (need 1.26+)') }
    Info ('using: ' + $go)
}

# --- 3. SDL3 (needed for -Target sim) -----------------------------------------
$sdl = Get-ChildItem -Path 'D:\DevProgramsSDK\SDL3\*\x86_64-w64-mingw32\bin\SDL3.dll' -ErrorAction SilentlyContinue |
    Select-Object -First 1
if ($sdl) { Ok ('SDL3.dll found: ' + $sdl.FullName) }
else {
    Bad 'SDL3.dll not found (needed for -Target sim). Expected under D:\DevProgramsSDK\SDL3\*\x86_64-w64-mingw32\bin\; if yours is elsewhere, update Get-SDL3Dll in build.ps1.'
}

# --- 4. Python + Pillow + numpy (only for -Target check) ----------------------
$py = Get-Command python -ErrorAction SilentlyContinue
if (-not $py) { $py = Get-Command python3 -ErrorAction SilentlyContinue }
if (-not $py) {
    Warn 'Python not found (only needed for -Target check frame cross-check)'
}
else {
    & $py.Source -c 'import PIL, numpy' 2>$null
    if ($LASTEXITCODE -eq 0) { Ok 'Python + Pillow + numpy present' }
    else { Warn 'Pillow/numpy missing (pip install pillow numpy) - only needed for -Target check' }
}

# --- summary ------------------------------------------------------------------
Write-Host ''
if ($CheckOnly) {
    if ($script:blocking -gt 0) {
        Write-Host "Dependency check FAILED: $script:blocking blocking issue(s) - see [FAIL] above." -ForegroundColor Red
        exit 1
    }
    else {
        Write-Host 'Dependency check passed.' -ForegroundColor Green
        exit 0
    }
}
else {
    if ($script:blocking -gt 0) {
        Write-Host "Setup finished with $script:blocking blocking issue(s) - see [FAIL] above." -ForegroundColor Yellow
    }
    else {
        Write-Host 'Setup complete.' -ForegroundColor Green
        Write-Host 'Next: .\build.ps1 -Target sim' -ForegroundColor DarkGray
    }
}
