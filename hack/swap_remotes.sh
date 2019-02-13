#!/usr/bin/env bash
set -e
set -o pipefail

main() {
  mapfile -t dirs < <(find "${HOME}" -maxdepth 4 -name ".git" ! -path "*vim*" -type d -exec dirname {} \; -prune 2>/dev/null)

  for dir in "${dirs[@]}"; do
    (
      cd "$dir"

      local base_url
      local repo
      base_url=$(git config --get remote.origin.url || true)

      if [[ "$base_url" != git@gitlab.twopoint.io* ]]; then
        exit $?
      fi

      repo=$(basename "$base_url")
      repo_sans_git=$(basename -s .git "$base_url")

      echo -e "\033[33m${base_url}\033[0m -> \033[32mgit@github.com:twopt/$repo\033[0m"
      git remote set-url origin "git@github.com:twopt/$repo" &>/dev/null || (echo "failed to set remote URL for $repo_sans_git" && exit 1)

    )
  done

  echo
  echo "All remotes updated!"
  echo
}

main