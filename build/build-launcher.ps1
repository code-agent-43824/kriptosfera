param(
  [string]$Version = "0.1.0",
  [string]$OutputDir = "dist",
  [ValidateSet("embedded", "remote")]
  [string]$PayloadMode = "embedded",
  [string]$PayloadZip = "dist/payload.zip",
  [string]$PayloadMetadata = "dist/payload.json",
  [string]$PayloadBaseUrl = "",
  [string]$PayloadLockPath = "build/payload-lock.json",
  [switch]$UsePayloadLock,
  [string]$CryptoProPluginLockPath = "build/cryptopro-plugin-lock.json"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
pwsh ./build/set-build-version.ps1 -Version $Version
pwsh ./build/fetch-cryptopro-plugin.ps1 -LockPath $CryptoProPluginLockPath -OutputPath "internal/bootstrap/cryptopro-plugin.zip" -MetadataOutputPath (Join-Path $OutputDir "cryptopro-plugin.json")

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
  if ($UsePayloadLock) {
    if (-not (Test-Path $PayloadLockPath)) {
      throw "Payload lock not found: $PayloadLockPath"
    }

    $payloadLock = Get-Content $PayloadLockPath -Raw | ConvertFrom-Json
    if (-not $payloadLock.url) {
      throw "Payload lock is missing url: $PayloadLockPath"
    }
    if (-not $payloadLock.url.StartsWith("https://")) {
      throw "Payload lock url must start with https://"
    }
    if (-not $payloadLock.sha256) {
      throw "Payload lock is missing sha256: $PayloadLockPath"
    }
    if ([long]$payloadLock.size -le 0) {
      throw "Payload lock has invalid size: $PayloadLockPath"
    }

    pwsh ./build/generate-runtime-config.ps1 -PayloadMode remote -Version $Version -PayloadUrl $payloadLock.url -PayloadSha256 $payloadLock.sha256 -PayloadSize ([long]$payloadLock.size)
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
  }
  $buildTags = @("-tags", "remote")
  $exeName = "KriptosferaDemo-remote.exe"
}

$env:GOOS = "windows"
$env:GOARCH = "amd64"
& go build @buildTags -trimpath -ldflags "-H=windowsgui -s -w" -o (Join-Path $OutputDir $exeName) ./cmd/kriptosfera-launcher

Copy-Item README.md (Join-Path $OutputDir "README.txt")
Write-Host "Launcher build completed: $(Join-Path $OutputDir $exeName)"
