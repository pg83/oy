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

func TestParseLibcxxWithCommaFlags(t *testing.T) {
	content := `LIBRARY()

IF (OS_EMSCRIPTEN)
    SET(CXX_RT "libcxxabi")
    LDFLAGS(-Wl,-Bdynamic)
    CXXFLAGS(-nostdinc++)
ENDIF()

END()
`

	module := NewParser(content, "test/ya.make").parseModule(ModuleTypeLibrary, "test")
	if module == nil {
		t.Fatal("expected non-nil module")
	}

	if module.Type != ModuleTypeLibrary {
		t.Errorf("expected ModuleTypeLibrary, got %v", module.Type)
	}
}

func TestParseMuslWithCommaFlags(t *testing.T) {
	content := `LIBRARY()

CFLAGS(
    GLOBAL -D_musl_=1
    -nostdinc
)

LDFLAGS(-static)

IF (NOT WITH_VALGRIND)
    LDFLAGS(-Wl,--no-dynamic-linker)
ENDIF()

END()
`

	module := NewParser(content, "test/ya.make").parseModule(ModuleTypeLibrary, "test")
	if module == nil {
		t.Fatal("expected non-nil module")
	}

	if module.Type != ModuleTypeLibrary {
		t.Errorf("expected ModuleTypeLibrary, got %v", module.Type)
	}
}

func TestParseYaMakeFile_WhenBlockSimple(t *testing.T) {
	content := `PROGRAM()

PEERDIR(test/lib) WHEN($VAR == yes)

END()
`

	module := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if module == nil {
		t.Fatal("expected non-nil module")
	}

	if len(module.ConditionalPeerdirs) != 1 {
		t.Fatalf("expected 1 conditional PEERDIR, got %d", len(module.ConditionalPeerdirs))
	}

	cp := module.ConditionalPeerdirs[0]
	if len(cp.Paths) != 1 {
		t.Fatalf("expected 1 path in conditional PEERDIR, got %d", len(cp.Paths))
	}

	if cp.Paths[0] != "test/lib" {
		t.Errorf("expected path 'test/lib', got '%s'", cp.Paths[0])
	}

	if cp.WhenClause == nil {
		t.Fatal("expected non-nil WhenClause")
	}

	if cp.WhenClause.Condition != "$VAR == yes" {
		t.Errorf("expected condition '$VAR == yes', got '%s'", cp.WhenClause.Condition)
	}
}

func TestParseYaMakeFile_WhenBlockMultiplePaths(t *testing.T) {
	content := `PROGRAM()

PEERDIR(
    lib/a
    lib/b
) WHEN($X == val)

END()
`

	module := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(module.ConditionalPeerdirs) != 1 {
		t.Fatalf("expected 1 conditional PEERDIR, got %d", len(module.ConditionalPeerdirs))
	}

	cp := module.ConditionalPeerdirs[0]
	if len(cp.Paths) != 2 {
		t.Fatalf("expected 2 paths in conditional PEERDIR, got %d", len(cp.Paths))
	}

	if cp.Paths[0] != "lib/a" {
		t.Errorf("expected path 'lib/a', got '%s'", cp.Paths[0])
	}

	if cp.Paths[1] != "lib/b" {
		t.Errorf("expected path 'lib/b', got '%s'", cp.Paths[1])
	}
}

func TestParseYaMakeFile_MixedPeerdirs(t *testing.T) {
	content := `PROGRAM()

PEERDIR(lib/always)
PEERDIR(lib/conditional) WHEN($FLAG == yes)

END()
`

	module := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(module.Dependencies) != 1 {
		t.Fatalf("expected 1 unconditional dependency, got %d", len(module.Dependencies))
	}

	if module.Dependencies[0] != "lib/always" {
		t.Errorf("expected dependency 'lib/always', got '%s'", module.Dependencies[0])
	}

	if len(module.ConditionalPeerdirs) != 1 {
		t.Fatalf("expected 1 conditional PEERDIR, got %d", len(module.ConditionalPeerdirs))
	}
}

func TestParseYaMakeFile_ComplexWhenCondition(t *testing.T) {
	content := `PROGRAM()

PEERDIR(lib/x) WHEN($A == val && $B == val2)

END()
`

	module := NewParser(content, "test/ya.make").parseModule(ModuleTypeProgram, "test")
	if len(module.ConditionalPeerdirs) != 1 {
		t.Fatalf("expected 1 conditional PEERDIR, got %d", len(module.ConditionalPeerdirs))
	}

	cp := module.ConditionalPeerdirs[0]
	if cp.WhenClause == nil {
		t.Fatal("expected non-nil WhenClause")
	}

	if cp.WhenClause.Condition != "$A == val && $B == val2" {
		t.Errorf("expected condition '$A == val && $B == val2', got '%s'", cp.WhenClause.Condition)
	}
}

func TestParseJoinSrcs(t *testing.T) {
	content := `LIBRARY()
JOIN_SRCS(
    all_charset.cpp
    generated/unidata.cpp
    recode_result.cpp
)
END()
`

	file := ParseYaMakeString(content, "test/ya.make")
	t.Logf("Number of modules parsed: %d", len(file.Modules))
	if len(file.Modules) != 1 {
		t.Fatalf("Expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]
	t.Logf("Module type: %v", module.Type)
	t.Logf("Number of JOIN_SRCS directives: %d", len(module.JoinSrcsDirectives))

	if len(module.JoinSrcsDirectives) != 1 {
		t.Fatalf("Expected 1 JOIN_SRCS directive, got %d", len(module.JoinSrcsDirectives))
	}

	jsd := module.JoinSrcsDirectives[0]
	if jsd.OutputFile != "all_charset.cpp" {
		t.Errorf("Expected output 'all_charset.cpp', got '%s'", jsd.OutputFile)
	}

	expectedInputs := []string{"generated/unidata.cpp", "recode_result.cpp"}
	if len(jsd.InputFiles) != len(expectedInputs) {
		t.Fatalf("Expected %d input files, got %d", len(expectedInputs), len(jsd.InputFiles))
	}

	for i, input := range jsd.InputFiles {
		if input != expectedInputs[i] {
			t.Errorf("Input %d: expected '%s', got '%s'", i, expectedInputs[i], input)
		}
	}
}

func TestParserNoPlatform(t *testing.T) {
	input := `PROGRAM()
NO_PLATFORM()
END()
`
	parser := NewParser(input, "test.make")
	file := parser.Parse()

	if len(file.Modules) != 1 {
		t.Fatal("expected 1 module")
	}

	module := file.Modules[0]
	if !module.NoPlatform {
		t.Errorf("expected NoPlatform to be true")
	}
}

func TestParserNoPlatformMultipleDirectives(t *testing.T) {
	input := `LIBRARY()
PEERDIR(util)
NO_PLATFORM()
SRCS(main.cpp)
END()
`
	parser := NewParser(input, "test.make")
	file := parser.Parse()

	if len(file.Modules) != 1 {
		t.Fatal("expected 1 module")
	}

	module := file.Modules[0]
	if !module.NoPlatform {
		t.Errorf("expected NoPlatform to be true")
	}

	if len(module.Dependencies) != 1 || module.Dependencies[0] != "util" {
		t.Errorf("expected PEERDIR util")
	}
}
