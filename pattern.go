package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Quick and dirty parse function for setp patterns.
// TODO: might replace with something that can produce decent error messages.
func parsePattern(timeSig timeSig, input string) ([]*node, error) {
	var nodes []*node
	patterns := strings.Split(input, "|")
	for _, p := range patterns {
		root := &node{}
		curr := root
		divs := strings.Split(strings.TrimSpace(p), "/")
		for i, div := range divs {
			curr.level = int(float64(timeSig.denom) * math.Pow(2.0, float64(i)))

			switch d := strings.TrimSpace(div); {
			case regexp.MustCompile("\\d-\\d").MatchString(d):
				parts := strings.Split(d, "-")
				start, err := strconv.Atoi(parts[0])
				if err != nil {
					return nil, err
				}
				end, err := strconv.Atoi(parts[1])
				if err != nil {
					return nil, err
				}
				curr.selector = rangeExpr{
					start: start,
					end:   end,
				}
			case regexp.MustCompile("\\d,?").MatchString(d):
				parts := strings.Split(d, ",")
				list := listExpr{}
				for _, s := range parts {
					n, err := strconv.Atoi(s)
					if err != nil {
						return nil, err
					}
					list = append(list, n)
				}
				curr.selector = list
			case d == "*":
				curr.selector = starExpr{}
			default:
				if d == "" {
					continue
				}
				return nil, fmt.Errorf("syntax error: %q", d)
			}

			if i == len(divs)-1 {
				// don't create new node if this is the last one
				break
			}

			next := &node{}
			curr.next = next
			curr = next
		}
		nodes = append(nodes, root)
	}
	return nodes, nil
}

type node struct {
	level    int        // 1 is whole note, 4 is quarter note etc.
	selector selectExpr // selects notes on this division level
	next     *node
}

type selectExpr interface {
	match(i int) bool
}

type rangeExpr struct {
	start, end int
}

func (r rangeExpr) match(i int) bool {
	return i >= r.start && i <= r.end
}

type starExpr struct{}

func (starExpr) match(_ int) bool { return true }

type listExpr []int

func (l listExpr) match(i int) bool {
	for _, k := range l {
		if k == i {
			return true
		}
	}
	return false
}

func (n *node) sequence(t timeSig, stepSize int, seq []int) {
	if n.next != nil {
		n.next.sequence(t, stepSize, seq)
	}

	skip := stepSize / n.level
	notesPerBeat := n.level / t.denom

	for note, steps := 0, 0; note < len(seq); note += skip {
		// calculate a note number relative to other notes on the same division, e.g.
		// the 16th notes within a beat are numbered 0 to 3
		noteNum := steps % notesPerBeat
		if notesPerBeat == 1 {
			noteNum = steps
		}
		steps++

		// add 1 because match expects note numbers to start at 1
		if n.selector.match(noteNum + 1) {
			// only write to sequence when on a leaf node
			if n.next == nil {
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
