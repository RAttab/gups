package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/nlopes/slack"
)

const IconURL = "https://github.com/RAttab/gups/blob/master/gups.png?raw=true"
const MsgLimit = 40000
const TruncateFooter = "\n..."

type Category int32

const (
	CategoryReady Category = iota
	CategoryPending
	CategoryRequested
	CategoryOpen
)

func (cat Category) String() string {
	switch cat {
	case CategoryReady:
		return "*Ready* (pull requests ready to merge)"
	case CategoryPending:
		return "*Pending* (pull requests for repos you own awaiting review)"
	case CategoryRequested:
		return "*Requested* (pull requests that you've been asked to review)"
	case CategoryOpen:
		return "*Open* (pull requests awaiting reviews that you opened)"
	}
	return "*Whoops* (something went horribly wrong)"
}

type Notification struct {
	Category Category
	Path     string
	PR       *PullRequest
}

type Notifications []Notification

func (n Notifications) Len() int {
	return len(n)
}

func (n Notifications) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n Notifications) Less(i, j int) bool {
	if n[i].Category < n[j].Category {
		return true
	} else if n[i].Category == n[j].Category {
		return n[i].PR.Age.Delta < n[j].PR.Age.Delta
	}

	return false
}

func ConnectSlack() (*slack.Client, error) {
	client := slack.New(os.Getenv("SLACK_TOKEN"))
	if _, err := client.AuthTest(); err != nil {
		return nil, err
	}
	return client, nil
}

type SlackUsers map[string]string

func SlackMapUsers(client *slack.Client, config *Config) SlackUsers {
	users := make(SlackUsers)

	slackUsers, err := client.GetUsers()
	if err != nil {
		log.Fatalf("unable to get slack users: %v", err)
	}

	slackIds := make(map[string]string)
	for _, user := range slackUsers {
		slackIds[user.Name] = user.ID
	}

	for github, slack := range config.Users {
		if id, ok := slackIds[slack]; ok {
			users[github] = id
		} else {
			log.Printf("unknown slack user '%v'", slack)
		}
	}

	return users
}

func SlackDumpUsers(client *slack.Client) {
	users, err := client.GetUsers()
	if err != nil {
		log.Fatalf("unable to get slack users: %v", err)
	}

	for _, user := range users {
		fmt.Printf("%v: %v (%v)\n", user.ID, user.Name, user.RealName)
	}
}

func NotifySlack(client *slack.Client, user string, notif Notifications, dryRun bool) error {
	sort.Sort(notif)

	if false { // DEBUG
		bytes, _ := json.MarshalIndent(notif, "", "    ")
		log.Printf("Notifications: %v", string(bytes))
	}

	var currCategory Category = -1
	buffer := bytes.Buffer{}

	for _, entry := range notif {

		if currCategory != entry.Category {
			currCategory = entry.Category
			buffer.WriteString(fmt.Sprintf("%v:\n", currCategory))
		}

		Line := fmt.Sprintf("[%v] *<https://github.com/%v/pull/%v|%v/%v>*: %v\n",
			entry.PR.Age,
			entry.Path, entry.PR.Number,
			entry.Path, entry.PR.Number,
			entry.PR.Title)

		if buffer.Len()+len(Line) <= MsgLimit {
			buffer.WriteString(Line)
		} else {
			buffer.WriteString(TruncateFooter)
			log.Printf("truncated")
			break
		}
	}

	if dryRun {
		log.Printf("%v", buffer.String())

	} else {
		_, _, err := client.PostMessage(user,
			slack.MsgOptionUsername("GUPS"),
			slack.MsgOptionAsUser(false),
			slack.MsgOptionText(buffer.String(), false),
			slack.MsgOptionIconURL(IconURL),
			slack.MsgOptionDisableLinkUnfurl())

		if err != nil {
			return err
		}
	}

	return nil
}
