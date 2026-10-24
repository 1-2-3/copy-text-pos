#requires -Version 5.1
<#
.SYNOPSIS
  Rebuild the Windows icon/manifest resource object from resource.rc.
#>
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$target = Join-Path $root "cmd\copy-text-pos"
Push-Location $target
try {
  if (-not (Get-Command windres -ErrorAction SilentlyContinue)) {
    throw "windres (GNU binutils) is required. Install MSYS2 or use the checked-in .syso."
  }
  windres -F pe-x86-64 -I. resource.rc -O coff -o rsrc_windows_amd64.syso
  if ($LASTEXITCODE -ne 0) { throw "windres failed with exit code $LASTEXITCODE" }
  Get-Item rsrc_windows_amd64.syso | Select-Object FullName, Length
} finally {
  Pop-Location
}
