package gups

import (
	"context"
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

type PullRequest struct {
	Number int32
	Title  string
	Author string

	Labels         []string
	Reviews        Reviews
	ReviewRequests []string
}

type Variables struct {
	Owner      string
	Repository string
}

type query struct {
	repository struct {
		pullRequests struct {
			totalCount githubv4.Int
			nodes      []struct {
				number githubv4.Int
				title  githubv4.String
				author struct {
					login githubv4.String
				}

				labels struct {
					totalCount githubv4.Int
					nodes      []struct {
						name githubv4.String
					}
				} `graphql:"labels(first: $labelCount)"`

				reviews struct {
					totalCount githubv4.Int
					nodes      []struct {
						state       githubv4.String
						submittedAt githubv4.DateTime
						author      struct {
							login githubv4.String
						}
					}
				} `graphql:"reviews(first: $reviewCount)"`

				reviewRequests struct {
					totalCount githubv4.Int
					nodes      []struct {
						user struct {
							login githubv4.String
						} `graphql:"... on User"`
					}
				} `graphql:"reviewRequests(first: $reviewReqCount)"`
			}
		} `graphql:"pullRequests(states: OPEN, first: $prCount)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

func QueryPullRequests(ctx context.Context, client *githubv4.Client, vars Variables) ([]PullRequest, error) {
	variables := map[string]interface{}{
		"owner":          githubv4.String(vars.Owner),
		"repo":           githubv4.String(vars.Repository),
		"prCount":        githubv4.Int(prCount),
		"labelCount":     githubv4.Int(labelCount),
		"reviewCount":    githubv4.Int(reviewCount),
		"reviewReqCount": githubv4.Int(reviewReqCount),
	}

	var raw query
	if err := client.Query(ctx, &raw, variables); err != nil {
		return nil, err
	}

	if count := raw.repository.pullRequests.totalCount; count > prCount {
		log.Printf("Truncated PR result for %v/%v (%v > %v)",
			vars.Owner, vars.Repository, count, prCount)
	}

	var pullRequests []PullRequest
	for _, rawPullRequest := range raw.repository.pullRequests.nodes {

		pullRequest := PullRequest{
			Number: int32(rawPullRequest.number),
			Title:  string(rawPullRequest.title),
			Author: string(rawPullRequest.author.login),
		}

		if count := rawPullRequest.labels.totalCount; count > labelCount {
			log.Printf("Truncated label result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		for _, rawLabels := range rawPullRequest.labels.nodes {
			pullRequest.Labels = append(pullRequest.Labels, string(rawLabels.name))
		}

		if count := rawPullRequest.reviews.totalCount; count > reviewCount {
			log.Printf("Truncated reviews result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		for _, rawReview := range rawPullRequest.reviews.nodes {
			review := Review{
				Author: string(rawReview.author.login),
				State:  string(rawReview.state),
				Time:   rawReview.submittedAt.Time,
			}

			pullRequest.Reviews = append(pullRequest.Reviews, review)
		}

		sort.Sort(pullRequest.Reviews)

		if count := rawPullRequest.reviewRequests.totalCount; count > reviewReqCount {
			log.Printf("Truncated review request result for %v/%v PR %v (%v > %v)",
				vars.Owner, vars.Repository, pullRequest.Number, count, prCount)
		}

		for _, reviewRequests := range rawPullRequest.reviewRequests.nodes {
			pullRequest.ReviewRequests =
				append(pullRequest.ReviewRequests, string(reviewRequests.user.login))
		}

		pullRequests = append(pullRequests, pullRequest)
	}

	return pullRequests, nil
}
