Param(
  [switch]$Release
)

# Simple helper to build cross-platform artifacts using GoReleaser.
# - Snapshot build (default): ./scripts/build_release.ps1
# - Full release (requires tag, configured publishing): ./scripts/build_release.ps1 -Release

$ErrorActionPreference = 'Stop'

# Check goreleaser availability
$goreleaser = Get-Command goreleaser -ErrorAction SilentlyContinue
if (-not $goreleaser) {
  Write-Host 'GoReleaser not found. Install from https://goreleaser.com/install/ or via scoop: scoop install goreleaser' -ForegroundColor Yellow
  exit 1
}

# Ensure we are in repo root (this script lives in scripts/) and run goreleaser from root
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location (Join-Path $scriptDir '..')

if ($Release) {
  $hasToken = $false
  if ($env:GITHUB_TOKEN -and $env:GITHUB_TOKEN.Trim() -ne '') { $hasToken = $true }
  if ($env:GITLAB_TOKEN -and $env:GITLAB_TOKEN.Trim() -ne '') { $hasToken = $true }
  if ($env:GITEA_TOKEN -and $env:GITEA_TOKEN.Trim() -ne '') { $hasToken = $true }

  if (-not $hasToken) {
    Write-Host 'No repo token found (GITHUB_TOKEN/GITLAB_TOKEN/GITEA_TOKEN). Running release with --skip=publish --skip=announce.' -ForegroundColor Yellow
    goreleaser release --clean --skip=publish --skip=announce
  } else {
    goreleaser release --clean
  }
} else {
  goreleaser build --snapshot --clean
}
