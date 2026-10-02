# Frame decode cross-check: Go decoder vs Python oracle (pixel by pixel)
#
# Without real hardware this is the only way to prove we draw correctly:
# feed the same 5624-byte frame to both implementations and compare pixels.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$golden = Join-Path $root 'tests/golden'

$go = 'D:\DevProgramsSDK\go1.26\bin\go.exe'
if (-not (Test-Path $go)) { $go = 'go' }

Push-Location $PSScriptRoot
try {
    Write-Host '==> Generating golden frames' -ForegroundColor Cyan
    & $go run ./genframes -out $golden
    if ($LASTEXITCODE -ne 0) { throw 'genframes failed' }

    Write-Host '==> Building Go frame2png' -ForegroundColor Cyan
    $toolDir = Join-Path $root 'build/tools'
    New-Item -ItemType Directory -Force -Path $toolDir | Out-Null
    & $go build -o "$toolDir/frame2png.exe" ./frame2png
    if ($LASTEXITCODE -ne 0) { throw 'frame2png build failed' }

    Write-Host '==> Rendering with both implementations' -ForegroundColor Cyan
    Get-ChildItem -Path $golden -Filter '*.bin' | ForEach-Object {
        $name = $_.BaseName
        python (Join-Path $PSScriptRoot 'oracle/frame2png.py') $_.FullName (Join-Path $golden "$name.oracle.png") | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "oracle render failed: $name" }
        & "$toolDir/frame2png.exe" -in $_.FullName -out (Join-Path $golden "$name.go.png")
        if ($LASTEXITCODE -ne 0) { throw "Go render failed: $name" }
    }

    Write-Host '==> Comparing pixel by pixel' -ForegroundColor Cyan
    python (Join-Path $PSScriptRoot 'oracle/pngdiff.py') $golden
    if ($LASTEXITCODE -ne 0) { throw 'pixel comparison failed' }
}
finally { Pop-Location }
