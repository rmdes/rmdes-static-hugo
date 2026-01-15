package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pojntfx/felicitas.pojtinger.com/pkg/forges"
	"gopkg.in/yaml.v3"
)

func main() {
	verbosity := flag.String("verbosity", "info", "Log level (debug, info, warn, error)")
	forgesFile := flag.String("forges", filepath.Join("data", "forges.yaml"), "Forges configuration file")
	tokens := flag.String("tokens", "", "Forge tokens as JSON object, e.g. {\"github.com\": \"token\"} (can also be set using the FORGE_TOKENS env variable)")
	username := flag.String("username", "", "GitHub username to fetch starred repos for")
	limit := flag.Int("limit", 50, "Maximum number of starred repos to fetch (0 for unlimited)")
	groupByLang := flag.Bool("group", false, "Group starred repos by language")

	flag.Parse()

	var level slog.Level
	if err := level.UnmarshalText([]byte(*verbosity)); err != nil {
		panic(err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	if *username == "" {
		log.Error("Username is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info("Reading forges configuration", "file", *forgesFile)

	forgesData, err := os.ReadFile(*forgesFile)
	if err != nil {
		panic(err)
	}

	var forgesList []forges.ForgeConfig
	if err := yaml.Unmarshal(forgesData, &forgesList); err != nil {
		panic(err)
	}

	if tokensEnv := os.Getenv("FORGE_TOKENS"); tokensEnv != "" {
		*tokens = tokensEnv
	}

	secrets := map[string]string{}
	if *tokens != "" {
		log.Info("Parsing forge tokens")

		if err := json.Unmarshal([]byte(*tokens), &secrets); err != nil {
			panic(fmt.Errorf("failed to parse tokens: %w", err))
		}
	}

	f, err := forges.OpenForges(ctx, forgesList, secrets)
	if err != nil {
		panic(err)
	}

	log.Info("Fetching starred repos", "username", *username, "limit", *limit)

	starred, err := f.FetchStarredRepos("github.com", *username, *limit)
	if err != nil {
		panic(err)
	}

	log.Info("Fetched starred repos", "count", len(starred))

	var output []byte
	if *groupByLang {
		categories := forges.GroupStarredByLanguage(starred)
		output, err = yaml.Marshal(categories)
	} else {
		output, err = yaml.Marshal(starred)
	}
	if err != nil {
		panic(err)
	}

	fmt.Print(string(output))
}
