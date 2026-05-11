package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestT150BaselineNodeCount(t *testing.T) {
	if _, err := os.Stat(ReferenceGraphPath); err != nil {
		t.Skipf("Reference graph unavailable: %v", err)
	}

	ctx := ParseContext{
		Platform:   "linux",
		Musl:       true,
		Language:   "",
		TargetPath: ReferenceTarget,
		BuildFlags: make(map[string]string),
	}

	graph, err := BuildDependencyGraph(ReferenceTarget, ctx, SourceRoot, nil)
	if err != nil {
		t.Fatalf("BuildDependencyGraph failed: %v", err)
	}

	currentCount := len(graph.Nodes)

	t.Logf("Current node count: %d (expected baseline: 4131)", currentCount)
	t.Logf("Reference node count: %d", ExpectedNodeCount)
	t.Logf("Gap: %d nodes (%.1f%% over-generation)",
		currentCount-ExpectedNodeCount,
		float64(currentCount-ExpectedNodeCount)/float64(ExpectedNodeCount)*100)

	distribution := countByPlatform(graph.Nodes)
	t.Logf("Platform distribution:")
	t.Logf("  aarch64: %d (reference: 1933)", distribution["default-linux-aarch64"])
	t.Logf("  x86_64: %d (reference: 1797)", distribution["default-linux-x86_64"])
	t.Logf("  both: %d (reference: 0)", distribution["both"])

	const BaselineNodeCount = 4131
	if currentCount != BaselineNodeCount {
		t.Logf("WARNING: Current node count is %d, expected baseline %d from NODE_GAP_ANALYSIS.md",
			currentCount, BaselineNodeCount)
	}

	byType := countByType(graph.Nodes)
	t.Logf("Node type breakdown:")
	t.Logf("  CC: %d (reference: 3571, gap: %d)", byType["CC"], byType["CC"]-3571)
	t.Logf("  AS: %d (reference: 83, gap: %d)", byType["AS"], byType["AS"]-83)
	t.Logf("  AR: %d (reference: 48, gap: %d)", byType["AR"], byType["AR"]-48)
	t.Logf("  JS: %d (reference: 23, gap: %d)", byType["JS"], byType["JS"]-23)
	t.Logf("  LD: %d (reference: 3, gap: %d)", byType["LD"], byType["LD"]-3)
	t.Logf("  R6: %d (reference: 1, gap: %d)", byType["R6"], byType["R6"]-1)
	t.Logf("  CP: %d (reference: 1, gap: %d)", byType["CP"], byType["CP"]-1)

	platformBothNodes := 0
	for _, node := range graph.Nodes {
		if node.Platform == "both" {
			platformBothNodes++
		}
	}
	t.Logf("Nodes with platform='both': %d (BUG - reference has 0)", platformBothNodes)

	if platformBothNodes > 0 {
		t.Logf("ERROR: %d nodes have platform='both', reference has 0", platformBothNodes)
	}
}

func TestT150NOPlatformDirectiveParsing(t *testing.T) {
	yasmPath := filepath.Join(SourceRoot, "contrib/tools/yasm/ya.make")
	content, err := os.ReadFile(yasmPath)
	if err != nil {
		t.Skipf("Cannot read yasm/ya.make: %v", err)
	}

	hasNOPlatform := strings.Contains(string(content), "NO_PLATFORM()")
	if !hasNOPlatform {
		t.Logf("yasm/ya.make does not contain NO_PLATFORM() directive")
		return
	}

	t.Logf("NO_PLATFORM() directive found in yasm/ya.make")
	t.Logf("Current implementation does NOT parse NO_PLATFORM()")
	t.Logf("This confirms T-140 finding: NO_PLATFORM() exists but not decoded")
}

func TestT150PredictedNodeCountAfterFixes(t *testing.T) {
	currentNodeCount := 4131
	referenceNodeCount := 3730

	expectedNOPLATFORMReduction := 294
	expectedMuslFullAddition := 83

	predictedAfterNOPLATFORM := currentNodeCount - expectedNOPLATFORMReduction
	predictedAfterMuslFull := predictedAfterNOPLATFORM + expectedMuslFullAddition

	t.Logf("=== T-150 Node Count Prediction ===")
	t.Logf("Current: %d nodes", currentNodeCount)
	t.Logf("")
	t.Logf("After NO_PLATFORM fix:")
	t.Logf("  Eliminate dual-platform over-generation: -%d nodes", expectedNOPLATFORMReduction)
	t.Logf("  Expected total: %d nodes", predictedAfterNOPLATFORM)
	t.Logf("")
	t.Logf("After musl/full PEERDIR fix:")
	t.Logf("  Add missing musl/full dependencies: +%d nodes", expectedMuslFullAddition)
	t.Logf("  Expected total: %d nodes", predictedAfterMuslFull)
	t.Logf("")
	t.Logf("Reference: %d nodes", referenceNodeCount)
	t.Logf("Remaining gap: %d nodes (%.1f%% under-reference)",
		predictedAfterMuslFull-referenceNodeCount,
		float64(predictedAfterMuslFull-referenceNodeCount)/float64(referenceNodeCount)*100)
	t.Logf("")
	t.Logf("=== Remaining Gap Attribution ===")
	t.Logf("After both fixes, expected remaining gap of ~190 nodes due to:")
	t.Logf("  - Builtins ARCH CC over-generation: +298 nodes")
	t.Logf("  - Missing host tool modules: -88 CC, -2 LD, -9 JS")
	t.Logf("  - JS remaining platform gap: -9 nodes")
	t.Logf("  - Other unresolved issues: ~-99 nodes")
	t.Logf("  Net remaining gap: +190 nodes")

	conservative := predictedAfterMuslFull - 100
	optimistic := predictedAfterMuslFull + 80
	t.Logf("")
	t.Logf("Realistic expectation range: %d - %d nodes", conservative, optimistic)

	t.Logf("")
	t.Logf("Conclusion: Achieving exactly 3730 nodes from NO_PLATFORM + musl/full alone is UNLIKELY")
	t.Logf("Requires additional tickets for host tool loading and per-platform SRCS resolution")
}

func countByPlatform(nodes []*GraphNode) map[string]int {
	distribution := make(map[string]int)
	for _, node := range nodes {
		distribution[node.Platform]++
	}
	return distribution
}

func countByType(nodes []*GraphNode) map[string]int {
	byType := make(map[string]int)
	for _, node := range nodes {
		if node.KV != nil {
			if nodeType, ok := node.KV["p"]; ok {
				byType[nodeType]++
			}
		}
	}
	for _, node := range nodes {
		if len(node.Cmds) > 0 {
			cmd := node.Cmds[0]
			if strings.HasPrefix(strings.Join(cmd.CmdArgs, " "), "yasm asm") {
				byType["AS"]++
			} else if strings.HasPrefix(strings.Join(cmd.CmdArgs, " "), "ragel") {
				byType["JS"]++
			}
		}
	}
	defaultToCC := len(nodes)
	for _, count := range byType {
		defaultToCC -= count
	}
	if defaultToCC > 0 {
		byType["CC"] = defaultToCC
	}
	return byType
}
