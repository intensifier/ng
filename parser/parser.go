// Copyright 2015 The Numgrad Authors. All rights reserved.
// See the LICENSE file for rights to use this source code.

package parser

import (
	"bytes"
	"fmt"
	"io"
)

func ParseExpr(src []byte) (expr Expr, err error) {
	p := newParser(src)
	if err := p.s.Next(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	expr = p.parseExpr(false)
	if len(p.err) > 0 {
		err = Errors(p.err)
	}
	if err == nil && p.s.err != io.EOF {
		err = p.s.err
	}
	return expr, err
}

type parser struct {
	s   *Scanner
	err []Error
}

func newParser(src []byte) *parser {
	p := &parser{
		s: NewScanner(src),
	}

	return p
}

func (p *parser) next() {
	p.s.Next()
	if p.s.Token == Comment {
		p.next()
	}
}

func (p *parser) parseExpr(lhs bool) Expr {
	return p.parseBinaryExpr(lhs, 1)
}

func (p *parser) parseBinaryExpr(lhs bool, minPrec int) Expr {
	x := p.parseUnaryExpr(lhs)
	for prec := p.s.Token.Precedence(); prec >= minPrec; prec-- {
		for {
			op := p.s.Token
			if op.Precedence() != prec {
				break
			}
			p.next()
			y := p.parseBinaryExpr(false, prec+1)
			// TODO: distinguish expr from types, when we have types
			// TODO record position
			x = &BinaryExpr{
				Op:    op,
				Left:  x,
				Right: y,
			}
		}
	}
	return x
}

func (p *parser) parseUnaryExpr(lhs bool) Expr {
	switch p.s.Token {
	case Add, Sub, Not:
		op := p.s.Token
		p.next()
		if p.s.err != nil {
			return &BadExpr{Error: p.s.err}
		}
		x := p.parseUnaryExpr(false)
		// TODO: distinguish expr from types, when we have types
		return &UnaryExpr{Op: op, Expr: x}
	case Mul:
		p.next()
		x := p.parseUnaryExpr(false)
		return &UnaryExpr{Op: Mul, Expr: x}
	default:
		return p.parsePrimaryExpr(lhs)
	}
}

func (p *parser) expectCommaOr(otherwise Token, msg string) bool {
	switch {
	case p.s.Token == Comma:
		return true
	case p.s.Token != otherwise:
		p.error("missing ',' in " + msg + " (got " + p.s.Token.String() + ")")
		return true // fake it
	default:
		return false
	}
}

func (p *parser) parseArgs() []Expr {
	p.expect(LeftParen)
	p.next()
	var args []Expr
	for p.s.Token != RightParen && p.s.r > 0 {
		args = append(args, p.parseExpr(false))
		if !p.expectCommaOr(RightParen, "arguments") {
			break
		}
		p.next()
	}
	p.expect(RightParen)
	p.next()
	return args
}

func (p *parser) parsePrimaryExpr(lhs bool) Expr {
	x := p.parseOperand(lhs)
	for {
		switch p.s.Token {
		case Period:
			p.next()
			switch p.s.Token {
			case Identifier:
				panic("TODO parse selector")
			case LeftParen:
				panic("TODO parse type assertion")
			default:
				panic("TODO expect selector type assertion")
			}
		case LeftBracket:
			panic("TODO array index")
		case LeftParen:
			args := p.parseArgs()
			return &CallExpr{Func: x, Args: args}
		case LeftBrace:
			panic("TODO could be composite literal")
			return x
		default:
			return x
		}
	}

	return x
}

func (p *parser) parseOperand(lhs bool) Expr {
	switch p.s.Token {
	case Identifier:
		return p.parseIdent()
	case Int, Float, Imaginary, String:
		x := &BasicLiteral{Value: p.s.Literal}
		p.next()
		return x
	case LeftParen:
		p.next()
		expr := p.parseExpr(false) // TODO or a type?
		p.expect(RightParen)
		return &UnaryExpr{Op: LeftParen, Expr: expr}
	}
	// TODO: other cases, eventually Func, etc

	p.next()
	return &BadExpr{p.error("expected operand")}
}

type Errors []Error

func (e Errors) Error() string {
	buf := new(bytes.Buffer)
	buf.WriteString("numgrad: parser erorrs:\n")
	for _, err := range e {
		fmt.Fprintf(buf, "off %5d: %v\n", err.Offset, err.Msg)
	}
	return buf.String()
}

type Error struct {
	Offset int
	Msg    string
}

func (e Error) Error() string {
	return fmt.Sprintf("numgrad: parser: %s (off %d)", e.Msg, e.Offset)
}

func (p *parser) error(msg string) error {
	err := Error{
		Offset: p.s.Offset,
		Msg:    msg,
	}
	p.err = append(p.err, err)
	return err
}

func (p *parser) expect(t Token) bool {
	met := t == p.s.Token
	if !met {
		p.error(fmt.Sprintf("expected %q, found %q", t, p.s.Token))
	}
	return met
}

func (p *parser) parseIdent() *Ident {
	name := "_"
	if p.expect(Identifier) {
		name = p.s.Literal.(string)
	}
	p.next()
	return &Ident{Name: name}
}