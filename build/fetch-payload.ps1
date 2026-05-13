param(
  [string]$PayloadLockPath = "build/payload-lock.json",
  [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $PayloadLockPath)) {
  throw "Payload lock file not found: $PayloadLockPath"
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

$lock = Get-Content $PayloadLockPath -Raw | ConvertFrom-Json

if (-not $lock.url) {
  throw "Payload lock file does not contain payload URL"
}
if (-not $lock.metadataUrl) {
  throw "Payload lock file does not contain metadata URL"
}
if (-not $lock.sha256) {
  throw "Payload lock file does not contain payload SHA256"
}

$payloadZip = Join-Path $OutputDir "payload.zip"
$payloadJson = Join-Path $OutputDir "payload.json"

Invoke-WebRequest -Uri $lock.metadataUrl -OutFile $payloadJson
Invoke-WebRequest -Uri $lock.url -OutFile $payloadZip

$downloadedHash = (Get-FileHash -Algorithm SHA256 -Path $payloadZip).Hash.ToLowerInvariant()
if ($downloadedHash -ne $lock.sha256.ToLowerInvariant()) {
  throw "Downloaded payload hash mismatch. Expected $($lock.sha256), got $downloadedHash"
}

if ($lock.PSObject.Properties.Name -contains 'size' -and [long]$lock.size -gt 0) {
  $downloadedSize = (Get-Item $payloadZip).Length
  if ($downloadedSize -ne [long]$lock.size) {
    throw "Downloaded payload size mismatch. Expected $($lock.size), got $downloadedSize"
  }
}

$metadata = Get-Content $payloadJson -Raw | ConvertFrom-Json
if ($metadata.sha256.ToLowerInvariant() -ne $lock.sha256.ToLowerInvariant()) {
  throw "Downloaded payload metadata SHA256 mismatch"
}

Copy-Item README.md (Join-Path $OutputDir "README.txt")
Write-Host "Fetched stable payload: $($lock.payloadVersion) $($lock.sha256)"
