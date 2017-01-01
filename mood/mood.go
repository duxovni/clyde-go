// Copyright 2016 Sam Dukhovni <dukhovni@mit.edu>
//
// Adapted from clyde.pl by cat@mit.edu
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
// mood defines a set of moods for clyde.

package mood

// Mood is a type for Clyde's moods.
type Mood int

// Clyde's 8 moods.
const (
	Yucky	Mood = 0
	Angry	Mood = 1
	Unhappy	Mood = 2
	Lonely	Mood = 3
	Turnip	Mood = 4
	Ok	Mood = 5
	Good	Mood = 6
	Great	Mood = 7
	max	Mood = 7
)

// Better returns the first mood better than the current mood.
func (m Mood) Better() Mood {
	if m + 1 > max {
		return max
	} else {
		return m + 1
	}
}

// Worse returns the first mood worse than the current mood.
func (m Mood) Worse() Mood {
	if m - 1 < 0 {
		return 0
	} else {
		return m - 1
	}
}

// AtLeastOk returns Ok if the current mood is less than Ok, otherwise
// it returns the current mood.
func (m Mood) AtLeastOk() Mood {
	if m < Ok {
		return Ok
	} else {
		return m
	}
}

// String returns a string describing the current mood, suitable for
// use in the sentence "I am _____".
func (m Mood) String() string {
	switch m {
	case Yucky:
		return "yucky"
	case Angry:
		return "angry"
	case Unhappy:
		return "unhappy"
	case Lonely:
		return "lonely"
	case Turnip:
		return "a turnip"
	case Ok:
		return "ok"
	case Good:
		return "good"
	case Great:
		return "great"
	default:
		return "ok"
	}
}

// Punc returns punctuation corresponding to the current mood,
// suitable for finishing the sentence "I am $mood".
func (m Mood) Punc() string {
	switch m {
	case Yucky:
		return "."
	case Angry:
		return "!"
	case Unhappy:
		return "."
	case Lonely:
		return " :("
	case Turnip:
		return "."
	case Ok:
		return "."
	case Good:
		return " :)"
	case Great:
		return "!"
	default:
		return "."
	}
}
