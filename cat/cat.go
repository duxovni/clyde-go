// Copyright 2016 Sam Dukhovni <dukhovni@mit.edu>
//
// Adapted from clyde.pl by cat@mit.edu
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
// cat defines structures and constants relating to zeroday, the
// zephyr cat.

package cat

import (
	"fmt"
	"regexp"
)

// Cat is a structure for keeping track of the cat.
type Cat struct {
	Class string
	Instance string
	State CatState
}

// CatState represents different states the cat can be in, with
// respect to Clyde.
type CatState int

const (
	Normal		CatState = 0
	TryScoop	CatState = 1
	WeScooped	CatState = 2
	WeCarrying	CatState = 3
	TryDeposit	CatState = 4
	TryPlay		CatState = 5
	Traveling	CatState = 6
)

// CatAction represents different actions the cat can perform.
type CatAction int

const (
	React		CatAction = 0
	Scooped		CatAction = 1
	ScoopFailed	CatAction = 2
	Leave		CatAction = 3
	Enter		CatAction = 4
	Deposited	CatAction = 5
	Bored		CatAction = 6
)

var ActionPatterns = map[CatAction]string {
	React: "((bats|scratches) at|rubs up against|snuggles up to|looks at) (?P<user>\\w*)|slips out of (?P<user>\\w*)'s arms|purrs|meows|is confused",
	Scooped: "(?P<user>\\w*) scoops",
	ScoopFailed: "slips out of (?P<user>\\w*)'s grip",
	Leave: "carried away by (?P<user>\\w*)",
	Enter: "(?P<user>\\w*) carries",
	Deposited: "(?P<user>\\w*) sets",
	Bored: "rolls around|curls up|plays with her tail|mews softly",
}

// ParseAction parses a message from the cat to determine what action
// is being performed, and possibly what user it's being performed
// with (if the user cannot be determined, the second return value is
// empty).
func ParseAction(msg string) (CatAction, string) {
	for action,pattern := range ActionPatterns {
		rex := regexp.MustCompile(pattern)
		match := rex.FindStringSubmatchIndex(msg)
		if match == nil {
			continue
		}
		user := string(rex.ExpandString([]byte(""), "$user", msg, match))
		return action, user
	}

	return Bored, ""
}

const CatName = "zeroday"

func CatCmd(cmd string) string {
	return fmt.Sprintf("%s::%s", CatName, cmd)
}

var PlayCmds = []string {
	"pet",
	"skritch",
	"cuddle",
	"treat",
	"play",
}
