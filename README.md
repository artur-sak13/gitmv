# gitmv

[![Travis CI](https://img.shields.io/travis/artur-sak13/gitmv.svg?style=for-the-badge)](https://travis-ci.org/artur-sak13/gitmv)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/artur-sak13/gitmv)
[![Codacy](https://img.shields.io/codacy/grade/0aea23ce159f436c867d136635e2beff.svg?style=for-the-badge)](https://app.codacy.com/app/artur-sak13/gitmv)

A command line tool to migrate repos between GitLab and GitHub

*   [Installation](README.md#installation)
*   [Binaries](README.md#binaries)
*   [Via Go](README.md#via-go)
*   [Usage](README.md#usage)

## Installation

### Binaries

For installation instructions from binaries please visit the [Releases Page](https://github.com/artur-sak13/gitmv/releases).

#### Via Go

```console
go get github.com/artur-sak13/gitmv
```

## Usage

```console
gitmv -  A command line tool to migrate repos between GitLab and GitHub.

Usage: gitmv <command>

Flags:

  -d, --debug     enable debug logging (default: false)
  --dry-run       do not run migration just print the changes that would occur (default: false)
  --github-token  GitHub API token (or env var GITHUB_TOKEN) (default: none)
  --gitlab-token  GitLab API token (or env var GITLAB_TOKEN) (default: none)
  --gitlab-user   GitLab Username (default: none)
  --org           GitHub org to move repositories (default: none)
  --ssh-key       SSH private key path to push Wikis (default: none)
  -u, --url       Custom GitLab URL (default: none)

Commands:

  repos    Migrate all repos from one Git provider to another.
  issues   Migrate all issues from one Git provider to another.
  wikis    Migrate all wikis from one Git provider to another.
  version  Show the version information.
```
