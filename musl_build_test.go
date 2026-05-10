package main

import (
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

	t.Logf("Generated %d graph nodes", len(graph.Nodes))

	for _, node := range graph.Nodes {
		if node.TargetProperties.ModuleDir != "" {
			t.Logf("Module dir: %s", node.TargetProperties.ModuleDir)
		}
	}
}
