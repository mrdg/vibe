package dub

import (
	"fmt"
	"strconv"
)

type Node interface {
	isNode()
}

func (Identifier) isNode() {}
func (Int) isNode()        {}
func (Float) isNode()      {}
func (String) isNode()     {}
func (MatchExpr) isNode()  {}

type Command struct {
	Name Identifier
	Args []Node
}

type Identifier string
type Int int
type Float float64
type String string
type MatchExpr struct {
	matchers []matchItem
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

func (p *parser) peek() token {
	t := p.next()
	p.pos--
	return t
}

func (p *parser) backup() {
	p.pos--
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
		case typeFloat:
			f, err := strconv.ParseFloat(token.text, 64)
			if err != nil {
				return cmd, err
			}
			arg = Float(f)
		case typeInt:
			n, err := strconv.Atoi(token.text)
			if err != nil {
				return cmd, err
			}
			arg = Int(n)
		case typeQuote:
			matchExpr, err := p.matchExpr(p.next())
			if err != nil {
				return cmd, err
			}
			arg = matchExpr
		default:
			return cmd, unexpected(token)
		}
		cmd.Args = append(cmd.Args, arg)
	}
	return cmd, nil
}

func (p *parser) matchExpr(start token) (MatchExpr, error) {
	match := MatchExpr{}
	current := matchItem{}

	for token := start; token.typ != typeEOF; token = p.next() {
		switch token.typ {
		case typeInt:
			switch next := p.peek(); next.typ {
			case typeComma, typeSlash, typeEOF:
				list, err := p.listMatch(token)
				if err != nil {
					return match, err
				}
				current.matcher = list
			case typeColon:
				p.next()
				start, err := strconv.Atoi(token.text)
				if err != nil {
					return match, err
				}
				t := p.next()
				if t.typ != typeInt {
					return match, unexpected(token)
				}
				end, err := strconv.Atoi(t.text)
				if err != nil {
					return match, err
				}
				current.matcher = rangeMatch{start: start, end: end}
			default:
				return match, unexpected(token)
			}
		case typeAsterisk:
			current.matcher = matchAll
		default:
			return match, unexpected(token)
		}

		if p.peek().typ == typeSlash {
			match.matchers = append(match.matchers, current)
			current = matchItem{level: current.level + 1}
			p.next()
		}
		for p.peek().typ == typeSlash {
			p.next()
			current.level++
		}
	}

	p.backup()
	match.matchers = append(match.matchers, current)
	return match, nil
}

func (p *parser) listMatch(start token) (listMatch, error) {
	var list listMatch
	current := start
	for {
		switch current.typ {
		case typeInt:
			n, err := strconv.Atoi(current.text)
			if err != nil {
				return list, err
			}
			list = append(list, n)
		case typeComma: // ignore
		default:
			p.backup()
			if current.typ != typeEOF && current.typ != typeSlash {
				return list, unexpected(current)
			}
			return list, nil
		}
		current = p.next()
	}
}

func unexpected(t token) error {
	return fmt.Errorf("unexpected token %q at position %d", t.text, t.pos)
}
