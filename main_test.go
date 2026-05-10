package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCLIBuildTargetAbsoluteDirectory(t *testing.T) {
	sourceRoot := makeTestSourceRoot(t)
	targetPath := filepath.Join(sourceRoot, "tools", "archiver")

	resolvedTarget, resolvedSourceRoot := resolveCLIBuildTarget(targetPath, "/unused")

	if resolvedSourceRoot != sourceRoot {
		t.Fatalf("expected source root %q, got %q", sourceRoot, resolvedSourceRoot)
	}

	if resolvedTarget != "tools/archiver" {
		t.Fatalf("expected target tools/archiver, got %q", resolvedTarget)
	}
}

func TestResolveCLIBuildTargetAbsoluteYaMake(t *testing.T) {
	sourceRoot := makeTestSourceRoot(t)
	targetPath := filepath.Join(sourceRoot, "tools", "archiver", "ya.make")

	resolvedTarget, resolvedSourceRoot := resolveCLIBuildTarget(targetPath, "/unused")

	if resolvedSourceRoot != sourceRoot {
		t.Fatalf("expected source root %q, got %q", sourceRoot, resolvedSourceRoot)
	}

	if resolvedTarget != "tools/archiver" {
		t.Fatalf("expected target tools/archiver, got %q", resolvedTarget)
	}
}

func TestResolveCLIBuildTargetRelative(t *testing.T) {
	cwd := t.TempDir()

	resolvedTarget, resolvedSourceRoot := resolveCLIBuildTarget("tools/archiver/ya.make", cwd)

	if resolvedSourceRoot != cwd {
		t.Fatalf("expected source root %q, got %q", cwd, resolvedSourceRoot)
	}

	if resolvedTarget != "tools/archiver" {
		t.Fatalf("expected target tools/archiver, got %q", resolvedTarget)
	}
}

func TestFindTargetArgSkipsSupportedFlags(t *testing.T) {
	args := []string{
		"-G",
		"--graph-file=/tmp/graph.json",
		"--musl",
		"--target-platform=linux",
		"--host-platform-flag=OS_LINUX=true",
		"--target-platform-flag=ARCH_X86_64=true",
		"--language=cpp",
		"tools/archiver",
	}

	target := findTargetArg(args)

	if target != "tools/archiver" {
		t.Fatalf("expected target tools/archiver, got %q", target)
	}
}

func TestBuildParseContextUsesResolvedTargetAndFlags(t *testing.T) {
	result := &ParseFlagsResult{
		Ctx: &BuildContext{
			Language: "cpp",
			Musl:     true,
		},
		PlatformFlag: NewPlatformFlags(),
	}
	result.PlatformFlag.SetTargetFlag("OS_LINUX", "true")

	ctx := buildParseContext(result, "tools/archiver")

	if ctx.Platform != "linux" {
		t.Fatalf("expected linux platform, got %q", ctx.Platform)
	}

	if ctx.TargetPath != "tools/archiver" {
		t.Fatalf("expected target tools/archiver, got %q", ctx.TargetPath)
	}

	if ctx.Language != "cpp" {
		t.Fatalf("expected language cpp, got %q", ctx.Language)
	}

	if !ctx.Musl {
		t.Fatal("expected musl to be true")
	}

	if ctx.BuildFlags["OS_LINUX"] != "true" {
		t.Fatalf("expected OS_LINUX=true, got %q", ctx.BuildFlags["OS_LINUX"])
	}
}

func TestCLIBuildTargetMatchesIntegrationTarget(t *testing.T) {
	if _, err := os.Stat(SourceRoot); err != nil {
		t.Skipf("reference source root unavailable: %v", err)
	}

	absTarget := filepath.Join(SourceRoot, "tools", "archiver")
	resolvedTarget, resolvedSourceRoot := resolveCLIBuildTarget(absTarget, "/unused")

	if resolvedTarget != ReferenceTarget {
		t.Fatalf("expected target %q, got %q", ReferenceTarget, resolvedTarget)
	}

	if resolvedSourceRoot != SourceRoot {
		t.Fatalf("expected source root %q, got %q", SourceRoot, resolvedSourceRoot)
	}

	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: resolvedTarget,
		BuildFlags: make(map[string]string),
	}
	cliGraph, cliErr := BuildDependencyGraph(resolvedTarget, ctx, resolvedSourceRoot, nil)
	if cliErr != nil {
		t.Fatalf("CLI-style BuildDependencyGraph failed: %v", cliErr)
	}

	integrationCtx := ParseContext{
		Platform:   "linux",
		TargetPath: ReferenceTarget,
		BuildFlags: make(map[string]string),
	}
	integrationGraph, integrationErr := BuildDependencyGraph(ReferenceTarget, integrationCtx, SourceRoot, nil)
	if integrationErr != nil {
		t.Fatalf("integration BuildDependencyGraph failed: %v", integrationErr)
	}

	if len(cliGraph.Nodes) != len(integrationGraph.Nodes) {
		t.Fatalf("expected CLI graph node count %d, got %d", len(integrationGraph.Nodes), len(cliGraph.Nodes))
	}
}

func makeTestSourceRoot(t *testing.T) string {
	t.Helper()

	sourceRoot := t.TempDir()
	Throw(os.MkdirAll(filepath.Join(sourceRoot, "build"), 0755))
	Throw(os.MkdirAll(filepath.Join(sourceRoot, "tools", "archiver"), 0755))
	Throw(os.WriteFile(filepath.Join(sourceRoot, "tools", "archiver", "ya.make"), []byte("PROGRAM()\nEND()\n"), 0644))

	return sourceRoot
}
