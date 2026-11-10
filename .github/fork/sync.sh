#!/usr/bin/env bash
set -euo pipefail

upstream=router-for-me/CLIProxyAPI
tag=$(gh api "repos/$upstream/releases/latest" --jq '.tag_name')
if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Unsupported upstream release tag: $tag" >&2
  exit 1
fi
git config user.name github-actions\[bot\]
git config user.email 41898282+github-actions\[bot\]@users.noreply.github.com
git remote add upstream "https://github.com/$upstream.git"
git fetch upstream main "refs/tags/$tag:refs/tags/$tag"
git merge --no-edit "$tag"
sha=$(git rev-parse HEAD)
publish=true
if [[ "$GITHUB_EVENT_NAME" == schedule ]] && [[ "$(python3 .github/fork/published_revision.py)" == "$sha" ]]; then
  publish=false
fi
{
  printf 'sha=%s\n' "$sha"
  printf 'upstream=%s\n' "$tag"
  printf 'version=%s-fork.%s.%s\n' "$tag" "$GITHUB_RUN_ID" "$GITHUB_RUN_ATTEMPT"
  printf 'publish=%s\n' "$publish"
} >> "$GITHUB_OUTPUT"
