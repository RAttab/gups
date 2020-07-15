package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
	"os"
	"sort"
	"time"
)

type GithubClient githubv4.Client

func (client *GithubClient) cast() *githubv4.Client {
	return (*githubv4.Client)(client)
}

func ConnectGithub() *GithubClient {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	return (*GithubClient)(githubv4.NewClient(httpClient))
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
	id     string
	Number int32
	Title  string
	Author string
	Age    Age

	Labels         Set
	Reviews        Reviews
	ReviewRequests Set
}

func (pr PullRequest) Reviewed() Set {
	reviewed := NewSet()
	pending := NewSet()

	for _, review := range pr.Reviews {
		if reviewed.Test(review.Author) || pending.Test(review.Author) {
			continue
		}

		if review.State == "APPROVED" {
			reviewed.Put(review.Author)
		} else {
			pending.Put(review.Author)
		}
	}

	return reviewed
}

type Variables struct {
	Owner      string
	Repository string
}

type queryPR struct {
	Repository struct {
		PullRequests struct {
			TotalCount githubv4.Int
			Nodes      []struct {
				Id        githubv4.String
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

func (client *GithubClient) QueryPullRequests(ctx context.Context, vars Variables) []*PullRequest {
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
		Debug("Vars: %v", string(bytes))
	}

	var raw queryPR
	if err := client.cast().Query(ctx, &raw, variables); err != nil {
		Fatal("unable to query github: %v", err)
		return nil
	}

	if count := raw.Repository.PullRequests.TotalCount; count > prCount {
		Warning("Truncated PR result for %v/%v (%v > %v)",
			vars.Owner, vars.Repository, count, prCount)
	}

	var pullRequests []*PullRequest
	for _, rawPullRequest := range raw.Repository.PullRequests.Nodes {

		pullRequest := &PullRequest{
			id:     string(rawPullRequest.Id),
			Number: int32(rawPullRequest.Number),
			Title:  string(rawPullRequest.Title),
			Author: string(rawPullRequest.Author.Login),
			Age:    NewAge(rawPullRequest.CreatedAt.Time),
		}

		if count := rawPullRequest.Labels.TotalCount; count > labelCount {
			Warning("Truncated label result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		pullRequest.Labels = NewSet()
		for _, rawLabels := range rawPullRequest.Labels.Nodes {
			pullRequest.Labels.Put(string(rawLabels.Name))
		}

		if count := rawPullRequest.Reviews.TotalCount; count > reviewCount {
			Warning("Truncated reviews result for %v/%v PR %v (%v > %v)",
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
			Warning("Truncated review request result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		pullRequest.ReviewRequests = NewSet()
		for _, reviewRequests := range rawPullRequest.ReviewRequests.Nodes {
			pullRequest.ReviewRequests.Put(string(reviewRequests.RequestedReviewer.User.Login))
		}

		pullRequests = append(pullRequests, pullRequest)
	}

	if false { // DEBUG
		bytes, _ := json.MarshalIndent(pullRequests, "", "    ")
		Debug("PullRequests: %v", string(bytes))
	}

	return pullRequests
}

var userId map[string]githubv4.ID = make(map[string]githubv4.ID)

func (client GithubClient) userId(ctx context.Context, user string) (githubv4.ID, error) {
	if id, ok := userId[user]; ok {
		return id, nil
	}

	var raw struct {
		User struct {
			Id githubv4.ID
		} `graphql:"user(login: $user)"`
	}

	vars := map[string]interface{}{
		"user": githubv4.String(user),
	}

	if err := client.cast().Query(ctx, &raw, vars); err != nil {
		return nil, err
	}

	id := raw.User.Id
	userId[user] = id
	return id, nil
}

func (client GithubClient) RequestReview(
	ctx context.Context, pr *PullRequest, users []string, dryRun bool) {

	if len(users) == 0 {
		return
	}

	var ids []githubv4.ID
	for _, user := range users {
		id, err := client.userId(ctx, user)
		if err != nil {
			Fatal("unable to translate user '%v' to github id: %v", user, err)
		}

		ids = append(ids, id)
	}

	var raw struct {
		RequestReviews struct {
			ClientMutationId githubv4.String
		} `graphql:"requestReviews(input: $input)"`
	}

	input := githubv4.RequestReviewsInput{
		PullRequestID: githubv4.ID(pr.id),
		UserIDs:       &ids,
	}

	if dryRun {
		return
	}

	if err := client.cast().Mutate(ctx, &raw, input, nil); err != nil {
		Fatal("unable to request reviews for '%v -> %v' on PR '%v': %v",
			users, ids, pr.Number, err)
	}
}
