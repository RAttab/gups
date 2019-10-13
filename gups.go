package main

import (
	"context"
	"log"
	"os"
)

func main() {
	if false { //debug
		log.Printf("CONFIG: %v", os.Getenv("CONFIG"))
		log.Printf("GITHUB_TOKEN: %v", os.Getenv("GITHUB_TOKEN"))
	}

	path := os.Getenv("CONFIG")
	config, err := ReadConfig(path)
	if err != nil {
		log.Fatalf("unable to read config '%v': %v", path, err)
	}

	githubClient := ConnectGithub()

	notifs := make(map[string][]Notification)

	for index, repo := range config.Repos {
		log.Printf("[%v/%v] processing %v...", index+1, len(config.Repos), repo.Path)

		vars, _ := PathToVariables(repo.Path)
		prs, err := QueryPullRequests(context.TODO(), githubClient, vars)
		if err != nil {
			log.Print(err)
			continue
		}

		for _, pr := range prs {
			check(&repo, pr, config, notifs)
		}
	}

	slackClient, err := ConnectSlack(context.TODO())
	if err != nil {
		log.Fatalf("Unable to connect to slack: %v", err)
	}

	index := 0
	for user, notif := range notifs {
		log.Printf("[%v/%v] notifying %v...", index+1, len(notifs), user)

		if err := NotifySlack(slackClient, user, notif); err != nil {
			log.Fatalf("Unable to notify slack: %v", err)
		}

		index++
	}
}

func check(repo *Repo, pr *PullRequest, config *Config, notifs map[string][]Notification) {
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

	reviewedOwners := make(map[string]struct{})
	for _, owner := range repo.Owners {
		github := config.Users[owner].Github
		if _, ok := reviewed[github]; ok {
			reviewedOwners[github] = struct{}{}
		}
	}

	notified := make(map[string]struct{})

	addNotification := func(name, github string) {
		if false { // debug
			log.Printf("add: name=%v, github=%v, notified=%v", name, github, notified)
		}

		if _, ok := notified[github]; ok {
			return
		}
		notified[github] = struct{}{}

		if slack, ok := config.TranslateGithubToSlack[github]; ok {
			notifs[slack] = append(notifs[slack], Notification{
				Type: name,
				Path: repo.Path,
				PR:   pr,
			})
		} else {
			log.Printf("unkown github user '%v' for '%v'", github, name)
		}
	}

	if len(reviewedOwners) < 2 {
		for _, owner := range repo.Owners {
			github := config.Users[owner].Github
			if _, ok := reviewed[github]; !ok {
				addNotification("Pending", github)
			}
		}
	} else {
		addNotification("Ready", pr.Author)
		for _, owner := range repo.Owners {
			addNotification("Ready", config.Users[owner].Github)
		}
	}

	for _, request := range pr.ReviewRequests {
		if _, ok := reviewed[request]; !ok {
			addNotification("Request", request)
		}
	}
}
