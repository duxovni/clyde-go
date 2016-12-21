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
	"log"
	"fmt"
	"strings"
	"regexp"
	"math/rand"
	"bufio"
	"os"
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
// zephyr (possibly generated using the markov chainer) on the same
// class and instance as the incoming zephyr.
func StandardBehavior(pattern string, keys []string, chain bool, resp func(*Clyde, zephyr.MessageReaderResult, map[string]string) string) Behavior {
	return func(c *Clyde, r zephyr.MessageReaderResult) bool {
		body := strings.Join(strings.Fields(r.Message.Body[1]), " ") // normalize spacing for regexp matches
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

		response := resp(c, r, keyvals)
		if chain {
			response = c.Chain.Generate(response, sentenceCounts[rand.Intn(len(sentenceCounts))], maxWords)
		}
		c.Send(r.Message.Header.Class, r.Message.Header.Instance, stringutil.BreakLines(response, stringutil.MaxLine))

		return true
	}
}

// maxWords is the maximum number of words that a behavior should
// generate using the markov chainer.
const maxWords = 100

// sentenceCounts is a set of sentence counts to request from the
// chainer; a number is chosen randomly from this list each time a
// number of sentences is needed.
var sentenceCounts = []int{1, 1, 1, 2, 2, 3}

// randomLine returns a random non-empty line from a file in Clyde's
// home directory.
func randomLine(c *Clyde, filename string) (string, error) {
	filepath := c.Path(filename)

	f, err := os.Open(filepath)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines[rand.Intn(len(lines))], nil
}

// addLine adds a line to a file in Clyde's home directory.
func addLine(c *Clyde, filename, line string) error {
	filepath := c.Path(filename)

	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, line)
	return nil
}


// Behaviors is a list of behaviors to be attempted in the order
// given.
var Behaviors = []Behavior{
	learnJob,
	story,
	fight,
	chat,
}

var learnJob = StandardBehavior("clyde.? (?P<job>.+) is an? (job|profession|occupation)",
	[]string{"job"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		addLine(c, "jobs", kvs["job"])
		return "That's what I wanna be when I grow up!"
	})

var story = StandardBehavior("tell me a story",
	nil,
	true,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		job, _ := randomLine(c, "jobs")
		sender := strings.Split(r.Message.Header.Sender, "@")[0]
		return fmt.Sprintf("Once upon a time, there was %s %s named %s who", stringutil.Article(job), job, sender)
	})

var fight = StandardBehavior("if (?P<fight1>.+) and (?P<fight2>.+) (fought|got in|were in|had)|between (?P<fight1>.+) and (?P<fight2>.+[^,\\?])(,|\\?|$| who| which| what)",
	[]string{"fight1", "fight2"},
	true,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		var winner string
		switch rand.Intn(2) {
		case 0:
			winner = "fight1"
		case 1:
			winner = "fight2"
		}
		return fmt.Sprintf("I think %s would win, because", kvs[winner])
	})

var chat = StandardBehavior("clyde, (?P<topic>[^ ]+)",
	[]string{"topic"},
	true,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		return stringutil.Capitalize(kvs["topic"])
	})
