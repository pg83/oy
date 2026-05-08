package main

import (
	"encoding/json"
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

func TestReferenceGraphParses(t *testing.T) {
	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &Graph{})

	if validator.reference == nil {
		t.Fatal("expected non-nil reference graph")
	}

	if len(validator.reference.Nodes) == 0 {
		t.Error("expected reference graph to have nodes")
	}

	if validator.reference.Conf == nil {
		t.Error("expected reference graph to have Conf section")
	}

	if validator.reference.Conf.GraphSize == 0 {
		t.Error("expected reference graph configuration to have non-zero graph_size")
	}
}

func TestStringSliceComparison(t *testing.T) {
	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &Graph{})

	refSlice := []string{"a", "b", "c"}
	genSlice := []string{"a", "b", "c"}

	err := validator.compareStringSlices("test_node", "test_field", refSlice, genSlice)

	if err != nil {
		t.Errorf("expected nil error for matching slices, got %v", err)
	}

	genSlice = []string{"a", "b", "d"}
	err = validator.compareStringSlices("test_node", "test_field", refSlice, genSlice)

	if err == nil {
		t.Error("expected error for non-matching slices")
	}
}

func TestStringMapComparison(t *testing.T) {
	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &Graph{})

	refMap := map[string]string{"key1": "value1", "key2": "value2"}
	genMap := map[string]string{"key1": "value1", "key2": "value2"}

	err := validator.compareStringMaps("test_node", "test_field", refMap, genMap)

	if err != nil {
		t.Errorf("expected nil error for matching maps, got %v", err)
	}

	genMap = map[string]string{"key1": "value1", "key2": "different"}
	err = validator.compareStringMaps("test_node", "test_field", refMap, genMap)

	if err == nil {
		t.Error("expected error for non-matching maps")
	}

	delete(genMap, "key2")
	gk2 := map[string]string{"key1": "value1"}
	err = validator.compareStringMaps("test_node", "test_field", refMap, gk2)

	if err == nil {
		t.Error("expected error for maps with different keys")
	}
}

func TestRequirementsComparison(t *testing.T) {
	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, &Graph{})

	refReqs := Requirements{CPU: 1, Network: "restricted", RAM: 32}
	genReqs := Requirements{CPU: 1, Network: "restricted", RAM: 32}

	err := validator.compareRequirements("test_node", refReqs, genReqs)

	if err != nil {
		t.Errorf("expected nil error for matching requirements, got %v", err)
	}

	genReqs.CPU = 2
	err = validator.compareRequirements("test_node", refReqs, genReqs)

	if err == nil {
		t.Error("expected error for non-matching CPU requirement")
	}

	genReqs = Requirements{CPU: 1, Network: "restricted", RAM: 32}
	genReqs.Network = "host"
	err = validator.compareRequirements("test_node", refReqs, genReqs)

	if err == nil {
		t.Error("expected error for non-matching Network requirement")
	}
}

func TestValidateWithEmptyGeneratedGraph(t *testing.T) {
	emptyGraph := &Graph{
		Nodes: []*GraphNode{},
	}

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, emptyGraph)

	err := validator.Validate()

	if err == nil {
		t.Error("expected validation error for empty generated graph")
	}

	expectedMsg := "node count mismatch"

	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error to start with '%s', got '%s'", expectedMsg, err.Error())
	}
}

func BenchmarkValidation3730Nodes(b *testing.B) {
	referenceData := Throw2(os.ReadFile(REFERENCE_GRAPH_PATH))
	var reference ReferenceGraph
	Throw(json.Unmarshal(referenceData, &reference))

	referenceGraph := ReferenceGraphToGraph(&reference)

	validator := NewGraphValidator(REFERENCE_GRAPH_PATH, referenceGraph)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validator.Validate()
		if err != nil {
			b.Fatalf("validation failed: %v", err)
		}
	}
}

func ReferenceGraphToGraph(ref *ReferenceGraph) *Graph {
	graph := &Graph{
		Context: ParseContext{
			Platform: "linux",
		},
		Nodes:  make([]*GraphNode, 0, len(ref.Nodes)),
		Inputs: make([]string, 0),
	}

	for _, node := range ref.Nodes {
		graph.Nodes = append(graph.Nodes, node)
	}

	return graph
}
