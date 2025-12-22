package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type Repository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stars"`
	UpdatedAt   string `json:"updated_at"`
}

func FetchRepositories(ctx context.Context, username string) ([]Repository, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated&per_page=50", username)

	c := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("make request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var repos []struct {
		Name        string `json:"name"`
		URL         string `json:"html_url"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Stars       int    `json:"stargazers_count"`
		Fork        bool   `json:"fork"`
		Private     bool   `json:"private"`
		UpdatedAt   string `json:"updated_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("json encode repos: %w", err)
	}

	var data []Repository
	for _, repo := range repos {
		if repo.Fork || repo.Private {
			continue
		}
		data = append(data, Repository{
			Name:        repo.Name,
			URL:         repo.URL,
			Description: repo.Description,
			Language:    repo.Language,
			Stars:       repo.Stars,
			UpdatedAt:   repo.UpdatedAt,
		})
	}

	sort.Slice(data, func(i, j int) bool { return data[i].Stars > data[j].Stars })
	if len(data) > 20 {
		data = data[:20]
	}

	return data, nil
}
