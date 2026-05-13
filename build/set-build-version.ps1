param(
  [string]$Version = "0.1.0",
  [string]$PayloadConfigPath = "payload-template/config/app-config.json"
)

$ErrorActionPreference = "Stop"

Set-Content -Path "internal/bootstrap/app-version.txt" -Value $Version -NoNewline
Set-Content -Path "internal/config/app-version.txt" -Value $Version -NoNewline

if (-not (Test-Path $PayloadConfigPath)) {
  throw "Payload config not found: $PayloadConfigPath"
}

$appConfig = Get-Content $PayloadConfigPath -Raw | ConvertFrom-Json
$appConfig.version = $Version
$appConfig | ConvertTo-Json -Depth 10 | Set-Content -Path $PayloadConfigPath -Encoding utf8NoBOM

Write-Host "Build version set to $Version"
