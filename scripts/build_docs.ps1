# Builds documentation PDF(s) from Markdown sources using Pandoc.
# Output: docs\gocomicwriter_tutorials_templates.pdf
# Requirements: Windows PowerShell 5+ or PowerShell 7+, Pandoc installed (https://pandoc.org/installing.html)

$ErrorActionPreference = 'Stop'

function Invoke-Pandoc {
    param(
        [Parameter(Mandatory=$true)][string] $InputMd,
        [Parameter(Mandatory=$true)][string] $OutputPdf
    )
    $pandoc = Get-Command pandoc -ErrorAction SilentlyContinue
    if (-not $pandoc) {
        Write-Host "Pandoc is not installed or not on PATH." -ForegroundColor Yellow
        Write-Host "Install from: https://pandoc.org/installing.html" -ForegroundColor Yellow
        exit 1
    }

    $args = @(
        '--from','markdown+smart',
        '--toc',
        '--toc-depth=3',
        '--pdf-engine','xelatex',
        '-V','geometry:margin=1in',
        '-V','linkcolor:blue',
        '-s',
        $InputMd,
        '-o', $OutputPdf
    )

    Write-Host "Running: pandoc $($args -join ' ')" -ForegroundColor Cyan
    & $pandoc.Path @args
}

# Resolve paths relative to repo root
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptDir '..')
$docsDir = Join-Path $repoRoot 'docs'
$input = Join-Path $docsDir 'tutorials_and_templates.md'
$output = Join-Path $docsDir 'gocomicwriter_tutorials_templates.pdf'

if (-not (Test-Path $input)) {
    Write-Error "Missing input Markdown: $input"
    exit 1
}

# Ensure docs directory exists
New-Item -ItemType Directory -Force -Path $docsDir | Out-Null

Invoke-Pandoc -InputMd $input -OutputPdf $output

if (Test-Path $output) {
    $size = (Get-Item $output).Length
    Write-Host "PDF generated: $output ($size bytes)" -ForegroundColor Green
    exit 0
} else {
    Write-Error "PDF generation failed: $output not found"
    exit 1
}
