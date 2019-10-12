package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"os"
	"strings"
)

type Notification struct {
	Path        string
	PullRequest int32
	Title       string
	Message     string
}

type Notifications map[string][]Notification

func ConnectSlack(ctx context.Context) (*http.Client, error) {
	config := &clientcredentials.Config{
		ClientID:     os.Getenv("SLACK_CLIENT_ID"),
		ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
		Scopes:       []string{"chat:write:bot"},
		TokenURL:     "https://slack.com/api/oauth.access",
	}
	return config.Client(ctx), nil
}

func NotifySlack(client *http.Client, notifs Notifications) error {
	for user, notif := range notifs {

		builder := strings.Builder{}
		for _, entry := range notif {
			builder.WriteString(fmt.Sprintf("[] %v: [%v](%v/pull/%v)\n",
				entry.Message, entry.Title, entry.Path, entry.PullRequest))
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
		client.Post("https://slack.com/api/chat.postMessage", "application/json", bytes.NewReader(data))
	}

	return nil
}
