# Windows PowerShell installation script for KomariX Agent

# Logging functions with colors
function Log-Info { param([string]$Message) Write-Host "$Message"    -ForegroundColor Cyan }
function Log-Success { param([string]$Message) Write-Host "$Message"    -ForegroundColor Green }
function Log-Warning { param([string]$Message) Write-Host "[WARNING] $Message"    -ForegroundColor Yellow }
function Log-Error { param([string]$Message) Write-Host "[ERROR] $Message"    -ForegroundColor Red }
function Log-Step { param([string]$Message) Write-Host "$Message"    -ForegroundColor Magenta }
function Log-Config { param([string]$Message) Write-Host "- $Message"    -ForegroundColor White }

# Default parameters
$InstallDir = Join-Path $Env:ProgramFiles "KomariX Agent"
$ServiceName = "komarix-agent"
$GitHubProxy = ""
$KomariXArgs = @()
$InstallVersion = ""

# Parse script arguments
for ($i = 0; $i -lt $args.Count; $i++) {
    switch ($args[$i]) {
        "--install-dir" { $InstallDir = $args[$i + 1]; $i++; continue }
        "--install-service-name" { $ServiceName = $args[$i + 1]; $i++; continue }
        "--install-ghproxy" { $GitHubProxy = $args[$i + 1]; $i++; continue }
        "--install-version" { $InstallVersion = $args[$i + 1]; $i++; continue }
        Default { $KomariXArgs += $args[$i] }
    }
}

# Ensure running as Administrator
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()
    ).IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator)) {
    Log-Error "Please run this script as Administrator."
    exit 1
}

# Prepare GitHub proxy display
if ($GitHubProxy -ne '') { $ProxyDisplay = $GitHubProxy } else { $ProxyDisplay = '(direct)' }

# Detect architecture early for constructing binary name
switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    'x86' { $arch = '386' }
    Default { Log-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; exit 1 }
}

# Ensure installation directory exists for nssm and agent
Log-Step "Ensuring installation directory exists: $InstallDir"
New-Item -ItemType Directory -Path $InstallDir -Force -ErrorAction SilentlyContinue | Out-Null # Ensure $InstallDir exists

# Check for nssm and download if not present
$nssmExeToUse = Join-Path $InstallDir "nssm.exe"

# First, check if nssm is in PATH and is functional
$nssmCmd = Get-Command nssm -ErrorAction SilentlyContinue
if ($nssmCmd) {
    Log-Info "nssm found in PATH at $($nssmCmd.Source)."
    try {
        $nssmVersionOutput = nssm version 2>&1
        Log-Info "Detected nssm version: $nssmVersionOutput"
    }
    catch {
        Log-Warning "nssm found in PATH failed to execute 'nssm version'. Will attempt to use/download local copy. Error: $_"
        $nssmCmd = $null # Force re-evaluation for local copy or download
    }
}

# If nssm not found in PATH or the one in PATH failed, check local $InstallDir
if (-not $nssmCmd) {
    if (Test-Path $nssmExeToUse) {
        Log-Info "nssm found at $nssmExeToUse. Attempting to use it by adding $InstallDir to PATH."
        $env:Path = "$($InstallDir);$($env:Path)"
        $nssmCmd = Get-Command nssm -ErrorAction SilentlyContinue
        if ($nssmCmd) {
            try {
                $nssmVersionOutput = nssm version 2>&1
            }
            catch {
                Log-Warning "nssm from $InstallDir failed to execute 'nssm version'. Error: $_"
                $nssmCmd = $null # Mark as unusable
            }
        }
        else {
            Log-Warning "Failed to make nssm from $nssmExeToUse available via PATH. Will attempt download."
        }
    }
}

# If still no usable nssm command, download a pinned, checksum-verified build.
if (-not $nssmCmd) {
    Log-Info "nssm not found or not usable. Attempting to download a verified build to $InstallDir..."
    $NssmVersion = "2.24-101-g897c7ad"
    $NssmZipUrl = "https://nssm.cc/ci/nssm-$NssmVersion.zip"
    $NssmZipSha256 = "99F5045FFFBFFB745D67FE3A065A953C4A3D9C253B868892D9B685B0EE7D07B8"
    $TempNssmZipPath = Join-Path $env:TEMP ("nssm-" + [IO.Path]::GetRandomFileName() + ".zip")
    $TempExtractDir = Join-Path $env:TEMP ("nssm-extract-" + [IO.Path]::GetRandomFileName())

    try {
        Log-Info "Downloading nssm from $NssmZipUrl..."
        Invoke-WebRequest -Uri $NssmZipUrl -OutFile $TempNssmZipPath -UseBasicParsing -TimeoutSec 120
        $ActualNssmZipSha256 = (Get-FileHash -Algorithm SHA256 -Path $TempNssmZipPath).Hash.ToUpperInvariant()
        if ($ActualNssmZipSha256 -ne $NssmZipSha256) {
            throw "nssm archive SHA256 verification failed. Expected $NssmZipSha256, got $ActualNssmZipSha256."
        }
        Log-Success "nssm archive SHA256 verified."

        New-Item -ItemType Directory -Path $TempExtractDir -Force | Out-Null
        Expand-Archive -Path $TempNssmZipPath -DestinationPath $TempExtractDir -Force

        $NssmPlatformDir = if ($arch -eq "amd64") { "win64" } else { "win32" }
        $NssmSourceExePath = Join-Path (Join-Path (Join-Path $TempExtractDir "nssm-$NssmVersion") $NssmPlatformDir) "nssm.exe"
        if (-not (Test-Path $NssmSourceExePath)) {
            throw "Could not find verified nssm.exe at expected path: $NssmSourceExePath"
        }

        Copy-Item -Path $NssmSourceExePath -Destination $nssmExeToUse -Force
        $env:Path = "$($InstallDir);$($env:Path)"
        $nssmCmd = Get-Command nssm -ErrorAction SilentlyContinue
        if (-not $nssmCmd) {
            throw "Downloaded nssm could not be added to PATH."
        }
        $nssmVersionOutput = nssm version 2>&1
        Log-Success "Verified nssm $nssmVersionOutput is configured."
    }
    catch {
        Log-Error "Failed to download or configure verified nssm: $_"
        Log-Error "Please install a trusted nssm build manually and ensure nssm.exe is in PATH."
        exit 1
    }
    finally {
        if (Test-Path $TempNssmZipPath) { Remove-Item $TempNssmZipPath -Force -ErrorAction SilentlyContinue }
        if (Test-Path $TempExtractDir) { Remove-Item $TempExtractDir -Recurse -Force -ErrorAction SilentlyContinue }
    }
}

# Final check that nssm is operational
try {
    $nssmVersionOutput = nssm version 2>&1
}
catch {
    Log-Error "nssm command failed to execute even after setup attempts. Please check the nssm installation and PATH. Error: $_"
    exit 1
}

Log-Step "Installation configuration:"
Log-Config "Service name: $ServiceName"
Log-Config "Install directory: $InstallDir"
Log-Config "GitHub proxy: $ProxyDisplay"
Log-Config "Agent arguments: $($KomariXArgs -join ' ')"
if ($InstallVersion -ne "") {
    Log-Config "Specified agent version: $InstallVersion"
} else {
    Log-Config "Agent version: Latest"
}

# Paths
$BinaryName = "komarix-agent-windows-$arch.exe"
$AgentPath = Join-Path $InstallDir "komarix-agent.exe"

# Uninstall previous service and binary
function Uninstall-Previous {
    Log-Step "Checking for existing service..."
    # Check if service exists using nssm status, as Get-Service might not work for nssm services if not properly registered
    $serviceStatus = nssm status $ServiceName 2>&1
    if ($serviceStatus -notmatch "SERVICE_STOPPED" -and $serviceStatus -notmatch "does not exist") {
        Log-Info "Stopping service $ServiceName..."
        nssm stop $ServiceName 2>&1 | Out-Null
    }
    # Attempt to remove the service using nssm
    # We check if it exists first by trying to get its status.
    # nssm remove will succeed if the service exists, and fail otherwise.
    # We add confirm to avoid interactive prompts.
    $removeOutput = nssm remove $ServiceName confirm 2>&1
    if ($LASTEXITCODE -eq 0) {
    }
    elseif ($removeOutput -match "Can't open service! (The specified service does not exist as an installed service.)" -or $removeOutput -match "No such service" -or $removeOutput -match "does not exist") {
        Log-Info "Service $ServiceName does not exist or was already removed."
    }
    else {
        # If nssm remove fails for other reasons, try sc.exe delete as a fallback for older installations
        $svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($svc) {
            Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue
            sc.exe delete $ServiceName | Out-Null
        }
    }

    if (Test-Path $AgentPath) {
        Log-Warning "Removing old binary..."
        Remove-Item $AgentPath -Force
    }
}
function Get-LatestSnapshotVersion {
    param([Parameter(Mandatory = $true)][string]$AssetName)

    $ApiUrl = "https://api.github.com/repos/kkx999/KomariX/releases?per_page=100"
    $ApiUrls = @($ApiUrl)
    if ($GitHubProxy -ne "") {
        $ApiUrls = @("$GitHubProxy/$ApiUrl", $ApiUrl)
    }

    for ($i = 0; $i -lt $ApiUrls.Count; $i++) {
        try {
            Log-Info "Fetching snapshot releases from GitHub API..."
            $releases = Invoke-RestMethod -Uri $ApiUrls[$i] -UseBasicParsing
        }
        catch {
            $releases = $null
        }

        if ($releases) {
            $latestSnapshot = $releases |
            Where-Object {
                $_.draft -eq $false -and
                $_.prerelease -eq $true -and
                $_.tag_name -like "Snapshot-*" -and
                (@($_.assets.name) -contains $AssetName)
            } |
            Sort-Object -Property @{ Expression = { [datetime]$_.published_at }; Descending = $true }, @{ Expression = { $_.tag_name }; Descending = $true } |
            Select-Object -First 1

            if ($latestSnapshot) {
                return $latestSnapshot.tag_name
            }
        }

        if ($i -lt ($ApiUrls.Count - 1)) {
            Log-Warning "Failed to resolve snapshot releases through GitHub proxy, retrying directly."
        }
    }

    throw "No snapshot release contains asset $AssetName."
}

$versionToInstall = ""
if ($InstallVersion -ne "") {
    Log-Info "Attempting to install specified version: $InstallVersion"
    if ($InstallVersion -ieq "snapshot") {
        Log-Info "Resolving the latest snapshot version..."
        try {
            $versionToInstall = Get-LatestSnapshotVersion -AssetName $BinaryName
            Log-Success "Latest snapshot version fetched: $versionToInstall"
        }
        catch {
            Log-Error "Failed to resolve the latest snapshot version: $_"
            exit 1
        }
    }
    else {
        $versionToInstall = $InstallVersion
    }
}
else {
    $ApiUrl = "https://api.github.com/repos/kkx999/KomariX/releases/latest"
    try {
        Log-Step "Fetching latest stable release version from GitHub API..."
        $release = Invoke-RestMethod -Uri $ApiUrl -Headers @{ Accept = "application/vnd.github+json"; "User-Agent" = "komarix-agent-installer" } -UseBasicParsing -TimeoutSec 60
        $versionToInstall = [string]$release.tag_name
        Log-Success "Latest stable version fetched: $versionToInstall"
    }
    catch {
        Log-Error "Failed to fetch latest version: $_"
        exit 1
    }
}
if ($versionToInstall -notmatch '^(v[0-9][0-9A-Za-z._-]*|Snapshot-[0-9A-Za-z._-]+)$') {
    Log-Error "Invalid KomariX release tag: $versionToInstall"
    exit 1
}
Log-Success "Installing KomariX Agent version: $versionToInstall"

$BinaryName = "komarix-agent-windows-$arch.exe"
$DirectDownloadUrl = "https://github.com/kkx999/KomariX/releases/download/$versionToInstall/$BinaryName"
$DownloadUrl = if ($GitHubProxy) { "$GitHubProxy/$DirectDownloadUrl" } else { $DirectDownloadUrl }
$DirectChecksumsUrl = "https://github.com/kkx999/KomariX/releases/download/$versionToInstall/SHA256SUMS"
$ChecksumUrls = @($DirectChecksumsUrl)
if ($GitHubProxy) { $ChecksumUrls += "$GitHubProxy/$DirectChecksumsUrl" }

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
$CandidatePath = Join-Path $InstallDir (".agent.download." + [IO.Path]::GetRandomFileName() + ".exe")
$ChecksumPath = Join-Path $InstallDir (".agent.checksums." + [IO.Path]::GetRandomFileName())
$BackupAgentPath = $null
$argString = $KomariXArgs -join ' '

try {
    $checksumDownloaded = $false
    foreach ($url in $ChecksumUrls) {
        try {
            Log-Info "Downloading release checksums from $url"
            Invoke-WebRequest -Uri $url -OutFile $ChecksumPath -UseBasicParsing -TimeoutSec 90
            $checksumDownloaded = $true
            break
        }
        catch {
            Log-Warning "Checksum download failed from $url"
        }
    }
    if (-not $checksumDownloaded) {
        throw "Failed to download SHA256SUMS. Existing Agent was not changed."
    }

    $ExpectedHash = $null
    foreach ($line in Get-Content -Path $ChecksumPath) {
        $parts = $line.Trim() -split '\s+', 2
        if ($parts.Count -eq 2 -and $parts[1].TrimStart('*') -eq $BinaryName) {
            $ExpectedHash = $parts[0].ToUpperInvariant()
            break
        }
    }
    if (-not $ExpectedHash -or $ExpectedHash -notmatch '^[0-9A-F]{64}$') {
        throw "SHA256SUMS does not contain a valid checksum for $BinaryName."
    }

    Log-Info "URL: $DownloadUrl"
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $CandidatePath -UseBasicParsing -TimeoutSec 300
    if (-not (Test-Path $CandidatePath) -or (Get-Item $CandidatePath).Length -le 0) {
        throw "Downloaded Agent binary is empty."
    }
    $ActualHash = (Get-FileHash -Algorithm SHA256 -Path $CandidatePath).Hash.ToUpperInvariant()
    if ($ActualHash -ne $ExpectedHash) {
        throw "SHA256 verification failed for $BinaryName."
    }
    Log-Success "SHA256 verified for $BinaryName"

    if (Test-Path $AgentPath) {
        $BackupAgentPath = Join-Path $InstallDir (".agent.previous." + $PID + ".exe")
        Copy-Item -Path $AgentPath -Destination $BackupAgentPath -Force
    }

    # The existing service/binary is touched only after the candidate is verified.
    Uninstall-Previous
    Move-Item -Path $CandidatePath -Destination $AgentPath -Force

    Log-Step "Configuring Windows service with nssm..."
    & nssm install $ServiceName $AgentPath $argString | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "nssm install failed with exit code $LASTEXITCODE" }
    & nssm set $ServiceName DisplayName "KomariX Agent Service" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "failed to set service display name" }
    & nssm set $ServiceName Start SERVICE_AUTO_START | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "failed to set service startup mode" }
    & nssm set $ServiceName AppExit Default Restart | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "failed to set service restart policy" }
    & nssm set $ServiceName AppRestartDelay 5000 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "failed to set service restart delay" }
    & nssm start $ServiceName | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "failed to start service" }

    Start-Sleep -Seconds 1
    $status = (& nssm status $ServiceName 2>&1 | Out-String).Trim()
    if ($status -notmatch "SERVICE_RUNNING") {
        throw "service status is $status"
    }

    if ($BackupAgentPath -and (Test-Path $BackupAgentPath)) {
        Remove-Item $BackupAgentPath -Force
        $BackupAgentPath = $null
    }
    Log-Success "Service $ServiceName installed and started using nssm."
}
catch {
    $failure = $_
    Log-Error "Agent installation or upgrade failed: $failure"
    if ($BackupAgentPath -and (Test-Path $BackupAgentPath)) {
        Log-Warning "Restoring the previous Agent binary..."
        & nssm stop $ServiceName 2>&1 | Out-Null
        & nssm remove $ServiceName confirm 2>&1 | Out-Null
        Copy-Item -Path $BackupAgentPath -Destination $AgentPath -Force

        & nssm install $ServiceName $AgentPath $argString | Out-Null
        & nssm set $ServiceName DisplayName "KomariX Agent Service" | Out-Null
        & nssm set $ServiceName Start SERVICE_AUTO_START | Out-Null
        & nssm set $ServiceName AppExit Default Restart | Out-Null
        & nssm set $ServiceName AppRestartDelay 5000 | Out-Null
        & nssm start $ServiceName | Out-Null
        Start-Sleep -Seconds 1

        $rollbackStatus = (& nssm status $ServiceName 2>&1 | Out-String).Trim()
        if ($rollbackStatus -match "SERVICE_RUNNING") {
            Log-Warning "Previous Agent binary restored and restarted."
        }
        else {
            Log-Error "Rollback restored the binary but the service is not running: $rollbackStatus"
        }
    }
    exit 1
}
finally {
    if (Test-Path $CandidatePath) { Remove-Item $CandidatePath -Force -ErrorAction SilentlyContinue }
    if (Test-Path $ChecksumPath) { Remove-Item $ChecksumPath -Force -ErrorAction SilentlyContinue }
}

Log-Success "KomariX Agent installation completed!"
Log-Config "Service name: $ServiceName"
Log-Config "Arguments: $argString"
