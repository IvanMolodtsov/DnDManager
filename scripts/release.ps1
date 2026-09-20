# Merge develop → main, tag vX.Y.Z, push, and create a GitHub Release.
# Usage: ./scripts/release.ps1 1.2.0
#        ./scripts/release.ps1 1.2.0 --from <sha>
# No history rewrite, no force-push. Requires a clean tree and gh.
$ErrorActionPreference = "Stop"
if ($PSVersionTable.PSVersion.Major -ge 7) {
    $PSNativeCommandUseErrorActionPreference = $false
}
Set-Location (Split-Path -Parent $PSScriptRoot)

function Fail([string]$Message) {
    Write-Error $Message
    exit 1
}

function Invoke-Git {
    git @args
    if ($LASTEXITCODE -ne 0) {
        Fail "git $($args -join ' ') failed (exit $LASTEXITCODE)"
    }
}

$Version = $null
$From = $null
for ($i = 0; $i -lt $args.Count; $i++) {
    if ($args[$i] -eq "--from") {
        if ($i + 1 -ge $args.Count) { Fail "--from requires a SHA" }
        $From = [string]$args[$i + 1]
        $i++
        continue
    }
    if ($null -eq $Version) {
        $Version = [string]$args[$i]
        continue
    }
    Fail "Unexpected argument: $($args[$i])"
}

if (-not $Version) {
    Fail "Usage: ./scripts/release.ps1 X.Y.Z [--from SHA]"
}
$Version = $Version.TrimStart("v", "V")
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    Fail "Version must be X.Y.Z (no leading v). Got: $Version"
}
$Tag = "v$Version"

foreach ($cmd in @("git", "gh")) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Fail "$cmd is required"
    }
}

$status = git status --porcelain
if ($LASTEXITCODE -ne 0) { Fail "git status failed" }
if ($status) {
    Fail "Working tree is not clean. Commit or stash first."
}

$OriginalBranch = (git branch --show-current).Trim()
if (-not $OriginalBranch) {
    Fail "Detached HEAD is not supported (checkout develop)."
}

Invoke-Git fetch origin

$Source = "develop"
if ($From) {
    git rev-parse --verify "$From^{commit}" | Out-Null
    if ($LASTEXITCODE -ne 0) { Fail "Unknown --from SHA: $From" }
    $Source = $From
} else {
    if ($OriginalBranch -ne "develop") {
        Fail "Checkout develop (or pass --from SHA). On $OriginalBranch."
    }
    Invoke-Git merge --ff-only origin/develop
}

git show-ref --quiet --tags "refs/tags/$Tag"
if ($LASTEXITCODE -eq 0) {
    Fail "Tag $Tag already exists"
}
git ls-remote --exit-code --tags origin "refs/tags/$Tag" | Out-Null
if ($LASTEXITCODE -eq 0) {
    Fail "Tag $Tag already exists on origin"
}

git rev-parse --verify origin/main 2>$null | Out-Null
if ($LASTEXITCODE -eq 0) {
    Invoke-Git checkout -B main origin/main
} else {
    git rev-parse --verify main 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Invoke-Git checkout main
    } else {
        Fail "origin/main is required (Vercel Production). Do not create a production branch."
    }
}

git merge --no-edit $Source
if ($LASTEXITCODE -ne 0) {
    Fail "Merge of $Source into main failed. Resolve locally (no rewrite / no force-push)."
}

New-Item -ItemType Directory -Force -Path tmp | Out-Null
$NotesFile = Join-Path $PWD "tmp\release-notes-$Tag.md"

$LastTag = (git describe --tags --abbrev=0 2>$null)
if ($LASTEXITCODE -ne 0) { $LastTag = "" } else { $LastTag = $LastTag.Trim() }

$lines = New-Object System.Collections.Generic.List[string]
[void]$lines.Add("## Changes")
[void]$lines.Add("")
if ($LastTag) {
    $logLines = @(git log --pretty=format:"- %s" "$LastTag..HEAD")
} else {
    $logLines = @(git log --pretty=format:"- %s" HEAD)
}
if ($logLines.Count -gt 0 -and $logLines[0]) {
    foreach ($line in $logLines) {
        if ($line) { [void]$lines.Add([string]$line) }
    }
} else {
    [void]$lines.Add("- No commits since previous tag.")
}

$prLines = @()
if ($LastTag) {
    $tagDate = (git log -1 --format=%cI $LastTag).Trim()
    if ($tagDate) {
        $prJson = gh pr list --state merged --base develop --search "merged:>=$tagDate" --limit 50 --json number,title
        if ($LASTEXITCODE -eq 0 -and $prJson) {
            $prs = $prJson | ConvertFrom-Json
            foreach ($pr in @($prs)) {
                $prLines += "- #$($pr.number) $($pr.title)"
            }
        }
    }
}
if ($prLines.Count -gt 0) {
    [void]$lines.Add("")
    [void]$lines.Add("## Pull requests")
    [void]$lines.Add("")
    foreach ($p in $prLines) { [void]$lines.Add($p) }
}

$utf8 = New-Object System.Text.UTF8Encoding $false
[System.IO.File]::WriteAllText($NotesFile, (($lines -join "`n") + "`n"), $utf8)

Invoke-Git tag -a $Tag -m "Release $Tag"
Invoke-Git push origin main
Invoke-Git push origin $Tag

gh release create $Tag --title $Tag --notes-file $NotesFile --target main
if ($LASTEXITCODE -ne 0) {
    Fail "gh release create failed. Tag $Tag is on origin; retry: gh release create $Tag --notes-file `"$NotesFile`" --target main"
}

Write-Host "Released $Tag. Vercel Production deploys from the main push."
git checkout $OriginalBranch
if ($LASTEXITCODE -ne 0) {
    Write-Host "Could not return to $OriginalBranch; you are on $(git branch --show-current)."
}
