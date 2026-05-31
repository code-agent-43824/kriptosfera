<#
.SYNOPSIS
  Snapshot CryptoPro-relevant Windows state (registry + Program Files layout) for
  the Mini CSP investigation. Run the SAME script in each phase so snapshots are
  directly diffable.

.DESCRIPTION
  Writes everything under docs/minicsp-snapshots/<Phase>/ :
    registry/*.reg        raw `reg export` of each relevant hive/key (UTF-16, as Windows writes)
    registry/*.txt        `reg query /s` text dumps (easier to diff/grep, may be empty if key absent)
    files/pf-x86-cryptopro.txt    recursive listing of Program Files (x86)\Crypto Pro with size+sha256
    files/minicsp-config.ini      copy of Mini CSP\config.ini if present
    files/minicsp-license.ini     copy of Mini CSP\license.ini if present
    summary.txt           which keys/paths existed, counts, quick verdict notes

  Nothing here downloads or installs anything. Read-only except for writing the
  snapshot output. Safe to run repeatedly.

.PARAMETER Phase
  Snapshot label, e.g. clean, installed-noflags, installed-addminicsp.

.PARAMETER RepoRoot
  Path to the cloned kriptosfera repo (defaults to two levels up from this script).

.EXAMPLE
  ./tools/windows/snapshot-cryptopro-state.ps1 -Phase clean
#>
param(
  [Parameter(Mandatory = $true)]
  [string]$Phase,
  [string]$RepoRoot
)

$ErrorActionPreference = "Stop"

if (-not $RepoRoot) {
  $RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
}
$outRoot = Join-Path $RepoRoot "docs/minicsp-snapshots/$Phase"
$regDir  = Join-Path $outRoot "registry"
$fileDir = Join-Path $outRoot "files"
New-Item -ItemType Directory -Force -Path $regDir, $fileDir | Out-Null

Write-Host "Snapshot phase '$Phase' -> $outRoot"

# --- Registry keys that matter for CryptoPro provider enumeration / install ---
# Both native (64-bit) and WOW6432Node (32-bit) views, HKLM and HKCU, plus the
# standard CryptoAPI provider registration and the Chrome native messaging host.
$regKeys = [ordered]@{
  "hklm-cryptopro"            = "HKLM\SOFTWARE\Crypto Pro"
  "hklm-wow64-cryptopro"      = "HKLM\SOFTWARE\WOW6432Node\Crypto Pro"
  "hkcu-cryptopro"            = "HKCU\SOFTWARE\Crypto Pro"
  "hkcu-wow64-cryptopro"      = "HKCU\SOFTWARE\WOW6432Node\Crypto Pro"
  "hklm-capi-defaults"        = "HKLM\SOFTWARE\Microsoft\Cryptography\Defaults\Provider"
  "hklm-capi-defaults-types"  = "HKLM\SOFTWARE\Microsoft\Cryptography\Defaults\Provider Types"
  "hklm-wow64-capi-defaults"  = "HKLM\SOFTWARE\WOW6432Node\Microsoft\Cryptography\Defaults\Provider"
  "hklm-wow64-capi-types"     = "HKLM\SOFTWARE\WOW6432Node\Microsoft\Cryptography\Defaults\Provider Types"
  "hkcu-nmhost"               = "HKCU\Software\Google\Chrome\NativeMessagingHosts\ru.cryptopro.nmcades"
  "hklm-nmhost"               = "HKLM\Software\Google\Chrome\NativeMessagingHosts\ru.cryptopro.nmcades"
}

$summary = New-Object System.Collections.Generic.List[string]
$summary.Add("Phase: $Phase")
$summary.Add("Timestamp (UTC): " + (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"))
$summary.Add("Machine: $env:COMPUTERNAME")
$summary.Add("")
$summary.Add("== Registry keys ==")

foreach ($name in $regKeys.Keys) {
  $key = $regKeys[$name]
  $regOut = Join-Path $regDir "$name.reg"
  $txtOut = Join-Path $regDir "$name.txt"
  # reg export (binary-accurate). Capture presence.
  & reg.exe export $key $regOut /y *> $null
  $exported = Test-Path $regOut
  # reg query /s (text, easy to diff). May fail if key missing.
  & reg.exe query $key /s *> $txtOut
  $present = ($LASTEXITCODE -eq 0)
  $summary.Add(("{0,-26} {1,-8} {2}" -f $name, ($(if ($present) {"EXISTS"} else {"absent"})), $key))
  if (-not $present -and (Test-Path $txtOut)) {
    Remove-Item $txtOut -ErrorAction SilentlyContinue
  }
}

# --- Program Files (x86)\Crypto Pro layout with sizes + hashes ---
$summary.Add("")
$summary.Add("== Program Files layout ==")
$pfRoots = @(
  (Join-Path ${env:ProgramFiles(x86)} "Crypto Pro"),
  (Join-Path $env:ProgramFiles "Crypto Pro")
)
$pfListing = New-Object System.Collections.Generic.List[string]
foreach ($pf in $pfRoots) {
  $pfListing.Add("### ROOT: $pf")
  if (Test-Path $pf) {
    $summary.Add("EXISTS  $pf")
    Get-ChildItem -Path $pf -Recurse -File -ErrorAction SilentlyContinue | ForEach-Object {
      $rel = $_.FullName.Substring($pf.Length).TrimStart('\')
      $hash = ""
      try { $hash = (Get-FileHash -Algorithm SHA256 -Path $_.FullName -ErrorAction Stop).Hash.ToLower() } catch { $hash = "(hash failed)" }
      $pfListing.Add(("{0,12}  {1,-64}  {2}" -f $_.Length, $rel, $hash))
    }
  } else {
    $summary.Add("absent  $pf")
    $pfListing.Add("(absent)")
  }
  $pfListing.Add("")
}
$pfListing | Set-Content -Path (Join-Path $fileDir "pf-x86-cryptopro.txt") -Encoding UTF8

# --- Copy Mini CSP config.ini / license.ini verbatim if present ---
foreach ($pf in $pfRoots) {
  $mini = Join-Path $pf "CAdES Browser Plug-in\Mini CSP"
  if (Test-Path (Join-Path $mini "config.ini")) {
    Copy-Item (Join-Path $mini "config.ini") (Join-Path $fileDir "minicsp-config.ini") -Force
    $summary.Add("Copied: $mini\config.ini")
  }
  if (Test-Path (Join-Path $mini "license.ini")) {
    Copy-Item (Join-Path $mini "license.ini") (Join-Path $fileDir "minicsp-license.ini") -Force
    $summary.Add("Copied: $mini\license.ini")
  }
}

# --- Bitness of the key binaries, if present ---
$summary.Add("")
$summary.Add("== Binary bitness ==")
function Get-PEBitness([string]$path) {
  if (-not (Test-Path $path)) { return "absent" }
  try {
    $fs = [System.IO.File]::OpenRead($path)
    $br = New-Object System.IO.BinaryReader($fs)
    $fs.Position = 0x3C
    $peOff = $br.ReadInt32()
    $fs.Position = $peOff + 4
    $machine = $br.ReadUInt16()
    $br.Close(); $fs.Close()
    switch ($machine) { 0x14c {"x86(32)"} 0x8664 {"x64(64)"} default {"0x{0:x}" -f $machine} }
  } catch { "(read error)" }
}
foreach ($pf in $pfRoots) {
  $plug = Join-Path $pf "CAdES Browser Plug-in"
  foreach ($b in @("nmcades.exe","npcades.dll","cplib.dll","Mini CSP\capi20.dll","Mini CSP\cpcspi.dll")) {
    $p = Join-Path $plug $b
    if (Test-Path $p) { $summary.Add(("{0,-22} {1}" -f (Get-PEBitness $p), "$plug\$b")) }
  }
}

$summary | Set-Content -Path (Join-Path $outRoot "summary.txt") -Encoding UTF8

Write-Host "Done. Review: $outRoot\summary.txt"
Write-Host "Registry dumps: $regDir"
Write-Host "File listing:   $fileDir\pf-x86-cryptopro.txt"
