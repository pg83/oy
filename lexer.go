package main

import (
	"fmt"
	"strings"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenLParen
	TokenRParen
	TokenString
	TokenAmpersandAmpersand
	TokenPipePipe
	TokenBang
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "IDENT"
	case TokenLParen:
		return "LPAREN"
	case TokenRParen:
		return "RPAREN"
	case TokenString:
		return "STRING"
	case TokenAmpersandAmpersand:
		return "&&"
	case TokenPipePipe:
		return "||"
	case TokenBang:
		return "!"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", t)
	}
}

type Token struct {
	Type  TokenType
	Value string
	Line  int32
	Col   int32
}

type Lexer struct {
	input []rune
	pos   int
	line  int32
	col   int32
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
		line:  1,
		col:   1,
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}

	return l.input[l.pos]
}

func (l *Lexer) read() rune {
	if l.pos >= len(l.input) {
		return 0
	}

	ch := l.input[l.pos]
	l.pos++

	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}

	return ch
}

func (l *Lexer) skipWhitespace() {
	for {
		ch := l.peek()
		if ch == ' ' || ch == '\t' {
			l.read()
		} else {
			break
		}
	}
}

func (l *Lexer) readIdent() string {
	var buf strings.Builder

	for {
		ch := l.peek()
		if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '/' || ch == '.' || ch == '-' || ch == '+' || ch == '*' {
			buf.WriteRune(ch)
			l.read()
		} else {
			break
		}
	}

	return buf.String()
}

func (l *Lexer) readString() string {
	l.read()

	var buf strings.Builder

	for {
		ch := l.peek()
		if ch == 0 {
			ThrowFmt("unterminated string at line %d:%d", l.line, l.col)
		}

		if ch == '"' {
			l.read()
			break
		}

		if ch == '\\' {
			l.read()
			escaped := l.peek()
			if escaped == 0 {
				ThrowFmt("unterminated escape sequence at line %d:%d", l.line, l.col)
			}

			switch escaped {
			case 'n':
				buf.WriteRune('\n')
			case 't':
				buf.WriteRune('\t')
			case 'r':
				buf.WriteRune('\r')
			case '\\':
				buf.WriteRune('\\')
			case '"':
				buf.WriteRune('"')
			default:
				buf.WriteRune(escaped)
			}
			l.read()
		} else {
			buf.WriteRune(ch)
			l.read()
		}
	}

	return buf.String()
}

func (l *Lexer) isKeyword(ident string) (TokenType, bool) {
	keywords := map[string]TokenType{
		"PROGRAM":           TokenIdent,
		"LIBRARY":           TokenIdent,
		"GO_LIBRARY":        TokenIdent,
		"PEERDIR":           TokenIdent,
		"SRCS":              TokenIdent,
		"RECURSE":           TokenIdent,
		"SET":               TokenIdent,
		"END":               TokenIdent,
		"RECURSE_FOR_TESTS": TokenIdent,
		"IF":                TokenIdent,
		"ELSE":              TokenIdent,
		"ELSEIF":            TokenIdent,
		"NOT":               TokenIdent,
		"ENDIF":             TokenIdent,
		"LICENSE":           TokenIdent,
		"VERSION":           TokenIdent,
		"BUILD_ONLY_IF":     TokenIdent,
	}

	tt, ok := keywords[ident]
	return tt, ok
}

func (l *Lexer) NextToken() (Token, error) {
	for {
		l.skipWhitespace()

		ch := l.peek()
		if ch == 0 {
			return Token{Type: TokenEOF, Line: l.line, Col: l.col}, nil
		}

		if ch == '(' {
			startCol := l.col
			l.read()
			return Token{Type: TokenLParen, Value: "(", Line: l.line, Col: startCol}, nil
		}

	if ch == ')' {
		startCol := l.col
		l.read()
		return Token{Type: TokenRParen, Value: ")", Line: l.line, Col: startCol}, nil
	}

	if ch == '"' {
		startLine := l.line
		startCol := l.col
		str := l.readString()
		return Token{Type: TokenString, Value: str, Line: startLine, Col: startCol}, nil
	}

	if ch == '&' {
		startCol := l.col
		l.read()
		if l.peek() == '&' {
			l.read()
			return Token{Type: TokenAmpersandAmpersand, Value: "&&", Line: l.line, Col: startCol}, nil
		}
	}

	if ch == '|' {
		startCol := l.col
		l.read()
		if l.peek() == '|' {
			l.read()
			return Token{Type: TokenPipePipe, Value: "||", Line: l.line, Col: startCol}, nil
		}
	}

	if ch == '!' {
		startCol := l.col
		l.read()
		return Token{Type: TokenBang, Value: "!", Line: l.line, Col: startCol}, nil
	}

		if ch == '\n' {
			l.read()
			continue
		}

		if ch == '\r' {
			l.read()
			if l.peek() == '\n' {
				l.read()
			}
			continue
		}

		if ch == '#' {
			for {
				consume := l.read()
				if consume == 0 || consume == '\n' || consume == '\r' {
					break
				}
			}
			continue
		}

		if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '-' || ch == '*' {
			startLine := l.line
			startCol := l.col
			ident := l.readIdent()
			return Token{Type: TokenIdent, Value: ident, Line: startLine, Col: startCol}, nil
		}

		if ch >= '0' && ch <= '9' {
			startLine := l.line
			startCol := l.col
			ident := l.readIdent()
			return Token{Type: TokenIdent, Value: ident, Line: startLine, Col: startCol}, nil
		}

		if ch == '/' {
			startLine := l.line
			startCol := l.col
			ident := l.readIdent()
			return Token{Type: TokenIdent, Value: ident, Line: startLine, Col: startCol}, nil
		}

		return Token{}, fmt.Errorf("invalid character '%c' at line %d:%d", ch, l.line, l.col)
	}
}
