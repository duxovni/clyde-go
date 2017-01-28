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
	"strconv"
	"regexp"
	"math/rand"
	"bufio"
	"os"
	"path"
	"time"
	"github.com/zephyr-im/zephyr-go"
	"github.com/sdukhovni/clyde-go/stringutil"
	"github.com/sdukhovni/clyde-go/mood"
	"github.com/sdukhovni/clyde-go/cat"
)

// behavior represents a zephyrbot behavior. A behavior takes a Clyde
// instance and an incoming zephyr, and either returns false to
// indicate that the behavior was not triggered by the message, or
// performs some action (possibly using or modifying the Clyde) and
// returns true to indicate that the behavior was triggered.
type behavior func(*Clyde, zephyr.MessageReaderResult) bool

// standardBehavior generates a behavior following a standard pattern
// of triggering based on a case-insensitive regular expression in a
// zephyr body, reading some named capturing groups from the regexp
// match, possibly performing some action, and replying with a single
// zephyr (possibly generated using the markov chainer) either on the
// same class and instance as the incoming zephyr or on Clyde's home
// class.
func standardBehavior(pattern string, keys []string, chain bool, resp func(*Clyde, zephyr.MessageReaderResult, map[string]string) string) behavior {
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
			response = c.chain.Generate(response, sentenceCounts[rand.Intn(len(sentenceCounts))], maxWords)
		}

		class := r.Message.Header.Class
		instance := r.Message.Header.Instance
		if class != homeClass || instance != homeInstance {
			switch c.subs[class] {
			case 0, LISTEN:
				return true
			case REPLYHOME:
				if !strings.HasPrefix(strings.ToLower(r.Message.Body[1]), "clyde") {
					class = homeClass
					instance = homeInstance
				}
			}
		}

		c.send(class, instance, response)

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
	filepath := c.path(filename)

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
	filepath := c.path(filename)

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
var behaviors = []behavior{
	watchCat,
	empathy,
	addActLike,
	actLike,
	addSub,
	checkSub,
	getMood,
	cheerup,
	learnJob,
	story,
	fight,
	fortune,
	dice,
	quip,
	ping,
	chat,
}


func tryPlayCat(c *Clyde) {
	c.cat.State = cat.TryPlay
	c.send(c.cat.Class, c.cat.Instance, cat.CatCmd(cat.PlayCmds[rand.Intn(len(cat.PlayCmds))]))
}

func tryScoopCat(c *Clyde) {
	c.cat.State = cat.TryScoop
	c.send(c.cat.Class, c.cat.Instance, cat.CatCmd("scoop"))
}

// watchCat is a special behavior for interacting with the cat and
// keeping track of her whereabouts.
func watchCat(c *Clyde, r zephyr.MessageReaderResult) bool {
	if shortSender(r) != cat.CatName {
		return false
	}

	body := r.Message.Body[1]

	c.cat.Class = r.Message.Header.Class
	c.cat.Instance = r.Message.Header.Instance

	action, user := cat.ParseAction(body)

	// Is the cat interacting with us?
	withUs := user == "clyde"

	switch action {
	case cat.React:
		if c.cat.State == cat.TryPlay && (withUs || user == "") {
			c.mood = c.mood.Better().Better().AtLeastOk()
			c.cat.State = cat.Normal
			return true
		}
		c.cat.State = cat.Normal
	case cat.Scooped:
		if withUs {
			c.cat.State = cat.WeScooped
			if c.cat.Stolen {
				c.send(c.cat.StolenClass, c.cat.StolenInstance, fmt.Sprintf("Thanks for visiting, %s!", cat.CatName))
				c.cat.Stolen = false
			} else {
				c.send(homeClass, homeInstance, fmt.Sprintf("Let's go over here, %s", cat.CatName))
				c.cat.Stolen = true
				c.cat.StolenTime = time.Now()
				c.cat.StolenClass = c.cat.Class
				c.cat.StolenInstance = c.cat.Instance
			}
		} else {
			c.cat.State = cat.Normal
		}
	case cat.ScoopFailed:
		if withUs {
			c.send(c.cat.Class, c.cat.Instance, ":(")
		}
		c.cat.State = cat.Normal
	case cat.Leave:
		if withUs {
			c.cat.State = cat.WeCarrying
		} else {
			c.cat.State = cat.Traveling
			c.cat.Stolen = false
		}
	case cat.Enter:
		if withUs {
			c.cat.State = cat.TryDeposit
			c.send(c.cat.Class, c.cat.Instance, cat.CatCmd("deposit"))
		} else {
			c.cat.State = cat.Normal
		}
	case cat.Deposited:
		if withUs {
			tryPlayCat(c)
		} else {
			c.cat.State = cat.Normal
		}
	case cat.Bored:
		c.cat.State = cat.Normal
		if time.Since(c.lastInteraction) > time.Hour {
			switch rand.Intn(8) {
			case 0:
				tryScoopCat(c)
			case 1:
				tryPlayCat(c)
			}
		}
	default:
		c.cat.State = cat.Normal
	}

	if c.mood == mood.Lonely && c.cat.State == cat.Normal {
		tryPlayCat(c)
	}

	return withUs
}

// Special behavior to update Clyde's mood based on incoming messages;
// always returns false.
func empathy(c *Clyde, r zephyr.MessageReaderResult) bool {
	rex := regexp.MustCompile("(?i)(?P<emote>:[\\(\\)D3]|;\\(|:,\\(|happy|smile|laugh|sad|frown|cry)")
	match := rex.FindStringSubmatchIndex(r.Message.Body[1])
	if match == nil {
		return false
	}

	emote := string(rex.ExpandString([]byte(""), "$emote", r.Message.Body[1], match))

	switch emote {
	case ":D", ":3", "laugh":
		if rand.Intn(2) == 0 {
			c.mood = c.mood.Better()
		}
		fallthrough
	case ":)", "happy", "smile":
		c.mood = c.mood.Better()

	case ";(", ":,(", "cry":
		if rand.Intn(2) == 0 {
			c.mood = c.mood.Worse()
		}
		fallthrough
	case ":(", "sad", "frown":
		c.mood = c.mood.Worse()
	}

	return false
}

var addActLike = standardBehavior("clyde.? (?P<person>.+) says,? (\"(?P<phrase>[^\"]+)\".?|'(?P<phrase>[^']+)'.?|(?P<phrase>[^\"']+)|(?P<phrase>.+[\"'].+))$",
	[]string{"person", "phrase"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		alDir := c.path("al")
		os.MkdirAll(alDir, 0755)
		filename := path.Join("al", stringutil.Escape(strings.ToLower(kvs["person"])))
		addLine(c, filename, kvs["phrase"])
		return "Ok!"
	})

var actLike = standardBehavior("clyde.? ((please )?act like (?P<person>.*[^\\.\\?!])(?P<punc>.*?)$|what does (?P<person>.+) say)",
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

var addSub = standardBehavior("clyde.*sub(scribe)? to (me|my class|(-c )?(?P<class>[^ !\\?]+[^ !\\?\\.]))",
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

		c.subscribe(class, REPLYHOME)
		return fmt.Sprintf("-c %s sounds awesome! Thanks for the invitation :)", class)
	})

var checkSub = standardBehavior("are you (on|sub(scri)?bed to) (me|my class|(-c )?(?P<class>[^ !\\?]+[^ !\\?\\.]))",
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

var getMood = standardBehavior("clyde.* how are you", []string{}, false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		return fmt.Sprintf("I'm %s%s", c.mood.String(), c.mood.Punc())
	})

var cheerup = standardBehavior("clyde.*[^a-z](hug|cuddle|s[ck]rit?ch)", []string{}, false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		c.mood = c.mood.Better()
		return "Thanks :)"
	})

var learnJob = standardBehavior("clyde.? (?P<job>.+) is an? (job|profession|occupation)",
	[]string{"job"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		addLine(c, "jobs", kvs["job"])
		return "That's what I wanna be when I grow up!"
	})

var story = standardBehavior("tell me a story",
	nil,
	true,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		job, _ := randomLine(c, "jobs")
		return fmt.Sprintf("Once upon a time, there was %s %s named %s who", stringutil.Article(job), job, shortSender(r))
	})

var fight = standardBehavior("if (?P<fight1>.+) and (?P<fight2>.+) (fought|duell?ed|got in|were in|had)|(who|which|what).* (win|happen).* between (?P<fight1>.+) and (?P<fight2>.+[^,\\?])(\\?|$)|between (?P<fight1>.+) and (?P<fight2>.+[^,\\?]),? (who|which|what).* (win|happen)",
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

var fortune = standardBehavior("fortune", []string{}, false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		var intros []string
		switch rand.Intn(3) {
		case 0:
			intros = []string{
				fmt.Sprintf("%s, yesterday you were", shortSender(r)),
				"Today you will",
				"Tomorrow you should",
			}
		case 1:
			planet, _ := randomLine(c, "planets")
			intros = []string{
				"The stars say",
				fmt.Sprintf("%s is aligned, so expect", planet),
				"Be careful of",
			}
		case 2:
			intros = []string{
				fmt.Sprintf("%s, in love, you will", shortSender(r)),
				"In work, you will",
				"For yourself, you should",
			}
		}
		var response []string
		for _, intro := range intros {
			response = append(response, c.chain.Generate(intro, 1, maxWords))
		}
		return strings.Join(response, " ")
	})

var dice = standardBehavior("( |^)(?P<count>[0-9]*)d(?P<faces>[0-9]+)",
	[]string{"count", "faces"},
	false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		var count int
		if kvs["count"] == "" {
			count = 1
		} else {
			count, _ = strconv.Atoi(kvs["count"])
		}
		faces, _ := strconv.Atoi(kvs["faces"])
		total := 0
		for i := 0; i < count; i++ {
			total += rand.Intn(faces) + 1
		}
		return strconv.Itoa(total)
	})

var simpleQuips = map[string]string{
	"wacky": "Aw, and me without my spork.",
	"too many secrets": "Setec Astronomy",
	"manna manna": "Do do dit do do.",
	"growl for me": "Grrrrrr",
	"(^| )(are|am) not[ ,\\.\\?!]": "Are too!",
	"(^| )(are|am) too[ ,\\.\\?!]": "Am not!",
	"i've been captured": "Yay!",
	"no, that's a bad thing": "Yay!",
	"morse": "dit, dit dah dah",
	"(^| )sing\\b": "la la la",
	"what makes the grass grow": "Fertilizer, sir!",
	"what is the meaning of life\\?": "42/3",
	"what do you want\\?": "Never ask that question.",
	"is there a god\\?": "There is now...",
	"elvis|bermuda triangle": "Elvis needs boats!!",
	"brains": "BRAAAAAAAAIIIIINNNNSSSSSS",
	"bonfire": "Bonfire is not a hivemind.",
	"(^| )los(e|t|ing) [^ ]+ way": "Don't lose your way!",
	"contract": "／人◕ ‿‿ ◕人＼",
	"clyde(::|\\.)(pet|play|cuddle|s[ck]rit?ch|treat|scoop|deposit)": "clyde climbs on top of the bookshelf and hisses",
}

var fileQuips = map[string]string{
	"(^| )ai[ ,\\.\\?]": "ai",
	"[\\*:](tickles?|poke)[\\*:]": "tickle",
	"what('| i)s wrong\\?": "wrong",
	"thank(s| you)": "welcome",
	"bye": "bye",
	"(good ?|')night": "night",
	"how do you like": "howlike",
	"(^| )(hi|hello)[ ,\\.\\?!]": "hello",
	"pull!": "pull",
}

func quip(c *Clyde, r zephyr.MessageReaderResult) bool {
	for k,v := range simpleQuips {
		if standardBehavior(k, []string{}, false,
			func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
				return v
			})(c, r) {
				return true
			}
	}

	for k,v := range fileQuips {
		if standardBehavior(k, []string{}, false,
			func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
				resp, _ := randomLine(c, v)
				return resp
			})(c, r) {
				return true
			}
	}

	return false
}

var ping = standardBehavior("^clyde\\?$", []string{}, false,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		return "Yes?"
	})

var chat = standardBehavior("^clyde,? (?P<topic>[^ ]+)",
	[]string{"topic"},
	true,
	func(c *Clyde, r zephyr.MessageReaderResult, kvs map[string]string) string {
		return stringutil.Capitalize(kvs["topic"])
	})
