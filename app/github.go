package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/zorcal/its-a-me-zorcal/pkg/github"
)

var (
	cachedRepos    []github.Repository
	reposCacheTime time.Time
	reposCacheTTL  = 10 * time.Minute
	reposCacheMu   sync.RWMutex
)

func fetchGitHubRepos(ctx context.Context, log *slog.Logger) []github.Repository {
	reposCacheMu.RLock()
	if time.Since(reposCacheTime) < reposCacheTTL && len(cachedRepos) > 0 {
		result := make([]github.Repository, len(cachedRepos))
		copy(result, cachedRepos)
		reposCacheMu.RUnlock()
		return result
	}
	reposCacheMu.RUnlock()

	repos, err := github.FetchRepositories(ctx, "Zorcal")
	if err != nil {
		log.ErrorContext(ctx, "Unable to fetch GitHub repositories", "error", err)
		reposCacheMu.RLock()
		if len(cachedRepos) > 0 {
			result := make([]github.Repository, len(cachedRepos))
			copy(result, cachedRepos)
			reposCacheMu.RUnlock()
			return result
		}
		reposCacheMu.RUnlock()
		return []github.Repository{}
	}

	reposCacheMu.Lock()
	cachedRepos = repos
	reposCacheTime = time.Now()
	reposCacheMu.Unlock()

	return repos
}
