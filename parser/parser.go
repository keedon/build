// Copyright 2015-2016 Sevki <s@sevki.org>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser // import "sevki.org/build/parser"

import (
	"fmt"
	"io"

	"sevki.org/build/token"

	"sevki.org/build/ast"
	"sevki.org/build/lexer"
)

type Parser struct {
	name     string
	path     string
	lexer    *lexer.Lexer
	state    stateFn
	peekTok  token.Token
	curTok   token.Token
	line     int
	Error    error
	Document *ast.File
	ptr      *ast.Func
	payload  map[string]interface{}
	typeName string
	stack    []stateFn
}

type stateFn func(*Parser) stateFn

func (p *Parser) peek() token.Token {
	return p.peekTok
}
func (p *Parser) next() token.Token {
	tok := p.peekTok
	p.peekTok = <-p.lexer.Tokens
	p.curTok = tok

	if tok.Type == token.Error {
		p.errorf("%q", tok)
	}

	return tok
}

func (p *Parser) errorf(format string, args ...interface{}) {
	p.curTok = token.Token{Type: token.Error}
	p.peekTok = token.Token{Type: token.EOF}
	p.Error = fmt.Errorf(format, args...)
}

func New(name, path string, r io.Reader) *Parser {
	p := &Parser{
		name:  name,
		path:  path,
		line:  0,
		lexer: lexer.New(name, r),
		Document: &ast.File{
			Path: path,
		},
	}

	return p
}

func (p *Parser) run() {
	p.next()
	for p.state = parseBuild; p.state != nil; {
		p.state = p.state(p)
	}
}

func parseBuild(p *Parser) stateFn {
	for p.peek().Type != token.EOF {
		return parseDecl
	}
	return nil
}

func parseDecl(p *Parser) stateFn {
	switch p.peek().Type {
	case token.Func:
		p.Document.Funcs = append(p.Document.Funcs, p.consumeFunc())
		return parseDecl
	case token.String:
		return parseVar
	}
	return nil
}
func parseVar(p *Parser) stateFn {
	t := p.next()
	if !p.isExpected(t, token.String) {
		return nil
	}
	if !p.isExpected(p.next(), token.Equal) {
		return nil
	}

	if p.Document.Vars == nil {
		p.Document.Vars = make(map[string]interface{})
	}

	switch p.peek().Type {
	case token.LeftBrac, token.String, token.Quote, token.True, token.False:
		p.Document.Vars[t.String()] = p.consumeNode()
	case token.Func:
		p.Document.Vars[t.String()] = p.consumeFunc()
	}
	if p.peek().Type == token.Plus {

		f := &ast.Func{
			Name: "addition",
		}
		f.File = p.name
		f.Line = t.Line
		f.Position = t.Start

		f.AnonParams = []interface{}{p.Document.Vars[t.String()]}

		p.Document.Vars[t.String()] = f

		for p.peek().Type == token.Plus {
			p.next()
			switch p.peek().Type {
			case token.String:
				f.AnonParams = append(
					f.AnonParams,
					ast.Variable{Key: p.next().String()},
				)
			case token.Quote:
				f.AnonParams = append(
					f.AnonParams,
					p.next().String(),
				)
			}

		}

	}

	return parseDecl
}
func (p *Parser) consumeNode() interface{} {
	switch p.peek().Type {
	case token.Quote:
		return p.next().String()
	case token.LeftBrac:
		x := p.consumeSlice()
		return x
	case token.Func:
		return p.consumeFunc()
	case token.True:
		return true
	case token.False:
		return false
	case token.String:
		return ast.Variable{Key: p.next().String()}
	}
	return nil
}
func (p *Parser) consumeParams(f *ast.Func) {
	for {
		switch p.peek().Type {
		case token.Quote, token.LeftBrac, token.Func:
			f.AnonParams = append(f.AnonParams, p.consumeNode())
		case token.String:
			t := p.next()
			if f.Params == nil {
				f.Params = make(map[string]interface{})
			}

			if p.expects(p.peek(), []token.Type{token.Colon, token.Equal}) {
				switch p.next().Type {
				case token.Colon:
				case token.Equal:
					f.Params[t.String()] = p.consumeNode()
				}
			} else {
				return
			}
		default:
			return
		}

		if p.expects(p.peek(), []token.Type{token.RightParen, token.Comma}) {
		DANGLING_COMMA:
			switch p.peek().Type {
			case token.RightParen:
				p.next()
				return
			case token.Comma:
				p.next()
				if p.peek().Type == token.RightParen {
					goto DANGLING_COMMA
				}
				continue
			}
		}

	}
}

func (p *Parser) consumeFunc() *ast.Func {
	t := p.next()
	if !p.isExpected(t, token.Func) {
		return nil
	}
	f := ast.Func{
		Name: t.String(),
	}

	f.File = p.name
	f.Line = t.Line
	f.Position = t.Start

	t = p.next()
	if !p.isExpected(t, token.LeftParen) {
		return nil
	}
	p.consumeParams(&f)
	return &f
}

func (p *Parser) consumeSlice() []interface{} {
	if !p.isExpected(p.next(), token.LeftBrac) {
		return nil
	}
	var slc []interface{}

	for p.peek().Type != token.RightBrac {
		slc = append(slc, p.next().String())
		if p.peek().Type == token.Comma {
			p.next()
		} else if !p.isExpected(p.peek(), token.RightBrac) {
			return nil
		}
	}

	// advance ]
	p.next()

	return slc
}

// Decode decodes a bazel/buck ast.
func (p *Parser) Decode(i interface{}) (err error) {
	p.Document = (i.(*ast.File))
	p.Document.Path = p.path
	p.run()
	if p.curTok.Type == token.Error {
		return p.Error
	}
	return nil
}
