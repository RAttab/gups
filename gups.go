package main

import (
	"context"
	"log"
	"os"
)

func main() {
	log.Printf("CONFIG: %v", os.Getenv("CONFIG"))
	log.Printf("GITHUB_TOKEN: %v", os.Getenv("GITHUB_TOKEN"))

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

	reviewed := make(map[string]struct{})
	for _, review := range pr.Reviews {
		if review.State == "APPROVED" {
			reviewed[review.Author] = struct{}{}
		}
	}

	for _, request := range pr.ReviewRequests {
		if _, ok := reviewed[request]; !ok {
			if slack, ok := config.TranslateGithubToSlack[request]; ok {
				notifs[slack] = append(notifs[slack], Notification{
					Path:        repo.Path,
					PullRequest: pr.Number,
					Message:     "Request",
				})
			} else {
				log.Printf("unkown github user '%v' for review request", request)
			}
		}
	}

	owners := make(map[string]struct{})
	for _, owner := range repo.Owners {
		if _, ok := reviewed[owner]; ok {
			owners[owner] = struct{}{}
		}
	}

	if len(owners) < 2 {
		for _, owner := range repo.Owners {
			github := config.Users[owner].Github

			if _, ok := reviewed[github]; !ok {
				if slack, ok := config.TranslateGithubToSlack[github]; ok {
					notifs[slack] = append(notifs[slack], Notification{
						Path:        repo.Path,
						PullRequest: pr.Number,
						Message:     "Pending",
					})
				}
			} else {
				log.Printf("unkown github user '%v' for pending review", owner)
			}
		}
	}
}
