package main

import (
	"strings"
	"testing"
)

func TestMUSLBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	sourceRoot := "/home/pg/monorepo/yatool_orig"

	ctx := ParseContext{
		Platform:   "linux",
		Musl:       true,
		Language:   "",
		TargetPath: "contrib/libs/musl",
		BuildFlags: make(map[string]string),
	}

	graph, err := BuildDependencyGraph("contrib/libs/musl", ctx, sourceRoot, nil)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}

	nodeCount := len(graph.Nodes)
	t.Logf("Generated %d graph nodes", nodeCount)

	expectedNodes := 3544

	actualDiff := nodeCount - expectedNodes
	if actualDiff != 0 {
		t.Logf("Node count discrepancy: got %d, expected %d (diff: %d, %.2f%%)", nodeCount, expectedNodes, actualDiff, float64(actualDiff)/float64(expectedNodes)*100)
	}

	muslNodeCount := 0
	for _, node := range graph.Nodes {
		if strings.Contains(node.TargetProperties.ModuleDir, "musl") &&
			!strings.Contains(node.TargetProperties.ModuleDir, "contrib/libs/musl/") {
			muslNodeCount++
		}
	}

	t.Logf("MUSL-specific nodes: %d of %d total (%.1f%%)", muslNodeCount, nodeCount, float64(muslNodeCount)/float64(nodeCount)*100)

	moduleDirs := make(map[string]bool)
	for _, node := range graph.Nodes {
		if node.TargetProperties.ModuleDir != "" {
			moduleDirs[node.TargetProperties.ModuleDir] = true
		}
	}

	t.Logf("Total unique module directories: %d", len(moduleDirs))
	for _, node := range graph.Nodes {
		if node.TargetProperties.ModuleDir != "" {
			t.Logf("Module dir: %s", node.TargetProperties.ModuleDir)
			break
		}
	}
}
