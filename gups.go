package main

import (
	"context"
	"flag"
	"log"
	"os"
)

var full = flag.Bool("full", false, "notify with full summary")
var dumpUsers = flag.Bool("dump-users", false, "dumps the slack users and exits")
var dryRun = flag.Bool("dry-run", false, "print slack notifications without sending them")

func main() {
	flag.Parse()

	if false { //debug
		Debug("CONFIG: %v", os.Getenv("CONFIG"))
		Debug("GITHUB_TOKEN: %v", os.Getenv("GITHUB_TOKEN"))
		Debug("SLACK_TOKEN: %v", os.Getenv("SLACK_TOKEN"))
	}

	if *dryRun {
		Info("DRY RUN: no messages will be posted to slack")
	}

	slackClient, err := ConnectSlack()
	if err != nil {
		if *dryRun {
			Info("unable to connect to slack: %v", err)
		} else {
			Fatal("unable to connect to slack: %v", err)
		}
	}

	if *dumpUsers {
		SlackDumpUsers(slackClient)
		return
	}

	path := os.Getenv("CONFIG")
	config := ReadConfig(path)

	githubClient := ConnectGithub()
	if err != nil {
		Fatal("unable to connect to slack: %v", err)
	}

	ruleset := NewRuleset(config)

	notifs := make(UserNotifications)
	slackUsers := SlackMapUsers(slackClient, config)

	for index, repo := range config.Repos {
		Info("[%v/%v] processing %v...", index+1, len(config.Repos), repo.Path)

		vars := PathToVariables(repo.Path)
		for _, pr := range githubClient.QueryPullRequests(context.TODO(), vars) {
			result := ruleset.Apply(repo.Rule, pr)

			githubClient.RequestReview(context.TODO(), pr, result.New.ToArray())
			for user, _ := range result.New {
				notifs.Add(CategoryAssigned, slackUsers[user], repo.Path, pr)
			}

			if *full {
				if result.Ready {
					notifs.Add(CategoryReady, slackUsers[pr.Author], repo.Path, pr)
				} else {
					notifs.Add(CategoryOpen, slackUsers[pr.Author], repo.Path, pr)
					for user, _ := range result.Pending {
						notifs.Add(CategoryReady, slackUsers[user], repo.Path, pr)
					}
				}
				for user, _ := range result.Requested {
					notifs.Add(CategoryRequested, slackUsers[user], repo.Path, pr)
				}
			}
		}
	}

	index := 0
	for githubUser, notif := range notifs {
		Info("[%v/%v] notifying %v...", index+1, len(notifs), githubUser)

		if slackUser, ok := slackUsers[githubUser]; ok {
			if err := NotifySlack(slackClient, slackUser, notif, *dryRun); err != nil {
				Fatal("Unable to notify slack: %v", err)
			}
		} else {
			Warning("unconfigured github user '%v'", githubUser)
		}

		index++
	}
}

func Fatal(fmt string, args ...interface{}) {
	log.Fatalf("<FATAL> "+fmt, args...)
}

func Warning(fmt string, args ...interface{}) {
	log.Printf("<WARN> "+fmt, args...)
}

func Info(fmt string, args ...interface{}) {
	log.Printf("<INFO> "+fmt, args...)
}

func Debug(fmt string, args ...interface{}) {
	log.Printf("<DEBUG> "+fmt, args...)
}
