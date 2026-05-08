package main

import (
	"os"
	"path/filepath"
	"strings"
)

type Parser struct {
	lexer *Lexer
	ctx   *ParseContext
}

func NewParser(ctx *ParseContext) *Parser {
	return &Parser{
		ctx: ctx,
	}
}

func (p *Parser) ParseFile(path string) (*File, error) {
	content := Throw2(os.ReadFile(path))

	p.lexer = NewLexer(string(content))

	file := &File{
		Modules: make([]*Module, 0),
		Imports: make([]*RecurseDirective, 0),
	}

	for {
		token := Throw2(p.lexer.NextToken())

		if token.Type == TokenEOF {
			break
		}

		if token.Type == TokenIdent {
			switch token.Value {
			case "PROGRAM", "LIBRARY", "GO_LIBRARY", "DLL", "PY_LIBRARY", "PY23_LIBRARY":
				module := p.parseModule(token.Value, path)
				file.Modules = append(file.Modules, module)

			case "RECURSE", "RECURSE_FOR_TESTS":
				recurse := p.parseRecurse(token.Value, token.Line, token.Col)
				file.Imports = append(file.Imports, recurse)

			default:
				ThrowFmt("unexpected token %s at line %d:%d (expected module type or RECURSE)", token.Value, token.Line, token.Col)
			}
		}
	}

	return file, nil
}

func (p *Parser) parseModule(moduleType string, filePath string) *Module {
	module := &Module{
		Type:         p.parseModuleType(moduleType),
		SourcePath:   filepath.Dir(filePath),
		Dependencies: make([]string, 0),
		Sources:      make([]string, 0),
		Properties:   make(map[string]string),
		Conditionals: make([]*ConditionalBlock, 0),
		Recursions:   make([]*RecurseDirective, 0),
	}

	for {
		token := Throw2(p.lexer.NextToken())

		if token.Type == TokenEOF {
			ThrowFmt("unexpected EOF while parsing module at line %d", token.Line)
		}

		if token.Type == TokenIdent {
			switch token.Value {
			case "PEERDIR":
				peerDeps := p.parsePeerDir()
				module.Dependencies = append(module.Dependencies, peerDeps...)

			case "SRCS", "PY_SRCS", "CPP_SRCS", "C_SRCS":
				sources := p.parseSrcs()
				module.Sources = append(module.Sources, sources...)

			case "SET":
				key, value := p.parseSet()
				module.Properties[key] = value

			case "IF":
				conditional := p.parseConditional()
				module.Conditionals = append(module.Conditionals, conditional)

			case "RECURSE", "RECURSE_FOR_TESTS":
				recurse := p.parseRecurse(token.Value, token.Line, token.Col)
				module.Recursions = append(module.Recursions, recurse)

			case "BUILD_ONLY_IF":
				module.BuildCondition = p.parseBuildCondition()

			case "END":
				return module

			default:
				ThrowFmt("unexpected token %s in module at line %d:%d", token.Value, token.Line, token.Col)
			}
		}
	}
}

func (p *Parser) parseModuleType(typeStr string) ModuleType {
	switch typeStr {
	case "PROGRAM":
		return ModuleTypeProgram
	case "LIBRARY":
		return ModuleTypeLibrary
	case "GO_LIBRARY":
		return ModuleTypeGoLibrary
	case "DLL":
		return ModuleTypeDLL
	case "PY_LIBRARY":
		return ModuleTypePyLibrary
	case "PY23_LIBRARY":
		return ModuleTypePy23Library
	default:
		return ModuleTypeUnknown
	}
}

func (p *Parser) parsePeerDir() []string {
	Throw2(p.expectLParen())

	paths := make([]string, 0)

	for {
		token := Throw2(p.lexer.NextToken())

		if token.Type == TokenRParen {
			break
		}

		if token.Type == TokenString || token.Type == TokenIdent {
			paths = append(paths, token.Value)
		} else {
			ThrowFmt("expected path or ')' in PEERDIR at line %d:%d, got %s", token.Line, token.Col, token.Value)
		}
	}

	return paths
}

func (p *Parser) parseSrcs() []string {
	Throw2(p.expectLParen())

	sources := make([]string, 0)

	for {
		token := Throw2(p.lexer.NextToken())

		if token.Type == TokenRParen {
			break
		}

		if token.Type == TokenString || token.Type == TokenIdent {
			sources = append(sources, token.Value)
		}
	}

	return sources
}

func (p *Parser) parseSet() (string, string) {
	Throw2(p.expectLParen())

	keyToken := Throw2(p.lexer.NextToken())
	if keyToken.Type != TokenIdent && keyToken.Type != TokenString {
		ThrowFmt("expected key in SET at line %d:%d, got %s", keyToken.Line, keyToken.Col, keyToken.Value)
	}

	valueToken := Throw2(p.lexer.NextToken())
	if valueToken.Type != TokenIdent && valueToken.Type != TokenString {
		ThrowFmt("expected value in SET at line %d:%d, got %s", valueToken.Line, valueToken.Col, valueToken.Value)
	}

	Throw2(p.expectRParen())

	return keyToken.Value, valueToken.Value
}

func (p *Parser) parseConditional() *ConditionalBlock {
	condition := p.parseConditionExpression()

	ifBranch := &ConditionalBranch{
		Condition: condition,
		Module: &Module{
			Type:         ModuleTypeUnknown,
			Dependencies: make([]string, 0),
			Sources:      make([]string, 0),
			Properties:   make(map[string]string),
			Recursions:   make([]*RecurseDirective, 0),
		},
	}

	p.parseModuleContentsInConditional(ifBranch.Module)

	block := &ConditionalBlock{
		IfBranch: ifBranch,
		ElseIfs:  make([]*ConditionalBranch, 0),
		Location: SourceLocation{},
	}

	for {
		token := p.peekNextToken()

		if token.Type == TokenIdent && token.Value == "ELSEIF" {
			p.lexer.NextToken()

			elseifCondition := p.parseConditionExpression()

			elseifBranch := &ConditionalBranch{
				Condition: elseifCondition,
				Module: &Module{
					Type:         ModuleTypeUnknown,
					Dependencies: make([]string, 0),
					Sources:      make([]string, 0),
					Properties:   make(map[string]string),
					Recursions:   make([]*RecurseDirective, 0),
				},
			}

			p.parseModuleContentsInConditional(elseifBranch.Module)

			block.ElseIfs = append(block.ElseIfs, elseifBranch)

		} else if token.Type == TokenIdent && token.Value == "ELSE" {
			p.lexer.NextToken()

			elseToken := Throw2(p.lexer.NextToken())

			if elseToken.Type != TokenLParen && elseToken.Type != TokenRParen {
				ThrowFmt("expected '(' or ')' after ELSE at line %d:%d, got %s", elseToken.Line, elseToken.Col, elseToken.Value)
			}

			elseBranch := &ConditionalBranch{
				Module: &Module{
					Type:         ModuleTypeUnknown,
					Dependencies: make([]string, 0),
					Sources:      make([]string, 0),
					Properties:   make(map[string]string),
					Recursions:   make([]*RecurseDirective, 0),
				},
			}

			if elseToken.Type == TokenLParen {
				p.parseModuleContentsInConditional(elseBranch.Module)
			}

			block.ElseBranch = elseBranch

		} else if token.Type == TokenIdent && token.Value == "ENDIF" {
			p.lexer.NextToken()
			break
		} else {
			ThrowFmt("expected ELSEIF, ELSE, or ENDIF at line %d:%d, got %s", token.Line, token.Col, token.Value)
		}
	}

	return block
}

func (p *Parser) parseModuleContentsInConditional(module *Module) {
	for {
		token := p.peekNextToken()

		if token.Type == TokenIdent && (token.Value == "ENDIF" || token.Value == "ELSE" || token.Value == "ELSEIF") {
			return
		}

		p.lexer.NextToken()

		if token.Type == TokenIdent {
			switch token.Value {
			case "PEERDIR":
				peerDeps := p.parsePeerDir()
				module.Dependencies = append(module.Dependencies, peerDeps...)

			case "SRCS", "PY_SRCS", "CPP_SRCS", "C_SRCS":
				sources := p.parseSrcs()
				module.Sources = append(module.Sources, sources...)

			case "SET":
				key, value := p.parseSet()
				module.Properties[key] = value

			case "RECURSE", "RECURSE_FOR_TESTS":
				recurse := p.parseRecurse(token.Value, token.Line, token.Col)
				module.Recursions = append(module.Recursions, recurse)

			default:
				ThrowFmt("unexpected token %s in conditional block at line %d:%d", token.Value, token.Line, token.Col)
			}
		}
	}
}

func (p *Parser) peekNextToken() Token {
	savedPos := p.lexer.pos
	savedLine := p.lexer.line
	savedCol := p.lexer.col

	token, err := p.lexer.NextToken()
	if err != nil {
		return token
	}

	p.lexer.pos = savedPos
	p.lexer.line = savedLine
	p.lexer.col = savedCol

	return token
}

func (p *Parser) parseConditionExpression() string {
	var expr strings.Builder

	token := Throw2(p.lexer.NextToken())

	if token.Type != TokenLParen {
		ThrowFmt("expected '(' in condition expression at line %d:%d, got %s", token.Line, token.Col, token.Value)
	}

	expr.WriteString(token.Value)

	for {
		token := Throw2(p.lexer.NextToken())

		expr.WriteString(token.Value)

		if token.Type == TokenRParen {
			break
		}
	}

	return expr.String()
}

func (p *Parser) parseModuleContents(module *Module) {
	for {
		token := Throw2(p.lexer.NextToken())

		if token.Type == TokenIdent {
			switch token.Value {
			case "PEERDIR":
				peerDeps := p.parsePeerDir()
				module.Dependencies = append(module.Dependencies, peerDeps...)

			case "SRCS", "PY_SRCS", "CPP_SRCS", "C_SRCS":
				sources := p.parseSrcs()
				module.Sources = append(module.Sources, sources...)

			case "SET":
				key, value := p.parseSet()
				module.Properties[key] = value

			case "RECURSE", "RECURSE_FOR_TESTS":
				recurse := p.parseRecurse(token.Value, token.Line, token.Col)
				module.Recursions = append(module.Recursions, recurse)

			case "END":
				return

			default:
				ThrowFmt("unexpected token %s in module at line %d:%d", token.Value, token.Line, token.Col)
			}
		}
	}
}

func (p *Parser) parseRecurse(recurseType string, line, col int32) *RecurseDirective {
	Throw2(p.expectLParen())

	paths := make([]string, 0)

	for {
		token := Throw2(p.lexer.NextToken())

		if token.Type == TokenRParen {
			break
		}

		if token.Type == TokenString || token.Type == TokenIdent {
			paths = append(paths, token.Value)
		} else {
			ThrowFmt("expected path or ')' in %s at line %d:%d, got %s", recurseType, line, col, token.Value)
		}
	}

	return &RecurseDirective{
		Paths:    paths,
		Location: SourceLocation{File: "", Line: line, Column: col},
	}
}

func (p *Parser) parseBuildCondition() *BuildCondition {
	Throw2(p.expectLParen())

	token := Throw2(p.lexer.NextToken())
	if token.Type != TokenIdent && token.Type != TokenString {
		ThrowFmt("expected condition expression in BUILD_ONLY_IF at line %d:%d, got %s", token.Line, token.Col, token.Value)
	}

	Throw2(p.expectRParen())

	return &BuildCondition{
		Expression: token.Value,
	}
}

func (p *Parser) expectLParen() (Token, error) {
	token, err := p.lexer.NextToken()
	Throw(err)

	if token.Type != TokenLParen {
		return Token{}, Fmt("expected '(' at line %d:%d, got %s", token.Line, token.Col, token.Value)
	}

	return token, nil
}

func (p *Parser) expectRParen() (Token, error) {
	token, err := p.lexer.NextToken()
	Throw(err)

	if token.Type != TokenRParen {
		return Token{}, Fmt("expected ')' at line %d:%d, got %s", token.Line, token.Col, token.Value)
	}

	return token, nil
}
