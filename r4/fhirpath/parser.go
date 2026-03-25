// Copyright 2026 the gofhir Authors
// SPDX-License-Identifier: Apache-2.0

package fhirpath

import "fmt"

// Node is an AST node in a FHIRPath expression.
type Node interface {
	nodeType() string
}

// LiteralNode represents a constant value.
type LiteralNode struct {
	Value any    // string, float64, int64, bool
	Raw   string // original text
}

func (n *LiteralNode) nodeType() string { return "literal" }

// IdentNode represents a field name or type name.
type IdentNode struct {
	Name string
}

func (n *IdentNode) nodeType() string { return "ident" }

// DotNode represents member access: left.right
type DotNode struct {
	Left  Node
	Right Node
}

func (n *DotNode) nodeType() string { return "dot" }

// IndexNode represents indexing: expr[index]
type IndexNode struct {
	Expr  Node
	Index Node
}

func (n *IndexNode) nodeType() string { return "index" }

// FunctionNode represents a function call: expr.func(args...)
type FunctionNode struct {
	Name string
	Args []Node
}

func (n *FunctionNode) nodeType() string { return "function" }

// UnaryNode represents a unary operation: -expr, not expr
type UnaryNode struct {
	Op   string
	Expr Node
}

func (n *UnaryNode) nodeType() string { return "unary" }

// BinaryNode represents a binary operation: left op right
type BinaryNode struct {
	Op    string
	Left  Node
	Right Node
}

func (n *BinaryNode) nodeType() string { return "binary" }

// TypeNode represents a type expression: expr is Type, expr as Type
type TypeNode struct {
	Op       string // "is" or "as"
	Expr     Node
	TypeName string
}

func (n *TypeNode) nodeType() string { return "type" }

// QuantityNode represents a quantity literal: 5 'mg'
type QuantityNode struct {
	Value float64
	Unit  string
}

func (n *QuantityNode) nodeType() string { return "quantity" }

// EmptyNode represents the empty collection {}
type EmptyNode struct{}

func (n *EmptyNode) nodeType() string { return "empty" }

// Parser builds an AST from tokens.
type Parser struct {
	tokens []Token
	pos    int
}

// Parse parses a FHIRPath expression string into an AST.
func Parse(expr string) (Node, error) {
	tokens, err := Lex(expr)
	if err != nil {
		return nil, fmt.Errorf("fhirpath lex: %w", err)
	}
	p := &Parser{tokens: tokens}
	node, err := p.parseExpression(0)
	if err != nil {
		return nil, fmt.Errorf("fhirpath parse: %w", err)
	}
	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("fhirpath: unexpected token %v at position %d", p.current(), p.current().Pos)
	}
	return node, nil
}

func (p *Parser) current() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: TokenEOF}
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	tok := p.current()
	if tok.Type != tt {
		return tok, fmt.Errorf("expected %v, got %v at position %d", tt, tok.Type, tok.Pos)
	}
	p.advance()
	return tok, nil
}

// Operator precedence (lower = binds looser)
func precedence(op string) int {
	switch op {
	case "implies":
		return 1
	case "or", "xor":
		return 2
	case "and":
		return 3
	case "in", "contains":
		return 4
	case "is", "as":
		return 5
	case "=", "!=", "~", "!~":
		return 6
	case "<", "<=", ">", ">=":
		return 7
	case "|":
		return 8
	case "&":
		return 9
	case "+", "-":
		return 10
	case "*", "/", "mod", "div":
		return 11
	default:
		return 0
	}
}

func (p *Parser) parseExpression(minPrec int) (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.current()
		op := tokenToOp(tok)
		if op == "" {
			break
		}
		prec := precedence(op)
		if prec < minPrec {
			break
		}

		p.advance()

		if op == "is" || op == "as" {
			// Type expression: expr is TypeName
			typeTok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, fmt.Errorf("expected type name after '%s'", op)
			}
			left = &TypeNode{Op: op, Expr: left, TypeName: typeTok.Value}
			continue
		}

		right, err := p.parseExpression(prec + 1)
		if err != nil {
			return nil, err
		}
		left = &BinaryNode{Op: op, Left: left, Right: right}
	}

	return left, nil
}

func tokenToOp(tok Token) string {
	switch tok.Type {
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
	case TokenEqual:
		return "="
	case TokenNotEqual:
		return "!="
	case TokenLess:
		return "<"
	case TokenLessEq:
		return "<="
	case TokenGreater:
		return ">"
	case TokenGreaterEq:
		return ">="
	case TokenEquivalent:
		return "~"
	case TokenNotEquivalent:
		return "!~"
	case TokenAnd:
		return "and"
	case TokenOr:
		return "or"
	case TokenXor:
		return "xor"
	case TokenImplies:
		return "implies"
	case TokenIn:
		return "in"
	case TokenContains:
		return "contains"
	case TokenIs:
		return "is"
	case TokenAs:
		return "as"
	case TokenMod:
		return "mod"
	case TokenDiv:
		return "div"
	case TokenPipe:
		return "|"
	case TokenAmpersand:
		return "&"
	default:
		return ""
	}
}

func (p *Parser) parseUnary() (Node, error) {
	tok := p.current()
	if tok.Type == TokenMinus {
		p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: "-", Expr: expr}, nil
	}
	if tok.Type == TokenNot {
		p.advance()
		// not can be a unary prefix: not expr
		// But in FHIRPath, not() is usually a function. Handle as unary.
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryNode{Op: "not", Expr: expr}, nil
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (Node, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.current()
		switch tok.Type {
		case TokenDot:
			p.advance()
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			node = &DotNode{Left: node, Right: right}

		case TokenLBracket:
			p.advance()
			index, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokenRBracket); err != nil {
				return nil, err
			}
			node = &IndexNode{Expr: node, Index: index}

		default:
			return node, nil
		}
	}
}

func (p *Parser) parsePrimary() (Node, error) {
	tok := p.current()

	switch tok.Type {
	case TokenNumber:
		p.advance()
		// Check for quantity literal: number followed by string (unit)
		if p.current().Type == TokenString {
			unit := p.advance()
			num := parseNumber(tok.Value)
			var f float64
			switch n := num.(type) {
			case int64:
				f = float64(n)
			case float64:
				f = n
			}
			return &QuantityNode{Value: f, Unit: unit.Value}, nil
		}
		return &LiteralNode{Value: parseNumber(tok.Value), Raw: tok.Value}, nil

	case TokenString:
		p.advance()
		return &LiteralNode{Value: tok.Value, Raw: tok.Value}, nil

	case TokenTrue:
		p.advance()
		return &LiteralNode{Value: true, Raw: "true"}, nil

	case TokenFalse:
		p.advance()
		return &LiteralNode{Value: false, Raw: "false"}, nil

	case TokenIdent, TokenNot, TokenContains, TokenIs, TokenAs, TokenIn:
		p.advance()
		name := tok.Value

		// Check if this is a function call: name(args)
		if p.current().Type == TokenLParen {
			p.advance() // skip (
			args, err := p.parseArgList()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokenRParen); err != nil {
				return nil, err
			}
			return &FunctionNode{Name: name, Args: args}, nil
		}

		// Keywords used as identifiers in dot context
		return &IdentNode{Name: name}, nil

	case TokenLParen:
		p.advance()
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token %v at position %d", tok, tok.Pos)
	}
}

func (p *Parser) parseArgList() ([]Node, error) {
	if p.current().Type == TokenRParen {
		return nil, nil
	}

	var args []Node
	arg, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	args = append(args, arg)

	for p.current().Type == TokenComma {
		p.advance()
		arg, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return args, nil
}

func parseNumber(s string) any {
	// Try integer first
	var i int64
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil && !containsDot(s) {
		return i
	}
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func containsDot(s string) bool {
	for _, c := range s {
		if c == '.' {
			return true
		}
	}
	return false
}
