package main

import (
	"os"
	"path/filepath"
	"strings"
)

type Parser struct {
	lexer    *Lexer
	tokens   []Token
	pos      int
	filePath string
}

func NewParser(input string, filePath string) *Parser {
	lexer := NewLexer(input)

	tokens := []Token{}
	for {
		tok, err := lexer.NextToken()
		Throw(err)

		if tok.Type == TokenEOF {
			break
		}

		tokens = append(tokens, tok)
	}

	return &Parser{
		lexer:    lexer,
		tokens:   tokens,
		pos:      0,
		filePath: filePath,
	}
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}

	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *Parser) expect(expectedType TokenType) Token {
	tok := p.peek()
	if tok.Type != expectedType {
		ThrowFmt("%s:%d:%d: syntax error: expected %s, got %s", p.filePath, tok.Line, tok.Col, expectedType, tok.Type)
	}

	return p.advance()
}

func (p *Parser) expectIdent(expectedValue string) Token {
	tok := p.expect(TokenIdent)
	if tok.Value != expectedValue {
		ThrowFmt("%s:%d:%d: syntax error: expected identifier '%s', got '%s'", p.filePath, tok.Line, tok.Col, expectedValue, tok.Value)
	}

	return tok
}

func (p *Parser) sourceLocation() SourceLocation {
	tok := p.peek()
	return SourceLocation{
		File:   p.filePath,
		Line:   tok.Line,
		Column: tok.Col,
	}
}

func (p *Parser) parseParenthesizedValues() []string {
	p.expect(TokenLParen)

	values := []string{}

	for {
		tok := p.peek()
		if tok.Type == TokenRParen {
			p.advance()
			break
		}

		if len(values) > 0 {
			if tok.Type == TokenString {
				ThrowFmt("%s:%d:%d: syntax error: expected identifier, got string", p.filePath, tok.Line, tok.Col)
			}
			values = append(values, tok.Value)
			p.advance()
		} else {
			if tok.Type == TokenString {
				values = append(values, tok.Value)
				p.advance()
			} else {
				values = append(values, tok.Value)
				p.advance()
			}
		}
	}

	return values
}

func (p *Parser) parseModuleType() ModuleType {
	tok := p.peek()

	switch tok.Value {
	case "PROGRAM":
		p.advance()
		return ModuleTypeProgram
	case "LIBRARY":
		p.advance()
		return ModuleTypeLibrary
	case "GO_LIBRARY":
		p.advance()
		return ModuleTypeGoLibrary
	case "DLL":
		p.advance()
		return ModuleTypeDLL
	case "PY23_LIBRARY":
		p.advance()
		return ModuleTypePy23Library
	case "PY_LIBRARY":
		p.advance()
		return ModuleTypePyLibrary
	default:
		return ModuleTypeUnknown
	}
}

func (p *Parser) parseModule(moduleType ModuleType, sourcePath string) *Module {
	module := &Module{
		Type:       moduleType,
		SourcePath: sourcePath,
		Properties: make(map[string]string),
	}

	for {
		tok := p.peek()
		if tok.Type == TokenEOF {
			break
		}

		if tok.Value == "END" {
			p.advance()
			p.expect(TokenLParen)
			p.expect(TokenRParen)
			break
		}

		if tok.Value == "END()" {
			p.advance()
			break
		}

		if tok.Value == "IF" {
			p.parseConditional(module)
			continue
		}

		if tok.Value == "PEERDIR" {
			tok = p.advance()
			deps := p.parseParenthesizedValues()

			if p.peek().Value == "WHEN" {
				condPeerdir := &ConditionalPeerdir{
					Paths: deps,
					Location: SourceLocation{
						File:   p.filePath,
						Line:   tok.Line,
						Column: tok.Col,
					},
				}
				whenBlock := p.parseWhenBlock()
				condPeerdir.WhenClause = whenBlock
				module.AddConditionalPeerdir(condPeerdir)
			} else {
				for _, dep := range deps {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						module.AddDependency(dep)
					}
				}
			}
			continue
		}

		if strings.HasPrefix(tok.Value, "SRCS") {
			tok = p.advance()
			sources := p.parseParenthesizedValues()
			for _, src := range sources {
				src = strings.TrimSpace(src)
				if src != "" {
					module.AddSource(src)
				}
			}
			continue
		}

		if tok.Value == "SET" {
			p.parseSet(module)
			continue
		}

		if tok.Value == "BUILD_ONLY_IF" {
			p.parseBuildOnlyIf(module)
			continue
		}

		if tok.Value == "RECURSE" || tok.Value == "RECURSE_FOR_TESTS" {
			p.parseRecurse(module)
			continue
		}

		if tok.Value == "ENABLE" {
			p.parseEnable(module)
			continue
		}

		if tok.Value == "DISABLE" {
			p.parseDisable(module)
			continue
		}

		p.advance()
	}

	return module
}

func (p *Parser) parseSet(module *Module) {
	p.expectIdent("SET")
	p.expect(TokenLParen)

	tok := p.peek()
	if tok.Type != TokenIdent && tok.Type != TokenString {
		ThrowFmt("%s:%d:%d: syntax error: expected identifier or string for SET key, got %s", p.filePath, tok.Line, tok.Col, tok.Type)
	}

	key := tok.Value
	p.advance()

	tok = p.peek()
	value := ""
	if tok.Type == TokenString {
		value = tok.Value
		p.advance()
	} else {
		value = tok.Value
		p.advance()
	}

	p.expect(TokenRParen)

	module.AddProperty(key, value)
}

func (p *Parser) parseConditional(module *Module) {
	if tok := p.peek(); tok.Value != "IF" {
		ThrowFmt("%s:%d:%d: syntax error: expected IF, got %s", p.filePath, tok.Line, tok.Col, tok.Value)
	}

	loc := p.sourceLocation()
	p.expectIdent("IF")
	p.expect(TokenLParen)

	conditionTokens := []string{}
	for {
		tok := p.peek()
		if tok.Type == TokenRParen {
			p.advance()
			break
		}
		conditionTokens = append(conditionTokens, tok.Value)
		p.advance()
	}

	condition := strings.Join(conditionTokens, " ")

	ifBranch := &ConditionalBranch{
		Condition: condition,
		Module: &Module{
			Type:       module.Type,
			SourcePath: module.SourcePath,
			Properties: make(map[string]string),
		},
	}

	for {
		tok := p.peek()
		if tok.Type == TokenEOF {
			break
		}

		if tok.Value == "ELSEIF" || tok.Value == "ELSE" || tok.Value == "ENDIF" {
			break
		}

		if tok.Value == "PEERDIR" {
			p.expectIdent("PEERDIR")
			deps := p.parseParenthesizedValues()
			for _, dep := range deps {
				dep = strings.TrimSpace(dep)
				if dep != "" {
					ifBranch.Module.AddDependency(dep)
				}
			}
			continue
		}

		if strings.HasPrefix(tok.Value, "SRCS") {
			tok = p.advance()
			sources := p.parseParenthesizedValues()
			for _, src := range sources {
				src = strings.TrimSpace(src)
				if src != "" {
					ifBranch.Module.AddSource(src)
				}
			}
			continue
		}

		if tok.Value == "IF" {
			p.parseConditional(ifBranch.Module)
			continue
		}

		p.advance()
	}

	elseIfs := []*ConditionalBranch{}

	for {
		tok := p.peek()
		if tok.Value != "ELSEIF" {
			break
		}

		p.expectIdent("ELSEIF")
		p.expect(TokenLParen)

		conditionTokens := []string{}
		for {
			tok := p.peek()
			if tok.Type == TokenRParen {
				p.advance()
				break
			}
			conditionTokens = append(conditionTokens, tok.Value)
			p.advance()
		}

		condition := strings.Join(conditionTokens, " ")

		elseIfBranch := &ConditionalBranch{
			Condition: condition,
			Module: &Module{
				Type:       module.Type,
				SourcePath: module.SourcePath,
				Properties: make(map[string]string),
			},
		}

		for {
			tok := p.peek()
			if tok.Type == TokenEOF {
				break
			}

			if tok.Value == "ELSEIF" || tok.Value == "ELSE" || tok.Value == "ENDIF" {
				break
			}

			if tok.Value == "PEERDIR" {
				p.expectIdent("PEERDIR")
				deps := p.parseParenthesizedValues()
				for _, dep := range deps {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						elseIfBranch.Module.AddDependency(dep)
					}
				}
				continue
			}

			if strings.HasPrefix(tok.Value, "SRCS") {
				tok = p.advance()
				sources := p.parseParenthesizedValues()
				for _, src := range sources {
					src = strings.TrimSpace(src)
					if src != "" {
						elseIfBranch.Module.AddSource(src)
					}
				}
				continue
			}

			p.advance()
		}

		elseIfs = append(elseIfs, elseIfBranch)
	}

	var elseBranch *ConditionalBranch

	if p.peek().Value == "ELSE" {
		p.expectIdent("ELSE")
		p.expect(TokenLParen)
		p.expect(TokenRParen)

		elseBranch = &ConditionalBranch{
			Condition: "",
			Module: &Module{
				Type:       module.Type,
				SourcePath: module.SourcePath,
				Properties: make(map[string]string),
			},
		}

		for {
			tok := p.peek()
			if tok.Type == TokenEOF {
				break
			}

			if tok.Value == "ENDIF" {
				break
			}

			if tok.Value == "PEERDIR" {
				p.expectIdent("PEERDIR")
				deps := p.parseParenthesizedValues()
				for _, dep := range deps {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						elseBranch.Module.AddDependency(dep)
					}
				}
				continue
			}

			if strings.HasPrefix(tok.Value, "SRCS") {
				tok = p.advance()
				sources := p.parseParenthesizedValues()
				for _, src := range sources {
					src = strings.TrimSpace(src)
					if src != "" {
						elseBranch.Module.AddSource(src)
					}
				}
				continue
			}

			if tok.Value == "IF" {
				p.parseConditional(elseBranch.Module)
				continue
			}

			p.advance()
		}
	}

	p.expectIdent("ENDIF")
	p.expect(TokenLParen)
	p.expect(TokenRParen)

	module.AddConditional(&ConditionalBlock{
		IfBranch:   ifBranch,
		ElseIfs:    elseIfs,
		ElseBranch: elseBranch,
		Location:   loc,
	})
}

func (p *Parser) parseBuildOnlyIf(module *Module) {
	p.expectIdent("BUILD_ONLY_IF")
	p.expect(TokenLParen)

	exprTokens := []string{}
	for {
		tok := p.peek()
		if tok.Type == TokenRParen {
			p.advance()
			break
		}
		exprTokens = append(exprTokens, tok.Value)
		p.advance()
	}

	expression := strings.Join(exprTokens, " ")

	module.BuildCondition = &BuildCondition{
		Expression: expression,
	}
}

func (p *Parser) parseRecurse(module *Module) {
	tok := p.peek()
	tokenValue := tok.Value

	if tokenValue == "RECURSE" || tokenValue == "RECURSE_FOR_TESTS" {
		p.advance()
		loc := SourceLocation{
			File:   p.filePath,
			Line:   tok.Line,
			Column: tok.Col,
		}

		paths := p.parseParenthesizedValues()

		module.AddRecurse(&RecurseDirective{
			Paths:    paths,
			Location: loc,
		})
	}
}

func ParseYaMakeFile(path string) *File {
	content := Throw2(os.ReadFile(path))

	parser := NewParser(string(content), path)

	sourcePath := filepath.Dir(path)

	file := &File{
		Modules: []*Module{},
		Imports: []*RecurseDirective{},
	}

	for {
		tok := parser.peek()
		if tok.Type == TokenEOF {
			break
		}

		moduleType := parser.parseModuleType()

		if moduleType != ModuleTypeUnknown {
			module := parser.parseModule(moduleType, sourcePath)
			file.Modules = append(file.Modules, module)
			continue
		}

		if tok.Value == "RECURSE" || tok.Value == "RECURSE_FOR_TESTS" {
			tok := parser.peek()
			loc := SourceLocation{
				File:   parser.filePath,
				Line:   tok.Line,
				Column: tok.Col,
			}

			if tok.Value == "RECURSE" || tok.Value == "RECURSE_FOR_TESTS" {
				parser.advance()
				paths := parser.parseParenthesizedValues()
				file.Imports = append(file.Imports, &RecurseDirective{
					Paths:    paths,
					Location: loc,
				})
			}
			continue
		}

		parser.advance()
	}

	return file
}

func (p *Parser) parseEnable(module *Module) {
	p.expectIdent("ENABLE")
	p.expect(TokenLParen)

	tok := p.peek()
	if tok.Type != TokenIdent {
		ThrowFmt("%s:%d:%d: syntax error: expected identifier for ENABLE flag, got %s", p.filePath, tok.Line, tok.Col, tok.Type)
	}

	flag := tok.Value
	p.advance()

	p.expect(TokenRParen)

	module.AddEnabledFlag(flag)
}

func (p *Parser) parseDisable(module *Module) {
	p.expectIdent("DISABLE")
	p.expect(TokenLParen)

	tok := p.peek()
	if tok.Type != TokenIdent {
		ThrowFmt("%s:%d:%d: syntax error: expected identifier for DISABLE flag, got %s", p.filePath, tok.Line, tok.Col, tok.Type)
	}

	flag := tok.Value
	p.advance()

	p.expect(TokenRParen)

	module.AddDisabledFlag(flag)
}

func (p *Parser) parseWhenBlock() *WhenBlock {
	p.expectIdent("WHEN")
	p.expect(TokenLParen)

	conditionTokens := []string{}
	for {
		tok := p.peek()
		if tok.Type == TokenRParen {
			p.advance()
			break
		}
		conditionTokens = append(conditionTokens, tok.Value)
		p.advance()
	}

	condition := strings.Join(conditionTokens, " ")

	return &WhenBlock{
		Condition: condition,
	}
}
