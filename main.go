package gups

import (
	"context"
	"log"
	"os"
)

func main() {
	path := os.Getenv("CONFIG")
	config, err := ReadConfig(path)
	if err != nil {
		log.Fatalf("unable to read config '%v': %v", path, err)
	}

	client := ConnectGithub()

	notifs := make(Notifications)

	for _, repo := range config.Repos {
		vars, _ := PathToVariables(repo.Path)
		prs, err := QueryPullRequests(context.TODO(), client, vars)
		if err != nil {
			log.Print(err)
			continue
		}

		for _, pr := range prs {
			check(&repo, &pr, config, notifs)
		}
	}

}

func check(repo *Repo, pr *PullRequest, config *Config, notifs Notifications) {
	for _, label := range pr.Labels {
		if label == "wip" {
			return
		}
	}

}
