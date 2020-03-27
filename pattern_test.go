package main

import (
	"reflect"
	"testing"
)

type exprTest struct {
	root     node
	timeSig  timeSig
	stepSize int
	want     []int
	debug    bool
}

func TestExpression(t *testing.T) {
	tests := []exprTest{
		{
			root: node{
				level:    4,
				selector: rangeExpr{start: 1, end: 4},
			},
			timeSig:  timeSig{num: 4, denom: 4},
			stepSize: 16,
			want:     []int{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0},
		},
		{
			root: node{
				level:    4,
				selector: rangeExpr{start: 1, end: 2},
				next: &node{
					level:    16,
					selector: rangeExpr{start: 1, end: 4},
				},
			},
			timeSig:  timeSig{num: 4, denom: 4},
			stepSize: 16,
			want:     []int{1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			root: node{
				level:    4,
				selector: starExpr{},
				next: &node{
					level:    16,
					selector: listExpr{3, 4},
				},
			},
			timeSig:  timeSig{num: 4, denom: 4},
			stepSize: 16,
			want:     []int{0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1},
		},
		{
			root: node{
				level:    4,
				selector: starExpr{},
				next: &node{
					level:    8,
					selector: listExpr{2},
				},
			},
			timeSig:  timeSig{num: 4, denom: 4},
			stepSize: 16,
			want:     []int{0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0},
		},
		{
			root: node{
				level:    4,
				selector: listExpr{5},
			},
			timeSig:  timeSig{num: 5, denom: 4},
			stepSize: 16,
			want:     []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		},
		{
			root: node{
				level:    8,
				selector: starExpr{},
			},
			timeSig:  timeSig{num: 7, denom: 8},
			stepSize: 16,
			want:     []int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0},
		},
		{
			root: node{
				level:    8,
				selector: starExpr{},
				next: &node{
					level:    16,
					selector: listExpr{2},
				},
			},
			timeSig:  timeSig{num: 7, denom: 8},
			stepSize: 16,
			want:     []int{0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1},
		},
		{
			root: node{
				level:    4,
				selector: starExpr{},
			},
			timeSig:  timeSig{num: 4, denom: 4},
			stepSize: 32,
			want: []int{
				1, 0, 0, 0,
				0, 0, 0, 0,
				1, 0, 0, 0,
				0, 0, 0, 0,
				1, 0, 0, 0,
				0, 0, 0, 0,
				1, 0, 0, 0,
				0, 0, 0, 0,
			},
		},
	}

	for _, test := range tests {
		got := make([]int, len(test.want))
		test.root.sequence(test.timeSig, test.stepSize, got)
		if !reflect.DeepEqual(test.want, got) {
			t.Fatalf("wrong sequence: \nwant: %v\ngot:  %v", test.want, got)
		}
	}
}
