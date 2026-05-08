package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

		registry := NewModuleRegistry()
		ctx := ParseContext{
			Platform:   "linux",
			TargetPath: "",
			Language:   result.Ctx.Language,
			Musl:       result.Ctx.Musl,
		}
		graph := NewGraph(ctx)
		parser := NewFileParser(registry, graph, result.Ctx, fullVars)

		targetPath := ""
		for _, arg := range os.Args[1:] {
			if arg == "lex" || arg == "--musl" || strings.HasPrefix(arg, "--target-platform") ||
				strings.HasPrefix(arg, "--host-platform-flag") || strings.HasPrefix(arg, "--target-platform-flag") ||
				strings.HasPrefix(arg, "--language=") {
				continue
			}
			if arg != "" && arg[0] != '-' {
				targetPath = arg
				break
			}
		}

		if targetPath == "" {
			fmt.Printf("Usage: ymake [flags] <target-path>\n")
			fmt.Printf("Example: ymake --musl tools/archiver\n")
			os.Exit(1)
		}

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
