param (
    [string]$Version
)

if (-not $Version) {
    Write-Host "==========================================" -ForegroundColor Yellow
    Write-Host "HELP: Build script requires a version tag." -ForegroundColor Yellow
    Write-Host "Usage: .\scripts\build.ps1 -Version v1.2.3" -ForegroundColor Yellow
    Write-Host "This will inject the version into the app and build it." -ForegroundColor Yellow
    Write-Host "==========================================" -ForegroundColor Yellow
    exit 1
}

Write-Host "Building NetRasad version $Version ..." -ForegroundColor Cyan

# Use wails to build with ldflags injecting the version
wails build -clean -platform windows/amd64 -ldflags "-X main.Version=$Version"

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build complete for version $Version!" -ForegroundColor Green
} else {
    Write-Host "Build failed." -ForegroundColor Red
}
