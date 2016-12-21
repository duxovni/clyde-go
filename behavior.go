// Copyright 2016 Sam Dukhovni
//
// Adapted from clyde.pl by cat@mit.edu
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
//
// behavior.go defines combinator functions for writing chatbot
// behaviors, and provides a set of behaviors for Clyde.

package clyde

import (
	"fmt"
	"regexp"
	"math/rand"
	"github.com/zephyr-im/zephyr-go"
	"github.com/sdukhovni/clyde-go/stringutil"
)

// Behavior represents a zephyrbot behavior. A Behavior takes a Clyde
// instance and an incoming zephyr, and either returns false to
// indicate that the behavior was not triggered by the message, or
// performs some action (possibly using or modifying the Clyde) and
// returns true to indicate that the behavior was triggered.
type Behavior func(*Clyde, zephyr.MessageReaderResult) bool

// StandardBehavior generates a Behavior following a standard pattern
// of triggering based on a case-insensitive regular expression in a
// zephyr body, reading some named capturing groups from the regexp
// match, possibly performing some action, and replying with a single
// zephyr on the same class and instance as the incoming zephyr.
func StandardBehavior(pattern string, keys []string, resp func(*Clyde, zephyr.MessageReaderResult, map[string]string) string) Behavior {
	return func(c *Clyde, r zephyr.MessageReaderResult) bool {
		body := r.Message.Body[1]
		insPattern := fmt.Sprint("(?i)", pattern)
		rex := regexp.MustCompile(insPattern)
		match := rex.FindStringSubmatchIndex(body)
		if match == nil {
			return false
		}

		keyvals := make(map[string]string)
		for _, key := range keys {
			keyvals[key] = string(rex.ExpandString([]byte(""), fmt.Sprint("$", key), body, match))
		}

		c.Send(r.Message.Header.Class, r.Message.Header.Instance, stringutil.BreakLines(resp(c, r, keyvals), stringutil.MaxLine))

		return true
	}
}

// Behaviors is a list of behaviors to be attempted in the order
// given.
var Behaviors = []Behavior{
	fight,
}

// genWords is the number of words that a behavior should generate using the markov chainer.
const genWords = 20

var fight = StandardBehavior("if (?P<fight1>.+) and (?P<fight2>.+) (fought|got in|were in|had)|between (?P<fight1>.+) and (?P<fight2>.+[^,\\?])(,|\\?|$| who| which| what)",
	[]string{"fight1", "fight2"},
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		var winner string
		switch rand.Intn(2) {
		case 0:
			winner = "fight1"
		case 1:
			winner = "fight2"
		}
		return c.Chain.Generate(fmt.Sprintf("I think %s would win, because", kvs[winner]), genWords)
	})
