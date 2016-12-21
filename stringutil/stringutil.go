// Copyright 2016 Sam Dukhovni <dukhovni@mit.edu>
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)

package stringutil

import (
	"strings"
	"unicode/utf8"
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
