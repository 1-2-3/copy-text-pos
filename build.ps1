#requires -Version 5.1
<#
.SYNOPSIS
  Build Copy Text Pos for Windows x64.
.EXAMPLE
  .\build.ps1
  .\build.ps1 -Version 0.1.3 -Package
  .\build.ps1 -Console -Output bin\copy-text-pos-dev.exe
#>
param(
  [string]$Version = "0.1.3",
  [string]$Output = "bin\copy-text-pos.exe",
  [switch]$Console,
  [switch]$Package,
  [switch]$RegenerateResources
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

$go = Get-Command go -ErrorAction Stop
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

$commit = "unknown"
if ((Test-Path (Join-Path $root ".git")) -and (Get-Command git -ErrorAction SilentlyContinue)) {
  $previousPreference = $ErrorActionPreference
  $ErrorActionPreference = "SilentlyContinue"
  $candidate = (git -C $root rev-parse --short HEAD 2>$null)
  $gitExit = $LASTEXITCODE
  $ErrorActionPreference = $previousPreference
  if ($gitExit -eq 0 -and $candidate) { $commit = $candidate.Trim() }
}
$buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$syso = Join-Path $root "cmd\copy-text-pos\rsrc_windows_amd64.syso"
if ($RegenerateResources -or -not (Test-Path $syso)) {
  $windres = Get-Command windres -ErrorAction SilentlyContinue
  if ($windres) {
    Push-Location "cmd\copy-text-pos"
    try {
      & $windres.Source -F pe-x86-64 -I. resource.rc -O coff -o rsrc_windows_amd64.syso
      if ($LASTEXITCODE -ne 0) { throw "windres failed with exit code $LASTEXITCODE" }
    } finally {
      Pop-Location
    }
  } else {
    Write-Warning "windres was not found; building without the executable icon and manifest."
  }
}

$outPath = Join-Path $root $Output
$outDir = Split-Path -Parent $outPath
if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Path $outDir | Out-Null }

$ldflags = @(
  "-s", "-w",
  "-X", "copytextpos/internal/version.Version=$Version",
  "-X", "copytextpos/internal/version.Commit=$commit",
  "-X", "copytextpos/internal/version.BuildTime=$buildTime"
) -join " "
if (-not $Console) { $ldflags += " -H windowsgui" }

Write-Host "Building Copy Text Pos $Version ($commit)..."
& $go.Source build -trimpath -buildvcs=false -ldflags $ldflags -o $outPath .\cmd\copy-text-pos
if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

$hash = (Get-FileHash -Algorithm SHA256 $outPath).Hash.ToLowerInvariant()
Write-Host ("Built {0} ({1:N0} bytes)" -f $outPath, (Get-Item $outPath).Length)
Write-Host "SHA256 $hash"
$exeName = Split-Path -Leaf $outPath
Set-Content -Path "$outPath.sha256" -Value "$hash  $exeName" -Encoding ascii

if ($Package) {
  $zip = Join-Path $outDir "copy-text-pos-$Version-win-x64.zip"
  $stage = Join-Path $env:TEMP ("ctp-package-" + [guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path $stage | Out-Null
  Copy-Item $outPath (Join-Path $stage "copy-text-pos.exe")
  Copy-Item "$outPath.sha256" (Join-Path $stage "copy-text-pos.exe.sha256")
  foreach ($name in @("README.md", "README.zh-CN.md", "LICENSE")) {
    if (Test-Path $name) { Copy-Item $name $stage }
  }
  if (Test-Path "docs") {
    Copy-Item "docs" (Join-Path $stage "docs") -Recurse
  }
  $vsix = Get-ChildItem "integrations\vscode\*.vsix" -File -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTime -Descending | Select-Object -First 1
  if ($vsix) { Copy-Item $vsix.FullName (Join-Path $stage $vsix.Name) }
  if (Test-Path $zip) { Remove-Item $zip -Force }
  Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $zip
  Remove-Item $stage -Recurse -Force
  Write-Host "Package: $zip"
}
