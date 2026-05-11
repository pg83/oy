package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {

	if exc := Try(func() {

		if len(os.Args) > 1 && os.Args[1] == "lex" {
			lexToolMain()
			return
		}

		result := Throw2(ParseFlags(os.Args))

		targetPath := findTargetArg(os.Args[1:])

		if targetPath == "" {
			fmt.Printf("Usage: ymake [flags] <target-path>\n")
			fmt.Printf("Example: ymake --musl tools/archiver\n")
			fmt.Printf("Flags:\n")
			fmt.Printf("  --benchmark          Run performance benchmark (multiple iterations)\n")
			fmt.Printf("  --diag-peerdir       Log PEERDIR traversal diagnostics\n")
			fmt.Printf("  --diag-eval          Log variable evaluation diagnostics in conditionals\n")
			fmt.Printf("  --diag-tool-modules  Log tool module loading and node creation diagnostics\n")
			os.Exit(1)
		}

		cwd := Throw2(os.Getwd())
		resolvedTarget, sourceRoot := resolveCLIBuildTarget(targetPath, cwd)
		ctx := buildParseContext(result, resolvedTarget)

		var diag *TraversalLogger
		if result.DiagPeerdir || result.DiagEval || result.DiagToolMod {
			diag = NewTraversalLogger(true, result.DiagEval, result.DiagToolMod)
			SetGlobalTraversalLogger(diag)
		}
		defer func() {
			if diag != nil {
				diag.OutputSummary()
			}
		}()

		var graph *Graph
		if result.Benchmark {
			report := MeasureGraphGeneration(resolvedTarget, ctx, sourceRoot, diag, 5)
			graph = Throw2(BuildDependencyGraph(resolvedTarget, ctx, sourceRoot, diag))
			fmt.Printf("Benchmark results: median %v, %d runs, %d nodes\n", report.Median, len(report.Runs), len(graph.Nodes))
		} else {
			fmt.Printf("Building graph for %s from %s...\n", resolvedTarget, sourceRoot)
			graph = Throw2(BuildDependencyGraph(resolvedTarget, ctx, sourceRoot, diag))
			fmt.Printf("Successfully generated %d graph nodes\n", len(graph.Nodes))
		}

		if result.OutputJSON {
			outputPath := "sg.json"
			if result.JSONPath != "" {
				outputPath = result.JSONPath
			}

			fmt.Printf("Writing graph to %s...\n", outputPath)
			WriteGraphToFile(graph, outputPath)
			fmt.Printf("Graph written successfully\n")

			if result.ValidatePath != "" {
				comparison := CompareGraphFiles(result.ValidatePath, outputPath, GraphComparisonOptions{})
				if err := comparison.Err(); err != nil {
					Throw(err)
				}

				fmt.Printf("Graph validation passed against %s\n", result.ValidatePath)
			}

			os.Exit(0)
		}

	}); exc != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", exc.AsError())
		os.Exit(1)
	}

}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func findTargetArg(args []string) string {
	for _, arg := range args {
		if isCLIFlagArg(arg) {
			continue
		}

		return arg
	}

	return ""
}

func isCLIFlagArg(arg string) bool {
	if arg == "" {
		return true
	}

	switch arg {
	case "lex", "-G", "--graph", "--musl", "--benchmark", "--diag-peerdir", "--diag-eval", "--diag-tool-modules":
		return true
	}

	return arg[0] == '-'
}

func resolveCLIBuildTarget(targetPath string, cwd string) (string, string) {
	if !filepath.IsAbs(targetPath) {
		return normalizeCLITargetPath(targetPath), cwd
	}

	startDir := cliTargetStartDir(targetPath)
	sourceRoot := findSourceRoot(startDir)
	if sourceRoot == "" {
		return fallbackBuildTarget(targetPath), fallbackSourceRoot(targetPath)
	}

	relTarget, err := filepath.Rel(sourceRoot, targetPath)
	if err != nil {
		return targetPath, sourceRoot
	}

	return normalizeCLITargetPath(filepath.ToSlash(relTarget)), sourceRoot
}

func cliTargetStartDir(targetPath string) string {
	if filepath.Base(targetPath) == "ya.make" {
		return filepath.Dir(targetPath)
	}

	if IsDir(targetPath) {
		return targetPath
	}

	return filepath.Dir(targetPath)
}

func findSourceRoot(startDir string) string {
	dir := filepath.Clean(startDir)

	for {
		if IsDir(filepath.Join(dir, "build")) && IsDir(filepath.Join(dir, "tools")) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fallbackSourceRoot(targetPath string) string {
	if IsDir(targetPath) && hasYaMakeFile(targetPath) {
		return targetPath
	}

	return filepath.Dir(targetPath)
}

func fallbackBuildTarget(targetPath string) string {
	if IsDir(targetPath) && hasYaMakeFile(targetPath) {
		return "ya.make"
	}

	return targetPath
}

func hasYaMakeFile(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "ya.make"))

	return err == nil && !info.IsDir()
}

func normalizeCLITargetPath(targetPath string) string {
	targetPath = filepath.ToSlash(filepath.Clean(targetPath))

	if filepath.Base(targetPath) == "ya.make" {
		return filepath.ToSlash(filepath.Dir(targetPath))
	}

	return targetPath
}

func buildParseContext(result *ParseFlagsResult, targetPath string) ParseContext {
	archString := "x86_64"
	if runtime.GOARCH == "arm64" {
		archString = "aarch64"
	}
	return ParseContext{
		Platform:       "linux",
		ArchString:     archString,
		TargetPath:     targetPath,
		Language:       result.Ctx.Language,
		Musl:           result.Ctx.Musl,
		BuildFlags:     result.PlatformFlag.ToMap(),
		TargetPlatform: result.Ctx.TargetPlatform,
	}
}
