package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestNewGraphValidator(t *testing.T) {
	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &Graph{})

	if validator == nil {
		t.Fatal("expected non-nil validator")
	}

	if validator.reference == nil {
		t.Error("expected non-nil reference graph")
	}

	if len(validator.reference.Nodes) != EXPECTED_NODE_COUNT {
		t.Errorf("expected reference to have %d nodes, got %d", EXPECTED_NODE_COUNT, len(validator.reference.Nodes))
	}

	if validator.generated == nil {
		t.Error("expected non-nil generated graph")
	}

	if len(validator.uidMap) != 0 {
		t.Errorf("expected uidMap to be empty during init, got %d entries", len(validator.uidMap))
	}
}

func TestNodeCountMismatch(t *testing.T) {
	smallGraph := &Graph{
		Nodes: []*GraphNode{
			{TargetProperties: TargetProperties{ModuleDir: "test/module"}},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, smallGraph)

	err := validator.Validate()

	if err == nil {
		t.Error("expected validation error for node count mismatch")
	}

	expectedMsg := "node count mismatch"

	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error to start with '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestUIDNormalization(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{TargetProperties: TargetProperties{ModuleDir: "library/cpp/archive"}},
			{TargetProperties: TargetProperties{ModuleDir: "library/cpp/digest/md5"}},
			{TargetProperties: TargetProperties{ModuleDir: "tools/archiver"}},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	id1 := validator.getSequentialIDForModule("library/cpp/archive")
	id2 := validator.getSequentialIDForModule("library/cpp/digest/md5")
	id3 := validator.getSequentialIDForModule("tools/archiver")

	if id1 == id2 {
		t.Error("expected different IDs for different modules")
	}

	if id2 == id3 {
		t.Error("expected different IDs for different modules")
	}

	if id1 == id3 {
		t.Error("expected different IDs for different modules")
	}

	if id1 != validator.getSequentialIDForModule("library/cpp/archive") {
		t.Error("expected consistent IDs for same module")
	}
}

func TestPerfectMatch(t *testing.T) {
	graph := createSimpleTestGraph("test/module")

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	err := validator.Validate()

	if err == nil {
		t.Error("expected validation error for graph not matching reference")
	}
}

func TestStringSliceComparison(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{
				TargetProperties: TargetProperties{ModuleDir: "test"},
				Inputs:           []string{"a.cpp", "b.cpp", "c.cpp"},
				Outputs:          []string{"test.bin"},
			},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	err := validator.compareStringSlices("NODE_0000", "inputs", []string{"a.cpp", "b.cpp", "c.cpp"}, []string{"a.cpp", "b.cpp", "c.cpp"})

	if err != nil {
		t.Errorf("expected no error for matching slices, got: %v", err)
	}

	err = validator.compareStringSlices("NODE_0000", "inputs", []string{"a.cpp", "b.cpp"}, []string{"a.cpp", "b.cpp", "c.cpp"})

	if err == nil {
		t.Error("expected error for slice length mismatch")
	}

	err = validator.compareStringSlices("NODE_0000", "inputs", []string{"a.cpp", "b.cpp", "c.cpp"}, []string{"a.cpp", "b.cpp", "d.cpp"})

	if err == nil {
		t.Error("expected error for slice content mismatch")
	}
}

func TestStringMapComparison(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{
				TargetProperties: TargetProperties{ModuleDir: "test"},
			},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	refEnv := map[string]string{"CC": "clang", "CXX": "clang++"}
	genEnv := map[string]string{"CC": "clang", "CXX": "clang++"}

	err := validator.compareStringMaps("NODE_0000", "env", refEnv, genEnv)

	if err != nil {
		t.Errorf("expected no error for matching maps, got: %v", err)
	}

	genEnv["CXX"] = "g++"

	err = validator.compareStringMaps("NODE_0000", "env", refEnv, genEnv)

	if err == nil {
		t.Error("expected error for map value mismatch")
	}

	delete(genEnv, "CXX")

	err = validator.compareStringMaps("NODE_0000", "env", refEnv, genEnv)

	if err == nil {
		t.Error("expected error for map key mismatch")
	}
}

func TestCommandComparison(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{
				TargetProperties: TargetProperties{ModuleDir: "test"},
			},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	refCmds := []Command{
		{CmdArgs: []string{"clang++", "-o", "test.o", "-c", "test.cpp"}},
		{CmdArgs: []string{"ld", "-o", "test", "test.o"}},
	}

	genCmds := []Command{
		{CmdArgs: []string{"clang++", "-o", "test.o", "-c", "test.cpp"}},
		{CmdArgs: []string{"ld", "-o", "test", "test.o"}},
	}

	err := validator.compareCommands("NODE_0000", refCmds, genCmds)

	if err != nil {
		t.Errorf("expected no error for matching commands, got: %v", err)
	}

	genCmds[0].CmdArgs[3] = "test.cxx"

	err = validator.compareCommands("NODE_0000", refCmds, genCmds)

	if err == nil {
		t.Error("expected error for command mismatch")
	}
}

func TestRequirementsComparison(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{
				TargetProperties: TargetProperties{ModuleDir: "test"},
			},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	refReqs := Requirements{CPU: 2, Network: "restricted", RAM: 64}
	genReqs := Requirements{CPU: 2, Network: "restricted", RAM: 64}

	err := validator.compareRequirements("NODE_0000", refReqs, genReqs)

	if err != nil {
		t.Errorf("expected no error for matching requirements, got: %v", err)
	}

	genReqs.CPU = 4

	err = validator.compareRequirements("NODE_0000", refReqs, genReqs)

	if err == nil {
		t.Error("expected error for CPU mismatch")
	}

	genReqs.CPU = 2
	genReqs.Network = "unrestricted"

	err = validator.compareRequirements("NODE_0000", refReqs, genReqs)

	if err == nil {
		t.Error("expected error for network mismatch")
	}
}

func TestForeignDepsComparison(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{
				TargetProperties: TargetProperties{ModuleDir: "test"},
			},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	refFD := ForeignDeps{
		"LINUX":   []string{"libfoo.so"},
		"DARWIN":  []string{"libfoo.dylib"},
		"WIN32":   []string{"foo.lib"},
	}

	genFD := ForeignDeps{
		"LINUX":   []string{"libfoo.so"},
		"DARWIN":  []string{"libfoo.dylib"},
		"WIN32":   []string{"foo.lib"},
	}

	err := validator.compareForeignDeps("NODE_0000", refFD, genFD)

	if err != nil {
		t.Errorf("expected no error for matching foreign deps, got: %v", err)
	}

	genFD["LINUX"] = []string{"bar.so"}

	err = validator.compareForeignDeps("NODE_0000", refFD, genFD)

	if err == nil {
		t.Error("expected error for foreign deps mismatch")
	}
}

func TestDuplicateModules(t *testing.T) {
	graph := &Graph{
		Nodes: []*GraphNode{
			{
				UID:              "uid1",
				TargetProperties: TargetProperties{ModuleDir: "library/cpp/archive", ModuleLang: "cpp", ModuleType: "bin"},
			},
			{
				UID:              "uid2",
				TargetProperties: TargetProperties{ModuleDir: "library/cpp/digest/md5", ModuleLang: "cpp", ModuleType: "lib"},
			},
		},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, graph)

	normalized := validator.normalizeGraph(graph)

	if len(normalized) != 2 {
		t.Errorf("expected 2 normalized nodes, got %d", len(normalized))
	}

	if normalized["NODE_0031"] == nil {
		t.Error("expected to find NODE_0031")
	}

	if normalized["NODE_0033"] == nil {
		t.Error("expected to find NODE_0033")
	}
}

func getKeys(m map[string]*NormalizedNode) []string {
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func BenchmarkValidation3730Nodes(b *testing.B) {
	referenceData := Throw2(os.ReadFile(REFERENCE_GRAPH_PATH))

	var reference Graph

	Throw(json.Unmarshal(referenceData, &reference))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &reference)

		err := validator.Validate()

		if err != nil {
			b.Fatalf("unexpected validation error: %v", err)
		}
	}
}

func BenchmarkUIDNormalization(b *testing.B) {
	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &Graph{})
	graph := createLargeTestGraph(1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = validator.normalizeGraph(graph)
	}
}

func BenchmarkCommandComparison(b *testing.B) {
	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &Graph{})

	cmds := []Command{
		{CmdArgs: []string{"clang++", "-o", "test.o", "-c", "test.cpp", "-I", "/usr/include"}},
		{CmdArgs: []string{"ld", "-o", "test", "test.o", "-L", "/usr/lib"}},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := validator.compareCommands("NODE_0000", cmds, cmds)

		if err != nil {
			b.Fatalf("unexpected comparison error: %v", err)
		}
	}
}

func createSimpleTestGraph(moduleDir string) *Graph {
	node := &GraphNode{
		UID:      "test-uid",
		Inputs:   []string{moduleDir + "/test.cpp"},
		Outputs:  []string{moduleDir + "/test.bin"},
		Cmds:     []Command{{CmdArgs: []string{"g++", "-o", "test.bin", "test.cpp"}}},
		Env:      map[string]string{"PATH": "/usr/bin"},
		KV:       map[string]string{"key": "value"},
		Platform: "linux",
		Requirements: Requirements{
			CPU:     1,
			Network: "restricted",
			RAM:     32,
		},
		Sandboxing:   true,
		Tags:         []string{"test"},
		ForeignDeps:  ForeignDeps{"LINUX": []string{"libtest.so"}},
		HostPlatform: false,
		TargetProperties: TargetProperties{
			ModuleDir:  moduleDir,
			ModuleLang: "cpp",
			ModuleType: "bin",
		},
	}

	return &Graph{
		Nodes: []*GraphNode{node},
	}
}

func createLargeTestGraph(nodeCount int) *Graph {
	nodes := make([]*GraphNode, nodeCount)

	for i := 0; i < nodeCount; i++ {
		moduleDir := fmt.Sprintf("library/module%04d", i)

		nodes[i] = &GraphNode{
			UID:      fmt.Sprintf("uid-%04d", i),
			Inputs:   []string{moduleDir + "/src.cpp"},
			Outputs:  []string{moduleDir + "/lib.so"},
			Cmds:     []Command{{CmdArgs: []string{"clang++", "-shared", "-o", "lib.so", "src.cpp"}}},
			Platform: "linux",
			TargetProperties: TargetProperties{
				ModuleDir:  moduleDir,
				ModuleLang: "cpp",
				ModuleType: "lib",
			},
		}
	}

	return &Graph{
		Nodes: nodes,
	}
}