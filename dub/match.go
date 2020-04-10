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

func EvalMatchExpr(expr MatchExpr, numerator, denominator, stepSize int) ([]int, error) {
	seq := make([]int, (stepSize/denominator)*numerator)

	for i := len(expr.matchers) - 1; i >= 0; i-- {
		item := expr.matchers[i]
		level := int(float64(denominator) * math.Pow(2.0, float64(item.level)))
		if level > stepSize {
			return nil, fmt.Errorf("can't match on %d notes with step size %d", level, stepSize)
		}
		skip := stepSize / level
		notesPerBeat := level / denominator

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
				for i := note; i < note+skip; i++ {
					seq[i] = 0
				}
			}
		}
	}
	return seq, nil
}
