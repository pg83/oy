package main

import (
	"fmt"
	"os"
)

func main() {

	if exc := Try(func() {

		fmt.Println("ymake - Ya Make build system reimplementation")

	}); exc != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", exc.AsError())
		os.Exit(1)
	}

}
