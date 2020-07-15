package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"sort"
	"time"
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

	rand.Seed(time.Now().UnixNano())

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

			if !result.New.Empty() {
				Info("<%v> review request: %v", pr.Number, result.New)
				requests := pr.ReviewRequests.Union(result.New).ToArray()
				githubClient.RequestReview(context.TODO(), pr, requests, *dryRun)
			}

			for user, _ := range result.New {
				notifs.Add(CategoryAssigned, user, repo.Path, pr)
			}

			if *full {
				if result.Ready {
					if ruleset.KnownUser(pr.Author) {
						notifs.Add(CategoryReady, pr.Author, repo.Path, pr)
					}
				} else {
					if ruleset.KnownUser(pr.Author) {
						notifs.Add(CategoryOpen, pr.Author, repo.Path, pr)
					}
					for user, _ := range result.Pending.Difference(result.Assigned) {
						notifs.Add(CategoryPending, user, repo.Path, pr)
					}
				}
				for user, _ := range result.Requested {
					notifs.Add(CategoryRequested, user, repo.Path, pr)
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

	stats(notifs)
}

type Stat struct {
	key string
	val int
}

type Stats []Stat

func (s Stats) Len() int           { return len(s) }
func (s Stats) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Stats) Less(i, j int) bool { return s[i].val > s[j].val }

func NewStats(m map[string]int) Stats {
	var stats Stats
	for key, val := range m {
		stats = append(stats, Stat{key, val})
	}
	sort.Sort(stats)
	return stats
}

func stats(notifs UserNotifications) {
	perRepo := make(map[string]int)
	perUser := make(map[string]int)

	for user, list := range notifs {
		for _, item := range list {
			if item.Category != CategoryAssigned && item.Category != CategoryPending {
				continue
			}

			perUser[user] += 1
			perRepo[item.Path] += 1
		}
	}

	Info("pending reviews per repo:")
	for _, stat := range NewStats(perRepo) {
		Info("  %2d %v", stat.val, stat.key)
	}

	Info("pending reviews per user:")
	for _, stat := range NewStats(perUser) {
		Info("  %2d %v", stat.val, stat.key)
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
