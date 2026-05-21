param(
  [string]$AppDir = "",
  [string]$OutputPath = "",
  [string]$ProcessName = "nmcades",
  [switch]$IncludeAllModules
)

$ErrorActionPreference = "Stop"

$filterTerms = @(
  "crypto pro",
  "cades",
  "npcades",
  "xades",
  "cplib",
  "capi10",
  "capi20",
  "capilite",
  "cpcspi",
  "cpsuprt",
  "cpui",
  "rutoken",
  "jacarta",
  "pcsc",
  "safenet"
)

function Normalize-PathForCompare {
  param([string]$Path)

  if (-not $Path) {
    return ""
  }
  return [System.IO.Path]::GetFullPath($Path).TrimEnd([System.IO.Path]::DirectorySeparatorChar).ToLowerInvariant()
}

function Test-InterestingModule {
  param(
    [string]$Name,
    [string]$Path
  )

  if ($IncludeAllModules) {
    return $true
  }

  $haystack = "$Name $Path".ToLowerInvariant()
  foreach ($term in $filterTerms) {
    if ($haystack.Contains($term)) {
      return $true
    }
  }
  return $false
}

function New-ModuleReport {
  param(
    [System.Diagnostics.ProcessModule]$Module,
    [string]$ResolvedAppDir
  )

  [ordered]@{
    name = $Module.ModuleName
    path = $Module.FileName
    origin = Resolve-Origin -Path $Module.FileName -ResolvedAppDir $ResolvedAppDir
    fileVersion = $Module.FileVersionInfo.FileVersion
    productVersion = $Module.FileVersionInfo.ProductVersion
    filterMatch = Test-InterestingModule -Name $Module.ModuleName -Path $Module.FileName
  }
}

function Get-ProcessPathSafe {
  param([System.Diagnostics.Process]$Process)

  try {
    return $Process.Path
  } catch {
    return ""
  }
}

function Get-ProcessStartTimeUtcString {
  param([System.Diagnostics.Process]$Process)

  try {
    if ($Process.StartTime) {
      return $Process.StartTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    }
  } catch {
    return $null
  }

  return $null
}

function Resolve-Origin {
  param(
    [string]$Path,
    [string]$ResolvedAppDir
  )

  $normalizedPath = Normalize-PathForCompare $Path
  if (-not $normalizedPath) {
    return "unknown"
  }

  $normalizedAppDir = Normalize-PathForCompare $ResolvedAppDir
  if ($normalizedAppDir -and $normalizedPath.StartsWith($normalizedAppDir)) {
    return "app"
  }

  $programFiles = Normalize-PathForCompare $env:ProgramFiles
  if ($programFiles -and $normalizedPath.StartsWith($programFiles)) {
    return "system"
  }

  $programFilesX86 = Normalize-PathForCompare ${env:ProgramFiles(x86)}
  if ($programFilesX86 -and $normalizedPath.StartsWith($programFilesX86)) {
    return "system"
  }

  $windowsDir = Normalize-PathForCompare $env:windir
  if ($windowsDir -and $normalizedPath.StartsWith($windowsDir)) {
    return "windows"
  }

  return "other"
}

function Infer-AppDirFromProcessPath {
  param([string]$Path)

  if (-not $Path) {
    return ""
  }

  $marker = "\cryptopro\plugin\"
  $index = $Path.ToLowerInvariant().IndexOf($marker)
  if ($index -lt 0) {
    return ""
  }

  return $Path.Substring(0, $index)
}

function Resolve-OutputPath {
  param(
    [string]$RequestedOutputPath,
    [string]$ResolvedAppDir
  )

  if ($RequestedOutputPath) {
    return [System.IO.Path]::GetFullPath($RequestedOutputPath)
  }

  if ($ResolvedAppDir) {
    return Join-Path (Join-Path $ResolvedAppDir "diagnostics") "cryptopro-modules.json"
  }

  return Join-Path (Get-Location) "cryptopro-modules.json"
}

$processes = @(Get-Process -Name $ProcessName -ErrorAction SilentlyContinue)
$resolvedAppDir = $AppDir
if (-not $resolvedAppDir) {
  foreach ($process in $processes) {
    $inferred = Infer-AppDirFromProcessPath $process.Path
    if ($inferred) {
      $resolvedAppDir = $inferred
      break
    }
  }
}

$processReports = @()
foreach ($process in $processes) {
  $processError = ""
  $allModules = @()
  $modules = @()

  try {
    foreach ($module in @($process.Modules)) {
      $moduleReport = New-ModuleReport -Module $module -ResolvedAppDir $resolvedAppDir
      $allModules += $moduleReport

      if ($moduleReport.filterMatch) {
        $modules += $moduleReport
      }
    }
  } catch {
    $processError = $_.Exception.Message
  }

  $processReports += [ordered]@{
    id = $process.Id
    name = $process.ProcessName
    path = Get-ProcessPathSafe -Process $process
    startTime = Get-ProcessStartTimeUtcString -Process $process
    moduleAccessError = $processError
    allModuleCount = $allModules.Count
    filteredModules = $modules
    modules = $allModules
  }
}

$relatedProcesses = @()
foreach ($process in @(Get-Process -ErrorAction SilentlyContinue)) {
  $path = Get-ProcessPathSafe -Process $process

  if (-not (Test-InterestingModule -Name $process.ProcessName -Path $path)) {
    continue
  }

  $relatedProcesses += [ordered]@{
    id = $process.Id
    name = $process.ProcessName
    path = $path
    startTime = Get-ProcessStartTimeUtcString -Process $process
    origin = Resolve-Origin -Path $path -ResolvedAppDir $resolvedAppDir
  }
}

$status = "ok"
if ($processReports.Count -eq 0) {
  $status = "process_not_found"
}

$report = [ordered]@{
  generatedAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
  status = $status
  computerName = $env:COMPUTERNAME
  userName = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
  processName = $ProcessName
  appDir = $resolvedAppDir
  includeAllModules = [bool]$IncludeAllModules
  filterTerms = $filterTerms
  processes = $processReports
  relatedProcesses = $relatedProcesses
}

$resolvedOutputPath = Resolve-OutputPath -RequestedOutputPath $OutputPath -ResolvedAppDir $resolvedAppDir
$outputDir = Split-Path -Parent $resolvedOutputPath
if ($outputDir -and -not (Test-Path $outputDir)) {
  New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$json = $report | ConvertTo-Json -Depth 8
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($resolvedOutputPath, $json + [Environment]::NewLine, $utf8NoBom)
Write-Host "CryptoPro module diagnostics written to $resolvedOutputPath"
