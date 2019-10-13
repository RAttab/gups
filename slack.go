package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func NotifySlack(user string, notif Notifications) error {
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

	message := map[string]string{
		"access_token": os.Getenv("SLACK_TOKEN"),
		"username":     "Gups",
		"channel":      user,
		"text":         buffer.String(),
	}

	fmt.Printf("slack: %v", message)

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := http.Post("https://slack.com/api/chat.postMessage", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("Response: %v", resp)
	return nil
}
