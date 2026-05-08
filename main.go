package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {

	if exc := Try(func() {

		if len(os.Args) > 1 && os.Args[1] == "lex" {
			lexToolMain()
			return
		}

		result := Throw2(ParseFlags(os.Args))

		vars := NewMemoryVariableSet()

		for k, v := range result.PlatformFlag.ToMap() {
			vars.SetValue(k, v)
		}

		fullVars := NewBuildContextVariableSet(result.Ctx, vars)

		if len(result.Args) == 0 {
			fmt.Printf("Usage: ymake [flags] <target-path>\n")
			fmt.Printf("Example: ymake --musl tools/archiver\n")
			os.Exit(1)
		}

		targetPath := result.Args[0]

		registry := NewModuleRegistry()
		graph := NewGraph(result.Ctx)
		parser := NewFileParser(registry, graph, result.Ctx, fullVars)

		yaMakePath := ""
		if IsDir(targetPath) {
			yaMakePath = filepath.Join(targetPath, "ya.make")
		} else {
			yaMakePath = targetPath
		}

		fmt.Printf("Parsing %s...\n", yaMakePath)
		parser.Parse(yaMakePath)

		fmt.Printf("Successfully parsed %d modules and generated %d graph nodes\n", registry.Count(), len(graph.Nodes))

	}); exc != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", exc.AsError())
		os.Exit(1)
	}

}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
