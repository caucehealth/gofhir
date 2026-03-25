// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package fhirpath

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents the type of a lexer token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent         // field names, function names, type names
	TokenNumber        // integer or decimal literal
	TokenString        // 'quoted string'
	TokenDot           // .
	TokenLParen        // (
	TokenRParen        // )
	TokenLBracket      // [
	TokenRBracket      // ]
	TokenComma         // ,
	TokenPlus          // +
	TokenMinus         // -
	TokenStar          // *
	TokenSlash         // /
	TokenEqual         // =
	TokenNotEqual      // !=
	TokenLess          // <
	TokenLessEq        // <=
	TokenGreater       // >
	TokenGreaterEq     // >=
	TokenEquivalent    // ~
	TokenNotEquivalent // !~
	TokenAnd           // and
	TokenOr            // or
	TokenNot           // not (unary)
	TokenXor           // xor
	TokenImplies       // implies
	TokenIs            // is
	TokenAs            // as
	TokenIn            // in
	TokenContains      // contains (operator, not function)
	TokenTrue          // true
	TokenFalse         // false
	TokenMod           // mod
	TokenDiv           // div (integer division)
	TokenPipe          // |
	TokenAmpersand     // &
)

// Token is a single lexical token.
type Token struct {
	Type    TokenType
	Value   string
	Pos     int
}

func (t Token) String() string {
	return fmt.Sprintf("%v(%q)", t.Type, t.Value)
}

// keywords maps keyword strings to their token types.
var keywords = map[string]TokenType{
	"and":      TokenAnd,
	"or":       TokenOr,
	"not":      TokenNot,
	"xor":      TokenXor,
	"implies":  TokenImplies,
	"is":       TokenIs,
	"as":       TokenAs,
	"in":       TokenIn,
	"contains": TokenContains,
	"true":     TokenTrue,
	"false":    TokenFalse,
	"mod":      TokenMod,
	"div":      TokenDiv,
}

// Lexer tokenizes a FHIRPath expression.
type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

// Lex tokenizes the input expression.
func Lex(input string) ([]Token, error) {
	l := &Lexer{input: input}
	if err := l.lex(); err != nil {
		return nil, err
	}
	return l.tokens, nil
}

func (l *Lexer) lex() error {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		switch {
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			l.pos++

		case ch == '.':
			l.emit(TokenDot, ".")

		case ch == '(':
			l.emit(TokenLParen, "(")

		case ch == ')':
			l.emit(TokenRParen, ")")

		case ch == '[':
			l.emit(TokenLBracket, "[")

		case ch == ']':
			l.emit(TokenRBracket, "]")

		case ch == ',':
			l.emit(TokenComma, ",")

		case ch == '+':
			l.emit(TokenPlus, "+")

		case ch == '-':
			l.emit(TokenMinus, "-")

		case ch == '*':
			l.emit(TokenStar, "*")

		case ch == '/':
			l.emit(TokenSlash, "/")

		case ch == '|':
			l.emit(TokenPipe, "|")

		case ch == '&':
			l.emit(TokenAmpersand, "&")

		case ch == '~':
			l.emit(TokenEquivalent, "~")

		case ch == '=':
			l.emit(TokenEqual, "=")

		case ch == '!':
			if l.peek() == '=' {
				l.pos++
				l.emit(TokenNotEqual, "!=")
			} else if l.peek() == '~' {
				l.pos++
				l.emit(TokenNotEquivalent, "!~")
			} else {
				return fmt.Errorf("unexpected character '!' at position %d", l.pos)
			}

		case ch == '<':
			if l.peek() == '=' {
				l.pos++
				l.emit(TokenLessEq, "<=")
			} else {
				l.emit(TokenLess, "<")
			}

		case ch == '>':
			if l.peek() == '=' {
				l.pos++
				l.emit(TokenGreaterEq, ">=")
			} else {
				l.emit(TokenGreater, ">")
			}

		case ch == '\'':
			s, err := l.readString()
			if err != nil {
				return err
			}
			l.tokens = append(l.tokens, Token{Type: TokenString, Value: s, Pos: l.pos})

		case ch >= '0' && ch <= '9':
			num := l.readNumber()
			l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: num, Pos: l.pos})

		case isIdentStart(ch):
			ident := l.readIdent()
			tok := Token{Value: ident, Pos: l.pos}
			if tt, ok := keywords[ident]; ok {
				tok.Type = tt
			} else {
				tok.Type = TokenIdent
			}
			l.tokens = append(l.tokens, tok)

		case ch == '`':
			// Backtick-quoted identifier (for reserved words as field names)
			ident, err := l.readBacktickIdent()
			if err != nil {
				return err
			}
			l.tokens = append(l.tokens, Token{Type: TokenIdent, Value: ident, Pos: l.pos})

		default:
			return fmt.Errorf("unexpected character %q at position %d", ch, l.pos)
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Pos: l.pos})
	return nil
}

func (l *Lexer) emit(tt TokenType, val string) {
	l.tokens = append(l.tokens, Token{Type: tt, Value: val, Pos: l.pos})
	l.pos++
}

func (l *Lexer) peek() byte {
	if l.pos+1 < len(l.input) {
		return l.input[l.pos+1]
	}
	return 0
}

func (l *Lexer) readString() (string, error) {
	l.pos++ // skip opening '
	var b strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\\' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			switch next {
			case '\'':
				b.WriteByte('\'')
			case '\\':
				b.WriteByte('\\')
			case '/':
				b.WriteByte('/')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte('\\')
				b.WriteByte(next)
			}
			l.pos += 2
			continue
		}
		if ch == '\'' {
			l.pos++
			return b.String(), nil
		}
		b.WriteByte(ch)
		l.pos++
	}
	return "", fmt.Errorf("unterminated string at position %d", l.pos)
}

func (l *Lexer) readNumber() string {
	start := l.pos
	for l.pos < len(l.input) && (l.input[l.pos] >= '0' && l.input[l.pos] <= '9') {
		l.pos++
	}
	if l.pos < len(l.input) && l.input[l.pos] == '.' && l.pos+1 < len(l.input) && l.input[l.pos+1] >= '0' && l.input[l.pos+1] <= '9' {
		l.pos++ // skip '.'
		for l.pos < len(l.input) && (l.input[l.pos] >= '0' && l.input[l.pos] <= '9') {
			l.pos++
		}
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readIdent() string {
	start := l.pos
	for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
		l.pos++
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readBacktickIdent() (string, error) {
	l.pos++ // skip opening `
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '`' {
		l.pos++
	}
	if l.pos >= len(l.input) {
		return "", fmt.Errorf("unterminated backtick identifier at position %d", start)
	}
	ident := l.input[start:l.pos]
	l.pos++ // skip closing `
	return ident, nil
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '$' || ch > 127
}

func isIdentPart(ch byte) bool {
	r := rune(ch)
	return unicode.IsLetter(r) || unicode.IsDigit(r) || ch == '_'
}
