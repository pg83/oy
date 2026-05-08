package main

import (
	"testing"
)

func TestParseUtilYaMake(t *testing.T) {
	path := "/home/pg/monorepo/yatool_orig/util/ya.make"
	file := ParseYaMakeFile(path)

	if file == nil {
		t.Fatal("expected non-nil file")
	}

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	if module.Type != ModuleTypeLibrary {
		t.Errorf("expected ModuleTypeLibrary, got %v", module.Type)
	}

	if len(module.Conditionals) == 0 {
		t.Error("expected at least some conditionals in util/ya.make")
	}

	if len(file.Imports) == 0 {
		t.Error("expected at least some recurse directives in util/ya.make")
	}

	hasWindowsIF := false
	for _, cond := range module.Conditionals {
		if cond.IfBranch.Condition == "OS_WINDOWS" {
			hasWindowsIF = true
			break
		}
	}

	if !hasWindowsIF {
		t.Error("expected to find IF (OS_WINDOWS) condition in util/ya.make")
	}
}