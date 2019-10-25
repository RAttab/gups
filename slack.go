package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/nlopes/slack"
)

type Notification struct {
	Type string
	Path string
	PR   *PullRequest
}

type Notifications []Notification

func (n Notifications) Len() int {
	return len(n)
}

func (n Notifications) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n Notifications) Less(i, j int) bool {
	if n[i].Type > n[j].Type {
		return true
	} else if n[i].Type == n[j].Type {
		if n[i].Path > n[j].Path {
			return true
		} else if n[i].Path == n[j].Path {
			if n[i].PR.Number > n[j].PR.Number {
				return true
			}
		}
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

func SlackTranslateUsers(client *slack.Client, config *Config) error {
	for _, user := range config.Users {
		if !strings.Contains(user.Slack, "@") {
			continue
		}

		slackUser, err := client.GetUserByEmail(user.Slack)
		if err != nil {
			return fmt.Errorf("failed to get '%v' slack user: %v", user.Slack, err)
		}

		user.Slack = slackUser.ID
	}

	return nil
}

func NotifySlack(client *slack.Client, user string, notif Notifications) error {
	sort.Sort(notif)

	if false { // DEBUG
		bytes, _ := json.MarshalIndent(notif, "", "    ")
		log.Printf("Notifications: %v", string(bytes))
	}

	currType := ""
	buffer := bytes.Buffer{}

	for _, entry := range notif {

		if currType != entry.Type {
			currType = entry.Type
			buffer.WriteString(fmt.Sprintf("**%v:**\n", currType))
		}

		buffer.WriteString(fmt.Sprintf("-[**%v/%v**](https://github.com/%v/pull/%v) (%v): %v\n",
			entry.Path, entry.PR.Number, entry.Path, entry.PR.Number, entry.PR.Age, entry.PR.Title))
	}

	if true {
		log.Printf("buffer: %v", buffer.String())
	}

	a, b, err := client.PostMessage(user,
		slack.MsgOptionUsername("GUPS"),
		slack.MsgOptionAsUser(false),
		slack.MsgOptionText(buffer.String(), false),
		slack.MsgOptionDisableLinkUnfurl())

	if err != nil {
		return err
	}

	log.Printf("slack.reply: %v, %v", a, b)
	return nil
}
