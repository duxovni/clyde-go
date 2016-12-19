// Copyright 2016 Sam Dukhovni
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
// Some code snippets copied from the zephyr-go library
// (https://github.com/zephyr-im/zephyr-go), (c) 2014 The zephyr-go
// authors, licensed under the Apache License, Version 2.0
// (http://www.apache.org/licenses/LICENSE-2.0)

package main

import (
	"log"
	"time"
	"os"
	"os/signal"
	"syscall"
	"github.com/zephyr-im/krb5-go"
	"github.com/zephyr-im/zephyr-go"
)

var session *zephyr.Session
var sender = "clyde"
var zsig = "Clyde"
var subs = []zephyr.Subscription{
	{"", "clyde-dev", "*"},
}

func send(class, instance, body string) {
	uid := session.MakeUID(time.Now())
	msg := &zephyr.Message{
		Header: zephyr.Header{
			Kind:	zephyr.ACKED,
			UID:	uid,
			Port:	session.Port(),
			Class:	class, Instance: instance, OpCode: "",
			Sender:		sender,
			Recipient:	"",
			DefaultFormat:	"http://mit.edu/df/",
			SenderAddress:	session.LocalAddr().IP,
			Charset:	zephyr.CharsetUTF8,
			OtherFields:	nil,
		},
		Body: []string{zsig, body},
	}
	sendTime := time.Now()
	var ack *zephyr.Notice
	var err error
	ack, err = session.SendMessageUnauth(msg)
	if err != nil {
		log.Printf("Send error: %v", err)
	} else {
		log.Printf("Received ack in %v: %v",
			time.Now().Sub(sendTime), ack)
	}
}

func handleMessage(auth zephyr.AuthStatus, msg *zephyr.Message) {
	if msg.Header.Sender == sender {
		return
	}

	send(msg.Header.Class, msg.Header.Instance, "hi!")
}

func main() {
	var err error

	session, err = zephyr.DialSystemDefault()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	go func() {
		for r := range session.Messages() {
			handleMessage(r.AuthStatus, r.Message)
		}
	}()

	log.Printf("Subscribing to %v", subs)
	ctx, err := krb5.NewContext()
	if err != nil {
		log.Fatal(err)
	}
	defer ctx.Free()

	ack, err := session.SendSubscribeNoDefaults(ctx, subs)
	log.Printf(" -> %v %v", ack, err)
	defer func() {
		log.Printf("Canceling subs")
		ack, err := session.SendCancelSubscriptions(ctx)
		log.Printf(" -> %v %v", ack, err)
	}()

	// Keep listening until a SIGINT or SIGTERM.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
