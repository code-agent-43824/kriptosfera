param(
  [string]$Version = "0.1.0",
  [string]$OutputDir = "dist",
  [ValidateSet("embedded", "remote")]
  [string]$PayloadMode = "embedded",
  [string]$PayloadBaseUrl = "https://agent.invalid/payloads/win64/demo"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
Set-Content -Path "internal/bootstrap/app-version.txt" -Value $Version -NoNewline
Set-Content -Path "internal/config/app-version.txt" -Value $Version -NoNewline

pwsh ./build/prepare-payload.ps1

$payloadZip = Join-Path $OutputDir "payload.zip"
$payloadJson = Join-Path $OutputDir "payload.json"
pwsh ./build/package-payload.ps1 -PayloadDir "payload" -OutputZip $payloadZip -MetadataPath $payloadJson
pwsh ./build/publish-payload.ps1 -PayloadZip $payloadZip -PayloadMetadata $payloadJson

$payloadMeta = Get-Content $payloadJson -Raw | ConvertFrom-Json
$payloadUrl = "$PayloadBaseUrl/$($payloadMeta.payloadVersion)/$($payloadMeta.sha256)/payload.zip"

if ($PayloadMode -eq "embedded") {
  pwsh ./build/generate-runtime-config.ps1 -PayloadMode embedded -Version $Version
  pwsh ./build/embed-payload.ps1 -OutputZip "internal/bootstrap/payload.zip"
  $buildTags = @()
  $exeName = "KriptosferaDemo.exe"
  go test ./...
} else {
  pwsh ./build/generate-runtime-config.ps1 -PayloadMode remote -Version $Version -PayloadUrl $payloadUrl -PayloadSha256 $payloadMeta.sha256 -PayloadSize ([long]$payloadMeta.size)
  $buildTags = @("-tags", "remote")
  $exeName = "KriptosferaDemo-remote.exe"
}

$env:GOOS = "windows"
$env:GOARCH = "amd64"
& go build @buildTags -trimpath -ldflags "-H=windowsgui -s -w" -o (Join-Path $OutputDir $exeName) ./cmd/kriptosfera-launcher

Copy-Item README.md (Join-Path $OutputDir "README.txt")
Write-Host "Build completed: $(Join-Path $OutputDir $exeName)"
