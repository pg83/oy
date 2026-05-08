package main

import (
	"testing"
)

func TestParseYaMakeFile_SimpleProgram(t *testing.T) {
	content := `PROGRAM()

PEERDIR(
    library/cpp/archive
)

SRCS(
    main.cpp
)

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if file == nil {
		t.Fatal("expected non-nil module")
	}

	if file.Type != ModuleTypeProgram {
		t.Errorf("expected ModuleTypeProgram, got %v", file.Type)
	}

	if len(file.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(file.Dependencies))
	}

	if file.Dependencies[0] != "library/cpp/archive" {
		t.Errorf("expected dependency 'library/cpp/archive', got '%s'", file.Dependencies[0])
	}

	if len(file.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(file.Sources))
	}

	if file.Sources[0] != "main.cpp" {
		t.Errorf("expected source 'main.cpp', got '%s'", file.Sources[0])
	}
}

func TestParseYaMakeFile_MultiplePeerdirs(t *testing.T) {
	content := `PROGRAM()

PEERDIR(
    library/cpp/archive
    library/cpp/digest/md5
    library/cpp/getopt/small
)

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(file.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(file.Dependencies))
	}

	expectedDeps := []string{"library/cpp/archive", "library/cpp/digest/md5", "library/cpp/getopt/small"}
	for i, dep := range file.Dependencies {
		if dep != expectedDeps[i] {
			t.Errorf("expected dependency '%s', got '%s'", expectedDeps[i], dep)
		}
	}
}

func TestParseYaMakeFile_SetStatement(t *testing.T) {
	content := `PROGRAM()

SET(IDE_FOLDER "_Builders")

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	value, ok := file.GetProperty("IDE_FOLDER")
	if !ok || value != "_Builders" {
		t.Errorf("expected IDE_FOLDER='_Builders', got '%s', ok=%v", value, ok)
	}
}

func TestParseYaMakeFile_IfBlock(t *testing.T) {
	content := `PROGRAM()

IF (OS_WINDOWS)
    PEERDIR(library/cpp/windows)
ENDIF()

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(file.Conditionals) != 1 {
		t.Fatalf("expected 1 conditional, got %d", len(file.Conditionals))
	}

	cond := file.Conditionals[0]
	if cond.IfBranch.Condition != "OS_WINDOWS" {
		t.Errorf("expected condition 'OS_WINDOWS', got '%s'", cond.IfBranch.Condition)
	}

	if len(cond.IfBranch.Module.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency in if branch, got %d", len(cond.IfBranch.Module.Dependencies))
	}

	if cond.IfBranch.Module.Dependencies[0] != "library/cpp/windows" {
		t.Errorf("expected dependency 'library/cpp/windows', got '%s'", cond.IfBranch.Module.Dependencies[0])
	}
}

func TestParseYaMakeFile_IfElseIfElse(t *testing.T) {
	content := `PROGRAM()

IF (OS_WINDOWS)
    PEERDIR(library/cpp/windows)
ELSEIF (OS_DARWIN)
    PEERDIR(library/cpp/darwin)
ELSE()
    PEERDIR(library/cpp/unix)
ENDIF()

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(file.Conditionals) != 1 {
		t.Fatalf("expected 1 conditional, got %d", len(file.Conditionals))
	}

	cond := file.Conditionals[0]
	if len(cond.ElseIfs) != 1 {
		t.Fatalf("expected 1 elseif, got %d", len(cond.ElseIfs))
	}

	if cond.ElseIfs[0].Condition != "OS_DARWIN" {
		t.Errorf("expected condition 'OS_DARWIN', got '%s'", cond.ElseIfs[0].Condition)
	}

	if cond.ElseBranch == nil {
		t.Fatal("expected non-nil else branch")
	}

	if len(cond.ElseBranch.Module.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency in else branch, got %d", len(cond.ElseBranch.Module.Dependencies))
	}

	if cond.ElseBranch.Module.Dependencies[0] != "library/cpp/unix" {
		t.Errorf("expected dependency 'library/cpp/unix', got '%s'", cond.ElseBranch.Module.Dependencies[0])
	}
}

func TestParseYaMakeFile_BuildOnlyIf(t *testing.T) {
	content := `PROGRAM()

BUILD_ONLY_IF(OS_LINUX)

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if file.BuildCondition == nil {
		t.Fatal("expected non-nil BuildCondition")
	}

	if file.BuildCondition.Expression != "OS_LINUX" {
		t.Errorf("expected expression 'OS_LINUX', got '%s'", file.BuildCondition.Expression)
	}
}

func TestParseYaMakeFile_Recurse(t *testing.T) {
	content := `PROGRAM()

RECURSE(ut bench)

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(file.Recursions) != 1 {
		t.Fatalf("expected 1 recurse, got %d", len(file.Recursions))
	}

	recurse := file.Recursions[0]
	if len(recurse.Paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(recurse.Paths))
	}

	if recurse.Paths[0] != "ut" {
		t.Errorf("expected path 'ut', got '%s'", recurse.Paths[0])
	}

	if recurse.Paths[1] != "bench" {
		t.Errorf("expected path 'bench', got '%s'", recurse.Paths[1])
	}
}

func TestParseYaMakeFile_Library(t *testing.T) {
	content := `LIBRARY()

PEERDIR(library/cpp/base)

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeLibrary, "test")
	if file.Type != ModuleTypeLibrary {
		t.Errorf("expected ModuleTypeLibrary, got %v", file.Type)
	}
}

func TestParseYaMakeFile_GoLibrary(t *testing.T) {
	content := `GO_LIBRARY()

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeGoLibrary, "test")
	if file.Type != ModuleTypeGoLibrary {
		t.Errorf("expected ModuleTypeGoLibrary, got %v", file.Type)
	}
}

func TestParseYaMakeFile_ComplexCondition(t *testing.T) {
	content := `PROGRAM()

IF (OS_WINDOWS AND ARCH_X86_64)
    PEERDIR(library/cpp/windows_x64)
ENDIF()

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(file.Conditionals) != 1 {
		t.Fatalf("expected 1 conditional, got %d", len(file.Conditionals))
	}

	cond := file.Conditionals[0]
	
	expectedCondition := "OS_WINDOWS AND ARCH_X86_64"
	if cond.IfBranch.Condition != expectedCondition {
		t.Errorf("expected condition '%s', got '%s'", expectedCondition, cond.IfBranch.Condition)
	}
}

func TestParseYaMakeFile_NestedConditionals(t *testing.T) {
	content := `PROGRAM()

IF (OS_WINDOWS)
    IF (CLANG)
        PEERDIR(library/cpp/clang)
    ENDIF()
ENDIF()

END()
`

	file := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(file.Conditionals) != 1 {
		t.Fatalf("expected 1 conditional, got %d", len(file.Conditionals))
	}
}