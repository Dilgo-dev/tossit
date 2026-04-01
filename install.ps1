$repo = "Dilgo-dev/tossit"
$binary = "tossit"

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "arm64" }
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $arch = "arm64" }

$latest = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
if (-not $latest) {
    Write-Host "Failed to get latest version"
    exit 1
}

$url = "https://github.com/$repo/releases/download/$latest/$binary-windows-$arch.exe"
Write-Host "Downloading tossit $latest for windows/$arch..."

$installDir = "$env:LOCALAPPDATA\tossit"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

$dest = "$installDir\$binary.exe"
Invoke-WebRequest -Uri $url -OutFile $dest

$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$installDir;$currentPath", "User")
    Write-Host "Added $installDir to PATH (restart terminal to apply)"
}

Write-Host "Installed $(& $dest --version)"
