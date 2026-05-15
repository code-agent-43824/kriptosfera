param(
  [string]$Version = "0.1.0",
  [string]$OutputDir = "dist",
  [ValidateSet("embedded", "remote")]
  [string]$PayloadMode = "embedded",
  [string]$PayloadBaseUrl = "",
  [string]$PayloadLockPath = "build/payload-lock.json",
  [switch]$UseStablePayload
)

$ErrorActionPreference = "Stop"

if ($UseStablePayload) {
  pwsh ./build/fetch-payload.ps1 -PayloadLockPath $PayloadLockPath -OutputDir $OutputDir
  Write-Host "Using stable published payload from lock file"
} else {
  pwsh ./build/build-payload.ps1 -OutputDir $OutputDir
  Write-Host "Using payload built from current checkout"
}

pwsh ./build/build-launcher.ps1 -Version $Version -OutputDir $OutputDir -PayloadMode $PayloadMode -PayloadZip (Join-Path $OutputDir "payload.zip") -PayloadMetadata (Join-Path $OutputDir "payload.json") -PayloadBaseUrl $PayloadBaseUrl

Write-Host "Launcher build completed for mode=$PayloadMode"
