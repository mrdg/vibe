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
