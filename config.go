package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"
)

type Pool []string

func (pool Pool) Contains(user string) bool {
	for _, item := range pool {
		if item == user {
			return true
		}
	}
	return false
}

type Repo struct {
	Path string `json:"path"`
	Rule string `json:"rule"`
}

type Config struct {
	Users      map[string]string              `json:"github_to_slack_user"`
	Pools      map[string]Pool                `json:"pools"`
	Rules      map[string]map[string][]string `json:"rules"`
	Repos      []Repo                         `json:"repos"`
	SkipLabels []string                       `json:"skip_pr_labels"`
}

func ReadConfig(file string) *Config {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		Fatal("unable to open '%v': %v", file, err)
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		Fatal("unable to parse '%v': %v", file, err)
	}

	if len(config.Users) == 0 {
		Fatal("missing field 'github_to_slack_user' in '%v'", file)
	}

	if len(config.Pools) == 0 {
		Fatal("missing field 'pools' in '%v'", file)
	}

	for poolName, pool := range config.Pools {
		for _, user := range pool {
			if _, ok := config.Users[user]; !ok {
				Fatal("unknown user '%v' in pool '%v'", user, poolName)
			}
		}
	}

	if len(config.Repos) == 0 {
		Fatal("missing field 'repos' in '%v'", file)
	}

	for _, repo := range config.Repos {
		PathToVariables(repo.Path)
		if _, ok := config.Rules[repo.Rule]; !ok {
			Fatal("unknown rule '%v' in repo '%v'", repo.Rule, repo.Path)
		}

	}
	return config
}

func PathToVariables(path string) Variables {
	split := strings.Split(path, "/")

	if len(split) != 2 {
		Fatal("invalid repo path '%v'", path)
	}

	vars := Variables{
		Owner:      split[0],
		Repository: split[1],
	}

	return vars
}
