package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParserProgramModule(t *testing.T) {
	parser := NewParser(&ParseContext{})

	tmpDir, err := os.MkdirTemp("", "ya-make-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yaMakePath := filepath.Join(tmpDir, "ya.make")
	testContent := `PROGRAM()

PEERDIR(
    library/cpp/archive
)

SRCS(
    main.cpp
    util.cpp
)

SET(CXX_FLAGS "-O2")

END()
`
	err = os.WriteFile(yaMakePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	file, err := parser.ParseFile(yaMakePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]
	if module.Type != ModuleTypeProgram {
		t.Errorf("expected ModuleTypeProgram, got %v", module.Type)
	}

	if len(module.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(module.Dependencies))
	}

	if module.Dependencies[0] != "library/cpp/archive" {
		t.Errorf("expected dependency 'library/cpp/archive', got '%s'", module.Dependencies[0])
	}

	if len(module.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(module.Sources))
	}

	if module.Sources[0] != "main.cpp" {
		t.Errorf("expected first source 'main.cpp', got '%s'", module.Sources[0])
	}

	if module.Properties["CXX_FLAGS"] != "-O2" {
		t.Errorf("expected CXX_FLAGS '-O2', got '%s'", module.Properties["CXX_FLAGS"])
	}
}

func TestParserLibraryModule(t *testing.T) {
	parser := NewParser(&ParseContext{})

	tmpDir, err := os.MkdirTemp("", "ya-make-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yaMakePath := filepath.Join(tmpDir, "ya.make")
	testContent := `LIBRARY()

PEERDIR(
    library/cpp/base
)

SRCS(
    archive.cpp
)

END()
`
	err = os.WriteFile(yaMakePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	file, err := parser.ParseFile(yaMakePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]
	if module.Type != ModuleTypeLibrary {
		t.Errorf("expected ModuleTypeLibrary, got %v", module.Type)
	}
}

func TestParserRecurseDirective(t *testing.T) {
	parser := NewParser(&ParseContext{})

	tmpDir, err := os.MkdirTemp("", "ya-make-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yaMakePath := filepath.Join(tmpDir, "ya.make")
	testContent := `PROGRAM()

RECURSE(
    ut
    bench
)

END()
`
	err = os.WriteFile(yaMakePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	file, err := parser.ParseFile(yaMakePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]
	if len(module.Recursions) != 1 {
		t.Errorf("expected 1 recursion directive, got %d", len(module.Recursions))
	}

	if len(module.Recursions[0].Paths) != 2 {
		t.Errorf("expected 2 recurse paths, got %d", len(module.Recursions[0].Paths))
	}

	if module.Recursions[0].Paths[0] != "ut" {
		t.Errorf("expected first recurse path 'ut', got '%s'", module.Recursions[0].Paths[0])
	}
}

func TestParserConditionalBlock(t *testing.T) {
	parser := NewParser(&ParseContext{})

	tmpDir, err := os.MkdirTemp("", "ya-make-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yaMakePath := filepath.Join(tmpDir, "ya.make")
	testContent := `PROGRAM()

IF(OS_LINUX)
SET(PLATFORM linux)
ENDIF()

END()
`
	err = os.WriteFile(yaMakePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	file, err := parser.ParseFile(yaMakePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]
	if len(module.Conditionals) != 1 {
		t.Errorf("expected 1 conditional block, got %d", len(module.Conditionals))
	}

	block := module.Conditionals[0]
	if block.IfBranch == nil {
		t.Fatal("expected IfBranch")
	}
}

func TestParserBuildOnlyIf(t *testing.T) {
	parser := NewParser(&ParseContext{})

	tmpDir, err := os.MkdirTemp("", "ya-make-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yaMakePath := filepath.Join(tmpDir, "ya.make")
	testContent := `PROGRAM()

BUILD_ONLY_IF(OS_LINUX)

SRCS(
    main.cpp
)

END()
`
	err = os.WriteFile(yaMakePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	file, err := parser.ParseFile(yaMakePath)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]
	if module.BuildCondition == nil {
		t.Fatal("expected BuildCondition")
	}

	if module.BuildCondition.Expression != "OS_LINUX" {
		t.Errorf("expected BuildCondition.Expression 'OS_LINUX', got '%s'", module.BuildCondition.Expression)
	}
}
