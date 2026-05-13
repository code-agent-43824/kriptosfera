param(
  [string]$Version = "0.1.0",
  [string]$OutputDir = "dist",
  [ValidateSet("embedded", "remote")]
  [string]$PayloadMode = "embedded",
  [string]$PayloadZip = "dist/payload.zip",
  [string]$PayloadMetadata = "dist/payload.json",
  [string]$PayloadBaseUrl = ""
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
pwsh ./build/set-build-version.ps1 -Version $Version

if ($PayloadMode -eq "embedded") {
  if (-not (Test-Path $PayloadZip)) {
    throw "Payload zip not found: $PayloadZip"
  }

  pwsh ./build/generate-runtime-config.ps1 -PayloadMode embedded -Version $Version
  Copy-Item -Force $PayloadZip "internal/bootstrap/payload.zip"
  $buildTags = @()
  $exeName = "KriptosferaDemo.exe"
  go test ./...
} else {
  if (-not (Test-Path $PayloadMetadata)) {
    throw "Payload metadata not found: $PayloadMetadata"
  }
  if (-not $PayloadBaseUrl) {
    throw "PayloadBaseUrl is required for remote mode"
  }
  if (-not $PayloadBaseUrl.StartsWith("https://")) {
    throw "PayloadBaseUrl must start with https://"
  }
  if ($PayloadBaseUrl -match "agent\.invalid") {
    throw "PayloadBaseUrl points to placeholder host agent.invalid; set a real HTTPS payload base URL"
  }

  $payloadMeta = Get-Content $PayloadMetadata -Raw | ConvertFrom-Json
  $payloadBase = $PayloadBaseUrl.TrimEnd('/')
  $payloadUrl = "$payloadBase/$($payloadMeta.payloadVersion)/$($payloadMeta.sha256)/payload.zip"

  pwsh ./build/generate-runtime-config.ps1 -PayloadMode remote -Version $Version -PayloadUrl $payloadUrl -PayloadSha256 $payloadMeta.sha256 -PayloadSize ([long]$payloadMeta.size)
  $buildTags = @("-tags", "remote")
  $exeName = "KriptosferaDemo-remote.exe"
}

$env:GOOS = "windows"
$env:GOARCH = "amd64"
& go build @buildTags -trimpath -ldflags "-H=windowsgui -s -w" -o (Join-Path $OutputDir $exeName) ./cmd/kriptosfera-launcher

Copy-Item README.md (Join-Path $OutputDir "README.txt")
Write-Host "Launcher build completed: $(Join-Path $OutputDir $exeName)"
