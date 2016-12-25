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
	"path"
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
// zephyr (possibly generated using the markov chainer) either on the
// same class and instance as the incoming zephyr or on Clyde's home
// class.
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

		class := r.Message.Header.Class
		instance := r.Message.Header.Instance
		if class != homeClass || instance != homeInstance {
			switch c.subs[class] {
			case 0, LISTEN:
				return true
			case REPLYHOME:
				if !strings.Contains(strings.ToLower(r.Message.Body[1]), "clyde") {
					class = homeClass
					instance = homeInstance
				}
			}
		}

		c.Send(class, instance, stringutil.BreakLines(response, stringutil.MaxLine))

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

// shortSender returns just the kerberos principal (with no realm) of
// the sender of a zephyr.
func shortSender(r zephyr.MessageReaderResult) string {
	return strings.Split(r.Message.Header.Sender, "@")[0]
}

// allLines returns a list of non-empty lines in a file in Clyde's
// home directory.
func allLines(c *Clyde, filename string) ([]string, error) {
	filepath := c.Path(filename)

	f, err := os.Open(filepath)
	if err != nil {
		log.Println(err)
		return nil, err
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

	return lines, nil
}

// randomLine returns a random non-empty line from a file in Clyde's
// home directory.
func randomLine(c *Clyde, filename string) (string, error) {
	lines, err := allLines(c, filename)
	if err != nil {
		return "", err
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
	addActLike,
	actLike,
	addSub,
	checkSub,
	learnJob,
	story,
	fight,
	chat,
}

var addActLike = StandardBehavior("clyde.? (?P<person>.+) says,? (\"(?P<phrase>[^\"]+)\".?|'(?P<phrase>[^']+)'.?|(?P<phrase>[^\"']+)|(?P<phrase>.+[\"'].+))$",
	[]string{"person", "phrase"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		alDir := c.Path("al")
		os.MkdirAll(alDir, 0755)
		filename := path.Join("al", stringutil.Escape(strings.ToLower(kvs["person"])))
		addLine(c, filename, kvs["phrase"])
		return "Ok!"
	})

var actLike = StandardBehavior("clyde.? ((please )?act like (?P<person>.*[^\\.\\?!])(?P<punc>.*?)$|what does (?P<person>.+) say)",
	[]string{"person", "punc"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		filename := path.Join("al", stringutil.Escape(strings.ToLower(kvs["person"])))
		phrase, err := randomLine(c, filename)
		if err != nil {
			filename = path.Join("al", stringutil.Escape(strings.ToLower(fmt.Sprint(kvs["person"], kvs["punc"]))))
			phrase, err = randomLine(c, filename)
			if err != nil {
				return fmt.Sprintf("I don't know how to act like %s.", kvs["person"])
			}
		}
		return phrase
	})

var addSub = StandardBehavior("clyde.*sub(scribe)? to (me|my class|(-c )?(?P<class>[^ !\\?]+[^ !\\?\\.]))",
	[]string{"class"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		class := kvs["class"]
		if class == "" {
			class = shortSender(r)
		}

		if r.Message.Header.Class != homeClass || r.Message.Header.Instance != homeInstance {
			return "I'm subbed to a lot of classes right now; maybe another time..."
		}

		if c.subs[class] != 0 {
			return fmt.Sprintf("I'm already subbed to -c %s!", class)
		}

		if r.AuthStatus != zephyr.AuthYes {
			return "You look sketchy, I don't trust you..."
		}

		c.Subscribe(class, REPLYHOME)
		return fmt.Sprintf("-c %s sounds awesome! Thanks for the invitation :)", class)
	})

var checkSub = StandardBehavior("are you (on|sub(scri)?bed to) (me|my class|(-c )?(?P<class>[^ !\\?]+[^ !\\?\\.]))",
	[]string{"class"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		class := kvs["class"]
		if class == "" {
			class = shortSender(r)
		}

		if c.subs[class] == 0 {
			return fmt.Sprintf("I'm not subbed to -c %s.", class)
		} else {
			return fmt.Sprintf("Yup, I'm subbed to -c %s! It's my favorite class :)", class)
		}
	})

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
		return fmt.Sprintf("Once upon a time, there was %s %s named %s who", stringutil.Article(job), job, shortSender(r))
	})

var fight = StandardBehavior("if (?P<fight1>.+) and (?P<fight2>.+) (fought|duell?ed|got in|were in|had)|(who|which|what) .* between (?P<fight1>.+) and (?P<fight2>.+[^,\\?])(\\?|$)|between (?P<fight1>.+) and (?P<fight2>.+[^,\\?]),? (who|which|what)",
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
