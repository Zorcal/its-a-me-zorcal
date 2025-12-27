package app

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/zorcal/its-a-me-zorcal/pkg/github"
)

type cachedGitHubFetcher struct {
	mu        sync.RWMutex
	repos     []github.Repository
	cacheTime time.Time
	ttl       time.Duration
	username  string
}

func newCachedGitHubFetcher(username string, ttl time.Duration) *cachedGitHubFetcher {
	return &cachedGitHubFetcher{
		username: username,
		ttl:      ttl,
	}
}

func (g *cachedGitHubFetcher) FetchRepositories(ctx context.Context, log *slog.Logger) []github.Repository {
	g.mu.RLock()
	if time.Since(g.cacheTime) < g.ttl && len(g.repos) > 0 {
		result := slices.Clone(g.repos)
		g.mu.RUnlock()
		return result
	}
	g.mu.RUnlock()

	repos, err := github.FetchRepositories(ctx, g.username)
	if err != nil {
		log.ErrorContext(ctx, "Unable to fetch GitHub repositories", "error", err)
		g.mu.RLock()
		if len(g.repos) > 0 {
			result := slices.Clone(g.repos)
			g.mu.RUnlock()
			return result
		}
		g.mu.RUnlock()
		return []github.Repository{}
	}

	g.mu.Lock()
	g.repos = repos
	g.cacheTime = time.Now()
	g.mu.Unlock()

	return repos
}
