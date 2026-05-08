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

$appConfig = Get-Content (Join-Path $OutputDir "config/app-config.json") -Raw | ConvertFrom-Json
if (-not $appConfig.version) {
  throw "Payload app-config.json must contain version"
}

$manifestFiles = @()
Get-ChildItem -Path $OutputDir -File -Recurse |
  Sort-Object FullName |
  ForEach-Object {
    $relativePath = [System.IO.Path]::GetRelativePath((Resolve-Path $OutputDir), $_.FullName).Replace("\", "/")
    if ($relativePath -eq "manifest.json") {
      return
    }

    $manifestFiles += [ordered]@{
      path = $relativePath
      sha256 = (Get-FileHash -Algorithm SHA256 -Path $_.FullName).Hash.ToLowerInvariant()
    }
  }

$manifest = [ordered]@{
  version = [string]$appConfig.version
  files = $manifestFiles
}

$manifest | ConvertTo-Json -Depth 4 | Set-Content -Path (Join-Path $OutputDir "manifest.json") -Encoding utf8NoBOM
Write-Host "Payload prepared at $OutputDir with manifest.json"
