param(
  [string]$Version = "0.1.0",
  [string]$OutputDir = "dist",
  [string]$PublishRoot = "dist/published/payloads/win64/demo"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

pwsh ./build/set-build-version.ps1 -Version $Version
pwsh ./build/prepare-payload.ps1

$payloadZip = Join-Path $OutputDir "payload.zip"
$payloadJson = Join-Path $OutputDir "payload.json"

pwsh ./build/package-payload.ps1 -PayloadDir "payload" -OutputZip $payloadZip -MetadataPath $payloadJson
pwsh ./build/publish-payload.ps1 -PayloadZip $payloadZip -PayloadMetadata $payloadJson -PublishRoot $PublishRoot

Copy-Item README.md (Join-Path $OutputDir "README.txt")

Write-Host "Payload build completed: $payloadZip"
