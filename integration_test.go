package main

import (
	"testing"
	"time"
)

const (
	ReferenceGraphPath = "/home/pg/monorepo/yatool_orig/sg.json"
	ReferenceTarget    = "tools/archiver"
	SourceRoot         = "/home/pg/monorepo/yatool_orig"
	ExpectedNodeCount  = 3730
)

func TestFullGraphGenerationToolsArchiver(t *testing.T) {
	ctx := ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: ReferenceTarget,
		BuildFlags: make(map[string]string),
	}

	generatorStart := time.Now()
	graph, err := BuildDependencyGraph(ReferenceTarget, ctx, SourceRoot)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}
	generationTime := time.Since(generatorStart)

	t.Logf("Graph generation took: %v", generationTime)
	t.Logf("Generated graph has %d nodes", len(graph.Nodes))
	t.Logf("Generated graph has %d inputs", len(graph.Inputs))

	if len(graph.Nodes) == 0 {
		t.Fatal("Generated graph has zero nodes - this indicates parsing or traversal issue")
	}

	validatorStart := time.Now()
	validator := NewGraphValidator(ReferenceGraphPath, graph)
	validationErr := validator.Validate()
	validationTime := time.Since(validatorStart)

	if validationErr != nil {
		t.Fatalf("Graph validation failed: %v (validation took: %v)", validationErr, validationTime)
	}

	t.Logf("Validation passed: %d nodes match", len(validator.reference.Nodes))

	if generationTime >= time.Second {
		t.Logf("WARNING: Graph generation took %v (target: <1s)", generationTime)
	}
}

func BenchmarkFullGraphGenerationToolsArchiver(b *testing.B) {
	ctx := ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: ReferenceTarget,
		BuildFlags: make(map[string]string),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		graph, err := BuildDependencyGraph(ReferenceTarget, ctx, SourceRoot)
		if err != nil {
			b.Fatalf("BuildDependencyGraph failed: %v", err)
		}

		if len(graph.Nodes) == 0 {
			b.Fatal("Generated graph has zero nodes")
		}
	}
}

func TestGenerateAndValidateGraphInOneStep(t *testing.T) {
	ctx := ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: ReferenceTarget,
		BuildFlags: make(map[string]string),
	}

	graph, err := BuildDependencyGraph(ReferenceTarget, ctx, SourceRoot)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}

	validator := NewGraphValidator(ReferenceGraphPath, graph);

	nodeCountDiff := len(validator.reference.Nodes) - len(graph.Nodes)
	if nodeCountDiff != 0 {
		t.Errorf("Node count mismatch: reference has %d, generated has %d (diff: %d)",
			len(validator.reference.Nodes), len(graph.Nodes), nodeCountDiff)
	}

	if err := validator.Validate(); err != nil {
		t.Errorf("Graph validation failed: %v", err)
	}
}

func TestGraphGenerationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := ParseContext{
		Platform:   "linux",
		Musl:       false,
		Language:   "",
		TargetPath: ReferenceTarget,
		BuildFlags: make(map[string]string),
	}

	start := time.Now()
	graph, err := BuildDependencyGraph(ReferenceTarget, ctx, SourceRoot)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}
	duration := time.Since(start)

	t.Logf("Graph generation completed in %v", duration)
	t.Logf("Generated graph has %d nodes", len(graph.Nodes))

	if len(graph.Nodes) == 0 {
		t.Fatal("Generated graph has zero nodes")
	}

	if duration >= time.Second {
		t.Errorf("Graph generation took %v, which is slower than the target of <1s", duration)
	}

	threshold := 500 * time.Millisecond
	if duration > threshold {
		t.Logf("WARNING: Graph generation took %v, which is above the recommended threshold of %v", duration, threshold)
	}
}
