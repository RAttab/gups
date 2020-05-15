package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type Repo struct {
	Path   string   `json:"path"`
	Owners []string `json:"owners"`
}

type Config struct {
	Users      map[string]string `json:"github_to_slack_user" yaml:"github_to_slack_user"`
	Repos      []Repo            `json:"repos" yaml:"repos"`
	SkipLabels []string          `json:"skip_pr_labels" yaml:"skip_pr_labels"`
}

func ReadConfig(file, format string) (*Config, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	switch format {
	case ".yml", ".yaml":
		fmt.Println("File is in yaml format")
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	case ".json":
		fmt.Println("File is in json format")
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("config file is not a valid format")
	}

	if len(config.Users) == 0 {
		return nil, fmt.Errorf("no configured users")
	}

	if len(config.Repos) == 0 {
		return nil, fmt.Errorf("no configured repos")
	}

	for _, repo := range config.Repos {
		if _, err := PathToVariables(repo.Path); err != nil {
			return nil, err
		}

		for _, owner := range repo.Owners {
			if _, ok := config.Users[owner]; !ok {
				return nil, fmt.Errorf("unconfigured repo owner: %v", owner)
			}
		}
	}

	return config, err
}

func PathToVariables(path string) (Variables, error) {
	split := strings.Split(path, "/")

	if len(split) != 2 {
		return Variables{}, fmt.Errorf("invalid path: %v", path)
	}

	vars := Variables{
		Owner:      split[0],
		Repository: split[1],
	}

	return vars, nil
}
