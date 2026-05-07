param(
  [string]$PayloadTemplate = "payload-template",
  [string]$OutputDir = "payload"
)

$ErrorActionPreference = "Stop"

if (Test-Path $OutputDir) {
  Remove-Item -Recurse -Force $OutputDir
}

New-Item -ItemType Directory -Path $OutputDir | Out-Null
Copy-Item -Recurse -Force (Join-Path $PayloadTemplate "*") $OutputDir

$required = @(
  "config/app-config.json",
  "diagnostics/diagnostics.html"
)

foreach ($item in $required) {
  $path = Join-Path $OutputDir $item
  if (-not (Test-Path $path)) {
    throw "Missing payload file: $item"
  }
}

Write-Host "Payload prepared at $OutputDir"
