package main

import (
	"fmt"
	"os"
	"testing"
)

const (
	REFERENCE_NODE_COUNT_CC     = 3571
	REFERENCE_NODE_COUNT_AS     = 83
	REFERENCE_NODE_COUNT_AR     = 48
	REFERENCE_NODE_COUNT_JS     = 23
	REFERENCE_NODE_COUNT_LD     = 3
	REFERENCE_NODE_COUNT_R6     = 1
	REFERENCE_NODE_COUNT_CP     = 1
	REFERENCE_NODE_COUNT_TOTAL  = 3730
	PLATFORM_NODE_COUNT_AARCH64 = 1933
	PLATFORM_NODE_COUNT_X86_64  = 1797
)

func TestFinalGraphValidation(t *testing.T) {
	if _, err := os.Stat(REFERENCE_GRAPH_PATH); os.IsNotExist(err) {
		t.Skipf("Reference graph not found at %s, skipping final validation", REFERENCE_GRAPH_PATH)
	}

	os.Getenv("OY_STRICT_GRAPH_VALIDATION")
	if os.Getenv("OY_STRICT_GRAPH_VALIDATION") != "1" {
		t.Skip("Skipping strict graph validation - set OY_STRICT_GRAPH_VALIDATION=1 to enable")
	}

	ctx := ParseContext{
		Platform:   "linux",
		Musl:       true,
		Language:   "",
		TargetPath: REFERENCE_TARGET,
		BuildFlags: make(map[string]string),
	}

	graph := Throw2(BuildDependencyGraph(ctx.TargetPath, ctx, SourceRoot, nil))

	if len(graph.Nodes) != REFERENCE_NODE_COUNT_TOTAL {
		t.Errorf("Node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_TOTAL, len(graph.Nodes), len(graph.Nodes)-REFERENCE_NODE_COUNT_TOTAL)
	}

	validationGraph := graphToValidationGraph(graph)
	typeDistribution := ExtractNodeTypeDistribution(validationGraph.Nodes)

	if ccCount := typeDistribution["CC"]; ccCount != REFERENCE_NODE_COUNT_CC {
		t.Errorf("CC node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_CC, ccCount, ccCount-REFERENCE_NODE_COUNT_CC)
	}

	if asCount := typeDistribution["AS"]; asCount != REFERENCE_NODE_COUNT_AS {
		t.Errorf("AS node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_AS, asCount, asCount-REFERENCE_NODE_COUNT_AS)
	}

	if arCount := typeDistribution["AR"]; arCount != REFERENCE_NODE_COUNT_AR {
		t.Errorf("AR node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_AR, arCount, arCount-REFERENCE_NODE_COUNT_AR)
	}

	if jsCount := typeDistribution["JS"]; jsCount != REFERENCE_NODE_COUNT_JS {
		t.Errorf("JS node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_JS, jsCount, jsCount-REFERENCE_NODE_COUNT_JS)
	}

	if ldCount := typeDistribution["LD"]; ldCount != REFERENCE_NODE_COUNT_LD {
		t.Errorf("LD node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_LD, ldCount, ldCount-REFERENCE_NODE_COUNT_LD)
	}

	if r6Count := typeDistribution["R6"]; r6Count != REFERENCE_NODE_COUNT_R6 {
		t.Errorf("R6 node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_R6, r6Count, r6Count-REFERENCE_NODE_COUNT_R6)
	}

	if cpCount := typeDistribution["CP"]; cpCount != REFERENCE_NODE_COUNT_CP {
		t.Errorf("CP node count mismatch: expected %d, got %d (delta: %+d)",
			REFERENCE_NODE_COUNT_CP, cpCount, cpCount-REFERENCE_NODE_COUNT_CP)
	}

	platformDistribution := ExtractPlatformDistribution(validationGraph.Nodes)

	if aarch64Count := platformDistribution["default-linux-aarch64"]; aarch64Count != PLATFORM_NODE_COUNT_AARCH64 {
		t.Errorf("aarch64 platform node count mismatch: expected %d, got %d (delta: %+d)",
			PLATFORM_NODE_COUNT_AARCH64, aarch64Count, aarch64Count-PLATFORM_NODE_COUNT_AARCH64)
	}

	if x86Count := platformDistribution["default-linux-x86_64"]; x86Count != PLATFORM_NODE_COUNT_X86_64 {
		t.Errorf("x86_64 platform node count mismatch: expected %d, got %d (delta: %+d)",
			PLATFORM_NODE_COUNT_X86_64, x86Count, x86Count-PLATFORM_NODE_COUNT_X86_64)
	}

	referenceGraph := loadValidationGraph(REFERENCE_GRAPH_PATH)
	comparison := CompareGraphs(referenceGraph, validationGraph, GraphComparisonOptions{MaxMismatches: 0})

	if comparison.TotalMismatches > 0 {
		t.Errorf("Graph structural comparison failed with %d mismatches:\n%s",
			comparison.TotalMismatches, comparison.Err())
	}

	fmt.Println("\n=== Final Graph Validation Results ===")
	fmt.Printf("Total nodes: %d (expected %d)\n", len(graph.Nodes), REFERENCE_NODE_COUNT_TOTAL)
	fmt.Println("\n=== Node Type Distribution ===")
	for nodeType, count := range typeDistribution {
		fmt.Printf("  %s: %d\n", nodeType, count)
	}
	fmt.Println("\n=== Platform Distribution ===")
	for platform, count := range platformDistribution {
		fmt.Printf("  %s: %d\n", platform, count)
	}
	if comparison.TotalMismatches == 0 {
		fmt.Println("\n=== Graph Structural Equality: PASS ===")
	}
}

func TestFinalGraphValidationWithReport(t *testing.T) {
	if _, err := os.Stat(REFERENCE_GRAPH_PATH); os.IsNotExist(err) {
		t.Skipf("Reference graph not found at %s, skipping final validation", REFERENCE_GRAPH_PATH)
	}

	ctx := ParseContext{
		Platform:   "linux",
		Musl:       true,
		Language:   "",
		TargetPath: REFERENCE_TARGET,
		BuildFlags: make(map[string]string),
	}

	graph := Throw2(BuildDependencyGraph(ctx.TargetPath, ctx, SourceRoot, nil))
	validationGraph := graphToValidationGraph(graph)

	typeDistribution := ExtractNodeTypeDistribution(validationGraph.Nodes)
	platformDistribution := ExtractPlatformDistribution(validationGraph.Nodes)

	fmt.Println("\n=== Current Graph State ===")
	fmt.Printf("Total nodes: %d (target: %d, delta: %+d)\n",
		len(graph.Nodes), REFERENCE_NODE_COUNT_TOTAL, len(graph.Nodes)-REFERENCE_NODE_COUNT_TOTAL)

	fmt.Println("\nCurrent vs Reference Type Distribution:")
	types := []string{"CC", "AS", "AR", "JS", "LD", "R6", "CP"}
	for _, nodeType := range types {
		refCount := map[string]int{
			"CC": REFERENCE_NODE_COUNT_CC,
			"AS": REFERENCE_NODE_COUNT_AS,
			"AR": REFERENCE_NODE_COUNT_AR,
			"JS": REFERENCE_NODE_COUNT_JS,
			"LD": REFERENCE_NODE_COUNT_LD,
			"R6": REFERENCE_NODE_COUNT_R6,
			"CP": REFERENCE_NODE_COUNT_CP,
		}[nodeType]
		actCount := typeDistribution[nodeType]
		delta := actCount - refCount
		status := "✓"
		if delta != 0 {
			status = "✗"
		}
		fmt.Printf("  %s %s: current=%d reference=%d delta=%+d\n", status, nodeType, actCount, refCount, delta)
	}

	fmt.Println("\nCurrent vs Reference Platform Distribution:")
	for platform, actCount := range platformDistribution {
		if platform == "unknown" || platform == "both" {
			continue
		}
		refCount := map[string]int{
			"default-linux-aarch64": PLATFORM_NODE_COUNT_AARCH64,
			"default-linux-x86_64":  PLATFORM_NODE_COUNT_X86_64,
		}[platform]
		if refCount == 0 {
			continue
		}
		delta := actCount - refCount
		status := "✓"
		if delta != 0 {
			status = "✗"
		}
		fmt.Printf("  %s %s: current=%d reference=%d delta=%+d\n", status, platform, actCount, refCount, delta)
	}
}
