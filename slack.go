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

	currType := ""
	buffer := bytes.Buffer{}

	for _, entry := range notif {

		if currType != entry.Type {
			currType = entry.Type
			buffer.WriteString(fmt.Sprintf("**%v:**\n", currType))
		}

		buffer.WriteString(fmt.Sprintf("-[**%v/%v**](%v/pull/%v) (%v): %v\n",
			entry.Path, entry.PR.Number, entry.Path, entry.PR.Number, entry.PR.Age, entry.PR.Title))
	}

	if true {
		log.Printf("buffer: %v", buffer.String())
	}

	message := map[string]string{
		"channel": user,
		"text":    buffer.String(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	client.Post("https://slack.com/api/chat.postMessage", "application/json", bytes.NewReader(data))

	return nil
}
