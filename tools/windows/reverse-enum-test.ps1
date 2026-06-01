<#
.SYNOPSIS
  Windows mirror of the Linux enumeration experiment. Shows WHERE provider
  enumeration comes from on Windows: CryptoPro's own config-based view vs the
  Microsoft registry-based view that the browser plugin's About.CSPName uses.

.DESCRIPTION
  Run on a machine where the CryptoPro plugin is installed with ADDMINICSP=1
  (so Mini CSP exists). No compiler needed. Writes results under
  docs/minicsp-snapshots/windows-reverse-enum/ :

    cpconfig-view-type.txt        Mini CSP cpconfig.exe -defprov -view_type (run FROM Mini CSP)
    cpconfig-view-type-elsewhere.txt  same, run from a different cwd (config-path probe)
    cpconfig-view-prov80.txt      Mini CSP cpconfig.exe -defprov -view -provtype 80
    certutil-csplist.txt          Microsoft registry view of installed CSPs
    advapi32-enum.txt             advapi32 CryptEnumProviderTypes/CryptGetDefaultProvider via P/Invoke
    summary.txt                   side-by-side verdict

  Interpretation:
    - If cpconfig (config-based) lists GOST 75/80/81 but certutil/advapi32
      (registry-based) do NOT, the split is confirmed: Mini CSP providers live in
      config.ini and are invisible to the OS registry-based enumeration the plugin
      uses -> About.CSPName(80)=0x80090017 is expected, not a config bug.

  Read-only. Does not install/modify anything.
#>
param(
  [string]$RepoRoot
)
$ErrorActionPreference = "Stop"
if (-not $RepoRoot) { $RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot) }
$out = Join-Path $RepoRoot "docs/minicsp-snapshots/windows-reverse-enum"
New-Item -ItemType Directory -Force -Path $out | Out-Null

$mini = Join-Path ${env:ProgramFiles(x86)} "Crypto Pro\CAdES Browser Plug-in\Mini CSP"
$cpconfig = Join-Path $mini "cpconfig.exe"
$summary = New-Object System.Collections.Generic.List[string]
$summary.Add("Reverse enumeration test - " + (Get-Date).ToUniversalTime().ToString("o"))
$summary.Add("Mini CSP: $mini  (exists: $(Test-Path $mini))")
$summary.Add("")

# --- 1. CryptoPro config-based enumeration via the bundled cpconfig.exe ---
if (Test-Path $cpconfig) {
  Push-Location $mini
  try {
    & $cpconfig -defprov -view_type *>&1 | Tee-Object -FilePath (Join-Path $out "cpconfig-view-type.txt") | Out-Null
    & $cpconfig -defprov -view -provtype 80 *>&1 | Tee-Object -FilePath (Join-Path $out "cpconfig-view-prov80.txt") | Out-Null
  } finally { Pop-Location }
  # config-path probe: run from a neutral cwd with full path
  Push-Location $env:SystemRoot
  try {
    & $cpconfig -defprov -view_type *>&1 | Tee-Object -FilePath (Join-Path $out "cpconfig-view-type-elsewhere.txt") | Out-Null
  } finally { Pop-Location }
  $vt = Get-Content (Join-Path $out "cpconfig-view-type.txt") -Raw
  $summary.Add("cpconfig -defprov -view_type (from Mini CSP): " + (if ($vt -match '\b80\b') {"lists type 80 (config-based: PROVIDERS VISIBLE)"} else {"did NOT list type 80"}))
} else {
  $summary.Add("cpconfig.exe NOT FOUND - is the plugin installed with ADDMINICSP=1?")
}

# --- 2. Microsoft registry view ---
& certutil -csplist *>&1 | Tee-Object -FilePath (Join-Path $out "certutil-csplist.txt") | Out-Null
$csp = Get-Content (Join-Path $out "certutil-csplist.txt") -Raw
$summary.Add("certutil -csplist (registry): " + (if ($csp -match 'GOST R 34.10-2012') {"shows a GOST 2012 provider"} else {"NO GOST 2012 provider (registry view empty of Mini CSP)"}))

# --- 3. advapi32 (the OS base CryptoAPI the plugin's About.CSPName uses) ---
$cs = @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class CapiProbe {
  [DllImport("advapi32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
  static extern bool CryptEnumProviderTypes(uint dwIndex, IntPtr pdwReserved, uint dwFlags, ref uint pdwProvType, StringBuilder pszTypeName, ref uint pcbTypeName);
  [DllImport("advapi32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
  static extern bool CryptGetDefaultProvider(uint dwProvType, IntPtr pdwReserved, uint dwFlags, StringBuilder pszProvName, ref uint pcbProvName);
  public static string Run() {
    var sb = new StringBuilder();
    sb.AppendLine("== advapi32 CryptEnumProviderTypes (Microsoft, registry-based) ==");
    for (uint i=0;;i++){
      uint type=0; var name=new StringBuilder(512); uint cb=512;
      if(!CryptEnumProviderTypes(i,IntPtr.Zero,0,ref type,name,ref cb)){
        sb.AppendLine(String.Format("  stop at index {0}, GetLastError=0x{1:X8}", i, Marshal.GetLastWin32Error()));
        break;
      }
      sb.AppendLine(String.Format("  type={0}  name={1}", type, name));
    }
    sb.AppendLine("== advapi32 CryptGetDefaultProvider(80) ==");
    var n=new StringBuilder(512); uint c=512;
    if(CryptGetDefaultProvider(80,IntPtr.Zero,1,n,ref c)) sb.AppendLine("  type80 = "+n);
    else sb.AppendLine(String.Format("  type80 FAILED GetLastError=0x{0:X8}", Marshal.GetLastWin32Error()));
    return sb.ToString();
  }
}
'@
Add-Type -TypeDefinition $cs -Language CSharp
$advOut = [CapiProbe]::Run()
$advOut | Set-Content -Path (Join-Path $out "advapi32-enum.txt") -Encoding UTF8
$summary.Add("advapi32 CryptGetDefaultProvider(80): " + (if ($advOut -match '0x80090017') {"0x80090017 (NTE_PROV_TYPE_NOT_DEF) - exactly the plugin's symptom"} else {"see advapi32-enum.txt"}))

$summary.Add("")
$summary.Add("VERDICT: if cpconfig lists 80 but certutil/advapi32 do not, the Mini CSP")
$summary.Add("providers live ONLY in config.ini and are invisible to the registry-based")
$summary.Add("enumeration the plugin uses -> About.CSPName(80)=0x80090017 is by design.")
$summary | Set-Content -Path (Join-Path $out "summary.txt") -Encoding UTF8
Write-Host "Done. Review: $out\summary.txt"
