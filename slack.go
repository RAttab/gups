package gups

type Notification struct {
	Path        string
	PullRequest int32
	Message     string
}

type Notifications map[string][]Notification
