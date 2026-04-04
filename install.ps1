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

# Install PowerShell completions
$profileDir = Split-Path $PROFILE
if (-not (Test-Path $profileDir)) {
    New-Item -ItemType Directory -Force -Path $profileDir | Out-Null
}

$completionBlock = @'

# tossit completions
Register-ArgumentCompleter -Native -CommandName tossit -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $commands = @('send', 'receive', 'relay', 'update', 'completion', 'help')
    $globalOpts = @('--relay', '--relay-token', '--stream', '--password', '--dir', '--version', '--help')

    $tokens = $commandAst.ToString().Split()
    $cmd = $null
    foreach ($t in $tokens[1..($tokens.Length-1)]) {
        if ($t -in @('send', 's', 'receive', 'recv', 'r', 'relay')) {
            $cmd = $t; break
        }
    }

    $opts = switch ($cmd) {
        { $_ -in 'send', 's' }       { @('--relay', '--relay-token', '--stream', '--password', '-p') }
        { $_ -in 'receive', 'recv', 'r' } { @('--relay', '--relay-token', '--password', '-p', '--dir', '-d') }
        'relay'                       { @('--config', '--port', '--storage', '--expire', '--max-size', '--rate-limit', '--auth-token', '--allow-ips', '--ui', '--ui-password', '--admin-password', '--help') }
        default                       { $commands + $globalOpts }
    }

    $opts | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
'@

if (-not (Test-Path $PROFILE)) {
    Set-Content $PROFILE $completionBlock
    Write-Host "PowerShell completions installed to $PROFILE"
} elseif (-not (Select-String -Path $PROFILE -Pattern "tossit completions" -Quiet)) {
    Add-Content $PROFILE $completionBlock
    Write-Host "PowerShell completions installed to $PROFILE"
}

Write-Host "Installed $(& $dest --version)"
