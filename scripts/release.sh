#!/usr/bin/env bash
# Merge develop → main, tag vX.Y.Z, push, and create a GitHub Release.
# Usage: ./scripts/release.sh 1.2.0
#        ./scripts/release.sh 1.2.0 --from <sha>
# No history rewrite, no force-push. Requires a clean tree and gh.
set -euo pipefail
cd "$(dirname "$0")/.."

fail() {
  echo "$1" >&2
  exit 1
}

VERSION=""
FROM=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --from)
      [[ $# -ge 2 ]] || fail "--from requires a SHA"
      FROM="$2"
      shift 2
      ;;
    -*)
      fail "Unexpected argument: $1"
      ;;
    *)
      [[ -z "$VERSION" ]] || fail "Unexpected argument: $1"
      VERSION="$1"
      shift
      ;;
  esac
done

[[ -n "$VERSION" ]] || fail "Usage: ./scripts/release.sh X.Y.Z [--from SHA]"
VERSION="${VERSION#v}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "Version must be X.Y.Z (no leading v). Got: $VERSION"
TAG="v$VERSION"

command -v git >/dev/null || fail "git is required"
command -v gh >/dev/null || fail "gh is required"

[[ -z "$(git status --porcelain)" ]] || fail "Working tree is not clean. Commit or stash first."

ORIGINAL_BRANCH="$(git branch --show-current)"
[[ -n "$ORIGINAL_BRANCH" ]] || fail "Detached HEAD is not supported (checkout develop)."

git fetch origin

SOURCE="develop"
if [[ -n "$FROM" ]]; then
  git rev-parse --verify "${FROM}^{commit}" >/dev/null || fail "Unknown --from SHA: $FROM"
  SOURCE="$FROM"
else
  [[ "$ORIGINAL_BRANCH" == "develop" ]] || fail "Checkout develop (or pass --from SHA). On $ORIGINAL_BRANCH."
  git merge --ff-only origin/develop
fi

if git show-ref --quiet --tags "refs/tags/$TAG"; then
  fail "Tag $TAG already exists"
fi
if git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
  fail "Tag $TAG already exists on origin"
fi

if git rev-parse --verify origin/main >/dev/null 2>&1; then
  git checkout -B main origin/main
elif git rev-parse --verify main >/dev/null 2>&1; then
  git checkout main
else
  fail "origin/main is required (Vercel Production). Do not create a production branch."
fi

if ! git merge --no-edit "$SOURCE"; then
  fail "Merge of $SOURCE into main failed. Resolve locally (no rewrite / no force-push)."
fi

mkdir -p tmp
NOTES_FILE="tmp/release-notes-${TAG}.md"
LAST_TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"

{
  echo "## Changes"
  echo
  if [[ -n "$LAST_TAG" ]]; then
    LOG="$(git log --pretty=format:"- %s" "${LAST_TAG}..HEAD" || true)"
  else
    LOG="$(git log --pretty=format:"- %s" HEAD || true)"
  fi
  if [[ -n "$LOG" ]]; then
    printf '%s\n' "$LOG"
  else
    echo "- No commits since previous tag."
  fi

  if [[ -n "$LAST_TAG" ]]; then
    TAG_DATE="$(git log -1 --format=%cI "$LAST_TAG" || true)"
    if [[ -n "$TAG_DATE" ]]; then
      PRS="$(gh pr list --state merged --base develop --search "merged:>=${TAG_DATE}" --limit 50 --json number,title \
        --jq '.[] | "- #\(.number) \(.title)"' 2>/dev/null || true)"
      if [[ -n "$PRS" ]]; then
        echo
        echo "## Pull requests"
        echo
        printf '%s\n' "$PRS"
      fi
    fi
  fi
} > "$NOTES_FILE"

git tag -a "$TAG" -m "Release $TAG"
git push origin main
git push origin "$TAG"

if ! gh release create "$TAG" --title "$TAG" --notes-file "$NOTES_FILE" --target main; then
  fail "gh release create failed. Tag $TAG is on origin; retry: gh release create $TAG --notes-file $NOTES_FILE --target main"
fi

echo "Released $TAG. Vercel Production deploys from the main push."
if ! git checkout "$ORIGINAL_BRANCH"; then
  echo "Could not return to $ORIGINAL_BRANCH; you are on $(git branch --show-current)."
fi
