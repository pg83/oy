package main

import (
	"fmt"
	"os"
)

func main() {

	if exc := Try(func() {

		if len(os.Args) > 1 && os.Args[1] == "lex" {
			lexToolMain()
			return
		}

		fmt.Println("ymake - Ya Make build system reimplementation")

	}); exc != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", exc.AsError())
		os.Exit(1)
	}

}
