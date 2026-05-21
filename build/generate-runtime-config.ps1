param(
  [ValidateSet("embedded", "remote")]
  [string]$PayloadMode = "embedded",
  [string]$Version = "0.5.0",
  [string]$ProductName = "Kriptosfera Demo",
  [string]$PayloadUrl = "",
  [string]$PayloadSha256 = "",
  [long]$PayloadSize = 0,
  [string]$OutputPath = "internal/config/runtime-config.json"
)

$ErrorActionPreference = "Stop"

$outDir = Split-Path -Parent $OutputPath
if ($outDir -and -not (Test-Path $outDir)) {
  New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

$payload = [ordered]@{
  mode = $PayloadMode
  version = $Version
}

if ($PayloadMode -eq "remote") {
  if (-not $PayloadUrl) { throw "PayloadUrl is required for remote mode" }
  if (-not $PayloadSha256) { throw "PayloadSha256 is required for remote mode" }
  $payload.url = $PayloadUrl
  $payload.sha256 = $PayloadSha256.ToLowerInvariant()
  $payload.size = $PayloadSize
}

$config = [ordered]@{
  productName = $ProductName
  version = $Version
  payload = $payload
}

$config | ConvertTo-Json -Depth 4 | Set-Content -Path $OutputPath -Encoding utf8NoBOM
Write-Host "Runtime config generated at $OutputPath for mode=$PayloadMode"
