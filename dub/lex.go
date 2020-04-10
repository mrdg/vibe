package dub

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokenType int

const (
	typeUnknown tokenType = iota
	typeInt
	typeFloat
	typeIdentifier
	typeQuote
	typeComma
	typeColon
	typeSlash
	typeAsterisk
	typeSemicolon
	typeEOF
)

const eof = -1

var simpleTokens = map[rune]tokenType{
	'\'': typeQuote,
	',':  typeComma,
	':':  typeColon,
	'/':  typeSlash,
	'*':  typeAsterisk,
	';':  typeSemicolon,
}

type token struct {
	typ  tokenType
	pos  int
	text string
}

func lex(input string) ([]token, error) {
	l := &lexer{input: input}
	return l.lex()
}

type lexer struct {
	input string

	width int
	start int
	pos   int

	tokens []token
	err    error
}

func (l *lexer) lex() ([]token, error) {
	for {
		switch r := l.next(); {
		case r == eof:
			l.yieldToken(typeEOF)
			return l.tokens, l.err
		case unicode.IsLetter(r):
			l.lexIdentifier()
		case l.isNumber(r):
			l.lexNumber()
		case r == ' ':
			l.ignoreSpace()
		default:
			if typ, ok := simpleTokens[r]; ok {
				l.yieldToken(typ)
			} else {
				l.invalidChar(r)
			}
		}
		if l.err != nil {
			return l.tokens, l.err
		}
	}
}

func (l *lexer) next() rune {
	if len(l.input) == l.pos {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) yieldToken(t tokenType) {
	s := l.input[l.start:l.pos]
	l.tokens = append(l.tokens, token{t, l.pos, s})
	l.start = l.pos
	l.width = 0
}

func (l *lexer) errorf(format string, args ...interface{}) {
	l.err = fmt.Errorf(format, args...)
}

func (l *lexer) invalidChar(r rune) {
	l.errorf("unexpected character: %#U", r)
}

func (l *lexer) ignoreSpace() {
	for l.peek() == ' ' {
		l.next()
	}
	l.start = l.pos
}

func (l *lexer) take(set string) int {
	var n int
	for strings.IndexRune(set, l.next()) >= 0 {
		n++
	}
	l.backup()
	return n
}

func (l *lexer) accept(set string) bool {
	if strings.IndexRune(set, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) lexIdentifier() {
	for {
		switch r := l.next(); {
		case unicode.IsLetter(r) || r == '_':
		default:
			if r == ' ' || r == eof {
				l.backup()
				l.yieldToken(typeIdentifier)
			} else {
				l.invalidChar(r)
			}
			return
		}
	}
}

const digits = "0123456789"

// lexNumber assumes input has been checked to contain at least one digit using isNumber
func (l *lexer) lexNumber() {
	// Back up to see a possible leading '.'
	l.backup()

	l.accept("-")
	l.take(digits)
	isFloat := l.accept(".")
	l.take(digits)

	r := l.peek()
	if r == ' ' || r == '/' || r == ':' || r == ',' || r == eof {
		if isFloat {
			l.yieldToken(typeFloat)
		} else {
			l.yieldToken(typeInt)
		}
	} else {
		l.invalidChar(r)
	}
}

func (l *lexer) isNumber(r rune) bool {
	if isDigit(r) {
		return true
	}
	peek := l.peek()
	if r == '-' {
		if isDigit(peek) {
			return true
		}
		if peek == '.' {
			l.next()
			defer l.backup()
			return isDigit(l.peek())
		}
	}
	return r == '.' && isDigit(peek)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
