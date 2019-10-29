package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"log"
	"os"
	"sort"
	"time"
)

func ConnectGithub() *githubv4.Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	return githubv4.NewClient(httpClient)
}

const (
	prCount        = 100
	labelCount     = 50
	reviewCount    = 50
	reviewReqCount = 50
)

type Review struct {
	Author string
	State  string
	Time   time.Time
}

type Reviews []Review

func (r Reviews) Len() int {
	return len(r)
}

func (r Reviews) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Reviews) Less(i, j int) bool {
	return r[i].Time.After(r[j].Time)
}

type Age struct {
	Delta time.Duration
}

func NewAge(ts time.Time) Age {
	return Age{Delta: time.Now().Sub(ts)}
}

func (age Age) String() string {
	if years := age.Delta / (time.Hour * 24 * 365); years >= 1 {
		return fmt.Sprintf("%vy", int64(years))
	} else if days := age.Delta / (time.Hour * 24); days >= 1 {
		return fmt.Sprintf("%vd", int64(days))
	} else if hours := age.Delta / time.Hour; hours >= 1 {
		return fmt.Sprintf("%vh", int64(hours))
	}
	return "1h"
}

type PullRequest struct {
	Number int32
	Title  string
	Author string
	Age    Age

	Labels         []string
	Reviews        Reviews
	ReviewRequests []string
}

type Variables struct {
	Owner      string
	Repository string
}

type query struct {
	Repository struct {
		PullRequests struct {
			TotalCount githubv4.Int
			Nodes      []struct {
				Number    githubv4.Int
				CreatedAt githubv4.DateTime
				Title     githubv4.String
				Author    struct {
					Login githubv4.String
				}

				Labels struct {
					TotalCount githubv4.Int
					Nodes      []struct {
						Name githubv4.String
					}
				} `graphql:"labels(first: $labelCount)"`

				Reviews struct {
					TotalCount githubv4.Int
					Nodes      []struct {
						State       githubv4.String
						SubmittedAt githubv4.DateTime
						Author      struct {
							Login githubv4.String
						}
					}
				} `graphql:"reviews(first: $reviewCount)"`

				ReviewRequests struct {
					TotalCount githubv4.Int
					Nodes      []struct {
						RequestedReviewer struct {
							User struct {
								Login githubv4.String
							} `graphql:"... on User"`
						}
					}
				} `graphql:"reviewRequests(first: $reviewReqCount)"`
			}
		} `graphql:"pullRequests(states: OPEN, first: $prCount)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func QueryPullRequests(ctx context.Context, client *githubv4.Client, vars Variables) ([]*PullRequest, error) {
	variables := map[string]interface{}{
		"owner":          githubv4.String(vars.Owner),
		"repo":           githubv4.String(vars.Repository),
		"prCount":        githubv4.Int(prCount),
		"labelCount":     githubv4.Int(labelCount),
		"reviewCount":    githubv4.Int(reviewCount),
		"reviewReqCount": githubv4.Int(reviewReqCount),
	}

	if false { // DEBUG
		bytes, _ := json.MarshalIndent(variables, "", "    ")
		log.Printf("Vars: %v", string(bytes))
	}

	var raw query
	if err := client.Query(ctx, &raw, variables); err != nil {
		return nil, err
	}

	if count := raw.Repository.PullRequests.TotalCount; count > prCount {
		log.Printf("Truncated PR result for %v/%v (%v > %v)",
			vars.Owner, vars.Repository, count, prCount)
	}

	var pullRequests []*PullRequest
	for _, rawPullRequest := range raw.Repository.PullRequests.Nodes {

		pullRequest := &PullRequest{
			Number: int32(rawPullRequest.Number),
			Title:  string(rawPullRequest.Title),
			Author: string(rawPullRequest.Author.Login),
			Age:    NewAge(rawPullRequest.CreatedAt.Time),
		}

		if count := rawPullRequest.Labels.TotalCount; count > labelCount {
			log.Printf("Truncated label result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		for _, rawLabels := range rawPullRequest.Labels.Nodes {
			pullRequest.Labels = append(pullRequest.Labels, string(rawLabels.Name))
		}

		if count := rawPullRequest.Reviews.TotalCount; count > reviewCount {
			log.Printf("Truncated reviews result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		for _, rawReview := range rawPullRequest.Reviews.Nodes {
			review := Review{
				Author: string(rawReview.Author.Login),
				State:  string(rawReview.State),
				Time:   rawReview.SubmittedAt.Time,
			}

			pullRequest.Reviews = append(pullRequest.Reviews, review)
		}

		sort.Sort(pullRequest.Reviews)

		if count := rawPullRequest.ReviewRequests.TotalCount; count > reviewReqCount {
			log.Printf("Truncated review request result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		for _, reviewRequests := range rawPullRequest.ReviewRequests.Nodes {
			pullRequest.ReviewRequests =
				append(pullRequest.ReviewRequests, string(reviewRequests.RequestedReviewer.User.Login))
		}

		pullRequests = append(pullRequests, pullRequest)
	}

	if false { // DEBUG
		bytes, _ := json.MarshalIndent(pullRequests, "", "    ")
		log.Printf("PullRequests: %v", string(bytes))
	}

	return pullRequests, nil
}
