package dub

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestEvalMatchExpr(t *testing.T) {
	type test struct {
		input    string
		time     string
		stepSize string
		expect   []int
	}
	tests := []test{
		{
			input:    "2,4/*",
			time:     "4/4",
			stepSize: "16",
			expect:   []int{0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 1, 0, 1, 0},
		},
		{
			input:    "1:4",
			time:     "4/4",
			stepSize: "16",
			expect:   []int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0},
		},
		{
			input:    "1:2//1:4",
			time:     "4/4",
			stepSize: "16",
			expect:   []int{1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			input:    "*//3,4",
			time:     "4/4",
			stepSize: "16",
			expect:   []int{0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1},
		},

		{
			input:    "*/2",
			time:     "4/4",
			stepSize: "16",
			expect:   []int{0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0},
		},
		{
			input:    "5",
			time:     "5/4",
			stepSize: "16",
			expect:   []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		},
		{
			input:    "*",
			time:     "7/8",
			stepSize: "16",
			expect:   []int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0},
		},
		{
			input:    "*/2",
			time:     "7/8",
			stepSize: "16",
			expect:   []int{0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0},
		},
		{
			input:    "*/2,3",
			time:     "4/4",
			stepSize: "16T",
			expect: []int{
				0, 0, 1, 0, 1, 0,
				0, 0, 1, 0, 1, 0,
				0, 0, 1, 0, 1, 0,
				0, 0, 1, 0, 1, 0,
			},
		},
		{
			input:    "*/*",
			time:     "9/8",
			stepSize: "16T",
			expect: []int{
				1, 0, 1, 0, 1, 0,
				1, 0, 1, 0, 1, 0,
				1, 0, 1, 0, 1, 0,
				1, 0, 1, 0, 1, 0,
				1, 0, 1,
			},
		},
		{
			input:    "*",
			time:     "4/4",
			stepSize: "32",
			expect: []int{
				1, 0, 0, 0, 0, 0, 0, 0,
				1, 0, 0, 0, 0, 0, 0, 0,
				1, 0, 0, 0, 0, 0, 0, 0,
				1, 0, 0, 0, 0, 0, 0, 0,
			},
		},
	}
	for _, test := range tests {
		input := "a '" + test.input // make the input a valid dub commad
		command, err := Parse(input)
		if err != nil {
			t.Error(err)
			continue
		}
		expr := command.Args[0].(MatchExpr)

		_, denom, err := parseTimeSignature(test.time)
		if err != nil {
			t.Error(err)
			continue
		}

		num := strings.TrimSuffix(test.stepSize, "T")
		triplets := len(num) != len(test.stepSize)
		stepSize, err := strconv.Atoi(num)
		if err != nil {
			t.Error(err)
			continue
		}

		got, _ := EvalMatchExpr(expr, denom, len(test.expect), stepSize, triplets)
		if !reflect.DeepEqual(test.expect, got) {
			t.Errorf("seq mismatch:\nwant %v\ngot: %v", test.expect, got)
		}
	}
}

func parseTimeSignature(s string) (int, int, error) {
	var num, denom int
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return num, denom, fmt.Errorf("not a valid time signature: %s", s)
	}
	var err error
	num, err = strconv.Atoi(parts[0])
	if err != nil {
		return num, denom, fmt.Errorf("bad numerator %s: %s", parts[0], err)
	}
	denom, err = strconv.Atoi(parts[1])
	if err != nil {
		return num, denom, fmt.Errorf("bad denominator %s: %s", parts[1], err)
	}
	return num, denom, nil
}
