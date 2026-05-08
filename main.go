package main

import (
	"fmt"

	"github.com/pg83/oy"
)

func main() {

	Try(func() {

		fmt.Println("ymake - Ya Make build system reimplementation")
	})

	if err := recover(); err != nil {
		if exc, ok := err.(*Exception); ok {
			fmt.Fprintf(stderr, "Error: %v\n", exc.AsError())
		} else {
			panic(err)
		}
	}
}
