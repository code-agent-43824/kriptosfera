param(
  [string]$LockPath = "build/cryptopro-plugin-lock.json",
  [string]$OutputPath = "internal/bootstrap/cryptopro-plugin.zip",
  [string]$MetadataOutputPath = "dist/cryptopro-plugin.json"
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.IO.Compression.FileSystem

function Get-CryptoProPluginSlimEntryName {
  param([string]$EntryName)

  $cleanName = $EntryName.Replace("\", "/")
  $parts = @($cleanName -split "/" | Where-Object { $_ -ne "" })
  if ($parts.Count -eq 0) {
    return $null
  }
  foreach ($part in $parts) {
    if ($part.Contains(":")) {
      return $null
    }
  }
  if ($parts.Count -ge 2 -and $parts[0] -eq "CAdES Browser Plug-in") {
    return ($parts -join "/")
  }
  for ($i = 0; $i -le ($parts.Count - 3); $i++) {
    if ($parts[$i] -eq "Program Files" -and $parts[$i + 1] -eq "Crypto Pro") {
      $relativeParts = $parts[($i + 2)..($parts.Count - 1)]
      if ($relativeParts.Count -eq 0) {
        return $null
      }
      return ($relativeParts -join "/")
    }
  }
  return $null
}

function New-SlimCryptoProPluginArchive {
  param(
    [string]$InputPath,
    [string]$OutputPath
  )

  $requiredEntries = @(
    "CAdES Browser Plug-in/nmcades.exe",
    "CAdES Browser Plug-in/nmcades.json",
    "CAdES Browser Plug-in/npcades.dll",
    "CAdES Browser Plug-in/cades.dll",
    "CAdES Browser Plug-in/xades.dll",
    "CAdES Browser Plug-in/cplib.dll",
    "CAdES Browser Plug-in/Mini CSP/capi10.dll",
    "CAdES Browser Plug-in/Mini CSP/capi20.dll",
    "CAdES Browser Plug-in/Mini CSP/cpcspi.dll",
    "CAdES Browser Plug-in/Mini CSP/cpsuprt.dll",
    "CAdES Browser Plug-in/Mini CSP/cpui.dll"
  )

  if (Test-Path $OutputPath) {
    Remove-Item -Force $OutputPath
  }

  $source = [System.IO.Compression.ZipFile]::OpenRead($InputPath)
  $destination = [System.IO.Compression.ZipFile]::Open($OutputPath, [System.IO.Compression.ZipArchiveMode]::Create)
  $seen = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
  try {
    foreach ($entry in $source.Entries) {
      if ($entry.FullName.EndsWith("/")) {
        continue
      }
      $targetName = Get-CryptoProPluginSlimEntryName -EntryName $entry.FullName
      if (-not $targetName) {
        continue
      }
      if (-not $seen.Add($targetName)) {
        throw "Duplicate CryptoPro slim archive entry: $targetName"
      }
      $newEntry = $destination.CreateEntry($targetName, [System.IO.Compression.CompressionLevel]::Optimal)
      $newEntry.LastWriteTime = $entry.LastWriteTime
      $inputStream = $entry.Open()
      $outputStream = $newEntry.Open()
      try {
        $inputStream.CopyTo($outputStream)
      } finally {
        $outputStream.Dispose()
        $inputStream.Dispose()
      }
    }
  } finally {
    $destination.Dispose()
    $source.Dispose()
  }

  $slim = [System.IO.Compression.ZipFile]::OpenRead($OutputPath)
  try {
    $present = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
    foreach ($entry in $slim.Entries) {
      if (-not $entry.FullName.EndsWith("/")) {
        [void]$present.Add($entry.FullName.Replace("\", "/"))
      }
    }
    foreach ($required in $requiredEntries) {
      if (-not $present.Contains($required)) {
        throw "Slim CryptoPro plugin archive is missing required entry: $required"
      }
    }
    return $present.Count
  } finally {
    $slim.Dispose()
  }
}

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
$tempSlimPath = "$OutputPath.slim"
$tempMetadataPath = "$MetadataOutputPath.download"
if (Test-Path $tempPath) {
  Remove-Item -Force $tempPath
}
if (Test-Path $tempSlimPath) {
  Remove-Item -Force $tempSlimPath
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

$slimEntryCount = New-SlimCryptoProPluginArchive -InputPath $tempPath -OutputPath $tempSlimPath
$slimHash = (Get-FileHash -Algorithm SHA256 -Path $tempSlimPath).Hash.ToLowerInvariant()
$slimSize = (Get-Item $tempSlimPath).Length

Move-Item -Force $tempSlimPath $OutputPath
Move-Item -Force $tempMetadataPath $MetadataOutputPath
Remove-Item -Force $tempPath

Write-Host "Fetched CryptoPro plugin bundle: $($lock.version) $($lock.sha256) size=$downloadedSize"
Write-Host "Slim CryptoPro plugin archive: $OutputPath size=$slimSize sha256=$slimHash entries=$slimEntryCount"
Write-Host "CryptoPro plugin metadata: $MetadataOutputPath"
