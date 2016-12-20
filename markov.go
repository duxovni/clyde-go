// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the golang-LICENSE file.
//
// Modified 2016 by Sam Dukhovni <dukhovni@mit.edu>, to generate
// continuations of user-provided input strings.

/*
Generating random text: a Markov chain algorithm

Based on the program presented in the "Design and Implementation" chapter
of The Practice of Programming (Kernighan and Pike, Addison-Wesley 1999).
See also Computer Recreations, Scientific American 260, 122 - 125 (1989).

A Markov chain algorithm generates text by creating a statistical model of
potential textual suffixes for a given prefix. Consider this text:

	I am not a number! I am a free man!

Our Markov chain algorithm would arrange this text into this set of prefixes
and suffixes, or "chain": (This table assumes a prefix length of two words.)

	Prefix       Suffix

	"" ""        I
	"" I         am
	I am         a
	I am         not
	a free       man!
	am a         free
	am not       a
	a number!    I
	number! I    am
	not a        number!

To generate text using this table we select an initial prefix ("I am", for
example), choose one of the suffixes associated with that prefix at random
with probability determined by the input statistics ("a"),
and then create a new prefix by removing the first word from the prefix
and appending the suffix (making the new prefix is "am a"). Repeat this process
until we can't find any suffixes for the current prefix or we exceed the word
limit. (The word limit is necessary as the chain table may contain cycles.)
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"encoding/json"
	"os"
)

// Prefix is a Markov chain prefix of one or more lowercase words.
type Prefix []string

// Shift removes the first word from the Prefix and appends the given word lowercased.
func (p Prefix) Shift(word string) {
	copy(p, p[1:])
	p[len(p)-1] = strings.ToLower(word)
}

// Chain contains a map ("chain") of prefixes to a map of suffixes to
// frequencies.  A prefix is a string of zero to prefixLen lowercase
// words joined with spaces.  A suffix is a single word.
type Chain struct {
	chain     map[string]map[string]int
	prefixLen int
}

// NewChain returns a new Chain with prefixes of prefixLen words.
func NewChain(prefixLen int) *Chain {
	return &Chain{make(map[string]map[string]int), prefixLen}
}

// Add increments the frequency count for a suffix following each
// distinct tail of a prefix
func (c *Chain) Add(p Prefix, s string) {
	for i := 0; i <= c.prefixLen; i++ {
		if i < c.prefixLen && p[i] == "" {
			continue
		}
		key := strings.Join(p[i:], " ")
		if c.chain[key] == nil {
			c.chain[key] = make(map[string]int)
		}
		c.chain[key][s]++
	}
}

// Build reads text from the provided Reader and
// parses it into prefixes and suffixes that are stored in Chain.
func (c *Chain) Build(r io.Reader) {
	br := bufio.NewReader(r)
	p := make(Prefix, c.prefixLen)
	for {
		var s string
		if _, err := fmt.Fscan(br, &s); err != nil {
			break
		}
		c.Add(p, s)
		p.Shift(s)
	}
}

func (c *Chain) NextWord(p Prefix) string {
	// Try each tail of the prefix, starting with the longest
	for i := 0; i <= c.prefixLen; i++ {
		key := strings.Join(p[i:], " ")
		if c.chain[key] == nil {
			continue
		}

		// Make a random choice weighted by frequency
		total := 0
		for _, freq := range c.chain[key] {
			total += freq
		}
		if total == 0 {
			continue
		}
		n := rand.Intn(total)
		for w, freq := range c.chain[key] {
			n -= freq
			if n <= 0 {
				return w
			}
		}
	}
	return ""
}

// Generate returns a string of at most n words generated from Chain.
func (c *Chain) Generate(start string, n int) string {
	words := strings.Fields(start)
	p := make(Prefix, c.prefixLen)
	lastWordsStart := len(words) - c.prefixLen
	if lastWordsStart < 0 {
		lastWordsStart = 0
	}
	for _,w := range words[lastWordsStart:] {
		p.Shift(w)
	}
	for i := 0; i < n; i++ {
		next := c.NextWord(p)
		if len(next) == 0 {
			break
		}
		words = append(words, next)
		p.Shift(next)
	}
	return strings.Join(words, " ")
}

// Save saves a chain's suffix frequency map to the given file in JSON
// format
func (c *Chain) Save(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(c.chain)
}

// LoadChain loads a suffix frequency map in JSON format from the
// given file if the file exists, and returns a new chain with the
// given prefix length and the loaded frequency map, or an empty map
// if loading failed.
func LoadChain(filename string, prefixLen int) *Chain {
	c := NewChain(prefixLen)

	f, err := os.Open(filename)
	if err == nil {
		defer f.Close()
		dec := json.NewDecoder(f)
		dec.Decode(&c.chain)
	}

	return c
}
