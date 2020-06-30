package main

import (
	"strconv"
	"strings"
)

type Ref struct {
	Pool  string
	Count int
}

type Rule map[string][]Ref

type Rules struct {
	users Set
	pools map[string]Set
	rules map[string]Rule

	skipLabels Set
}

func NewRules(config *Config) *Rules {
	rules := &Rules{
		users:      NewSet(),
		pools:      make(map[string]Set),
		rules:      make(map[string]Rule),
		skipLabels: NewSet(config.SkipLabels...),
	}

	for user, _ := range config.Users {
		rules.users.Put(user)
	}

	pools := NewSet()

	for poolName, pool := range config.Pools {
		set := NewSet(pool...)
		if diff := set.Difference(rules.users); !diff.Empty() {
			Fatal("unknown users '%v' in pool '%v'", diff, poolName)
		}

		pools.Put(poolName)
		rules.pools[poolName] = set
	}

	for ruleName, rawRule := range config.Rules {
		rule := make(Rule)

		for cond, rawRefs := range rawRule {
			if cond != "_" && !pools.Test(cond) {
				Fatal("unknown conditional pool '%v' in rule '%v'", cond, ruleName)
			}

			var refs []Ref
			for _, ref := range rawRefs {
				items := strings.Split(ref, ":")

				pool := items[0]
				if !pools.Test(pool) {
					Fatal("unknown pool name '%v' in rule '%v'", pool, ruleName)
				}

				count := 1
				if len(items) > 1 {
					if val, err := strconv.ParseInt(items[1], 10, 64); err != nil {
						Fatal("malformed ref '%v' in rule '%v': '%v' is not an int", ref, ruleName, items[1])
					} else if val < 1 {
						Fatal("malformed ref '%v' in rule '%v': '%v' is not a valid value", ref, ruleName, val)
					} else {
						count = int(val)
					}
				}

				refs = append(refs, Ref{pool, count})
			}
			rule[cond] = refs
		}
		rules.rules[ruleName] = rule
	}
	return rules
}

type Result struct {
	New      Set
	Pending  Set
	Assigned Set
}

func (rules Rules) Apply(ruleName string, pr *PullRequest) Result {

	if !pr.Labels.Intersect(rules.skipLabels).Empty() {
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

	for cond, refs := range rules.rules[ruleName] {
		if cond != "_" && !rules.pools[cond].Test(pr.Author) {
			continue
		}

		for _, ref := range refs {
			pool := rules.pools[ref.Pool]

			active := pool.Intersect(all).Difference(author)
			assigned := active.Copy()

			if missing := ref.Count - len(active); missing > 0 {
				picked := pool.Difference(active.Union(author)).Pick(missing)
				assigned.Add(picked)
				result.New.Add(picked)
			}

			result.Pending.Add(assigned.Difference(reviewed))
			result.Assigned.Add(assigned)
		}
	}

	return result
}
