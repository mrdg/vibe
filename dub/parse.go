package dub

import (
	"fmt"
	"strconv"
)

type Node interface {
	isNode()
}

func (Identifier) isNode() {}
func (Number) isNode()     {}
func (String) isNode()     {}
func (Array) isNode()      {}
func (Tuple) isNode()      {}

type Identifier string
type Number float64
type String string
type Array []Node
type Tuple []Node

type Command struct {
	Name Identifier
	Args []Node
}

func Parse(input string) (Command, error) {
	tokens, err := lex(input)
	if err != nil {
		return Command{}, err
	}
	p := parser{tokens: tokens}
	return p.parse()
}

type parser struct {
	pos    int
	tokens []token
}

func (p *parser) next() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) parse() (Command, error) {
	var cmd Command
	token := p.next()
	if token.typ != typeIdentifier {
		return cmd, unexpected(token)
	}
	cmd.Name = Identifier(token.text)
	for token := p.next(); token.typ != typeEOF; token = p.next() {
		var arg Node
		switch token.typ {
		case typeIdentifier:
			arg = Identifier(token.text)
		case typeString:
			arg = String(token.text[1 : len(token.text)-1])
		case typeNumber:
			f, err := strconv.ParseFloat(token.text, 64)
			if err != nil {
				return cmd, err
			}
			arg = Number(f)
		case typeLeftBracket:
			array, err := p.array()
			if err != nil {
				return cmd, err
			}
			arg = array
		default:
			return cmd, unexpected(token)
		}
		cmd.Args = append(cmd.Args, arg)
	}
	return cmd, nil
}

func (p *parser) array() (Array, error) {
	var array Array
	var token token
	for token = p.next(); token.typ != typeEOF; token = p.next() {
		switch token.typ {
		case typeNumber:
			f, err := strconv.ParseFloat(token.text, 64)
			if err != nil {
				return array, err
			}
			array = append(array, Number(f))
		case typeRightBracket:
			return array, nil
		case typeLeftBracket:
			nestedArray, err := p.array()
			if err != nil {
				return nil, err
			}
			array = append(array, nestedArray)
		case typeLeftCurly:
			tuple, err := p.tuple()
			if err != nil {
				return nil, err
			}
			array = append(array, tuple)
		default:
			return nil, unexpected(token)
		}
	}
	return nil, unexpected(token)
}

func (p *parser) tuple() (Tuple, error) {
	var tuple Tuple
	var token token
	for token = p.next(); token.typ != typeEOF; token = p.next() {
		switch token.typ {
		case typeNumber:
			f, err := strconv.ParseFloat(token.text, 64)
			if err != nil {
				return tuple, err
			}
			tuple = append(tuple, Number(f))
		case typeRightCurly:
			return tuple, nil
		default:
			return nil, unexpected(token)
		}
	}
	return nil, unexpected(token)
}

func unexpected(t token) error {
	return fmt.Errorf("unexpected token %q at position %d", t.text, t.pos)
}
