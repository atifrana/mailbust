# MailBust installer for Windows (PowerShell).
#   irm https://raw.githubusercontent.com/atifrana/mailbust/main/docs/install.ps1 | iex
#
# Env overrides:
#   $env:MAILBUST_VERSION       version tag to install (default: latest)
#   $env:MAILBUST_INSTALL_DIR   install location (default: %LOCALAPPDATA%\Programs\mailbust)
$ErrorActionPreference = "Stop"

$repo = "atifrana/mailbust"

$arch = if ([Environment]::Is64BitOperatingSystem) {
  if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else { throw "mailbust: 32-bit Windows is not supported" }

$version = if ($env:MAILBUST_VERSION) { $env:MAILBUST_VERSION } else { "latest" }
$base = if ($version -eq "latest") {
  "https://github.com/$repo/releases/latest/download"
} else {
  "https://github.com/$repo/releases/download/$version"
}
$url = "$base/mailbust_windows_$arch.zip"

$dir = if ($env:MAILBUST_INSTALL_DIR) { $env:MAILBUST_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\mailbust" }
New-Item -ItemType Directory -Force -Path $dir | Out-Null

$tmp = Join-Path $env:TEMP "mailbust.zip"
Write-Host "Downloading $url"
Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
Expand-Archive -Path $tmp -DestinationPath $dir -Force
Remove-Item $tmp

Write-Host "Installed mailbust to $dir\mailbust.exe"

# Add to the user PATH if missing.
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$dir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$dir", "User")
  Write-Host "Added $dir to your PATH. Open a new terminal to use 'mailbust'."
}
& "$dir\mailbust.exe" version
