package main

import (
	"testing"
)

func TestIncludeDirectiveParsing(t *testing.T) {
	input := `LIBRARY()
INCLUDE(test.inc)
END()
`

	file := ParseYaMakeString(input, "test.yamake")

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	if len(module.IncludeDirectives) != 1 {
		t.Fatalf("expected 1 include directive, got %d", len(module.IncludeDirectives))
	}

	include := module.IncludeDirectives[0]

	if include.FilePath != "test.inc" {
		t.Fatalf("expected include path 'test.inc', got '%s'", include.FilePath)
	}
}

func TestIncludeDirectiveInConditional(t *testing.T) {
	input := `LIBRARY()
IF (ARCH_X86_64)
  INCLUDE(x86_64.inc)
ENDIF()
END()
`

	file := ParseYaMakeString(input, "test.yamake")

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	if len(module.Conditionals) != 1 {
		t.Fatalf("expected 1 conditional block, got %d", len(module.Conditionals))
	}

	cond := module.Conditionals[0]

	if cond.IfBranch == nil {
		t.Fatal("expected IF branch")
	}

	if len(cond.IfBranch.Module.IncludeDirectives) != 1 {
		t.Fatalf("expected 1 include directive in IF branch, got %d", len(cond.IfBranch.Module.IncludeDirectives))
	}

	include := cond.IfBranch.Module.IncludeDirectives[0]

	if include.FilePath != "x86_64.inc" {
		t.Fatalf("expected include path 'x86_64.inc', got '%s'", include.FilePath)
	}
}

func TestIncludeDirectiveInElseBranch(t *testing.T) {
	input := `LIBRARY()
IF (ARCH_X86_64)
  SRCS(x86_64.cpp)
ELSE()
  INCLUDE(other.inc)
ENDIF()
END()
`

	file := ParseYaMakeString(input, "test.yamake")

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	if len(module.Conditionals) != 1 {
		t.Fatalf("expected 1 conditional block, got %d", len(module.Conditionals))
	}

	cond := module.Conditionals[0]

	if cond.ElseBranch == nil {
		t.Fatal("expected ELSE branch")
	}

	if len(cond.ElseBranch.Module.IncludeDirectives) != 1 {
		t.Fatalf("expected 1 include directive in ELSE branch, got %d", len(cond.ElseBranch.Module.IncludeDirectives))
	}

	include := cond.ElseBranch.Module.IncludeDirectives[0]

	if include.FilePath != "other.inc" {
		t.Fatalf("expected include path 'other.inc', got '%s'", include.FilePath)
	}
}

func TestIncludeDirectiveLocation(t *testing.T) {
	input := `LIBRARY()
INCLUDE(test.inc)
END()
`

	file := ParseYaMakeString(input, "test.yamake")

	module := file.Modules[0]
	include := module.IncludeDirectives[0]

	if include.Location.File != "test.yamake" {
		t.Fatalf("expected file 'test.yamake', got '%s'", include.Location.File)
	}

	if include.Location.Line != 2 {
		t.Fatalf("expected line 2, got %d", include.Location.Line)
	}
}
