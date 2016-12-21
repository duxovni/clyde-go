// Copyright 2016 Sam Dukhovni <dukhovni@mit.edu>
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
// stringutil contains miscellaneous string-manipulating functions
// useful for clyde-go.

package stringutil

import (
	"fmt"
	"strings"
	"unicode/utf8"
	"regexp"
)

const MaxLine = 70

func BreakLines(s string, maxLine int) string {
	words := strings.Fields(s)
	var lines []string
	var line []string
	length := -1

	for _,w := range words {
		wordLength := utf8.RuneCountInString(w) + 1
		if length + wordLength > maxLine && length != 0 {
			lines = append(lines, strings.Join(line, " "))
			line = line[:0]
			length = -1
		}
		line = append(line, w)
		length += wordLength
	}
	lines = append(lines, strings.Join(line, " "))

	return strings.Join(lines, "\n")
}

var endOfSentence = regexp.MustCompile("[\\.\\?!]['\"]?$")

// IsEndOfSentence returns a boolean indicating whether a word ends
// with sentence-ending punctuation marks.
var IsEndOfSentence = endOfSentence.MatchString

var vowelStart = regexp.MustCompile("^[aeiou]")

// Article returns the appropriate indefinite article for a noun.
func Article(w string) string {
	if vowelStart.MatchString(w) {
		return "an"
	} else {
		return "a"
	}
}

// Capitalize returns its input with the first letter uppercased.
func Capitalize(w string) string {
	parts := strings.SplitN(w, "", 2)
	if len(parts) < 2 {
		return strings.ToUpper(w)
	} else {
		return fmt.Sprint(strings.ToUpper(parts[0]), parts[1])
	}
}
