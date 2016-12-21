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
	"path"
	"os"
	"github.com/zephyr-im/krb5-go"
	"github.com/zephyr-im/zephyr-go"
	"github.com/sdukhovni/clyde-go/markov"
)

// Clyde (the struct) holds all of the internal state needed for Clyde
// (the zephyrbot) to send and receive zephyrs, generate text, and
// load/save persistent state data.
type Clyde struct {
	Chain *markov.Chain
	homeDir string
	session *zephyr.Session
	ctx *krb5.Context
	subs []zephyr.Subscription
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
	c.Chain = markov.NewChain(prefixLen)
	err = c.Chain.Load(c.path(chainFile))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return c, nil
}

// Listen receives and handles zephyrs on classes Clyde is subscribed
// to, and never returns until Clyde is shut down.
func (c *Clyde) Listen() {
	for r := range c.session.Messages() {
		c.handleMessage(r)
	}
}

// Subscribe subscribes Clyde to the given list of zephyr
// subscriptions.
func (c *Clyde) Subscribe(subs []zephyr.Subscription) {
	c.session.SendSubscribeNoDefaults(c.ctx, subs)
	c.subs = append(c.subs, subs...)
}

// Send sends a zephyr from Clyde with the given body to the given
// class and instance.
func (c *Clyde) Send(class, instance, body string) {
	uid := c.session.MakeUID(time.Now())
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

// Shutdown saves Clyde's persistent state to Clyde's home directory,
// closes Clyde's zephyr session, and performs any necessary cleanup
// for Clyde to shut down. Any program that uses a Clyde must call
// this method to cleanly shutdown Clyde before exiting.
func (c *Clyde) Shutdown() error {
	var err error

	err = c.Chain.Save(c.path(chainFile))
	if err != nil {
		return err
	}

	_, err = c.session.SendCancelSubscriptions(c.ctx)
	if err != nil {
		return err
	}

	c.ctx.Free()

	err = c.session.Close()
	if err != nil {
		return err
	}

	return nil
}


const chainFile = "chain.json"

const sender = "clyde"
const zsig = "Clyde"
const maxLine = 70
const prefixLen = 2


func (c *Clyde) path(filename string) string {
	return path.Join(c.homeDir, filename)
}

func (c *Clyde) handleMessage(r zephyr.MessageReaderResult) {
	// Ignore our own messages
	if r.Message.Header.Sender == sender {
		return
	}

	c.Chain.Build(strings.NewReader(r.Message.Body[1]))

	// Perform the first behavior that triggers, and exit
	for _, b := range Behaviors {
		if b(c, r) {
			return
		}
	}
}
