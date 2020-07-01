package main

import (
	"fmt"
	"testing"
)

func TestBasics(t *testing.T) {
	rules := MakeRules(`
    "pools": { "p1": [ "u1", "u2" ] },
    "rules": { "r1": { "_": ["p1"] } }`)

	Check(t, rules, "r1",
		PR("pr1", "u1"),
		New("u2"), Pending("u2"), Assigned("u2"))

	Check(t, rules, "r1",
		PR("pr2", "u1").Request("u2"),
		New(), Pending("u2"), Assigned("u2"))

	Check(t, rules, "r1",
		PR("pr3", "u1").Review("u2", true),
		New(), Pending(), Assigned("u2"))

	Check(t, rules, "r1",
		PR("pr4", "u1").Review("u2", false),
		New("u2"), Pending("u2"), Assigned("u2"))

	Check(t, rules, "r1",
		PR("pr5", "u1").Request("u2").Review("u2", true),
		New(), Pending(), Assigned("u2"))

	Check(t, rules, "r1",
		PR("pr6", "u1").Request("u2").Review("u2", false),
		New(), Pending("u2"), Assigned("u2"))
}

func TestCount(t *testing.T) {
	rules := MakeRules(`
    "pools": { "p1": [ "u1", "u2", "u3" ] },
    "rules": {
        "r1": { "_": ["p1:2"] },
        "r2": { "_": ["p1:3"] }
    }`)

	Check(t, rules, "r1",
		PR("pr1", "u1"),
		New("u2", "u3"), Pending("u2", "u3"), Assigned("u2", "u3"))

	Check(t, rules, "r2",
		PR("pr1", "u1"),
		New("u2", "u3"), Pending("u2", "u3"), Assigned("u2", "u3"))
}

func TestCond(t *testing.T) {
	rules := MakeRules(`
    "pools": {
        "p1": [ "u1" ],
        "p2": [ "u2" ],
        "p3": [ "u3" ]
    },

    "rules": {
        "r1": { "p1": ["p2"], "_": ["p3"] }
    }`)

	Check(t, rules, "r1",
		PR("pr1", "u1"),
		New("u2"), Pending("u2"), Assigned("u2"))

	Check(t, rules, "r1",
		PR("pr2", "u2"),
		New("u3"), Pending("u3"), Assigned("u3"))

	Check(t, rules, "r1",
		PR("pr2", "u3"),
		New(), Pending(), Assigned())
}

func MakeRules(body string) *Rules {
	json := fmt.Sprintf(`
{
    "repos": [{ "path": "gups/repo", "rule": "r1" }],
    "github_to_slack_user": { "u1": "s1", "u2": "s2", "u3": "s3", "u4": "s4", "u5": "s5" },
%v
}`, body)
	return NewRules(ParseConfig("test", []byte(json)))
}

func PR(title, user string) *PullRequest {
	return &PullRequest{Title: title, Author: user, ReviewRequests: NewSet()}
}

func (pr *PullRequest) Review(user string, approved bool) *PullRequest {
	state := "FUCKOFF"
	if approved {
		state = "APPROVED"
	}

	pr.Reviews = append(pr.Reviews, Review{Author: user, State: state})
	return pr
}

func (pr *PullRequest) Request(user string) *PullRequest {
	pr.ReviewRequests.Put(user)
	return pr
}

func New(users ...string) Set {
	return NewSet(users...)
}

func Pending(users ...string) Set {
	return NewSet(users...)
}

func Assigned(users ...string) Set {
	return NewSet(users...)
}

func Check(t *testing.T, rules *Rules, rule string, pr *PullRequest, new, pending, assigned Set) {
	result := rules.Apply(rule, pr)

	CheckSet(t, pr.Title+"-new", new, result.New)
	CheckSet(t, pr.Title+"-pending", pending, result.Pending)
	CheckSet(t, pr.Title+"-assigned", assigned, result.Assigned)
}

func CheckSet(t *testing.T, title string, exp, val Set) {
	if !val.Equals(exp) {
		t.Errorf("%v: val=%v exp=%v", title, val, exp)
	}
}
