# C1-Slim simulator: dual-target build script (ASCII only, no BOM needed)
#
#   .\build.ps1 -Target sim      build PC simulator (Windows window, SDL3)
#   .\build.ps1 -Target shot     headless: no window, dump the first frame to PNG
#   .\build.ps1 -Target seq      headless: dump a numbered frame sequence to build/shots/seq/
#   .\build.ps1 -Target regress  dump the sequence twice and compare pixel by pixel (CI gate)
#   .\build.ps1 -Target device   cross-compile device ELF (linux/mipsle, static)
#   .\build.ps1 -Target check    run all tests + verify device side has no SDL
#
# Notes:
#   * Neither target needs cgo (SDL3 is loaded dynamically via purego).
#   * The device build is free of SDL because the host files carry the build tag
#     `!linux || !mipsle`, so they are not compiled at all for linux/mipsle
#     (see the isolation check in -Target check).
param(
    [ValidateSet('sim', 'shot', 'seq', 'regress', 'device', 'check')]
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

# Run the app headless (no window, no SDL) and dump a frame sequence.
# $DumpPath is used as a filename prefix when $Frames is greater than 1.
function Invoke-Headless {
    param([string]$DumpPath, [int]$Frames)

    $env:C1SIM_HEADLESS = '1'
    $env:C1SIM_DUMP = $DumpPath
    $env:C1SIM_FRAMES = "$Frames"
    Push-Location (Join-Path $root 'build/sim')
    try {
        & ".\$App.exe"
        if ($LASTEXITCODE -ne 0) { throw "$App exited with $LASTEXITCODE" }
    }
    finally {
        Pop-Location
        Remove-Item Env:C1SIM_HEADLESS, Env:C1SIM_DUMP, Env:C1SIM_FRAMES -ErrorAction SilentlyContinue
    }
}

# Count differing pixels between two PNGs. Returns -1 if either file is missing,
# -2 if the dimensions differ, otherwise the number of differing pixels.
function Get-PngPixelDiff {
    param([string]$PathA, [string]$PathB)

    if (-not (Test-Path $PathA) -or -not (Test-Path $PathB)) { return -1 }
    Add-Type -AssemblyName System.Drawing
    $ba = [System.Drawing.Bitmap]::FromFile($PathA)
    $bb = [System.Drawing.Bitmap]::FromFile($PathB)
    try {
        if ($ba.Width -ne $bb.Width -or $ba.Height -ne $bb.Height) { return -2 }
        $rect = New-Object System.Drawing.Rectangle 0, 0, $ba.Width, $ba.Height
        $data = $ba.LockBits($rect, 'ReadOnly', 'Format32bppArgb')
        $dataB = $bb.LockBits($rect, 'ReadOnly', 'Format32bppArgb')
        try {
            $len = $data.Stride * $data.Height
            $buf = New-Object byte[] $len
            $bufB = New-Object byte[] $len
            [System.Runtime.InteropServices.Marshal]::Copy($data.Scan0, $buf, 0, $len)
            [System.Runtime.InteropServices.Marshal]::Copy($dataB.Scan0, $bufB, 0, $len)
            $diff = 0
            for ($i = 0; $i -lt $len; $i += 4) {
                if ($buf[$i] -ne $bufB[$i]) { $diff++ }
            }
            return $diff
        }
        finally {
            $ba.UnlockBits($data)
            $bb.UnlockBits($dataB)
        }
    }
    finally {
        $ba.Dispose()
        $bb.Dispose()
    }
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

        Invoke-Headless -DumpPath $dump -Frames 1
        Write-Host "==> Done: $dump" -ForegroundColor Green
    }

    'seq' {
        $frames = if ($env:C1SIM_FRAMES) { [int]$env:C1SIM_FRAMES } else { 16 }
        Write-Host "==> Headless frame sequence ($frames frames, no window)" -ForegroundColor Cyan
        Build-Sim | Out-Null

        $seqDir = Join-Path $root 'build/shots/seq'
        Remove-Item $seqDir -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Force -Path $seqDir | Out-Null

        Invoke-Headless -DumpPath (Join-Path $seqDir "$App.png") -Frames $frames
        $n = (Get-ChildItem $seqDir -Filter '*.png').Count
        Write-Host "==> Done: $n frames in $seqDir" -ForegroundColor Green
    }

    'regress' {
        $frames = if ($env:C1SIM_FRAMES) { [int]$env:C1SIM_FRAMES } else { 16 }
        Write-Host "==> Two-round frame regression ($frames frames each, no window)" -ForegroundColor Cyan
        Build-Sim | Out-Null

        $regDir = Join-Path $root 'build/shots/regress'
        Remove-Item $regDir -Recurse -Force -ErrorAction SilentlyContinue

        $rounds = @()
        foreach ($name in @('round1', 'round2')) {
            $dir = Join-Path $regDir $name
            New-Item -ItemType Directory -Force -Path $dir | Out-Null
            Write-Host "--> $name" -ForegroundColor DarkGray
            Invoke-Headless -DumpPath (Join-Path $dir "$App.png") -Frames $frames
            $rounds += ,$dir
        }

        # Frame count must match exactly: a short run means the exporter gave up early.
        foreach ($dir in $rounds) {
            $n = (Get-ChildItem $dir -Filter '*.png').Count
            if ($n -ne $frames) {
                throw "$(Split-Path -Leaf $dir) exported $n frames, expected $frames"
            }
        }

        Write-Host '==> Comparing the two rounds pixel by pixel' -ForegroundColor Cyan
        $mismatch = @()
        for ($i = 1; $i -le $frames; $i++) {
            $name = '{0}_{1:d4}.png' -f $App, $i
            $diff = Get-PngPixelDiff (Join-Path $rounds[0] $name) (Join-Path $rounds[1] $name)
            switch ($diff) {
                0 { Write-Host "OK      $name" }
                -1 { Write-Host "MISSING $name" -ForegroundColor Red; $mismatch += $name }
                -2 { Write-Host "SIZE    $name (dimensions differ)" -ForegroundColor Red; $mismatch += $name }
                default {
                    Write-Host "DIFF    $name  $diff pixels differ" -ForegroundColor Red
                    $mismatch += $name
                }
            }
        }

        if ($mismatch.Count -gt 0) {
            throw "frame regression FAILED: $($mismatch.Count)/$frames frames differ (non-deterministic render)"
        }
        Write-Host "==> Done: $frames/$frames frames identical across both rounds" -ForegroundColor Green
        Write-Host "    artifacts: $regDir" -ForegroundColor DarkGray
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

        Write-Host '==> Device build: vet + link + ELF header' -ForegroundColor Cyan
        # The device-tagged tests (input_lifecycle / visual_key) carry
        # `linux && mipsle`, so on Windows they are neither run nor type-checked.
        # `go vet` to GOOS=linux GOARCH=mipsle gives them a full type check, and
        # linking proves the device side actually compiles end to end.
        $env:GOOS = 'linux'; $env:GOARCH = 'mipsle'; $env:GOMIPS = 'hardfloat'
        $devOutDir = Join-Path $root 'build/device'
        New-Item -ItemType Directory -Force -Path $devOutDir | Out-Null
        Invoke-Go @('vet', './...') (Join-Path $root 'c1device')
        Invoke-Go @('build', '-trimpath', '-ldflags', '-s -w -buildid=', '-o', "$devOutDir/$App", '.') (Join-Path $root "apps/$App")
        Remove-Item Env:GOOS, Env:GOARCH, Env:GOMIPS -ErrorAction SilentlyContinue

        # ELF sanity: magic, 32-bit, little-endian, MIPS, EXEC (static, no interpreter)
        $elfPath = Join-Path $devOutDir $App
        $elf = [System.IO.File]::ReadAllBytes($elfPath)
        if ($elf.Length -lt 52 -or
            $elf[0] -ne 0x7F -or $elf[1] -ne 0x45 -or $elf[2] -ne 0x4C -or $elf[3] -ne 0x46) {
            throw "$App is not an ELF file"
        }
        if ($elf[4] -ne 1) { throw "ELF class is not 32-bit (EI_CLASS=$($elf[4]))" }
        if ($elf[5] -ne 1) { throw "ELF data is not little-endian (EI_DATA=$($elf[5]))" }
        $machine = [int]$elf[18] -bor([int]$elf[19] -shl 8)
        if ($machine -ne 8) { throw "e_machine is not EM_MIPS ($machine)" }
        $elfType = [int]$elf[16] -bor([int]$elf[17] -shl 8)
        if ($elfType -ne 2) { throw "ELF type is not ET_EXEC (static), got $elfType" }
        Write-Host "OK: device ELF $([int]$elf[18])/mips, $([math]::Round($elf.Length/1MB,1)) MB" -ForegroundColor Green

        Write-Host '==> Build tag dispatch: host/device files must not overlap' -ForegroundColor Cyan
        # Platform dispatch relies on build tags. A too-wide tag does not break the
        # build -- it silently compiles both implementations in -- so assert
        # explicitly which side was compiled instead of only checking "it builds".
        $c1dev = Join-Path $root 'c1device'

        # Let go decide whether a file is compiled in (compares names in its own
        # template), avoiding filename string handling in PowerShell.
        # PowerShell strips double quotes when passing args to native commands, which
        # would break any go template that compares against a quoted string.
        # So emit plain file names and match them here instead.
        $listFmt = '{{range .GoFiles}}{{.}}
{{end}}'

        foreach ($combo in @(
                @{ os = 'windows'; arch = 'amd64';    mips = $null;     wantDevice = $false },
                @{ os = 'linux';   arch = 'amd64';    mips = $null;     wantDevice = $false },
                @{ os = 'linux';   arch = 'mipsle'; mips = 'hardfloat'; wantDevice = $true })) {
            $env:CGO_ENABLED = '0'
            $env:GOOS = $combo.os
            $env:GOARCH = $combo.arch
            if ($combo.mips) { $env:GOMIPS = $combo.mips } else { Remove-Item Env:GOMIPS -ErrorAction SilentlyContinue }

            $flags = & $go -C $c1dev list -f $listFmt .
            $listExit = $LASTEXITCODE
            Remove-Item Env:GOOS, Env:GOARCH, Env:GOMIPS -ErrorAction SilentlyContinue

            if ($listExit -ne 0) { throw "go list failed for $($combo.os)/$($combo.arch)" }
            # Split on the .go suffix rather than on newlines: PowerShell strips the
        # newline characters out of the template argument passed to go.
            $compiled = @([regex]::Matches("$flags", '[\w./-]+\.go') | ForEach-Object { $_.Value })
            if ($compiled.Count -eq 0) { throw "go list returned no files for $($combo.os)/$($combo.arch)" }

            $hasHost = $compiled -contains 'platform_host.go'
            $hasDevice = $compiled -contains 'platform_device_linux_mipsle.go'
            $hasDesktopLinux = $compiled -contains 'desktop_linux.go'
            $hasDesktopStub = $compiled -contains 'desktop_stub.go'

            # A too-wide tag does not break the build, so assert explicitly.
            if ($combo.wantDevice) {
                if ($hasHost -or $hasDesktopStub) {
                    throw "build tag dispatch leak on $($combo.os)/$($combo.arch): host files compiled into the device build"
                }
                if (-not ($hasDevice -and $hasDesktopLinux)) {
                    throw "device implementation missing for $($combo.os)/$($combo.arch)"
                }
                Write-Host "OK      $($combo.os)/$($combo.arch): device only" -ForegroundColor Green
            }
            else {
                if ($hasDevice -or $hasDesktopLinux) {
                    throw "build tag dispatch leak on $($combo.os)/$($combo.arch): device files compiled into a host build"
                }
                if (-not ($hasHost -and $hasDesktopStub)) {
                    throw "host implementation missing for $($combo.os)/$($combo.arch)"
                }
                Write-Host "OK      $($combo.os)/$($combo.arch): host only" -ForegroundColor Green
            }
        }

        Write-Host '==> Frame decode cross-check (vs Python oracle)' -ForegroundColor Cyan
        & (Join-Path $root 'tools\crosscheck.ps1')
        if ($LASTEXITCODE -ne 0) { throw 'cross-check failed' }
    }
}
