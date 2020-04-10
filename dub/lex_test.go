package dub

import "testing"

func TestLexer(t *testing.T) {
	type test struct {
		input  string
		expect []token
	}
	tests := []test{
		{
			input: "A '* 2",
			expect: []token{
				token{typ: typeIdentifier, text: "A"},
				token{typ: typeQuote, text: "'"},
				token{typ: typeAsterisk, text: "*"},
				token{typ: typeInt, text: "2"},
				token{typ: typeEOF},
			},
		},
		{
			input: "A 1 2",
			expect: []token{
				token{typ: typeIdentifier, text: "A"},
				token{typ: typeInt, text: "1"},
				token{typ: typeInt, text: "2"},
				token{typ: typeEOF},
			},
		},
		{
			input: "'1:2 /    / 3,4",
			expect: []token{
				token{typ: typeQuote, text: "'"},
				token{typ: typeInt, text: "1"},
				token{typ: typeColon, text: ":"},
				token{typ: typeInt, text: "2"},
				token{typ: typeSlash, text: "/"},
				token{typ: typeSlash, text: "/"},
				token{typ: typeInt, text: "3"},
				token{typ: typeComma, text: ","},
				token{typ: typeInt, text: "4"},
				token{typ: typeEOF},
			},
		},
		{
			input: "1.0",
			expect: []token{
				token{typ: typeFloat, text: "1.0"},
				token{typ: typeEOF},
			},
		},
		{
			input: "-1.",
			expect: []token{
				token{typ: typeFloat, text: "-1."},
				token{typ: typeEOF},
			},
		},
		{
			input: "-.1",
			expect: []token{
				token{typ: typeFloat, text: "-.1"},
				token{typ: typeEOF},
			},
		},
		{
			input: `command "this is a string" 1`,
			expect: []token{
				token{typ: typeIdentifier, text: "command"},
				token{typ: typeString, text: `"this is a string"`},
				token{typ: typeInt, text: "1"},
				token{typ: typeEOF},
			},
		},
	}
	for _, test := range tests {
		t.Log(test.input)
		tokens, err := lex(test.input)
		if err != nil {
			t.Errorf("unexpected lex error: %v", err)
			continue
		}
		if len(tokens) != len(test.expect) {
			t.Fatalf("token mismatch: \nwant: %+v, \ngot:  %+v", test.expect, tokens)
		}
		for i, got := range tokens {
			want := test.expect[i]
			if want.typ != got.typ {
				t.Errorf("wrong type: want %v, got %v", want, got)
			}
			if want.text != got.text {
				t.Errorf("wrong text: want %v, got %v", want, got)
			}
		}
	}
}

func TestLexerErrors(t *testing.T) {
	for _, input := range []string{
		"a -",
		"a .-",
	} {
		_, err := lex(input)
		if err == nil {
			t.Errorf("expected error for input: %q", input)
		}
	}
}
