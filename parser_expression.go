package main

import (
	"strconv"
	"strings"
)

type exprParser struct {
	lexer  *Lexer
	tokens []Token
	pos    int
}

func newExprParser(conditionExpr string) *exprParser {
	lexer := NewLexer(conditionExpr)

	tokens := []Token{}
	for {
		tok, err := lexer.NextToken()
		Throw(err)

		if tok.Type == TokenEOF {
			break
		}

		tokens = append(tokens, tok)
	}

	return &exprParser{
		lexer:  lexer,
		tokens: tokens,
		pos:    0,
	}
}

func (p *exprParser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}

	return p.tokens[p.pos]
}

func (p *exprParser) advance() Token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *exprParser) expect(expectedType TokenType) Token {
	tok := p.peek()
	if tok.Type != expectedType {
		ThrowFmt("syntax error at line %d:%d: expected %s, got %s", tok.Line, tok.Col, expectedType, tok.Type)
	}

	return p.advance()
}

func (p *exprParser) parseExpression() *ExpressionAST {
	return p.parseOrExpr()
}

func (p *exprParser) parseOrExpr() *ExpressionAST {
	left := p.parseAndExpr()

	for {
		tok := p.peek()
		if tok.Type == TokenPipePipe || tok.Type == TokenOr {
			p.advance()
			right := p.parseAndExpr()
			left = &ExpressionAST{
				Op:   ExprOr,
				Args: []*ExpressionAST{left, right},
			}
		} else {
			break
		}
	}

	return left
}

func (p *exprParser) parseAndExpr() *ExpressionAST {
	left := p.parseNotExpr()

	for {
		tok := p.peek()
		if tok.Type == TokenAmpersandAmpersand || tok.Type == TokenAnd {
			p.advance()
			right := p.parseNotExpr()
			left = &ExpressionAST{
				Op:   ExprAnd,
				Args: []*ExpressionAST{left, right},
			}
		} else {
			break
		}
	}

	return left
}

func (p *exprParser) parseNotExpr() *ExpressionAST {
	tok := p.peek()

	if tok.Type == TokenBang || tok.Type == TokenNot {
		p.advance()
		operand := p.parseNotExpr()
		return &ExpressionAST{
			Op:   ExprNot,
			Args: []*ExpressionAST{operand},
		}
	}

	return p.parseComparisonExpr()
}

func (p *exprParser) parseComparisonExpr() *ExpressionAST {
	left := p.parsePrimary()

	for {
		tok := p.peek()
		switch tok.Type {
		case TokenEqualsEquals:
			p.advance()
			right := p.parsePrimary()
			left = &ExpressionAST{
				Op:   ExprEqual,
				Args: []*ExpressionAST{left, right},
			}
		case TokenLess:
			p.advance()
			right := p.parsePrimary()
			left = &ExpressionAST{
				Op:   ExprLessThan,
				Args: []*ExpressionAST{left, right},
			}
		case TokenGreater:
			p.advance()
			right := p.parsePrimary()
			left = &ExpressionAST{
				Op:   ExprGreaterThan,
				Args: []*ExpressionAST{left, right},
			}
		default:
			return left
		}
	}
}

func (p *exprParser) parsePrimary() *ExpressionAST {
	tok := p.peek()

	if tok.Type == TokenLParen {
		p.advance()
		expr := p.parseExpression()
		p.expect(TokenRParen)
		return expr
	}

	if tok.Type == TokenIdent {
		ident := p.advance()
		return &ExpressionAST{
			Op:    ExprIdent,
			Ident: ident.Value,
		}
	}

	if tok.Type == TokenNumber {
		num := p.advance()
		value, err := strconv.ParseFloat(num.Value, 64)
		if err != nil {
			ThrowFmt("invalid number %q at line %d:%d", num.Value, num.Line, num.Col)
		}
		return &ExpressionAST{
			Op:     ExprNumber,
			Number: value,
		}
	}

	ThrowFmt("syntax error at line %d:%d: expected identifier, number, or '(', got %s", tok.Line, tok.Col, tok.Type)
	return nil
}

func ParseConditionExpression(conditionExpr string) *ExpressionAST {
	if conditionExpr == "" {
		return nil
	}

	trimmed := strings.TrimSpace(conditionExpr)
	if trimmed == "" {
		return nil
	}

	parser := newExprParser(trimmed)
	expr := parser.parseExpression()

	if parser.peek().Type != TokenEOF {
		tok := parser.peek()
		ThrowFmt("syntax error at line %d:%d: unexpected token after expression: %s", tok.Line, tok.Col, tok.Type)
	}

	if expr != nil {
		expr.SourceText = conditionExpr
	}

	return expr
}
