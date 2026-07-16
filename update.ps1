# Update the installed coralline statusline from this repository:
# pull -> test -> build (to a temp file, so a failed build never touches the
# installed binary) -> deploy -> sync themes. Any failure aborts with a
# non-zero exit code and leaves the current installation untouched.
$ErrorActionPreference = 'Stop'

$repoRoot = $PSScriptRoot
$installDir = Join-Path $env:USERPROFILE '.claude\coralline'
$installedBin = Join-Path $installDir 'coralline.exe'

if (-not (Test-Path $installDir)) {
    Write-Error "Install dir not found: $installDir — run configure.sh for first-time setup."
    exit 1
}

Set-Location $repoRoot

git pull
if ($LASTEXITCODE -ne 0) { Write-Error 'git pull failed'; exit 1 }

go test ./...
if ($LASTEXITCODE -ne 0) { Write-Error 'tests failed — not deploying'; exit 1 }

$tmpBin = Join-Path $installDir 'coralline.exe.new'
go build -o $tmpBin ./cmd/coralline
if ($LASTEXITCODE -ne 0) { Write-Error 'build failed — installed binary unchanged'; exit 1 }

Move-Item -Force $tmpBin $installedBin

# Sync themes that differ (or are new) from repo to install dir.
$themeDstDir = Join-Path $installDir 'themes'
if (-not (Test-Path $themeDstDir)) { New-Item -ItemType Directory -Force $themeDstDir | Out-Null }
foreach ($src in Get-ChildItem (Join-Path $repoRoot 'themes') -Filter '*.conf') {
    $dst = Join-Path $themeDstDir $src.Name
    if (-not (Test-Path $dst) -or (Get-FileHash $src.FullName).Hash -ne (Get-FileHash $dst).Hash) {
        Copy-Item $src.FullName $dst
        Write-Host "theme updated: $($src.Name)"
    }
}

Write-Host 'coralline updated.'
