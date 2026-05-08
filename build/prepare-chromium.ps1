param(
  [string]$ConfigPath = "build/chromium-runtime.json",
  [string]$OutputDir = "payload/chromium",
  [string]$CacheDir = ".build-cache/chromium"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $ConfigPath)) {
  throw "Chromium config not found: $ConfigPath"
}

$config = Get-Content $ConfigPath -Raw | ConvertFrom-Json
if (-not $config.version -or -not $config.url -or -not $config.archiveName -or -not $config.extractRoot) {
  throw "Chromium config is incomplete"
}

$versionCacheDir = Join-Path $CacheDir $config.version
$archivePath = Join-Path $versionCacheDir $config.archiveName
$extractDir = Join-Path $versionCacheDir $config.extractRoot

New-Item -ItemType Directory -Force -Path $versionCacheDir | Out-Null

if (-not (Test-Path $archivePath)) {
  Write-Host "Downloading Chromium runtime $($config.version)"
  Invoke-WebRequest -Uri $config.url -OutFile $archivePath
} else {
  Write-Host "Using cached Chromium archive $archivePath"
}

if (-not (Test-Path $extractDir)) {
  Write-Host "Extracting Chromium runtime to $extractDir"
  Expand-Archive -Path $archivePath -DestinationPath $versionCacheDir -Force
} else {
  Write-Host "Using cached Chromium extraction $extractDir"
}

$chromeExe = Join-Path $extractDir "chrome.exe"
if (-not (Test-Path $chromeExe)) {
  throw "Chromium executable not found: $chromeExe"
}

if (Test-Path $OutputDir) {
  Remove-Item -Recurse -Force $OutputDir
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
Copy-Item -Recurse -Force (Join-Path $extractDir "*") $OutputDir
Write-Host "Chromium runtime prepared at $OutputDir"
