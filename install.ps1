# Install or update ctop on Windows from GitHub releases.
#
#   irm https://raw.githubusercontent.com/maxeo/ctop/master/install.ps1 | iex
#
# With options:
#
#   & ([scriptblock]::Create((irm https://raw.githubusercontent.com/maxeo/ctop/master/install.ps1))) -Version 0.8.12
#
# or set them as environment variables before the first command:
# CTOP_VERSION, CTOP_DIR, CTOP_REPO, CTOP_NO_VERIFY=1, CTOP_NO_PATH=1,
# CTOP_UNINSTALL=1.
#
# Works in Windows PowerShell 5.1 and PowerShell 7. Installs into the user's
# folder (no administrator rights needed) and adds it to the user's PATH.
# This file is ASCII only, so that Windows PowerShell reads it correctly.

param(
    [string]$Version = $env:CTOP_VERSION,
    [string]$Dir = $env:CTOP_DIR,
    [string]$Repo = $(if ($env:CTOP_REPO) { $env:CTOP_REPO } else { 'maxeo/ctop' }),
    [switch]$NoVerify = ($env:CTOP_NO_VERIFY -eq '1'),
    [switch]$NoPath = ($env:CTOP_NO_PATH -eq '1'),
    [switch]$Uninstall = ($env:CTOP_UNINSTALL -eq '1'),
    [switch]$Help
)

# Everything runs in a function that returns on errors: with "irm | iex" the
# script runs in the user's session, where "exit" would close the window.
function Install-Ctop {
    param($Version, $Dir, $Repo, $NoVerify, $NoPath, $Uninstall, $Help)

    $ErrorActionPreference = 'Stop'
    # Invoke-WebRequest is very slow with its progress bar in PowerShell 5.1
    $ProgressPreference = 'SilentlyContinue'
    # PowerShell 5.1 may not enable TLS 1.2, which github.com requires
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    if ($Help) {
        Write-Host @"
Usage: install.ps1 [options]

  -Version X    install version X (e.g. 0.8.12); default: latest release
  -Dir DIR      install into DIR; default: where ctop.exe already is,
                otherwise $env:LOCALAPPDATA\Programs\ctop
  -Repo O/N     GitHub repository; default: maxeo/ctop
  -NoVerify     skip the sha256 check against sha256sums.txt
  -NoPath       do not add DIR to the user's PATH
  -Uninstall    remove ctop.exe from DIR (or from where it is found), and
                DIR from the user's PATH
  -Help         show this help
"@
        return
    }

    # --- target directory ---------------------------------------------------

    if (-not $Dir) {
        $existing = Get-Command ctop.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($existing) {
            $Dir = Split-Path -Parent $existing.Source
        } else {
            $Dir = Join-Path $env:LOCALAPPDATA 'Programs\ctop'
        }
    }
    $Dir = [IO.Path]::GetFullPath($Dir)
    $target = Join-Path $Dir 'ctop.exe'

    if ($Uninstall) {
        if (-not (Test-Path $target)) {
            Write-Error "install.ps1: ctop.exe not found in $Dir"
            return
        }
        Remove-Item -Force $target
        Remove-Item -Force (Join-Path $Dir 'ctop.exe.old') -ErrorAction SilentlyContinue
        Write-Host "removed $target"
        if (Remove-UserPath $Dir) {
            Write-Host "removed $Dir from the user's PATH"
        }
        if (-not (Get-ChildItem -Force $Dir)) {
            Remove-Item -Force $Dir
        }
        return
    }

    # --- platform -----------------------------------------------------------

    # only an amd64 build exists; Windows on ARM runs it through emulation
    $arch = 'amd64'
    if (-not [Environment]::Is64BitOperatingSystem) {
        Write-Error 'install.ps1: ctop needs a 64-bit Windows'
        return
    }

    # --- version ------------------------------------------------------------

    if (-not $Version) {
        Write-Host "looking up the latest release of $Repo..."
        $Version = Get-LatestVersion $Repo
        if (-not $Version) {
            Write-Error "install.ps1: could not determine the latest release of $Repo (is github.com reachable? or pass -Version)"
            return
        }
    }
    $Version = $Version -replace '^v', ''

    $file = "ctop-$Version-windows-$arch.exe"
    $base = "https://github.com/$Repo/releases/download/v$Version"

    if (Test-Path $target) {
        $current = ''
        try {
            $out = & $target -v 2>$null
            if ("$out" -match '^ctop version ([^,]+)') { $current = $Matches[1] }
        } catch { }
        if ($current -eq $Version) {
            Write-Host "ctop $Version is already installed in $target"
            return
        }
        if (-not $current) { $current = 'unknown' }
        Write-Host "updating ${target}: $current -> $Version"
    } else {
        Write-Host "installing ctop $Version (windows/$arch) into $target"
    }

    # --- download, verify, install ------------------------------------------

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("ctop-" + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        $download = Join-Path $tmp 'ctop.exe'
        Write-Host "downloading $base/$file"
        try {
            Invoke-WebRequest -UseBasicParsing -Uri "$base/$file" -OutFile $download -TimeoutSec 300
        } catch {
            Write-Error "install.ps1: download failed: $base/$file ($($_.Exception.Message))"
            return
        }

        if (-not $NoVerify) {
            $expected = ''
            try {
                $sums = (Invoke-WebRequest -UseBasicParsing -Uri "$base/sha256sums.txt" -TimeoutSec 60).Content
                if ($sums -is [byte[]]) { $sums = [Text.Encoding]::ASCII.GetString($sums) }
                foreach ($line in ($sums -split "`n")) {
                    $parts = $line.Trim() -split '\s+'
                    if ($parts.Count -eq 2 -and ($parts[1] -eq $file -or $parts[1] -eq "*$file")) {
                        $expected = $parts[0].ToLower()
                    }
                }
            } catch { }
            if (-not $expected) {
                Write-Error "install.ps1: $file is not listed in $base/sha256sums.txt, so it cannot be verified (-NoVerify installs it anyway)"
                return
            }
            $actual = (Get-FileHash -Algorithm SHA256 $download).Hash.ToLower()
            if ($actual -ne $expected) {
                Write-Error "install.ps1: checksum mismatch for $file (expected $expected, got $actual)"
                return
            }
        }

        if (-not (Test-Path $Dir)) {
            New-Item -ItemType Directory -Path $Dir | Out-Null
        }
        # a running ctop.exe cannot be overwritten, but it can be renamed:
        # move it aside, then put the new one in its place
        $old = Join-Path $Dir 'ctop.exe.old'
        Remove-Item -Force $old -ErrorAction SilentlyContinue
        if (Test-Path $target) {
            Move-Item -Force $target $old
        }
        Move-Item -Force $download $target
        Remove-Item -Force $old -ErrorAction SilentlyContinue
    } finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }

    & $target -v

    if (-not $NoPath) {
        if (Add-UserPath $Dir) {
            Write-Host "added $Dir to the user's PATH (open a new terminal to use it everywhere)"
        }
    } elseif (-not (Test-InPath $Dir)) {
        Write-Host "note: $Dir is not in PATH"
    }
}

# Get-LatestVersion returns the latest release (without "v"), read from the
# redirect of github.com/<repo>/releases/latest: unlike api.github.com it has
# no 60-requests-per-hour limit. Falls back to the API.
function Get-LatestVersion($Repo) {
    try {
        $req = [Net.HttpWebRequest]::Create("https://github.com/$Repo/releases/latest")
        $req.AllowAutoRedirect = $false
        $req.Method = 'HEAD'
        $req.Timeout = 15000
        $resp = $req.GetResponse()
        $location = $resp.Headers['Location']
        $resp.Close()
        if ($location -match '/tag/v?([^/]+)$') { return $Matches[1] }
    } catch { }
    try {
        $rel = Invoke-RestMethod -UseBasicParsing -Uri "https://api.github.com/repos/$Repo/releases/latest" -TimeoutSec 60
        return ($rel.tag_name -replace '^v', '')
    } catch { }
    return ''
}

function Get-UserPath {
    $p = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not $p) { return @() }
    return @($p -split ';' | Where-Object { $_ })
}

function Test-SameDir($a, $b) {
    return $a.TrimEnd('\') -ieq $b.TrimEnd('\')
}

function Test-InPath($Dir) {
    foreach ($d in ($env:Path -split ';')) {
        if ($d -and (Test-SameDir $d $Dir)) { return $true }
    }
    return $false
}

# Add-UserPath adds Dir to the user's PATH (and to this session's), and
# reports whether it was missing.
function Add-UserPath($Dir) {
    $user = Get-UserPath
    foreach ($d in $user) {
        if (Test-SameDir $d $Dir) { return $false }
    }
    if (Test-InPath $Dir) { return $false } # in the system PATH
    [Environment]::SetEnvironmentVariable('Path', (($user + $Dir) -join ';'), 'User')
    $env:Path = $env:Path.TrimEnd(';') + ';' + $Dir
    return $true
}

# Remove-UserPath removes Dir from the user's PATH, and reports whether it
# was there.
function Remove-UserPath($Dir) {
    $user = Get-UserPath
    $kept = @($user | Where-Object { -not (Test-SameDir $_ $Dir) })
    if ($kept.Count -eq $user.Count) { return $false }
    [Environment]::SetEnvironmentVariable('Path', ($kept -join ';'), 'User')
    return $true
}

Install-Ctop -Version $Version -Dir $Dir -Repo $Repo -NoVerify $NoVerify -NoPath $NoPath -Uninstall $Uninstall -Help $Help
