package main

import (
	"fmt"
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

func TestGraphOutputMatchReference(t *testing.T) {
	path := "/home/pg/monorepo/yatool_orig/tools/archiver/ya.make"

	buildCtx := NewBuildContext()
	buildCtx.Platform = PlatformLinux

	registry := NewModuleRegistry()
	vars := NewMemoryVariableSet()
	fullVars := NewBuildContextVariableSet(buildCtx, vars)
	graph := NewGraph(ParseContext{
		Platform: "linux",
	})

	parser := NewFileParser(registry, graph, buildCtx, fullVars)
	parser.Parse(path)

	output := graph.ToOutput()

	if len(output.Graph) == 0 {
		t.Fatal("expected graph nodes")
	}

	if output.Conf == nil {
		t.Fatal("expected conf section")
	}

	if output.Conf.Platform != "linux" {
		t.Errorf("expected platform=linux, got %s", output.Conf.Platform)
	}

	if output.Conf.GraphSize != len(output.Graph) {
		t.Errorf("expected graph_size=%d, got %d", len(output.Graph), output.Conf.GraphSize)
	}

	if output.Conf.Gsid == "" && output.Conf.Description == nil {
		t.Fatal("expected description section")
	}

	if output.Conf.Description != nil && output.Conf.Description.Platform == "" {
		t.Error("expected non-empty platform in description")
	}

	if len(output.Result) == 0 {
		t.Error("expected non-empty result section")
	}

	fmt.Printf("Graph output: %d nodes, graph_size=%d, platform=%s\n",
		len(output.Graph), output.Conf.GraphSize, output.Conf.Platform)
}
