package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Pick struct {
	Pool  string
	Count int
}

func (pick *Pick) String() string {
	return fmt.Sprintf("%v:%v", pick.Pool, pick.Count)
}

func (pick *Pick) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	items := strings.Split(raw, ":")
	pick.Pool = items[0]

	pick.Count = 1
	if len(items) > 1 {
		if val, err := strconv.ParseInt(items[1], 10, 64); err != nil {
			Fatal("malformed pick '%v': %v", raw, err)
		} else if val < 1 {
			Fatal("malformed pick '%v': '%v' must be >= 1", items[1], val)
		} else {
			pick.Count = int(val)
		}
	}

	return nil
}

type Rule struct {
	If   string `json:"if"`
	Pick []Pick `json:"pick"`
}

func (rule *Rule) HasIf() bool {
	return rule.If != ""
}

type Rules []Rule

type Ruleset struct {
	users   Set
	pools   map[string]Set
	ruleset map[string]Rules

	skipLabels Set
}

func NewRuleset(config *Config) *Ruleset {
	ruleset := &Ruleset{
		users:      NewSet(),
		pools:      make(map[string]Set),
		ruleset:    config.Ruleset,
		skipLabels: NewSet(config.SkipLabels...),
	}

	for user, _ := range config.Users {
		ruleset.users.Put(user)
	}

	pools := NewSet()

	for poolName, pool := range config.Pools {
		set := NewSet(pool...)
		if diff := set.Difference(ruleset.users); !diff.Empty() {
			Fatal("unknown users '%v' in pool '%v'", diff, poolName)
		}

		pools.Put(poolName)
		ruleset.pools[poolName] = set
	}

	for ruleName, rules := range ruleset.ruleset {
		for _, rule := range rules {
			if rule.HasIf() && !pools.Test(rule.If) {
				Fatal("unknown if pool '%v' in rule '%v'", rule.If, ruleName)
			}

			for _, pick := range rule.Pick {
				if !pools.Test(pick.Pool) {
					Fatal("unknown pool name '%v' in rule '%v' for condition '%v'",
						pick.Pool, ruleName, rule.If)
				}
			}
		}
	}

	return ruleset
}

type Result struct {
	New       Set
	Pending   Set
	Assigned  Set
	Requested Set
}

func (ruleset *Ruleset) Apply(ruleName string, pr *PullRequest) Result {
	if !pr.Labels.Intersect(ruleset.skipLabels).Empty() {
		return Result{}
	}

	result := Result{
		New:      NewSet(),
		Pending:  NewSet(),
		Assigned: NewSet(),
	}

	author := NewSet(pr.Author)
	reviewed := pr.Reviewed()

	all := pr.ReviewRequests.Union(reviewed)

	for _, rule := range ruleset.ruleset[ruleName] {
		Debug("if: %v:%v <- %v", rule.If, ruleset.pools[rule.If], pr.Author)
		if rule.HasIf() && !ruleset.pools[rule.If].Test(pr.Author) {
			continue
		}

		for _, pick := range rule.Pick {
			pool := ruleset.pools[pick.Pool]

			active := pool.Intersect(all).Difference(author)
			assigned := active.Copy()

			Debug("pool: %v", pool)
			Debug("active: %v", active)

			if missing := pick.Count - len(active); missing > 0 {
				picked := pool.Difference(active.Union(author)).Pick(missing)
				Debug("picked: %v", picked)
				assigned.Add(picked)
				result.New.Add(picked)
			}

			result.Pending.Add(assigned.Difference(reviewed))
			result.Assigned.Add(assigned)
		}

		break
	}

	result.Requested = pr.ReviewRequests.Difference(result.Assigned)

	return result
}
