package dub

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	type test struct {
		input string
		want  Command
	}
	tests := []test{
		{
			input: "A '1",
			want: Command{
				Name: Identifier("A"),
				Args: []Node{
					MatchExpr{
						matchers: []matchItem{
							{level: 0, matcher: listMatch{1}},
						},
					},
				},
			},
		},
		{
			input: "A '*/*",
			want: Command{
				Name: Identifier("A"),
				Args: []Node{
					MatchExpr{
						matchers: []matchItem{
							{level: 0, matcher: matchAll},
							{level: 1, matcher: matchAll},
						},
					},
				},
			},
		},
		{
			input: "A '*//3,4",
			want: Command{
				Name: Identifier("A"),
				Args: []Node{
					MatchExpr{
						matchers: []matchItem{
							{level: 0, matcher: matchAll},
							{level: 2, matcher: listMatch{3, 4}},
						},
					},
				},
			},
		},
		{
			input: "A '1,2//3:4",
			want: Command{
				Name: Identifier("A"),
				Args: []Node{
					MatchExpr{
						matchers: []matchItem{
							{level: 0, matcher: listMatch{1, 2}},
							{level: 2, matcher: rangeMatch{start: 3, end: 4}},
						},
					},
				},
			},
		},
		{
			input: `load "a/file.wav"`,
			want: Command{
				Name: Identifier("load"),
				Args: []Node{String("a/file.wav")},
			},
		},
		{
			input: `load ""`,
			want: Command{
				Name: Identifier("load"),
				Args: []Node{String("")},
			},
		},
	}
	for _, test := range tests {
		t.Log(test.input)
		got, err := Parse(test.input)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(test.want, got) {
			t.Errorf("\nwant: %+v\ngot:  %+v", test.want, got)
		}
	}
}
