package main

import (
	"fmt"
	"os"
)

func lexToolMain() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run *.go lex <ya.make file>")
		os.Exit(1)
	}

	content := Throw2(os.ReadFile(os.Args[2]))
	l := NewLexer(string(content))

	for {
		tok, err := l.NextToken()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%d:%d %s: %q\n", tok.Line, tok.Col, tok.Type, tok.Value)

		if tok.Type == TokenEOF {
			break
		}
	}
}
