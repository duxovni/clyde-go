// Copyright 2016 Sam Dukhovni <dukhovni@mit.edu>
//
// Adapted from clyde.pl by cat@mit.edu
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
// Some code snippets copied from the zephyr-go library
// (https://github.com/zephyr-im/zephyr-go), (c) 2014 The zephyr-go
// authors, licensed under the Apache License, Version 2.0
// (http://www.apache.org/licenses/LICENSE-2.0)
//
//
// clyde is a markov-chain-based zephyr chatbot; this library defines
// structures and methods for running an instance of clyde.

package clyde

import (
	"strings"
	"log"
	"time"
	"math/rand"
	"path"
	"os"
	"encoding/json"
	"sync"
	"github.com/zephyr-im/krb5-go"
	"github.com/zephyr-im/zephyr-go"
	"github.com/sdukhovni/clyde-go/markov"
)

// Clyde (the struct) holds all of the internal state needed for Clyde
// (the zephyrbot) to send and receive zephyrs, generate text, and
// load/save persistent state data.
type Clyde struct {
	chain *markov.Chain
	zsigChain *markov.Chain
	homeDir string
	session *zephyr.Session
	ctx *krb5.Context
	subs map[string]classPolicy
	mood int
	lastInteraction time.Time
	ticker *time.Ticker
	shutdown chan struct{}
	wg sync.WaitGroup
}

const (
	yuckyMood	int = 0
	angryMood	int = 1
	unhappyMood	int = 2
	lonelyMood	int = 3
	turnipMood	int = 4
	okMood		int = 5
	goodMood	int = 6
	greatMood	int = 7
	maxMood		int = 7
)

func MoodString(m int) string {
	switch m {
	case yuckyMood:
		return "yucky"
	case angryMood:
		return "angry"
	case unhappyMood:
		return "unhappy"
	case lonelyMood:
		return "lonely"
	case turnipMood:
		return "a turnip"
	case okMood:
		return "ok"
	case goodMood:
		return "good"
	case greatMood:
		return "great"
	default:
		return "ok"
	}
}

func MoodPunc(m int) string {
	switch m {
	case yuckyMood:
		return "."
	case angryMood:
		return "!"
	case unhappyMood:
		return "."
	case lonelyMood:
		return " :("
	case turnipMood:
		return "."
	case okMood:
		return "."
	case goodMood:
		return " :)"
	case greatMood:
		return "!"
	default:
		return "."
	}
}

// LoadClyde initializes a Clyde by loading data files found in the
// given directory, returning an error if the directory does not
// exist and cannot be created.
func LoadClyde(dir string) (*Clyde, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	c := &Clyde{}

	c.homeDir = dir

	// Set up zephyr session
	c.session, err = zephyr.DialSystemDefault()
	if err != nil {
		return nil, err
	}

	// Create krb5 context for subscriptions
	c.ctx, err = krb5.NewContext()
	if err != nil {
		return nil, err
	}

	// Create markov chain, and try to load saved chain
	c.chain = markov.NewChain(prefixLen)
	err = c.chain.Load(c.path(chainFile))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Create zsig markov chain, and try to load saved chain
	c.zsigChain = markov.NewChain(zsigPrefixLen)
	err = c.zsigChain.Load(c.path(zsigChainFile))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	c.session.SendSubscribeNoDefaults(c.ctx, []zephyr.Subscription{{Class: homeClass, Instance: homeInstance, Recipient: ""}})
	c.subs = make(map[string]classPolicy)
	err = c.loadSubs()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	c.mood = okMood

	c.lastInteraction = time.Now()

	c.ticker = time.NewTicker(time.Minute)

	c.shutdown = make(chan struct{})

	return c, nil
}

// Run starts Clyde running; Clyde will begin receiving and responding
// to zephyrs on classes Clyde is subscribed to, as well as responding
// to clock ticks. After Clyde.Run() is called, Clyde.Shutdown() must
// be called before exiting.
func (c *Clyde) Run() {
	c.wg.Add(1)
	go func() {
		defer c.handleShutdown()
		for {
			// A shutdown should take priority over
			// pending messages/ticks
			select {
			case <-c.shutdown:
				return
			default:
			}
			select {
			case t := <-c.ticker.C:
				c.handleTick(t)
			case r := <-c.session.Messages():
				c.handleMessage(r)
			case <-c.shutdown:
				return
			}
		}
	}()
}

// Shutdown tells Clyde to save his persistent state to his home
// directory, close his zephyr session, and perform any other
// necessary cleanup for Clyde to shut down. Any program that uses a
// Clyde must call this method to cleanly shutdown Clyde before
// exiting.
func (c *Clyde) Shutdown() {
	close(c.shutdown)
	c.wg.Wait()
	c.session.Close() // Moved here to avoid lingering internal event loop issue
}


type classPolicy uint8

const (
	LISTEN classPolicy = 1
	REPLYHOME classPolicy = 2
	FULL classPolicy = 3
)

// subscribe subscribes Clyde to a new zephyr class.
func (c *Clyde) subscribe(class string, policy classPolicy) {
	if c.subs[class] != 0 {
		return
	}
	c.session.SendSubscribeNoDefaults(c.ctx, []zephyr.Subscription{{Class: class, Instance: "*", Recipient: ""}})
	c.subs[class] = policy
}

// send sends a zephyr from Clyde with the given body to the given
// class and instance.
func (c *Clyde) send(class, instance, body string) {
	uid := c.session.MakeUID(time.Now())
	zsig := c.zsigChain.Generate("", 1, rand.Intn(6)+2)
	msg := &zephyr.Message{
		Header: zephyr.Header{
			Kind:	zephyr.ACKED,
			UID:	uid,
			Port:	c.session.Port(),
			Class:	class, Instance: instance, OpCode: "",
			Sender:		sender,
			Recipient:	"",
			DefaultFormat:	"http://mit.edu/df/",
			SenderAddress:	c.session.LocalAddr().IP,
			Charset:	zephyr.CharsetUTF8,
			OtherFields:	nil,
		},
		Body: []string{zsig, body},
	}
	_, err := c.session.SendMessageUnauth(msg)
	if err != nil {
		log.Printf("Send error: %v", err)
	}
}

func (c *Clyde) path(filename string) string {
	return path.Join(c.homeDir, filename)
}


const homeClass = "ztoys-dev"
const homeInstance = "clyde"

const chainFile = "chain.json"
const zsigChainFile = "zsigChain.json"
const subsFile = "subs.json"

const sender = "clyde"
const prefixLen = 2
const zsigPrefixLen = 1 // Be more creative with less input data


func (c *Clyde) handleMessage(r zephyr.MessageReaderResult) {
	// Ignore our own messages
	if r.Message.Header.Sender == sender {
		return
	}

	c.chain.Build(strings.NewReader(r.Message.Body[1]))
	c.zsigChain.Build(strings.NewReader(r.Message.Body[0]))

	// Perform the first behavior that triggers, and exit
	for _, b := range behaviors {
		if b(c, r) {
			c.lastInteraction = time.Now()
			return
		}
	}
}

func (c *Clyde) handleTick(t time.Time) {
	aloneDuration := time.Since(c.lastInteraction)

	if aloneDuration >= time.Hour && rand.Intn(90) == 0 {
		var phrase string
		switch c.mood {
		case lonelyMood:
			phrase,_ = randomLine(c, "bored")
		case goodMood:
			phrase = "Hi, all."
		case greatMood:
			phrase = "*bounce*"
		}
		if phrase != "" {
			c.send(homeClass, homeInstance, phrase)
		}
	}
	if aloneDuration >= 2*time.Hour && rand.Intn(30) == 0 {
		c.mood = lonelyMood
	}
}

func (c *Clyde) handleShutdown() {
	c.ticker.Stop()
	c.chain.Save(c.path(chainFile))
	c.zsigChain.Save(c.path(zsigChainFile))
	c.saveSubs()
	c.session.SendCancelSubscriptions(c.ctx)
	c.ctx.Free()
	// c.session.Close()
	c.wg.Done()
}

// loadSubs attempts to load and subscribe to a list of subscriptions
// in JSON format from a file in Clyde's home directory.
func (c *Clyde) loadSubs() error {
	f, err := os.Open(c.path(subsFile))
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	err = dec.Decode(&(c.subs))
	if err != nil {
		return err
	}

	var subList []zephyr.Subscription
	for class, policy := range c.subs {
		if policy != 0 {
			subList = append(subList, zephyr.Subscription{Class: class, Instance: "*", Recipient: ""})
		}
	}

	c.session.SendSubscribeNoDefaults(c.ctx, subList)

	return nil
}

// saveSubs saves Clyde's subscriptions to a file in JSON format in
// Clyde's home directory.
func (c *Clyde) saveSubs() error {
	f, err := os.Create(c.path(subsFile))
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(c.subs)
	if err != nil {
		return err
	}

	return nil
}
