package forges

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"codeberg.org/mvdkleijn/forgejo-sdk/forgejo"
	"github.com/google/go-github/v42/github"
	"github.com/pojntfx/felicitas.pojtinger.com/data"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

const (
	typePushEvent = "PushEvent"
)

var (
	errNoActivityFoundFromAnyForge = errors.New("no activity found from any forge")
	errInvalidPushEvent            = errors.New("invalid push event")
)

type ForgeType string

const (
	ForgeTypeGitHub  ForgeType = "github"
	ForgeTypeForgejo ForgeType = "forgejo"
)

type ForgeConfig struct {
	Domain string    `yaml:"domain"`
	Type   ForgeType `yaml:"type"`
	API    string    `yaml:"api"`
	CDN    string    `yaml:"cdn"`
	Icon   string    `yaml:"icon"`
	Name   string    `yaml:"name"`
	Shield string    `yaml:"shield"`
}

type Output struct {
	UserName          string `json:"username"`
	UserFollowerCount int    `json:"userFollowerCount"`
	UserURL           string `json:"userURL"`

	LastCommitTime     string `json:"lastCommitTime"`
	LastCommitRepoName string `json:"lastCommitRepoName"`
	LastCommitRepoURL  string `json:"lastCommitRepoURL"`
	LastCommitMessage  string `json:"lastCommitMessage"`
	LastCommitURL      string `json:"lastCommitURL"`

	ForgeIcon   string `json:"forgeIcon"`
	ForgeDomain string `json:"forgeDomain"`
	ForgeName   string `json:"forgeName"`
	ForgeShield string `json:"forgeShield"`
}

func ForgesHandler(w http.ResponseWriter, r *http.Request, forgesYAML []byte, tokens map[string]string) {
	username := r.URL.Query().Get("username")
	if username == "" {
		w.Write([]byte("missing username query parameter"))

		panic("missing username query parameter")
	}

	// Support limit parameter for multiple results
	// Default from GITHUB_ACTIVITY_LIMIT env var, or 1 for backwards compatibility
	limit := 1
	if envLimit := os.Getenv("GITHUB_ACTIVITY_LIMIT"); envLimit != "" {
		if l, err := fmt.Sscanf(envLimit, "%d", &limit); err != nil || l != 1 {
			limit = 1
		}
	}
	// Query param overrides env var
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 {
			limit = 1
		}
	}
	// Enforce bounds
	if limit < 1 {
		limit = 1
	}
	if limit > 10 {
		limit = 10
	}

	var forgesList []ForgeConfig
	if err := yaml.Unmarshal(forgesYAML, &forgesList); err != nil {
		panic(err)
	}

	var results []Output

	for _, forge := range forgesList {
		token := tokens[forge.Domain]

		var outputs []Output
		var err error

		switch forge.Type {
		case ForgeTypeGitHub:
			outputs, err = fetchGitHubActivities(r, forge, username, token, limit)
		case ForgeTypeForgejo:
			output, ferr := fetchForgejoActivity(r, forge, username, token)
			if ferr == nil {
				outputs = []Output{output}
			}
			err = ferr
		}

		if err != nil {
			panic(err)
		}

		for i := range outputs {
			outputs[i].ForgeIcon = forge.Icon
			outputs[i].ForgeDomain = forge.Domain
			outputs[i].ForgeName = forge.Name
			outputs[i].ForgeShield = forge.Shield
		}
		results = append(results, outputs...)
	}

	if len(results) == 0 {
		panic(errNoActivityFoundFromAnyForge)
	}

	sort.Slice(results, func(i, j int) bool {
		ti, err := time.Parse(time.RFC3339, results[i].LastCommitTime)
		if err != nil {
			panic(err)
		}

		tj, err := time.Parse(time.RFC3339, results[j].LastCommitTime)
		if err != nil {
			panic(err)
		}

		return ti.After(tj)
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	// Return single object for backwards compatibility when limit=1
	if limit == 1 {
		j, err := json.Marshal(results[0])
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, "%v", string(j))
		return
	}

	// Return array when limit > 1
	j, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(w, "%v", string(j))
}

func fetchGitHubActivity(r *http.Request, forge ForgeConfig, username string, token string) (Output, error) {
	outputs, err := fetchGitHubActivities(r, forge, username, token, 1)
	if err != nil {
		return Output{}, err
	}
	if len(outputs) == 0 {
		return Output{}, errNoActivityFoundFromAnyForge
	}
	return outputs[0], nil
}

func fetchGitHubActivities(r *http.Request, forge ForgeConfig, username string, token string, limit int) ([]Output, error) {
	var httpClient *http.Client
	if token != "" {
		httpClient = oauth2.NewClient(
			r.Context(),
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: token,
				},
			),
		)
	}

	client := github.NewClient(httpClient)

	var err error
	client.BaseURL, err = url.Parse(forge.API)
	if err != nil {
		return nil, err
	}

	user, _, err := client.Users.Get(r.Context(), username)
	if err != nil {
		return nil, err
	}

	events, _, err := client.Activity.ListEventsPerformedByUser(r.Context(), username, true, nil)
	if err != nil {
		return nil, err
	}

	baseOutput := Output{
		UserName:          user.GetLogin(),
		UserFollowerCount: user.GetFollowers(),
		UserURL:           user.GetHTMLURL(),
	}

	// Collect push events up to limit
	var pushEvents []*github.Event
	for _, candidate := range events {
		if candidate.GetType() == typePushEvent {
			pushEvents = append(pushEvents, candidate)
			if len(pushEvents) >= limit {
				break
			}
		}
	}

	if len(pushEvents) == 0 {
		return nil, nil
	}

	var outputs []Output

	for _, event := range pushEvents {
		output := baseOutput
		output.LastCommitTime = event.GetCreatedAt().Format(time.RFC3339)

		owner, repoName := path.Split(event.GetRepo().GetName())
		owner = strings.TrimSuffix(owner, "/")

		repo, _, err := client.Repositories.Get(r.Context(), owner, repoName)
		if err != nil {
			// Skip this event if we can't get repo info
			continue
		}

		if repo != nil {
			output.LastCommitRepoName = repo.GetFullName()
			output.LastCommitRepoURL = repo.GetHTMLURL()
		}

		rawPayload, err := event.ParsePayload()
		if err != nil {
			continue
		}

		pushEvent, ok := rawPayload.(*github.PushEvent)
		if !ok {
			continue
		}

		if pushEvent.Head != nil {
			commit, _, err := client.Repositories.GetCommit(r.Context(), owner, repoName, pushEvent.GetHead(), nil)
			if err != nil {
				continue
			}

			output.LastCommitURL = commit.GetHTMLURL()

			if commit.Commit != nil {
				output.LastCommitMessage = commit.Commit.GetMessage()
			}
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	tokens := map[string]string{}
	if forgeTokens := os.Getenv("FORGE_TOKENS"); forgeTokens != "" {
		if err := json.Unmarshal([]byte(forgeTokens), &tokens); err != nil {
			panic(fmt.Errorf("failed to parse forge tokens: %w", err))
		}
	}

	ForgesHandler(w, r, data.ForgesYAML, tokens)
}

func fetchForgejoActivity(r *http.Request, forge ForgeConfig, username string, token string) (Output, error) {
	options := []forgejo.ClientOption{
		forgejo.SetContext(r.Context()),
	}
	if token != "" {
		options = append(options, forgejo.SetToken(token))
	}

	client, err := forgejo.NewClient(forge.API, options...)
	if err != nil {
		return Output{}, err
	}

	user, _, err := client.GetUserInfo(username)
	if err != nil {
		return Output{}, err
	}

	output := Output{
		UserName:          user.UserName,
		UserFollowerCount: user.FollowerCount,
		UserURL:           fmt.Sprintf("%s%s", forge.API, user.UserName),
	}

	repos, _, err := client.ListUserRepos(username, forgejo.ListReposOptions{})
	if err != nil {
		return Output{}, err
	}

	if len(repos) > 0 {
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].Updated.After(repos[j].Updated)
		})

		mostRecentRepo := repos[0]

		commits, _, err := client.ListRepoCommits(mostRecentRepo.Owner.UserName, mostRecentRepo.Name, forgejo.ListCommitOptions{
			ListOptions: forgejo.ListOptions{
				PageSize: 1,
			},
		})
		if err != nil {
			return Output{}, err
		}

		if len(commits) > 0 {
			commit := commits[0]
			output.LastCommitTime = commit.CommitMeta.Created.Format(time.RFC3339)
			output.LastCommitRepoName = mostRecentRepo.FullName
			output.LastCommitRepoURL = mostRecentRepo.HTMLURL
			output.LastCommitMessage = commit.RepoCommit.Message
			output.LastCommitURL = commit.HTMLURL
		}
	}

	return output, nil
}
