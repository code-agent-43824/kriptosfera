param(
  [string]$Version = "0.1.0",
  [string]$OutputDir = "dist",
  [ValidateSet("embedded", "remote")]
  [string]$PayloadMode = "embedded",
  [string]$PayloadBaseUrl = "",
  [string]$PayloadLockPath = "build/payload-lock.json"
)

$ErrorActionPreference = "Stop"

pwsh ./build/fetch-payload.ps1 -PayloadLockPath $PayloadLockPath -OutputDir $OutputDir
pwsh ./build/build-launcher.ps1 -Version $Version -OutputDir $OutputDir -PayloadMode $PayloadMode -PayloadZip (Join-Path $OutputDir "payload.zip") -PayloadMetadata (Join-Path $OutputDir "payload.json") -PayloadBaseUrl $PayloadBaseUrl

Write-Host "Launcher build completed against stable payload for mode=$PayloadMode"
