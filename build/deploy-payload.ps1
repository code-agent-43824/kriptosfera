param(
  [string]$PayloadZip = "dist/payload.zip",
  [string]$PayloadMetadata = "dist/payload.json",
  [string]$RemoteHost = "",
  [string]$RemoteUser = "",
  [string]$RemoteRoot = "",
  [string]$SshPrivateKey = ""
)

$ErrorActionPreference = "Stop"

if (-not $RemoteHost -or -not $RemoteUser -or -not $RemoteRoot -or -not $SshPrivateKey) {
  Write-Host "Skipping payload deploy: remote publish configuration is incomplete"
  exit 0
}

if (-not (Test-Path $PayloadZip)) {
  throw "Payload zip not found: $PayloadZip"
}
if (-not (Test-Path $PayloadMetadata)) {
  throw "Payload metadata not found: $PayloadMetadata"
}

$metadata = Get-Content $PayloadMetadata -Raw | ConvertFrom-Json
$remoteDir = "$($RemoteRoot.TrimEnd('/'))/$($metadata.payloadVersion)/$($metadata.sha256)"

$keyFile = Join-Path $env:RUNNER_TEMP "kriptosfera-payload-deploy.key"
$normalizedKey = $SshPrivateKey -replace "`r", ""
Set-Content -Path $keyFile -Value $normalizedKey -Encoding ascii -NoNewline

try {
  ssh -i $keyFile -o StrictHostKeyChecking=accept-new "$RemoteUser@$RemoteHost" "mkdir -p '$remoteDir'"
  if ($LASTEXITCODE -ne 0) {
    throw "Failed to create remote payload directory: $remoteDir"
  }

  scp -i $keyFile -o StrictHostKeyChecking=accept-new $PayloadZip "$RemoteUser@${RemoteHost}:$remoteDir/payload.zip"
  if ($LASTEXITCODE -ne 0) {
    throw "Failed to upload payload.zip"
  }

  scp -i $keyFile -o StrictHostKeyChecking=accept-new $PayloadMetadata "$RemoteUser@${RemoteHost}:$remoteDir/payload.json"
  if ($LASTEXITCODE -ne 0) {
    throw "Failed to upload payload.json"
  }

  Write-Host "Payload deployed to ${RemoteHost}:$remoteDir"
} finally {
  if (Test-Path $keyFile) {
    Remove-Item -Force $keyFile
  }
}
