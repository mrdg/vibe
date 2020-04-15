package dub

import (
	"fmt"
	"math"
)

type matchItem struct {
	level   int
	matcher matcher
}

type matcher interface {
	match(i int) bool
}

type rangeMatch struct {
	start, end int
}

func (r rangeMatch) match(i int) bool {
	return (i >= r.start || r.start == -1) && (i <= r.end || r.end == -1)
}

var matchAll = rangeMatch{-1, -1}

type listMatch []int

func (l listMatch) match(i int) bool {
	for _, k := range l {
		if k == i {
			return true
		}
	}
	return false
}

// NOTE: when playing triplets in a compound meter, '*/* will play each triplet eighth note instead the regular
// eighth notes on the beat.
func EvalMatchExpr(expr MatchExpr, denominator, numSteps, stepSize int, triplets bool) ([]int, error) {
	seq := make([]int, numSteps)

	for i := len(expr.matchers) - 1; i >= 0; i-- {
		item := expr.matchers[i]
		// NOTE: matching is always relative to quarter notes, i.e. '* always plays each quarter note
		// regardless of the time signature.
		level := int(4. * math.Pow(2., float64(item.level)))
		if level > stepSize {
			return nil, fmt.Errorf("can't match on %d notes with step size %d", level, stepSize)
		}
		skip := stepSize / level
		notesPerBeat := level / 4.
		if triplets {
			if item.level > 0 {
				notesPerBeat = level / 4 / 2 * 3
			} else {
				skip = stepSize / level / 2 * 3
			}
		}

		for note, steps := 0, 0; note < len(seq); note += skip {
			// calculate a note number relative to other notes on the same division, e.g.
			// the 16th notes within a beat are numbered 0 to 3
			noteNum := steps % notesPerBeat
			if notesPerBeat == 1 {
				noteNum = steps
			}
			steps++

			// add 1 because match expects note numbers to start at 1
			if item.matcher.match(noteNum + 1) {
				if i == len(expr.matchers)-1 {
					seq[note] = 1
				}
			} else {
				// zero steps that are unmatched by the current level
				max := note + skip
				if max >= len(seq) {
					max = len(seq)
				}
				for i := note; i < max; i++ {
					seq[i] = 0
				}
			}
		}
	}
	return seq, nil
}
