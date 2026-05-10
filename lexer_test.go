package main

import (
	"reflect"
	"testing"
)

func expectToken(t *testing.T, l *Lexer, typ TokenType, value string, line, col int32) {
	tok, err := l.NextToken()
	if err != nil {
		t.Fatalf("NextToken() error = %v", err)
	}

	if tok.Type != typ {
		t.Errorf("Token.Type = %v, want %v", tok.Type, typ)
	}

	if tok.Value != value {
		t.Errorf("Token.Value = %q, want %q", tok.Value, value)
	}

	if tok.Line != line {
		t.Errorf("Token.Line = %d, want %d", tok.Line, line)
	}

	if tok.Col != col {
		t.Errorf("Token.Col = %d, want %d", tok.Col, col)
	}
}

func expectEOF(t *testing.T, l *Lexer) {
	tok, err := l.NextToken()
	if err != nil {
		t.Fatalf("NextToken() error = %v", err)
	}

	if tok.Type != TokenEOF {
		t.Errorf("Token.Type = %v, want TokenEOF", tok.Type)
	}
}

func TestKeywords(t *testing.T) {
	keywords := []struct {
		input string
		value string
	}{
		{"PROGRAM", "PROGRAM"},
		{"LIBRARY", "LIBRARY"},
		{"GO_LIBRARY", "GO_LIBRARY"},
		{"PEERDIR", "PEERDIR"},
		{"SRCS", "SRCS"},
		{"RECURSE", "RECURSE"},
		{"SET", "SET"},
		{"END", "END"},
		{"RECURSE_FOR_TESTS", "RECURSE_FOR_TESTS"},
		{"IF", "IF"},
		{"ELSE", "ELSE"},
		{"ENDIF", "ENDIF"},
		{"LICENSE", "LICENSE"},
		{"VERSION", "VERSION"},
		{"BUILD_ONLY_IF", "BUILD_ONLY_IF"},
	}

	for _, tt := range keywords {
		t.Run(tt.value, func(t *testing.T) {
			l := NewLexer(tt.value)

			expectToken(t, l, TokenIdent, tt.value, 1, 1)
			expectEOF(t, l)
		})
	}
}

func TestModuleDefinition(t *testing.T) {
	input := `PROGRAM()`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "PROGRAM", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)
	expectToken(t, l, TokenRParen, ")", 1, 9)
	expectEOF(t, l)
}

func TestModuleWithDependencies(t *testing.T) {
	input := `PROGRAM()
PEERDIR(
    library/cpp/archive
    library/cpp/digest/md5
)`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "PROGRAM", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)
	expectToken(t, l, TokenRParen, ")", 1, 9)
	expectToken(t, l, TokenIdent, "PEERDIR", 2, 1)
	expectToken(t, l, TokenLParen, "(", 2, 8)
	expectToken(t, l, TokenIdent, "library/cpp/archive", 3, 5)
	expectToken(t, l, TokenIdent, "library/cpp/digest/md5", 4, 5)
	expectToken(t, l, TokenRParen, ")", 5, 1)
	expectEOF(t, l)
}

func TestConditionals(t *testing.T) {
	input := `IF(OS_WINDOWS)
ELSE()
ENDIF()`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "IF", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 3)
	expectToken(t, l, TokenIdent, "OS_WINDOWS", 1, 4)
	expectToken(t, l, TokenRParen, ")", 1, 14)
	expectToken(t, l, TokenIdent, "ELSE", 2, 1)
	expectToken(t, l, TokenLParen, "(", 2, 5)
	expectToken(t, l, TokenRParen, ")", 2, 6)
	expectToken(t, l, TokenIdent, "ENDIF", 3, 1)
	expectToken(t, l, TokenLParen, "(", 3, 6)
	expectToken(t, l, TokenRParen, ")", 3, 7)
	expectEOF(t, l)
}

func TestStrings(t *testing.T) {
	input := `SET(NAME "test")`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "SET", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 4)
	expectToken(t, l, TokenIdent, "NAME", 1, 5)
	expectToken(t, l, TokenString, "test", 1, 10)
	expectToken(t, l, TokenRParen, ")", 1, 16)
	expectEOF(t, l)
}

func TestStringEscapes(t *testing.T) {
	input := `"test\nstring\tend"`

	l := NewLexer(input)

	expectToken(t, l, TokenString, "test\nstring\tend", 1, 1)
	expectEOF(t, l)
}

func TestMultiLineArguments(t *testing.T) {
	input := `PEERDIR(
    library/cpp/archive
    library/cpp/digest/md5
    library/cpp/getopt/small
)`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "PEERDIR", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)
	expectToken(t, l, TokenIdent, "library/cpp/archive", 2, 5)
	expectToken(t, l, TokenIdent, "library/cpp/digest/md5", 3, 5)
	expectToken(t, l, TokenIdent, "library/cpp/getopt/small", 4, 5)
	expectToken(t, l, TokenRParen, ")", 5, 1)
	expectEOF(t, l)
}

func TestArchiverExample(t *testing.T) {
	input := `PROGRAM()

PEERDIR(
    library/cpp/archive
    library/cpp/digest/md5
    library/cpp/getopt/small
)

SRCS(
    main.cpp
)

SET(IDE_FOLDER "_Builders")

END()
`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "PROGRAM", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)
	expectToken(t, l, TokenRParen, ")", 1, 9)
	expectToken(t, l, TokenIdent, "PEERDIR", 3, 1)
	expectToken(t, l, TokenLParen, "(", 3, 8)
	expectToken(t, l, TokenIdent, "library/cpp/archive", 4, 5)
	expectToken(t, l, TokenIdent, "library/cpp/digest/md5", 5, 5)
	expectToken(t, l, TokenIdent, "library/cpp/getopt/small", 6, 5)
	expectToken(t, l, TokenRParen, ")", 7, 1)
	expectToken(t, l, TokenIdent, "SRCS", 9, 1)
	expectToken(t, l, TokenLParen, "(", 9, 5)
	expectToken(t, l, TokenIdent, "main.cpp", 10, 5)
	expectToken(t, l, TokenRParen, ")", 11, 1)
	expectToken(t, l, TokenIdent, "SET", 13, 1)
	expectToken(t, l, TokenLParen, "(", 13, 4)
	expectToken(t, l, TokenIdent, "IDE_FOLDER", 13, 5)
	expectToken(t, l, TokenString, "_Builders", 13, 16)
	expectToken(t, l, TokenRParen, ")", 13, 27)
	expectToken(t, l, TokenIdent, "END", 15, 1)
	expectToken(t, l, TokenLParen, "(", 15, 4)
	expectToken(t, l, TokenRParen, ")", 15, 5)
	expectEOF(t, l)
}

func TestLibraryExample(t *testing.T) {
	input := `LIBRARY()

SRCS(
    yarchive.cpp
    yarchive.h
)

END()

RECURSE_FOR_TESTS(
    ut
)`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "LIBRARY", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)
	expectToken(t, l, TokenRParen, ")", 1, 9)
	expectToken(t, l, TokenIdent, "SRCS", 3, 1)
	expectToken(t, l, TokenLParen, "(", 3, 5)
	expectToken(t, l, TokenIdent, "yarchive.cpp", 4, 5)
	expectToken(t, l, TokenIdent, "yarchive.h", 5, 5)
	expectToken(t, l, TokenRParen, ")", 6, 1)
	expectToken(t, l, TokenIdent, "END", 8, 1)
	expectToken(t, l, TokenLParen, "(", 8, 4)
	expectToken(t, l, TokenRParen, ")", 8, 5)
	expectToken(t, l, TokenIdent, "RECURSE_FOR_TESTS", 10, 1)
	expectToken(t, l, TokenLParen, "(", 10, 18)
	expectToken(t, l, TokenIdent, "ut", 11, 5)
	expectToken(t, l, TokenRParen, ")", 12, 1)
	expectEOF(t, l)
}

func TestPathIdentifiers(t *testing.T) {
	input := `library/cpp/archive`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "library/cpp/archive", 1, 1)
	expectEOF(t, l)
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		token TokenType
		want  string
	}{
		{TokenEOF, "EOF"},
		{TokenIdent, "IDENT"},
		{TokenLParen, "LPAREN"},
		{TokenRParen, "RPAREN"},
		{TokenString, "STRING"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.token.String(); got != tt.want {
				t.Errorf("TokenType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidCharacter(t *testing.T) {
	input := `PROGRAM($test)`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "PROGRAM", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)

	_, err := l.NextToken()
	if err == nil {
		t.Error("Expected error for invalid character, got nil")
	}

	wantErr := "invalid character '$' at line 1:9"
	if err.Error() != wantErr {
		t.Errorf("Error = %v, want %v", err, wantErr)
	}
}

func TestUnterminatedString(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for unterminated string")
		}
	}()

	input := `SET(NAME "test)`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "SET", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 4)
	expectToken(t, l, TokenIdent, "NAME", 1, 5)

	l.NextToken()
}

func TestRecursiveExample(t *testing.T) {
	input := `RECURSE(
    devtools/local_cache/toolscache/server
    devtools/ya/bin
)`

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "RECURSE", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)
	expectToken(t, l, TokenIdent, "devtools/local_cache/toolscache/server", 2, 5)
	expectToken(t, l, TokenIdent, "devtools/ya/bin", 3, 5)
	expectToken(t, l, TokenRParen, ")", 4, 1)
	expectEOF(t, l)
}

func TestTokenEquality(t *testing.T) {
	tok1 := Token{Type: TokenIdent, Value: "PROGRAM", Line: 1, Col: 1}
	tok2 := Token{Type: TokenIdent, Value: "PROGRAM", Line: 1, Col: 1}

	if !reflect.DeepEqual(tok1, tok2) {
		t.Error("Tokens with same values should be equal")
	}
}

func TestEmptyInput(t *testing.T) {
	l := NewLexer("")

	expectEOF(t, l)
}

func TestWhitespaceOnly(t *testing.T) {
	l := NewLexer("   \n\t   \n")

	expectEOF(t, l)
}

func TestCarriageReturn(t *testing.T) {
	input := "PROGRAM()\r\nPEERDIR(\r\ntest\r\n)"

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "PROGRAM", 1, 1)
	expectToken(t, l, TokenLParen, "(", 1, 8)
	expectToken(t, l, TokenRParen, ")", 1, 9)
	expectToken(t, l, TokenIdent, "PEERDIR", 2, 1)
	expectToken(t, l, TokenLParen, "(", 2, 8)
	expectToken(t, l, TokenIdent, "test", 3, 1)
	expectToken(t, l, TokenRParen, ")", 4, 1)
	expectEOF(t, l)
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		input     string
		tokenType TokenType
		value     string
	}{
		{"<", TokenLess, "<"},
		{">", TokenGreater, ">"},
		{"==", TokenEqualsEquals, "=="},
		{"=", TokenEquals, "="},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			l := NewLexer(tt.input)
			expectToken(t, l, tt.tokenType, tt.value, 1, 1)
			expectEOF(t, l)
		})
	}
}

func TestNumbers(t *testing.T) {
	tests := []struct {
		input string
		value string
	}{
		{"0", "0"},
		{"123", "123"},
		{"29", "29"},
		{"3.14", "3.14"},
		{"0.5", "0.5"},
		{"100.0", "100.0"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			l := NewLexer(tt.input)
			expectToken(t, l, TokenNumber, tt.value, 1, 1)
			expectEOF(t, l)
		})
	}
}

func TestComparisonExpressionTokens(t *testing.T) {
	input := "ANDROID_API < 29"

	l := NewLexer(input)

	expectToken(t, l, TokenIdent, "ANDROID_API", 1, 1)
	expectToken(t, l, TokenLess, "<", 1, 13)
	expectToken(t, l, TokenNumber, "29", 1, 15)
	expectEOF(t, l)
}
