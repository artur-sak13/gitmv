#!/usr/bin/env bash

set -e
set -o pipefail

main() {
  mapfile -t dirs < <(find "${HOME}" -name ".git")

  for dir in "${dirs[@]}"; do
    dir=$(dirname "$dir")
    # base=$(basename "$dir")

    (
      cd "$dir"
      local base_url
      base_url=$(git config --get remote.origin.url)
      # strip .git from end of url
      base_url=${base_url%\.git}
      base_url=${base_url//git@gitlab\.twopoint\.io:/git@github\.com:/twopt\/}
      
      # Validate that this folder is a git folder
      if ! git branch 2>/dev/null 1>&2 ; then
        echo "Not a git repo!"
        exit $?
      fi
      
      # Figure out current git branch
      # git_where=$(command git symbolic-ref -q HEAD || command git name-rev --name-only --no-undefined --always HEAD) 2>/dev/null
      # git_where=$(command git name-rev --name-only --no-undefined --always HEAD) 2>/dev/null

      # # Remove cruft from branchname
      # branch=${git_where#refs\/heads\/}

      url="$base_url"
      echo "Setting new remote for $url"
    )
  done
}
main