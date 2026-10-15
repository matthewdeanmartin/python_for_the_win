# Install pftw from PyPI on Windows, then hand over to pftw.
#
#   irm https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.ps1 | iex
#   & ([scriptblock]::Create((irm .../install.ps1))) install 3.15 --path none   # arguments go to pftw
#
# This script never edits PATH. It downloads the pftw wheel, checks its SHA-256 against the
# package index, extracts pftw.exe to a temporary folder and runs it. pftw installs Python and,
# if you agree, puts Python and itself on PATH. The temporary copy is deleted afterwards.
# Works in Windows PowerShell 5.1 and PowerShell 7.
#
# Settings (environment variables):
#   PFTW_VERSION        pftw version to install, e.g. 0.3.0 (default: newest final release)
#   PFTW_INDEX_URL      PEP 503 simple index (default: https://pypi.org/simple); mirrors work
#   PFTW_KEEP           copy pftw.exe (and pftww.exe) into this folder instead of running it
#   PFTW_ALLOW_INSECURE set to 1 to allow a non-HTTPS index (testing only)

function Install-Pftw {
    $ErrorActionPreference = 'Stop'
    $progress = $ProgressPreference
    $ProgressPreference = 'SilentlyContinue' # The 5.1 progress bar slows downloads drastically.
    # Windows PowerShell 5.1 may not offer TLS 1.2 by default; PyPI requires it.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $index = if ($env:PFTW_INDEX_URL) { $env:PFTW_INDEX_URL.TrimEnd('/') } else { 'https://pypi.org/simple' }
    if (-not $index.StartsWith('https://') -and $env:PFTW_ALLOW_INSECURE -ne '1') {
        throw 'PFTW_INDEX_URL must use https:// because the index supplies the checksum.'
    }
    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    $plat = switch ($arch) {
        'AMD64' { 'win_amd64' }
        'ARM64' { 'win_arm64' }
        default { throw "Unsupported Windows architecture: $arch" }
    }

    $page = (Invoke-WebRequest -UseBasicParsing -Uri "$index/pftw/").Content
    # Wheels for this platform, skipping yanked releases.
    $links = @()
    foreach ($m in [regex]::Matches($page, '<a\s[^>]*>[^<]*</a>')) {
        if ($m.Value -match 'data-yanked') { continue }
        if ($m.Value -match "href=""([^""]*)""[^>]*>(pftw-([^-<]+)-py3-none-$plat\.whl)</a>") {
            $links += [pscustomobject]@{ File = $Matches[2]; Version = $Matches[3]; Href = $Matches[1].Replace('&amp;', '&') }
        }
    }
    if ($links.Count -eq 0) { throw "No pftw wheel for $plat at $index" }

    $version = if ($env:PFTW_VERSION) { $env:PFTW_VERSION.TrimStart('v') } else { '' }
    if (-not $version) {
        $finals = @($links | Where-Object { $_.Version -match '^\d+(\.\d+)*$' } | Sort-Object {
                $parts = @($_.Version.Split('.') | ForEach-Object { [int64]$_ }) + @(0, 0, 0, 0)
                '{0:D12}.{1:D12}.{2:D12}.{3:D12}' -f $parts[0], $parts[1], $parts[2], $parts[3]
            })
        if ($finals.Count -eq 0) { throw "No final pftw release for $plat at $index; set PFTW_VERSION to choose a pre-release." }
        $version = $finals[-1].Version
    }
    $link = $links | Where-Object { $_.File -eq "pftw-$version-py3-none-$plat.whl" } | Select-Object -First 1
    if (-not $link) { throw "pftw $version has no wheel for $plat at $index" }
    $parts = $link.Href.Split('#', 2)
    if ($parts.Count -ne 2 -or $parts[1] -notmatch '^sha256=([0-9a-fA-F]{64})$') {
        throw "The index gave no SHA-256 for pftw $version; refusing to install."
    }
    $expected = $Matches[1].ToLowerInvariant()
    $url = ([Uri]::new([Uri]"$index/pftw/", $parts[0])).AbsoluteUri

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("pftw-install-" + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        $zip = Join-Path $tmp 'pftw.zip' # Expand-Archive only accepts the .zip extension.
        Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $zip
        if ((Get-FileHash -Algorithm SHA256 -Path $zip).Hash.ToLowerInvariant() -ne $expected) {
            throw "SHA-256 mismatch for $url; nothing was run."
        }
        Expand-Archive -Path $zip -DestinationPath (Join-Path $tmp 'x')
        $bin = Join-Path $tmp 'bin'
        New-Item -ItemType Directory -Path $bin | Out-Null
        Get-ChildItem -Path (Join-Path $tmp 'x') -Directory -Filter '*.data' |
            ForEach-Object { Get-ChildItem -Path (Join-Path $_.FullName 'scripts') -Filter '*.exe' } |
            Copy-Item -Destination $bin
        $exe = Join-Path $bin 'pftw.exe'
        if (-not (Test-Path $exe)) { throw 'The wheel did not contain pftw.exe' }
        [Console]::Error.WriteLine("pftw $version ($plat) from $index, SHA-256 verified.")

        if ($env:PFTW_KEEP) {
            New-Item -ItemType Directory -Force -Path $env:PFTW_KEEP | Out-Null
            Copy-Item -Path (Join-Path $bin '*.exe') -Destination $env:PFTW_KEEP
            [Console]::Error.WriteLine("Saved pftw.exe to $($env:PFTW_KEEP)")
            $script:PftwStatus = 0
            return
        }
        $pftwArgs = @($args)
        if ($pftwArgs.Count -eq 0 -and [Console]::IsInputRedirected) {
            $pftwArgs = @('install') # No terminal for the wizard.
        }
        & $exe @pftwArgs
        $status = $LASTEXITCODE
        $shim = Join-Path $env:LOCALAPPDATA 'pftw\shims\pftw.exe'
        if ($status -eq 0 -and (Test-Path $shim)) {
            [Console]::Error.WriteLine("pftw is now at $shim. Open a new terminal to pick up PATH changes.")
        }
        $script:PftwStatus = $status
    }
    finally {
        $ProgressPreference = $progress
        Remove-Item -Recurse -Force -Path $tmp -ErrorAction SilentlyContinue
    }
}

$script:PftwStatus = 1
Install-Pftw @args
$pftwStatus = $script:PftwStatus
$global:LASTEXITCODE = $pftwStatus
# Exiting would close the caller's window under "irm | iex"; only a script file run sets the process exit code.
if ($MyInvocation.MyCommand.Path) { exit $pftwStatus }
if ($pftwStatus -ne 0) { Write-Error "pftw exited with code $pftwStatus" -ErrorAction Continue }
