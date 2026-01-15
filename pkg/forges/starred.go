package forges

import (
	"time"

	"github.com/google/go-github/v68/github"
)

// StarredRepo represents a starred repository
type StarredRepo struct {
	URL         string `yaml:"url"`
	Name        string `yaml:"name"`
	FullName    string `yaml:"fullName"`
	Description string `yaml:"description"`
	Language    string `yaml:"language"`
	Stars       int    `yaml:"stars"`
	Forks       int    `yaml:"forks"`
	StarredAt   string `yaml:"starredAt"`
	OwnerAvatar string `yaml:"ownerAvatar"`
}

// StarredCategory groups starred repos by language
type StarredCategory struct {
	Language string        `yaml:"language"`
	Repos    []StarredRepo `yaml:"repos"`
}

// FetchStarredRepos fetches starred repositories for a GitHub user
func (f *Forges) FetchStarredRepos(domain, username string, limit int) ([]StarredRepo, error) {
	client, ok := f.githubClients[domain]
	if !ok {
		return nil, nil // No client for this domain
	}

	var allStarred []StarredRepo
	opts := &github.ActivityListStarredOptions{
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		starred, resp, err := client.Activity.ListStarred(f.ctx, username, opts)
		if err != nil {
			return nil, err
		}

		for _, s := range starred {
			repo := s.GetRepository()
			starredAt := ""
			if s.StarredAt != nil {
				starredAt = s.StarredAt.Time.Format(time.RFC3339)
			}

			allStarred = append(allStarred, StarredRepo{
				URL:         repo.GetHTMLURL(),
				Name:        repo.GetName(),
				FullName:    repo.GetFullName(),
				Description: repo.GetDescription(),
				Language:    repo.GetLanguage(),
				Stars:       repo.GetStargazersCount(),
				Forks:       repo.GetForksCount(),
				StarredAt:   starredAt,
				OwnerAvatar: repo.GetOwner().GetAvatarURL(),
			})

			if limit > 0 && len(allStarred) >= limit {
				return allStarred, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allStarred, nil
}

// GroupStarredByLanguage groups starred repos by programming language
func GroupStarredByLanguage(repos []StarredRepo) []StarredCategory {
	languageMap := make(map[string][]StarredRepo)

	for _, repo := range repos {
		lang := repo.Language
		if lang == "" {
			lang = "Other"
		}
		languageMap[lang] = append(languageMap[lang], repo)
	}

	var categories []StarredCategory
	for lang, repos := range languageMap {
		categories = append(categories, StarredCategory{
			Language: lang,
			Repos:    repos,
		})
	}

	return categories
}
