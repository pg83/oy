package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileParserBasic(t *testing.T) {
	tempDir := t.TempDir()

	yaMakeContent := `PROGRAM()
PEERDIR(library/cpp/archive)
SRCS(main.cpp)
END()
`

	yaMakePath := filepath.Join(tempDir, "ya.make")
	if err := os.WriteFile(yaMakePath, []byte(yaMakeContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := NewModuleRegistry()
	buildCtx := NewBuildContext()
	innerVars := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(buildCtx, innerVars)
	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: tempDir,
		Language:   "cpp",
	}
	graph := NewGraph(ctx)

	parser := NewFileParser(registry, graph, buildCtx, vars)
	parser.Parse(yaMakePath)

	if registry.Count() != 1 {
		t.Errorf("expected 1 module, got %d", registry.Count())
	}

	if len(graph.Nodes) != 1 {
		t.Errorf("expected 1 graph node, got %d", len(graph.Nodes))
	}

	firstModule := registry.Get(tempDir)
	if firstModule == nil {
		t.Fatal("expected to find module in registry")
	}

	if firstModule.Type != ModuleTypeProgram {
		t.Errorf("expected ModuleTypeProgram, got %v", firstModule.Type)
	}
}

func TestFileParserRecurse(t *testing.T) {
	tempDir := t.TempDir()

	rootYaMake := `PROGRAM()
PEERDIR(lib1/lib1)
SRCS(main.cpp)
RECURSE(lib1)
END()
`

	rootPath := filepath.Join(tempDir, "ya.make")
	if err := os.WriteFile(rootPath, []byte(rootYaMake), 0644); err != nil {
		t.Fatal(err)
	}

	lib1Path := filepath.Join(tempDir, "lib1")
	if err := os.Mkdir(lib1Path, 0755); err != nil {
		t.Fatal(err)
	}

	lib1YaMake := `LIBRARY()
PEERDIR(library/cpp/archive)
SRCS(lib1.cpp)
END()
`

	lib1YaMakePath := filepath.Join(lib1Path, "ya.make")
	if err := os.WriteFile(lib1YaMakePath, []byte(lib1YaMake), 0644); err != nil {
		t.Fatal(err)
	}

	registry := NewModuleRegistry()
	buildCtx := NewBuildContext()
	innerVars := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(buildCtx, innerVars)
	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: tempDir,
		Language:   "cpp",
	}

	graph := NewGraph(ctx)
	parser := NewFileParser(registry, graph, buildCtx, vars)
	parser.Parse(rootPath)

	if registry.Count() != 2 {
		t.Errorf("expected 2 modules, got %d", registry.Count())
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("expected 2 graph nodes, got %d", len(graph.Nodes))
	}

	if !registry.Has(tempDir) {
		t.Error("expected to find root module in registry")
	}

	if !registry.Has(lib1Path) {
		t.Error("expected to find lib1 module in registry")
	}
}

func TestFileParserConditional(t *testing.T) {
	tempDir := t.TempDir()

	yaMakeContent := `PROGRAM()
IF(OS_LINUX)
PEERDIR(library/cpp/archive)
ELSE()
PEERDIR(library/cpp/archive_win)
ENDIF()
SRCS(main.cpp)
END()
`

	yaMakePath := filepath.Join(tempDir, "ya.make")
	if err := os.WriteFile(yaMakePath, []byte(yaMakeContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := NewModuleRegistry()
	buildCtx := NewBuildContext()
	innerVars := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(buildCtx, innerVars)
	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: tempDir,
		Language:   "cpp",
	}

	graph := NewGraph(ctx)
	parser := NewFileParser(registry, graph, buildCtx, vars)
	parser.Parse(yaMakePath)

	module := registry.Get(tempDir)
	if module == nil {
		t.Fatal("expected to find module in registry")
	}

	hasLinuxDep := false
	for _, dep := range module.Dependencies {
		if dep == "library/cpp/archive" {
			hasLinuxDep = true
			break
		}
	}

	if !hasLinuxDep {
		t.Error("expected to find linux-specific dependency on Linux platform")
	}
}

func TestFileParserBuildOnlyIf(t *testing.T) {
	tempDir := t.TempDir()
	winBuildCtx := &BuildContext{Platform: PlatformWindows, Arch: Arch64, Compiler: CompilerMSVC}

	yaMakeContent := `PROGRAM()
BUILD_ONLY_IF(OS_WINDOWS)
PEERDIR(library/cpp/archive)
SRCS(main.cpp)
END()
`

	yaMakePath := filepath.Join(tempDir, "ya.make")
	if err := os.WriteFile(yaMakePath, []byte(yaMakeContent), 0644); err != nil {
		t.Fatal(err)
	}

	registry := NewModuleRegistry()
	innerVars := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(winBuildCtx, innerVars)
	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: tempDir,
		Language:   "cpp",
	}

	graph := NewGraph(ctx)
	parser := NewFileParser(registry, graph, winBuildCtx, vars)
	parser.Parse(yaMakePath)

	if registry.Count() != 0 {
		t.Errorf("expected 0 modules (BUILD_ONLY_IF should filter on Linux), got %d", registry.Count())
	}

	if len(graph.Nodes) != 0 {
		t.Errorf("expected 0 graph nodes (BUILD_ONLY_IF should filter on Linux), got %d", len(graph.Nodes))
	}
}

func TestFileParserVisitTracking(t *testing.T) {
	tempDir := t.TempDir()

	yaMakeContent := `PROGRAM()
RECURSE(lib1)
RECURSE(lib1)
PEERDIR(lib1/lib1)
SRCS(main.cpp)
END()
`

	rootPath := filepath.Join(tempDir, "ya.make")
	if err := os.WriteFile(rootPath, []byte(yaMakeContent), 0644); err != nil {
		t.Fatal(err)
	}

	lib1Path := filepath.Join(tempDir, "lib1")
	if err := os.Mkdir(lib1Path, 0755); err != nil {
		t.Fatal(err)
	}

	lib1YaMake := `LIBRARY()
SRCS(lib1.cpp)
END()
`

	lib1YaMakePath := filepath.Join(lib1Path, "ya.make")
	if err := os.WriteFile(lib1YaMakePath, []byte(lib1YaMake), 0644); err != nil {
		t.Fatal(err)
	}

	registry := NewModuleRegistry()
	buildCtx := NewBuildContext()
	innerVars := NewMemoryVariableSet()
	vars := NewBuildContextVariableSet(buildCtx, innerVars)
	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: tempDir,
		Language:   "cpp",
	}

	graph := NewGraph(ctx)
	parser := NewFileParser(registry, graph, buildCtx, vars)
	parser.Parse(rootPath)

	if registry.Count() != 2 {
		t.Errorf("expected 2 modules (duplicate RECURSE should not parse twice), got %d", registry.Count())
	}
}
