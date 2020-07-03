package main

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestBasics(t *testing.T) {
	Debug("[ basics ]==============================================")

	ruleset := MakeRuleset(`
    "pools": { "p1": [ "u1", "u2" ] },
    "ruleset": {
        "r1": [{ "pick": ["p1"] }]
    }`)

	Check(t, ruleset, "r1",
		PR("pr1", "u1"),
		New("u2"), Pending("u2"), Assigned("u2"), Requested(), Ready(false))

	Check(t, ruleset, "r1",
		PR("pr2", "u1").Request("u2"),
		New(), Pending("u2"), Assigned("u2"), Requested(), Ready(false))

	Check(t, ruleset, "r1",
		PR("pr3", "u1").Review("u2", true),
		New(), Pending(), Assigned("u2"), Requested(), Ready(true))

	Check(t, ruleset, "r1",
		PR("pr4", "u1").Review("u2", false),
		New("u2"), Pending("u2"), Assigned("u2"), Requested(), Ready(false))

	Check(t, ruleset, "r1",
		PR("pr5", "u1").Review("u2", false).Review("u2", true),
		New("u2"), Pending("u2"), Assigned("u2"), Requested(), Ready(false))

	Check(t, ruleset, "r1",
		PR("pr6", "u1").Review("u2", true).Review("u2", false),
		New(), Pending(), Assigned("u2"), Requested(), Ready(true))

	Check(t, ruleset, "r1",
		PR("pr7", "u1").Request("u2").Review("u2", true),
		New(), Pending(), Assigned("u2"), Requested(), Ready(true))

	Check(t, ruleset, "r1",
		PR("pr8", "u1").Request("u2").Review("u2", false),
		New(), Pending("u2"), Assigned("u2"), Requested(), Ready(false))
}

func TestCount(t *testing.T) {
	Debug("[ count ]==============================================")

	ruleset := MakeRuleset(`
    "pools": { "p1": [ "u1", "u2", "u3" ] },
    "ruleset": {
        "r1": [{ "pick": ["p1:2"] }],
        "r2": [{ "pick": ["p1:3"] }]
    }`)

	Check(t, ruleset, "r1",
		PR("pr1", "u1"),
		New("u2", "u3"), Pending("u2", "u3"), Assigned("u2", "u3"), Requested(), Ready(false))

	Check(t, ruleset, "r2",
		PR("pr1", "u1"),
		New("u2", "u3"), Pending("u2", "u3"), Assigned("u2", "u3"), Requested(), Ready(false))
}

func TestCond(t *testing.T) {
	Debug("[ if ]==============================================")
	ruleset := MakeRuleset(`
    "pools": {
        "p1": [ "u1" ],
        "p2": [ "u2" ],
        "p3": [ "u3" ]
    },
    "ruleset": {
         "r1": [{ "if":"p1", "pick":["p2"] }, { "pick": ["p3"] }]
    }`)

	Check(t, ruleset, "r1",
		PR("pr1", "u1"),
		New("u2"), Pending("u2"), Assigned("u2"), Requested(), Ready(false))

	Check(t, ruleset, "r1",
		PR("pr2", "u2"),
		New("u3"), Pending("u3"), Assigned("u3"), Requested(), Ready(false))

	Check(t, ruleset, "r1",
		PR("pr3", "u3"),
		New(), Pending(), Assigned(), Requested(), Ready(true))
}

func TestRequested(t *testing.T) {
	Debug("[ Requested ]==============================================")

	ruleset := MakeRuleset(`
    "pools": {
        "p1": [ "u2" ],
        "p2": [ "u2", "u3" ]
    },

    "ruleset": {
        "r1": [{ "pick": ["p1"] }],
        "r2": [{ "pick": ["p2"] }]
    }`)

	Check(t, ruleset, "r1",
		PR("pr1", "u1").Request("u3"),
		New("u2"), Pending("u2"), Assigned("u2"), Requested("u3"), Ready(false))

	Check(t, ruleset, "r1",
		PR("pr2", "u1").Request("u3").Review("u3", true),
		New("u2"), Pending("u2"), Assigned("u2"), Requested(), Ready(false))

	Check(t, ruleset, "r2",
		PR("pr3", "u1").Request("u3"),
		New(), Pending("u3"), Assigned("u3"), Requested(), Ready(false))

	Check(t, ruleset, "r2",
		PR("pr4", "u1").Request("u2").Request("u3"),
		New(), Pending("u2"), Assigned("u2"), Requested("u3"), Ready(false))
}

func MakeRuleset(body string) *Ruleset {
	json := fmt.Sprintf(`
{
    "repos": [{ "path": "gups/repo", "rule": "r1" }],
    "github_to_slack_user": { "u1": "s1", "u2": "s2", "u3": "s3", "u4": "s4", "u5": "s5" },
%v
}`, body)
	return NewRuleset(ParseConfig("test", []byte(json)))
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

func Requested(users ...string) Set {
	return NewSet(users...)
}

func Ready(val bool) bool {
	return val
}

func Check(t *testing.T, ruleset *Ruleset, rule string, pr *PullRequest,
	new, pending, assigned, requested Set, ready bool) {

	Debug("[ %v ]----------------------------------------------", pr.Title)

	rand.Seed(0)
	result := ruleset.Apply(rule, pr)

	CheckSet(t, pr.Title+"-new", new, result.New)
	CheckSet(t, pr.Title+"-pending", pending, result.Pending)
	CheckSet(t, pr.Title+"-assigned", assigned, result.Assigned)
	CheckSet(t, pr.Title+"-requested", requested, result.Requested)

	if ready != result.Ready {
		t.Errorf("%v-ready: val=%v exp=%v", pr.Title, result.Ready, ready)
	}
}

func CheckSet(t *testing.T, title string, exp, val Set) {
	if !val.Equals(exp) {
		t.Errorf("%v: val=%v exp=%v", title, val, exp)
	}
}
