package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
)

var dumpUsers = flag.Bool("dump-users", false, "dumps the slack users and exits")
var dryRun = flag.Bool("dry-run", false, "print slack notifications without sending them")

func main() {
	flag.Parse()

	if false { //debug
		log.Printf("CONFIG: %v", os.Getenv("CONFIG"))
		log.Printf("GITHUB_TOKEN: %v", os.Getenv("GITHUB_TOKEN"))
		log.Printf("SLACK_TOKEN: %v", os.Getenv("SLACK_TOKEN"))
	}

	if *dryRun {
		log.Printf("dry run...")
	}

	slackClient, err := ConnectSlack()
	if *dumpUsers {
		SlackDumpUsers(slackClient)
		return
	}

	path := os.Getenv("CONFIG")
	config, err := ReadConfig(path)
	if err != nil {
		log.Fatalf("unable to read config '%v': %v", path, err)
	}

	githubClient := ConnectGithub()
	if err != nil {
		log.Fatalf("unable to connect to slack: %v", err)
	}

	notifs := make(map[string][]Notification)
	slackUsers := SlackMapUsers(slackClient, config)

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

	index := 0
	for githubUser, notif := range notifs {
		log.Printf("[%v/%v] notifying %v...", index+1, len(notifs), githubUser)

		if slackUser, ok := slackUsers[githubUser]; ok {
			if err := NotifySlack(slackClient, slackUser, notif, *dryRun); err != nil {
				log.Fatalf("Unable to notify slack: %v", err)
			}
		} else {
			log.Printf("unconfigured github user '%v'", githubUser)
		}

		index++
	}
}

func check(repo *Repo, pr *PullRequest, config *Config, notifs map[string][]Notification) {
	for _, label := range pr.Labels {
		for _, toSkip := range config.SkipLabels {
			if strings.ToLower(label) == toSkip {
				return
			}
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
		if _, ok := reviewed[owner]; ok {
			reviewedOwners[owner] = struct{}{}
		}
	}

	notified := make(map[string]struct{})

	addNotification := func(name, user string) {
		if false { // debug
			log.Printf("add: name=%v, github=%v, notified=%v", name, user, notified)
		}

		if _, ok := notified[user]; ok {
			return
		}

		notified[user] = struct{}{}
		notifs[user] = append(notifs[user], Notification{
			Type: name,
			Path: repo.Path,
			PR:   pr,
		})
	}

	if len(reviewedOwners) < 2 {
		for _, owner := range repo.Owners {
			if _, ok := reviewed[owner]; !ok {
				addNotification("Pending", owner)
			}
		}
	} else {
		addNotification("Ready", pr.Author)
		for _, owner := range repo.Owners {
			addNotification("Ready", owner)
		}
	}

	for _, request := range pr.ReviewRequests {
		if _, ok := reviewed[request]; !ok {
			addNotification("Request", request)
		}
	}
}
