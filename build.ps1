# C1-Slim simulator: dual-target build script (ASCII only, no BOM needed)
#
#   .\build.ps1 -Target sim      build PC simulator (Windows window, SDL3)
#   .\build.ps1 -Target shot     headless: no window, dump the first frame to PNG
#   .\build.ps1 -Target device   cross-compile device ELF (linux/mipsle, static)
#   .\build.ps1 -Target check    run all tests + verify device side has no SDL
#
# Notes:
#   * Neither target needs cgo (SDL3 is loaded dynamically via purego).
#   * The device build is free of SDL because the host files carry the build tag
#     `!linux || !mipsle`, so they are not compiled at all for linux/mipsle
#     (see the isolation check in -Target check).
param(
    [ValidateSet('sim', 'shot', 'device', 'check')]
    [string]$Target = 'sim',
    [string]$App = 'demo'
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot

# Go may not be on PATH; prefer the known install location.
$go = 'D:\DevProgramsSDK\go1.26\bin\go.exe'
if (-not (Test-Path $go)) { $go = 'go' }

function Get-SDL3Dll {
    return Get-ChildItem -Path 'D:\DevProgramsSDK\SDL3\*\x86_64-w64-mingw32\bin\SDL3.dll' -ErrorAction SilentlyContinue |
        Select-Object -First 1
}

function Invoke-Go {
    param([string[]]$GoArgs, [string]$WorkDir)
    Push-Location $WorkDir
    try {
        & $go @GoArgs
        if ($LASTEXITCODE -ne 0) { throw "go $($GoArgs -join ' ') failed (exit=$LASTEXITCODE)" }
    }
    finally { Pop-Location }
}

function Build-Sim {
    Remove-Item Env:GOOS, Env:GOARCH, Env:GOMIPS -ErrorAction SilentlyContinue
    $env:CGO_ENABLED = '0'

    $outDir = Join-Path $root 'build/sim'
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
    Invoke-Go @('build', '-o', "$outDir/$App.exe", '.') (Join-Path $root "apps/$App")

    # purego loads SDL3.dll from the executable's directory at runtime.
    $dll = Get-SDL3Dll
    if (-not $dll) {
        throw 'SDL3.dll not found. Expected D:\DevProgramsSDK\SDL3\*\x86_64-w64-mingw32\bin\SDL3.dll'
    }
    Copy-Item $dll.FullName -Destination $outDir -Force
    return $outDir
}

switch ($Target) {
    'sim' {
        Write-Host '==> Building PC simulator (windows/amd64, zero cgo)' -ForegroundColor Cyan
        $outDir = Build-Sim
        Write-Host "==> Done: $outDir/$App.exe (SDL3.dll copied alongside)" -ForegroundColor Green
    }

    'shot' {
        Write-Host '==> Headless screenshot (no window)' -ForegroundColor Cyan
        Build-Sim | Out-Null

        $shotDir = Join-Path $root 'build/shots'
        New-Item -ItemType Directory -Force -Path $shotDir | Out-Null
        $dump = Join-Path $shotDir "$App.png"

        $env:C1SIM_HEADLESS = '1'
        $env:C1SIM_DUMP = $dump
        Push-Location (Join-Path $root 'build/sim')
        try {
            & ".\$App.exe"
            if ($LASTEXITCODE -ne 0) { throw "$App exited with $LASTEXITCODE" }
        }
        finally {
            Pop-Location
            Remove-Item Env:C1SIM_HEADLESS, Env:C1SIM_DUMP -ErrorAction SilentlyContinue
        }
        Write-Host "==> Done: $dump" -ForegroundColor Green
    }

    'device' {
        Write-Host '==> Cross-compiling device ELF (linux/mipsle, static)' -ForegroundColor Cyan
        Remove-Item Env:CGO_CFLAGS, Env:CGO_LDFLAGS -ErrorAction SilentlyContinue
        $env:CGO_ENABLED = '0'
        $env:GOOS = 'linux'
        $env:GOARCH = 'mipsle'
        $env:GOMIPS = 'hardfloat'

        $outDir = Join-Path $root 'build/device'
        New-Item -ItemType Directory -Force -Path $outDir | Out-Null
        Invoke-Go @('build', '-trimpath', '-ldflags', '-s -w -buildid=', '-o', "$outDir/$App", '.') (Join-Path $root "apps/$App")

        Remove-Item Env:GOOS, Env:GOARCH, Env:GOMIPS -ErrorAction SilentlyContinue
        Write-Host "==> Done: $outDir/$App" -ForegroundColor Green
    }

    'check' {
        Write-Host '==> Unit tests (c1device)' -ForegroundColor Cyan
        Remove-Item Env:GOOS, Env:GOARCH, Env:GOMIPS -ErrorAction SilentlyContinue
        $env:CGO_ENABLED = '0'
        Invoke-Go @('test', './...') (Join-Path $root 'c1device')
        Invoke-Go @('vet', './...') (Join-Path $root 'c1device')

        Write-Host "==> Unit tests (apps/$App)" -ForegroundColor Cyan
        Invoke-Go @('test', './...') (Join-Path $root "apps/$App")
        Invoke-Go @('vet', './...') (Join-Path $root "apps/$App")

        Write-Host '==> Isolation check: device deps must not contain sdl' -ForegroundColor Cyan
        $env:GOOS = 'linux'; $env:GOARCH = 'mipsle'; $env:GOMIPS = 'hardfloat'
        Push-Location (Join-Path $root "apps/$App")
        $deps = & $go list -deps .
        if ($LASTEXITCODE -ne 0) { Pop-Location; throw 'go list -deps failed' }
        Pop-Location
        Remove-Item Env:GOOS, Env:GOARCH, Env:GOMIPS -ErrorAction SilentlyContinue

        $sdlHits = $deps | Select-String -Pattern 'sdl' -SimpleMatch
        if ($sdlHits) {
            Write-Host $sdlHits -ForegroundColor Red
            throw 'device deps contain sdl - a host file leaked into the device build'
        }
        Write-Host 'OK: device deps contain no sdl package' -ForegroundColor Green

        Write-Host '==> Frame decode cross-check (vs Python oracle)' -ForegroundColor Cyan
        & (Join-Path $root 'tools\crosscheck.ps1')
        if ($LASTEXITCODE -ne 0) { throw 'cross-check failed' }
    }
}
