param(
  [string]$PayloadZip = "dist/payload.zip",
  [string]$PayloadMetadata = "dist/payload.json",
  [string]$PublishRoot = "dist/published/payloads/win64/demo"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $PayloadZip)) {
  throw "Payload zip not found: $PayloadZip"
}
if (-not (Test-Path $PayloadMetadata)) {
  throw "Payload metadata not found: $PayloadMetadata"
}

$metadata = Get-Content $PayloadMetadata -Raw | ConvertFrom-Json
if (-not $metadata.payloadVersion) { throw "payloadVersion missing in metadata" }
if (-not $metadata.sha256) { throw "sha256 missing in metadata" }

$targetDir = Join-Path $PublishRoot ([System.IO.Path]::Combine([string]$metadata.payloadVersion, [string]$metadata.sha256))
New-Item -ItemType Directory -Force -Path $targetDir | Out-Null

Copy-Item -Force $PayloadZip (Join-Path $targetDir "payload.zip")
Copy-Item -Force $PayloadMetadata (Join-Path $targetDir "payload.json")

Write-Host "Payload published to $targetDir"
