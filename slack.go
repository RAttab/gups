package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

type Notification struct {
	Path        string
	PullRequest int32
	Title       string
	Type        string
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
			if n[i].PullRequest > n[j].PullRequest {
				return true
			}
		}
	}
	return false
}

func ConnectSlack(ctx context.Context) (*http.Client, error) {
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SLACK_CLIENT_ID"),
		ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
		Scopes:       []string{"chat:write:bot"},
		TokenURL:     "https://slack.com/api/oauth.access",
	}
	return config.Client(ctx), nil
}

func NotifySlack(client *http.Client, user string, notif Notifications) error {
	sort.Sort(notif)

	if false { // DEBUG
		bytes, _ := json.MarshalIndent(notif, "", "    ")
		log.Printf("Notifications: %v", string(bytes))
	}

	builder := strings.Builder{}
	for _, entry := range notif {
		line := fmt.Sprintf("[] %v: [%v](%v/pull/%v)\n",
			entry.Type, entry.Title, entry.Path, entry.PullRequest)

		log.Printf("line: %v", line)

		builder.WriteString(line)
	}

	message := struct {
		channel string
		text    string
	}{
		channel: user,
		text:    builder.String(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if true { // DEBUG
		log.Printf("Slack Message:\n %v", string(data))
	}

	client.Post("https://slack.com/api/chat.postMessage", "application/json", bytes.NewReader(data))

	return nil
}
