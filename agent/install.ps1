# KomariX Agent installer wrapper
# Keeps the upstream Komari Agent installation logic intact and changes only user-visible branding.

$ErrorActionPreference = "Stop"
$UpstreamUrl = "https://raw.githubusercontent.com/komari-monitor/komari-agent/refs/heads/main/install.ps1"
$TempScript = Join-Path $env:TEMP ("komarix-agent-install-" + [Guid]::NewGuid().ToString("N") + ".ps1")

try {
    $Content = (Invoke-WebRequest -Uri $UpstreamUrl -UseBasicParsing).Content
    $Content = $Content.Replace("Komari Agent", "KomariX Agent")
    $Content = $Content.Replace("Komari-agent", "KomariX Agent")
    $Content = $Content.Replace("KOMARI Agent", "KomariX Agent")
    Set-Content -Path $TempScript -Value $Content -Encoding UTF8
    & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $TempScript @args
    exit $LASTEXITCODE
}
finally {
    Remove-Item -Path $TempScript -Force -ErrorAction SilentlyContinue
}
