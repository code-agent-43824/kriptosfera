param(
  [string]$PayloadDir = "payload",
  [string]$OutputZip = "internal/bootstrap/payload.zip"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $PayloadDir)) {
  throw "Payload directory not found: $PayloadDir"
}

$zipDir = Split-Path -Parent $OutputZip
if (-not (Test-Path $zipDir)) {
  New-Item -ItemType Directory -Path $zipDir | Out-Null
}

if (Test-Path $OutputZip) {
  Remove-Item -Force $OutputZip
}

Compress-Archive -Path (Join-Path $PayloadDir "*") -DestinationPath $OutputZip -CompressionLevel Optimal
Write-Host "Embedded payload archive created at $OutputZip"
