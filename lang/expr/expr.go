// Copyright 2015 The Numgrad Authors. All rights reserved.
// See the LICENSE file for rights to use this source code.

// Package expr defines data structures representing Numengrad expressions.
package expr

import (
	"bytes"
	"fmt"
	"strings"

	"numgrad.io/lang/tipe"
	"numgrad.io/lang/token"
)

type Expr interface {
	Sexp() string
	expr()
}

type Binary struct {
	Op    token.Token // Add, Sub, Mul, Div, Rem, Pow, And, Or, Equal, NotEqual, Less, Greater
	Left  Expr
	Right Expr
}

type Unary struct {
	Op   token.Token // Not, Mul (deref), Ref, LeftParen, Range
	Expr Expr
}

type Bad struct {
	Error error
}

type Selector struct {
	Left  Expr
	Right *Ident
}

type BasicLiteral struct {
	Value interface{} // string, *big.Int, *big.Float
}

type FuncLiteral struct {
	Name            string // may be empty
	ReceiverName    string // if non-empty, this is a method
	PointerReceiver bool
	Type            *tipe.Func
	ParamNames      []string
	ResultNames     []string
	Body            interface{} // *stmt.Block, breaking the package import cycle
}

type CompLiteral struct {
	Type     tipe.Type
	Names    []string // TODO use
	Elements []Expr
}

type TableLiteral struct {
	Type     *tipe.Table
	ColNames []Expr
	Rows     [][]Expr
}

type Ident struct {
	Name string
	// Type tipe.Type
}

type Call struct {
	Func Expr
	Args []Expr
}

type Range struct {
	Start Expr
	End   Expr
	Exact Expr
}

type TableIndex struct {
	Expr     Expr
	ColNames []string
	Cols     Range
	Rows     Range
}

var (
	_ = Expr((*Binary)(nil))
	_ = Expr((*Unary)(nil))
	_ = Expr((*Bad)(nil))
	_ = Expr((*Selector)(nil))
	_ = Expr((*BasicLiteral)(nil))
	_ = Expr((*FuncLiteral)(nil))
	_ = Expr((*CompLiteral)(nil))
	_ = Expr((*TableLiteral)(nil))
	_ = Expr((*Ident)(nil))
	_ = Expr((*Call)(nil))
	_ = Expr((*TableIndex)(nil))
)

func (e *Binary) expr()       {}
func (e *Unary) expr()        {}
func (e *Bad) expr()          {}
func (e *Selector) expr()     {}
func (e *BasicLiteral) expr() {}
func (e *FuncLiteral) expr()  {}
func (e *CompLiteral) expr()  {}
func (e *TableLiteral) expr() {}
func (e *Ident) expr()        {}
func (e *Call) expr()         {}
func (e *TableIndex) expr()   {}

func (e *Binary) Sexp() string {
	if e == nil {
		return "nilbin"
	}
	return fmt.Sprintf("(%s %s %s)", e.Op, exprSexp(e.Left), exprSexp(e.Right))
}
func (e *Unary) Sexp() string {
	if e == nil {
		return "nilunary"
	}
	return fmt.Sprintf("(%s %s)", e.Op, exprSexp(e.Expr))
}
func (e *Bad) Sexp() string { return fmt.Sprintf("(bad %v)", e.Error) }
func (e *Selector) Sexp() string {
	if e == nil {
		return "nilsel"
	}
	return fmt.Sprintf("(sel %s %s)", exprSexp(e.Left), exprSexp(e.Right))
}
func (e *BasicLiteral) Sexp() string {
	if e == nil {
		return "nillit"
	}
	return fmt.Sprintf("(lit %T %s)", e.Value, e.Value)
}
func (e *Ident) Sexp() string {
	if e == nil {
		return "nilident"
	}
	return fmt.Sprintf("%s", e.Name)
}
func (e *Call) Sexp() string {
	if e == nil {
		return "nilcall"
	}
	return fmt.Sprintf("(call %s %s)", exprSexp(e.Func), exprsStr(e.Args))
}

func (e *FuncLiteral) Sexp() string {
	body := "nilbody"
	if e.Body != nil {
		b, ok := e.Body.(interface {
			Sexp() string
		})
		if ok {
			body = b.Sexp()
		} else {
			body = fmt.Sprintf("badbody:%T", e.Body)
		}
	}
	if e.ReceiverName != "" {
		pointer := ""
		if e.PointerReceiver {
			pointer = "*"
		}
		return fmt.Sprintf("(method (%s%s) %s %s %s)", pointer, e.ReceiverName, e.Name, tipeSexp(e.Type), body)
	} else {
		return fmt.Sprintf("(func %s %s %s)", e.Name, tipeSexp(e.Type), body)
	}
}

func (e *CompLiteral) Sexp() string {
	names := ""
	if len(e.Names) > 0 {
		names = "(" + strings.Join(e.Names, " ") + ")"
	}
	return fmt.Sprintf("(comp %s %s %s)", tipeSexp(e.Type), names, exprsStr(e.Elements))
}
func (e *TableLiteral) Sexp() string {
	rows := ""
	for _, row := range e.Rows {
		rows += " " + exprsStr(row)
	}
	if rows != "" {
		rows = " (" + rows[1:] + ")"
	}
	return fmt.Sprintf("(table %s %s%s)", tipeSexp(e.Type), exprsStr(e.ColNames), rows)
}

func (e *TableIndex) Sexp() string {
	names := strings.Join(e.ColNames, `"|"`)
	if names != "" {
		names = ` "` + names + `"`
	}
	rangeSexp := func(r Range) string {
		rs := ""
		if r.Start != nil || r.End != nil {
			if r.Start != nil {
				rs += exprSexp(r.Start)
			}
			rs += ":"
			if r.End != nil {
				rs += exprSexp(r.End)
			}
		}
		exact := ""
		if r.Exact != nil {
			if rs != "" {
				rs += " "
			}
			exact = exprSexp(r.Exact)
		}
		return fmt.Sprintf("(%s%s)", rs, exact)
	}
	return fmt.Sprintf("(tableindex %s%s %s %s", exprSexp(e.Expr), names, rangeSexp(e.Cols), rangeSexp(e.Rows))
}

func tipeSexp(t tipe.Type) string {
	if t == nil {
		return "niltype"
	}
	return t.Sexp()
}

func exprSexp(e Expr) string {
	if e == nil {
		return "nilexpr"
	}
	return e.Sexp()
}

func exprsStr(e []Expr) string {
	buf := new(bytes.Buffer)
	for i, arg := range e {
		if i > 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(exprSexp(arg))
	}
	return buf.String()
}