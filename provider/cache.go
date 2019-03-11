package provider

import (
	"sync"
	"time"
)

// TODO: Pull out generic cache (possibly in GitCache interface)

type CachedIssue struct {
	Issue *GitIssue

	commentMu sync.RWMutex
	Comments  map[time.Time]*GitIssueComment
}

type CachedRepo struct {
	Repo *GitRepository

	issueMu sync.RWMutex
	Issues  map[string]*CachedIssue

	labelMu sync.RWMutex
	Labels  map[string]*GitLabel
}
