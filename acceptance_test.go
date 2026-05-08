package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseToolsArchiver(t *testing.T) {
	path := "/home/pg/monorepo/yatool_orig/tools/archiver/ya.make"
	file := ParseYaMakeFile(path)

	if file == nil {
		t.Fatal("expected non-nil file")
	}

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}

	module := file.Modules[0]

	if module.Type != ModuleTypeProgram {
		t.Errorf("expected ModuleTypeProgram, got %v", module.Type)
	}

	expectedDeps := []string{
		"library/cpp/archive",
		"library/cpp/digest/md5",
		"library/cpp/getopt/small",
	}

	if len(module.Dependencies) != len(expectedDeps) {
		t.Fatalf("expected %d dependencies, got %d", len(expectedDeps), len(module.Dependencies))
	}

	for i, dep := range module.Dependencies {
		if dep != expectedDeps[i] {
			t.Errorf("expected dependency '%s', got '%s'", expectedDeps[i], dep)
		}
	}

	expectedSources := []string{"main.cpp"}

	if len(module.Sources) != len(expectedSources) {
		t.Fatalf("expected %d sources, got %d", len(expectedSources), len(module.Sources))
	}

	for i, src := range module.Sources {
		if src != expectedSources[i] {
			t.Errorf("expected source '%s', got '%s'", expectedSources[i], src)
		}
	}

	value, ok := module.GetProperty("IDE_FOLDER")
	if !ok || value != "_Builders" {
		t.Errorf("expected IDE_FOLDER='_Builders', got '%s', ok=%v", value, ok)
	}
}

func TestGraphEqualityValidation(t *testing.T) {
	if _, err := os.Stat(REFERENCE_GRAPH_PATH); os.IsNotExist(err) {
		t.Skip("reference graph not found:", REFERENCE_GRAPH_PATH)
	}

	referenceData := Throw2(os.ReadFile(REFERENCE_GRAPH_PATH))

	var referenceGraph Graph
	Throw(json.Unmarshal(referenceData, &referenceGraph))

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &referenceGraph)

	if err := validator.Validate(); err != nil {
		t.Errorf("graph equality validation failed: %v", err)
	}
}
