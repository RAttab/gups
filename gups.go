package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"
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
		log.Printf("DRY RUN: no messages will be posted to slack")
	}

	slackClient, err := ConnectSlack()
	if err != nil {
		if *dryRun {
			log.Printf("unable to connect to slack: %v", err)
		} else {
			log.Fatalf("unable to connect to slack: %v", err)
		}
	}

	if *dumpUsers {
		SlackDumpUsers(slackClient)
		return
	}

	path := os.Getenv("CONFIG")
	config, err := ReadConfig(path, filepath.Ext(path))
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

	perUser, perRepo := calcStats(notifs)
	logStats("Users", perUser)
	logStats("Repo", perRepo)
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
		if owner == pr.Author {
			reviewedOwners[owner] = struct{}{}
		}
	}

	notified := make(map[string]struct{})
	skipped := make(map[string]struct{})

	addNotification := func(cat Category, user string) {
		if false { // debug
			log.Printf("add: cat=%v, github=%v, notified=%v", cat, user, notified)
		}

		if _, ok := notified[user]; ok {
			return
		}

		if _, ok := config.Users[user]; !ok {
			skipped[user] = struct{}{}
			return
		}

		notified[user] = struct{}{}
		notifs[user] = append(notifs[user], Notification{
			Category: cat,
			Path:     repo.Path,
			PR:       pr,
		})
	}

	if len(reviewedOwners) < 2 {
		addNotification(CategoryOpen, pr.Author)
		for _, owner := range repo.Owners {
			if _, ok := reviewed[owner]; !ok {
				addNotification(CategoryPending, owner)
			}
		}
	} else {
		addNotification(CategoryReady, pr.Author)
		for _, owner := range repo.Owners {
			addNotification(CategoryReady, owner)
		}
	}

	for _, request := range pr.ReviewRequests {
		if _, ok := reviewed[request]; !ok {
			addNotification(CategoryRequested, request)
		}
	}

	for user, _ := range skipped {
		log.Printf("skipped user %v", user)
	}
}

type stat struct {
	string
	int
}
type stats []stat

func (stats stats) Len() int {
	return len(stats)
}

func (stats stats) Swap(i, j int) {
	stats[i], stats[j] = stats[j], stats[i]
}

func (stats stats) Less(i, j int) bool {
	if stats[i].int < stats[j].int {
		return true
	} else if stats[i].int == stats[j].int {
		return stats[i].string < stats[j].string
	}
	return false
}

func calcStats(notifs map[string][]Notification) (perUser, perRepo stats) {
	repoMap := make(map[string]map[int32]struct{})

	for user, todo := range notifs {
		if len(todo) > 0 {
			perUser = append(perUser, stat{user, len(todo)})
		}

		for _, notif := range todo {
			if _, ok := repoMap[notif.Path]; !ok {
				repoMap[notif.Path] = make(map[int32]struct{})
			}
			repoMap[notif.Path][notif.PR.Number] = struct{}{}
		}
	}

	for repo, todo := range repoMap {
		if len(todo) > 0 {
			perRepo = append(perRepo, stat{repo, len(todo)})
		}
	}

	return
}

func logStats(title string, stats stats) {
	sort.Sort(stats)

	log.Printf("%v Stats:", title)
	if len(stats) == 0 {
		log.Printf("  Everybody is on top of their shit; Gupi-chan is disapointed.")
		return
	}

	for _, stat := range stats {
		log.Printf("  %2d %v", stat.int, stat.string)
	}
}
