param(
  [string]$LockPath = "build/cryptopro-plugin-lock.json",
  [string]$RutokenFkcLockPath = "build/rutoken-fkc-lock.json",
  [string]$OutputPath = "internal/bootstrap/cryptopro-plugin.zip",
  [string]$MetadataOutputPath = "dist/cryptopro-plugin.json"
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.IO.Compression.FileSystem

$rutokenFkcCarrierConfigFragment = @"

; --- Kriptosfera Rutoken FKC / PKCS#11 overlay ---
[KeyCarriers\rutokenfkc]
DLL = "cpfkc.dll"

[KeyCarriers\rutokenfkc\Default]
atr = hex: 3B,8B,01,52,75,74,6F,6B,65,6E,20,44,53,20,C1
mask = hex: FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF
Name = "Rutoken FKC"

[KeyCarriers\rutokenfkc_nfc]
DLL = "cpfkc.dll"

[KeyCarriers\rutokenfkc_nfc\Default]
atr = hex: 3B,88,80,01,52,74,53,43,77,81,83,20,6A
mask = hex: FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF
Name = "Rutoken FKC NFC"

[KeyCarriers\rutokenfkc_nfc\Contact]
atr = hex: 3B,9C,96,80,11,40,52,75,74,6F,6B,65,6E,45,43,50,73,63,C0
mask = hex: FF,FF,FE,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FE
Name = "Rutoken FKC NFC"
"@

$rutokenPkcs11ConfigFragment = @"

; --- Kriptosfera Rutoken PKCS#11 active overlay ---

[KeyDevices\cryptoki_rutoken]
"DLL"="cryptoki.dll"
"Group"=1

[KeyDevices\cryptoki_rutoken\"PNP cryptoki"]
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"\Default]
pkcs11_dll = "rtPKCS11ECP.dll"
"@

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

function Get-ZipFileEntryCount {
  param([string]$Path)

  $archive = [System.IO.Compression.ZipFile]::OpenRead($Path)
  try {
    $count = 0
    foreach ($entry in $archive.Entries) {
      if (-not $entry.FullName.EndsWith("/")) {
        $count++
      }
    }
    return $count
  } finally {
    $archive.Dispose()
  }
}

function Get-RutokenFkcOverlayFiles {
  param(
    [string]$LockPath,
    [string]$WorkDir
  )

  if (-not (Test-Path $LockPath)) {
    throw "Rutoken FKC lock file not found: $LockPath"
  }

  $lock = Get-Content $LockPath -Raw | ConvertFrom-Json
  if ($lock.component -ne "rutoken-fkc-pkcs11-mini-csp-overlay") {
    throw "Unexpected Rutoken FKC component: $($lock.component)"
  }
  if ($lock.platform -ne "windows-x86") {
    throw "Unsupported Rutoken FKC platform: $($lock.platform)"
  }
  if (-not $lock.files -or $lock.files.Count -eq 0) {
    throw "Rutoken FKC lock file does not contain files"
  }

  if (-not (Test-Path $WorkDir)) {
    New-Item -ItemType Directory -Force -Path $WorkDir | Out-Null
  }

  $downloaded = @()
  foreach ($file in $lock.files) {
    if (-not $file.targetName) {
      throw "Rutoken FKC lock file entry does not contain targetName"
    }
    if ([string]$file.targetName -notmatch "^[A-Za-z0-9_.-]+\.dll$") {
      throw "Rutoken FKC targetName is invalid: $($file.targetName)"
    }
    if (-not $file.url -or -not $file.url.StartsWith("https://")) {
      throw "Rutoken FKC file URL must start with https://: $($file.targetName)"
    }
    if (-not $file.sha256 -or [string]$file.sha256 -notmatch "^[a-fA-F0-9]{64}$") {
      throw "Rutoken FKC SHA256 is invalid: $($file.targetName)"
    }
    if ([long]$file.size -le 0) {
      throw "Rutoken FKC size must be positive: $($file.targetName)"
    }

    $downloadPath = Join-Path $WorkDir $file.targetName
    if (Test-Path $downloadPath) {
      Remove-Item -Force $downloadPath
    }
    Invoke-WebRequest -Uri $file.url -OutFile $downloadPath

    $hash = (Get-FileHash -Algorithm SHA256 -Path $downloadPath).Hash.ToLowerInvariant()
    if ($hash -ne $file.sha256.ToLowerInvariant()) {
      Remove-Item -Force $downloadPath
      throw "Downloaded Rutoken FKC file hash mismatch for $($file.targetName). Expected $($file.sha256), got $hash"
    }

    $size = (Get-Item $downloadPath).Length
    if ($size -ne [long]$file.size) {
      Remove-Item -Force $downloadPath
      throw "Downloaded Rutoken FKC file size mismatch for $($file.targetName). Expected $($file.size), got $size"
    }

    $downloaded += [pscustomobject]@{
      TargetName       = [string]$file.targetName
      Path             = $downloadPath
      LastWriteTimeUtc = [string]$file.lastWriteTimeUtc
      Size             = $size
      SHA256           = $hash
    }
  }

  return $downloaded
}

function Set-ZipEntryFromFile {
  param(
    [System.IO.Compression.ZipArchive]$Archive,
    [string]$EntryName,
    [string]$SourcePath,
    [string]$LastWriteTimeUtc
  )

  $existing = $Archive.GetEntry($EntryName)
  if ($existing) {
    $existing.Delete()
  }

  $newEntry = $Archive.CreateEntry($EntryName, [System.IO.Compression.CompressionLevel]::Optimal)
  if ($LastWriteTimeUtc) {
    $newEntry.LastWriteTime = [System.DateTimeOffset]::Parse($LastWriteTimeUtc).ToUniversalTime()
  }
  $inputStream = [System.IO.File]::OpenRead($SourcePath)
  $outputStream = $newEntry.Open()
  try {
    $inputStream.CopyTo($outputStream)
  } finally {
    $outputStream.Dispose()
    $inputStream.Dispose()
  }
}

function Add-RutokenFkcOverlayToArchive {
  param(
    [string]$ArchivePath,
    [object[]]$OverlayFiles
  )

  $archive = [System.IO.Compression.ZipFile]::Open($ArchivePath, [System.IO.Compression.ZipArchiveMode]::Update)
  try {
    foreach ($file in $OverlayFiles) {
      Set-ZipEntryFromFile `
        -Archive $archive `
        -EntryName "CAdES Browser Plug-in/Mini CSP/$($file.TargetName)" `
        -SourcePath $file.Path `
        -LastWriteTimeUtc $file.LastWriteTimeUtc
    }

    $configEntryName = "CAdES Browser Plug-in/Mini CSP/config.ini"
    $configEntry = $archive.GetEntry($configEntryName)
    if (-not $configEntry) {
      throw "Slim CryptoPro plugin archive is missing Mini CSP config.ini"
    }

    [System.Text.Encoding]::RegisterProvider([System.Text.CodePagesEncodingProvider]::Instance)
    $encoding = [System.Text.Encoding]::GetEncoding(1251)
    $configTimestamp = $configEntry.LastWriteTime
    $inputStream = $configEntry.Open()
    try {
      $reader = [System.IO.StreamReader]::new($inputStream, $encoding, $false)
      try {
        $configText = $reader.ReadToEnd()
      } finally {
        $reader.Dispose()
      }
    } finally {
      $inputStream.Dispose()
    }

    if ($configText -notmatch "\[KeyCarriers\\rutokenfkc\]") {
      $configText = $configText.TrimEnd() + "`r`n" + $rutokenFkcCarrierConfigFragment.TrimStart() + "`r`n"
    }
    if ($configText -notmatch "\[KeyDevices\\cryptoki_rutoken\]") {
      $configText = $configText.TrimEnd() + "`r`n" + $rutokenPkcs11ConfigFragment.TrimStart() + "`r`n"
    }

    $configEntry.Delete()
    $newConfigEntry = $archive.CreateEntry($configEntryName, [System.IO.Compression.CompressionLevel]::Optimal)
    $newConfigEntry.LastWriteTime = $configTimestamp
    $outputStream = $newConfigEntry.Open()
    try {
      $writer = [System.IO.StreamWriter]::new($outputStream, $encoding)
      try {
        $writer.Write($configText)
      } finally {
        $writer.Dispose()
      }
    } finally {
      $outputStream.Dispose()
    }
  } finally {
    $archive.Dispose()
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
$tempRutokenFkcDir = "$OutputPath.rutoken-fkc"
if (Test-Path $tempPath) {
  Remove-Item -Force $tempPath
}
if (Test-Path $tempSlimPath) {
  Remove-Item -Force $tempSlimPath
}
if (Test-Path $tempMetadataPath) {
  Remove-Item -Force $tempMetadataPath
}
if (Test-Path $tempRutokenFkcDir) {
  Remove-Item -Recurse -Force $tempRutokenFkcDir
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

[void](New-SlimCryptoProPluginArchive -InputPath $tempPath -OutputPath $tempSlimPath)
$rutokenFkcFiles = Get-RutokenFkcOverlayFiles -LockPath $RutokenFkcLockPath -WorkDir $tempRutokenFkcDir
Add-RutokenFkcOverlayToArchive -ArchivePath $tempSlimPath -OverlayFiles $rutokenFkcFiles
$slimEntryCount = Get-ZipFileEntryCount -Path $tempSlimPath
$slimHash = (Get-FileHash -Algorithm SHA256 -Path $tempSlimPath).Hash.ToLowerInvariant()
$slimSize = (Get-Item $tempSlimPath).Length

Move-Item -Force $tempSlimPath $OutputPath
Move-Item -Force $tempMetadataPath $MetadataOutputPath
Remove-Item -Force $tempPath
Remove-Item -Recurse -Force $tempRutokenFkcDir

Write-Host "Fetched CryptoPro plugin bundle: $($lock.version) $($lock.sha256) size=$downloadedSize"
Write-Host "Overlayed Rutoken FKC/PKCS#11 files: $($rutokenFkcFiles.Count)"
Write-Host "Slim CryptoPro plugin archive: $OutputPath size=$slimSize sha256=$slimHash entries=$slimEntryCount"
Write-Host "CryptoPro plugin metadata: $MetadataOutputPath"
