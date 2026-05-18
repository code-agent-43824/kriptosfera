param(
  [string]$LockPath = "build/cryptopro-plugin-lock.json",
  [string]$OutputPath = "internal/bootstrap/cryptopro-plugin.zip",
  [string]$MetadataOutputPath = "dist/cryptopro-plugin.json"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $LockPath)) {
  throw "CryptoPro plugin lock file not found: $LockPath"
}

$lock = Get-Content $LockPath -Raw | ConvertFrom-Json

if ($lock.component -ne "cryptopro-browser-plugin") {
  throw "Unexpected CryptoPro plugin component: $($lock.component)"
}
if ($lock.platform -ne "windows-amd64") {
  throw "Unsupported CryptoPro plugin platform: $($lock.platform)"
}
if (-not $lock.url) {
  throw "CryptoPro plugin lock file does not contain URL"
}
if (-not $lock.url.StartsWith("https://")) {
  throw "CryptoPro plugin URL must start with https://"
}
if (-not $lock.metadataUrl) {
  throw "CryptoPro plugin lock file does not contain metadata URL"
}
if (-not $lock.metadataUrl.StartsWith("https://")) {
  throw "CryptoPro plugin metadata URL must start with https://"
}
if (-not $lock.sha256) {
  throw "CryptoPro plugin lock file does not contain SHA256"
}
if ([string]$lock.sha256 -notmatch "^[a-fA-F0-9]{64}$") {
  throw "CryptoPro plugin SHA256 is invalid"
}
if ([long]$lock.size -le 0) {
  throw "CryptoPro plugin size must be positive"
}

$outputDir = Split-Path -Parent $OutputPath
if ($outputDir -and -not (Test-Path $outputDir)) {
  New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$metadataDir = Split-Path -Parent $MetadataOutputPath
if ($metadataDir -and -not (Test-Path $metadataDir)) {
  New-Item -ItemType Directory -Force -Path $metadataDir | Out-Null
}

$tempPath = "$OutputPath.download"
$tempMetadataPath = "$MetadataOutputPath.download"
if (Test-Path $tempPath) {
  Remove-Item -Force $tempPath
}
if (Test-Path $tempMetadataPath) {
  Remove-Item -Force $tempMetadataPath
}

Invoke-WebRequest -Uri $lock.metadataUrl -OutFile $tempMetadataPath
Invoke-WebRequest -Uri $lock.url -OutFile $tempPath

$downloadedHash = (Get-FileHash -Algorithm SHA256 -Path $tempPath).Hash.ToLowerInvariant()
if ($downloadedHash -ne $lock.sha256.ToLowerInvariant()) {
  Remove-Item -Force $tempPath
  throw "Downloaded CryptoPro plugin hash mismatch. Expected $($lock.sha256), got $downloadedHash"
}

$downloadedSize = (Get-Item $tempPath).Length
if ($downloadedSize -ne [long]$lock.size) {
  Remove-Item -Force $tempPath
  throw "Downloaded CryptoPro plugin size mismatch. Expected $($lock.size), got $downloadedSize"
}

$metadata = Get-Content $tempMetadataPath -Raw | ConvertFrom-Json
if ($metadata.sha256.ToLowerInvariant() -ne $lock.sha256.ToLowerInvariant()) {
  Remove-Item -Force $tempPath
  Remove-Item -Force $tempMetadataPath
  throw "Downloaded CryptoPro plugin metadata SHA256 mismatch"
}
if ([long]$metadata.size -ne [long]$lock.size) {
  Remove-Item -Force $tempPath
  Remove-Item -Force $tempMetadataPath
  throw "Downloaded CryptoPro plugin metadata size mismatch"
}

Move-Item -Force $tempPath $OutputPath
Move-Item -Force $tempMetadataPath $MetadataOutputPath

Write-Host "Fetched CryptoPro plugin bundle: $($lock.version) $($lock.sha256)"
Write-Host "CryptoPro plugin archive: $OutputPath"
Write-Host "CryptoPro plugin metadata: $MetadataOutputPath"
