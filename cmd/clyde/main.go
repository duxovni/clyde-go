// Copyright 2016 Sam Dukhovni <dukhovni@mit.edu>
//
// Adapted from clyde.pl by cat@mit.edu
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
package main

import (
	"log"
	"time"
	"path"
	"os"
	"os/user"
	"os/signal"
	"syscall"
	"math/rand"
	"github.com/sdukhovni/clyde-go"
)

func main() {
	// Seed RNG
	rand.Seed(time.Now().UnixNano())

	// Get directory path for Clyde files
	curUser, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	clydeDir := path.Join(curUser.HomeDir, ".clyde")

	// Load Clyde
	clyde, err := clyde.LoadClyde(clydeDir)
	if err != nil {
		log.Fatal(err)
	}
	defer clyde.Shutdown()

	// Start Clyde's listener goroutine
	go clyde.Listen()

	// Keep listening until a SIGINT or SIGTERM.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
