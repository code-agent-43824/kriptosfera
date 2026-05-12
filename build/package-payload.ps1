param(
  [string]$PayloadDir = "payload",
  [string]$OutputZip = "dist/payload.zip",
  [string]$MetadataPath = "dist/payload.json",
  [string]$AppId = "ru.kriptosfera.demo",
  [string]$Platform = "win64"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $PayloadDir)) {
  throw "Payload directory not found: $PayloadDir"
}

$outDir = Split-Path -Parent $OutputZip
if ($outDir -and -not (Test-Path $outDir)) {
  New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

$metaDir = Split-Path -Parent $MetadataPath
if ($metaDir -and -not (Test-Path $metaDir)) {
  New-Item -ItemType Directory -Force -Path $metaDir | Out-Null
}

if (Test-Path $OutputZip) {
  Remove-Item -Force $OutputZip
}

Compress-Archive -Path (Join-Path $PayloadDir "*") -DestinationPath $OutputZip -CompressionLevel Optimal

$hash = (Get-FileHash -Algorithm SHA256 -Path $OutputZip).Hash.ToLowerInvariant()
$size = (Get-Item $OutputZip).Length
$appConfig = Get-Content (Join-Path $PayloadDir "config/app-config.json") -Raw | ConvertFrom-Json

$metadata = [ordered]@{
  appId = $AppId
  platform = $Platform
  payloadVersion = [string]$appConfig.version
  archive = [System.IO.Path]::GetFileName($OutputZip)
  sha256 = $hash
  size = $size
  createdAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
}

$metadata | ConvertTo-Json -Depth 4 | Set-Content -Path $MetadataPath -Encoding utf8NoBOM

Write-Host "Payload archive created at $OutputZip"
Write-Host "Payload metadata created at $MetadataPath"
Write-Host "Payload SHA256: $hash"
Write-Host "Payload size: $size"
