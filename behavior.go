// Copyright 2016 Sam Dukhovni
//
// Adapted from clyde.pl by cat@mit.edu
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)

package main

import (
	"strings"
	"regexp"
	"math/rand"
)

const prefixLen = 2
const genWords = 20
var chain *Chain

type Behavior func(string) string

func plainBehavior(substr string, resp func() string) Behavior {
	return func(s string) string {
		if strings.Contains(s, substr) {
			return resp()
		} else {
			return ""
		}
	}
}

func regexpBehavior(pattern string, resp func(*regexp.Regexp, string, []int) string) Behavior {
	insPattern := strings.Join([]string{"(?i)", pattern}, "")
	rex := regexp.MustCompile(insPattern)
	return func(s string) string {
		m := rex.FindStringSubmatchIndex(s)
		if m == nil {
			return ""
		} else {
			return resp(rex, s, m)
		}
	}
}

func chainBehavior(b Behavior) Behavior {
	return func(s string) string {
		out := b(s)
		if out != "" {
			return chain.Generate(out, genWords)
		} else {
			return ""
		}
	}
}

// List of behaviors to be attempted in the order given; the final
// behavior should be able to trigger on any input, and should add its
// input to the chainer
var Behaviors = []Behavior{
	chainBehavior(regexpBehavior("if (?P<fight1>.+) and (?P<fight2>.+) (fought|got in|were in|had)|between (?P<fight1>.+) and (?P<fight2>.+[^,\\?])(,|\\?|$| who| which| what)",
		func(r *regexp.Regexp, s string, m []int) string {
			var template string
			switch rand.Intn(2) {
			case 0:
				template = "$fight1"
			case 1:
				template = "$fight2"
			}
			return string(r.ExpandString([]byte(""), template, s, m))
		})),
	chainBehavior(func (s string) string {
		chain.Build(strings.NewReader(s))
		return " "
	}),
}
