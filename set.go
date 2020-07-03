package main

import (
	"fmt"
	"math/rand"
	"sort"
)

type Set map[string]struct{}

func NewSet(items ...string) Set {
	set := make(Set)
	for _, item := range items {
		set.Put(item)
	}
	return set
}

func (set Set) Copy() Set {
	result := make(Set)
	for item, _ := range set {
		result.Put(item)
	}
	return result
}

func (set Set) Empty() bool {
	return len(set) == 0
}

func (set Set) ToArray() []string {
	var result []string
	for item, _ := range set {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func (set Set) Test(item string) bool {
	_, ok := set[item]
	return ok
}

func (set Set) Put(item string) {
	set[item] = struct{}{}
}

func (set Set) Del(item string) {
	delete(set, item)
}

func (set Set) Add(other Set) {
	for item, _ := range other {
		set.Put(item)
	}
}

func (set Set) Remove(other Set) {
	for item, _ := range other {
		set.Del(item)
	}
}

func (set Set) Equals(other Set) bool {
	if len(set) != len(other) {
		return false
	}

	array := other.ToArray()
	for i, item := range set.ToArray() {
		if item != array[i] {
			return false
		}
	}
	return true
}

func (set Set) Union(other Set) Set {
	result := make(Set)
	for item, _ := range set {
		result.Put(item)
	}
	for item, _ := range other {
		result.Put(item)
	}
	return result
}

func (set Set) Intersect(other Set) Set {
	result := make(Set)
	for item, _ := range set {
		if other.Test(item) {
			result.Put(item)
		}
	}
	return result
}

func (set Set) Difference(other Set) Set {
	result := make(Set)
	for item, _ := range set {
		if !other.Test(item) {
			result.Put(item)
		}
	}
	return result
}

func (set Set) Take(n int) Set {
	if n > len(set) {
		return set.Copy()
	}
	return NewSet(set.ToArray()[0:n]...)
}

func (set Set) Pick(n int) Set {
	if n >= len(set) {
		return set
	}

	arr := set.ToArray()
	rand.Shuffle(len(arr), func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
	return NewSet(arr[0:n]...)
}

func (set Set) String() string {
	return fmt.Sprintf("%v", set.ToArray())
}
