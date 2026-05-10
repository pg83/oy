package main

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestMUSLStandaloneVerification(t *testing.T) {
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

	t.Run("INCLUDE_processing", func(t *testing.T) {
		yaMakePath := "/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make"
		content := Throw2(os.ReadFile(yaMakePath))

		includePattern := regexp.MustCompile(`INCLUDE\s*\(\s*([^\)]+)\s*\)`)
		matches := includePattern.FindAllStringSubmatch(string(content), -1)

		t.Logf("Found %d INCLUDE directive(s) in ya.make", len(matches))
		for _, m := range matches {
			t.Logf("  INCLUDE(%s)", m[1])
		}

		if len(matches) == 0 {
			t.Error("Expected at least one INCLUDE directive in ya.make")
		}

		incPath := "/home/pg/monorepo/yatool_orig/contrib/libs/musl/ya.make.inc"
		if _, err := os.Stat(incPath); os.IsNotExist(err) {
			t.Errorf("ya.make.inc does not exist at expected path: %s", incPath)
		} else {
			incContent := Throw2(os.ReadFile(incPath))
			archPattern := regexp.MustCompile(`IF\s*\(\s*ARCH_X86_64\s*\)`)
			archMatches := archPattern.FindAllString(string(incContent), -1)

			t.Logf("Found %d ARCH_X86_64 conditional blocks in ya.make.inc", len(archMatches))

			sourcePattern := regexp.MustCompile(`^src/`)
			sourceLines := sourcePattern.FindAllString(string(incContent), -1)
			t.Logf("Found %d source file entries in ya.make.inc (src/ at start of line)", len(sourceLines))

			sourcePattern2 := regexp.MustCompile(`\bsrc/`)
			sourceLines2 := sourcePattern2.FindAllString(string(incContent), -1)
			t.Logf("Found %d source file entries in ya.make.inc (src/ anywhere)", len(sourceLines2))
		}
	})

	t.Run("graph_generation", func(t *testing.T) {
		graph, err := BuildDependencyGraph("contrib/libs/musl", ctx, sourceRoot, nil)
		if err != nil {
			t.Fatalf("BuildDependencyGraph failed: %v", err)
		}

		nodeCount := len(graph.Nodes)
		t.Logf("Generated %d graph nodes for MUSL standalone build", nodeCount)

		expectedNodes := 3544
		if nodeCount != expectedNodes {
			t.Logf("Node count discrepancy: got %d, expected %d (diff: %d)", nodeCount, expectedNodes, nodeCount-expectedNodes)
		}

		nodesByModule := make(map[string]int)
		nodesByKV := make(map[string]int)

		for _, node := range graph.Nodes {
			for k := range node.KV {
				nodesByKV[k]++
			}

			if node.TargetProperties.ModuleDir != "" {
				nodesByModule[node.TargetProperties.ModuleDir]++
			}
		}

		t.Log("KV key frequencies:")
		for _, kv := range sortedKeys(nodesByKV) {
			t.Logf("  %s: %d", kv, nodesByKV[kv])
		}

		t.Log("Node distribution by module (all modules):")
		sortedModules := sortedKeys(nodesByModule)
		topN := 10
		displayModules := sortedModules
		if len(sortedModules) > topN {
			displayModules = sortedModules[:topN]
		}
		for _, mod := range displayModules {
			t.Logf("  %s: %d", mod, nodesByModule[mod])
		}

		if len(sortedModules) > topN {
			topNSum := sumNodes(sortedModules[:topN], nodesByModule)
			remainingSum := sumNodes(sortedModules[topN:], nodesByModule)
			totalSum := sumNodes(sortedModules, nodesByModule)
			t.Logf("Showing %d of %d modules", topN, len(sortedModules))
			t.Logf("Top %d modules: %d nodes (excluding last module: %s with %d nodes)",
				topN, topNSum, sortedModules[topN], remainingSum)
			if totalSum != nodeCount {
				t.Errorf("Module node sum %d != total node count %d", totalSum, nodeCount)
			}
		}
	})

	t.Run("verification_flags", func(t *testing.T) {
		if !ctx.Musl {
			t.Error("Musl flag should be enabled for MUSL build")
		}
		if ctx.Platform != "linux" {
			t.Logf("Warning: platform is %s, expected linux for MUSL build", ctx.Platform)
		}
	})

	t.Run("dependency_verification", func(t *testing.T) {
		graph, err := BuildDependencyGraph("contrib/libs/musl", ctx, sourceRoot, nil)
		if err != nil {
			t.Fatalf("BuildDependencyGraph failed: %v", err)
		}

		moduleDirs := make(map[string]bool)
		for _, node := range graph.Nodes {
			if node.TargetProperties.ModuleDir != "" {
				moduleDirs[node.TargetProperties.ModuleDir] = true
			}
		}

		expectedDeps := []string{
			"contrib/libs/musl",
			"contrib/libs/cxxsupp/builtins",
			"contrib/libs/cxxsupp/libcxx",
			"contrib/libs/cxxsupp/libcxxabi-parts",
			"contrib/libs/cxxsupp/libcxxrt",
			"contrib/libs/libunwind",
			"contrib/libs/double-conversion",
			"contrib/libs/libc_compat",
			"contrib/libs/zlib",
			"util",
			"util/charset",
		}

		for _, dep := range expectedDeps {
			if moduleDirs[dep] {
				t.Logf("Found expected dependency: %s", dep)
			} else {
				t.Errorf("Missing expected dependency: %s", dep)
			}
		}

		t.Logf("Total unique module directories: %d", len(moduleDirs))
	})
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sumNodes(modules []string, counts map[string]int) int {
	total := 0
	for _, mod := range modules {
		total += counts[mod]
	}
	return total
}

func TestMUSLIncludeProcessingDetails(t *testing.T) {
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

	muslNodes := 0
	for _, node := range graph.Nodes {
		if strings.Contains(node.TargetProperties.ModuleDir, "musl") &&
			!strings.Contains(node.TargetProperties.ModuleDir, "contrib/libs/musl/") {
			muslNodes++
		}
	}

	t.Logf("Nodes containing 'musl' in module path: %d", muslNodes)

	totalNodes := len(graph.Nodes)
	nonMuslNodes := totalNodes - muslNodes

	t.Logf("Total nodes: %d", totalNodes)
	t.Logf("Non-MUSL nodes (dependencies): %d", nonMuslNodes)

	if muslNodes > 0 {
		percentage := float64(muslNodes) / float64(totalNodes) * 100
		t.Logf("MUSL-specific nodes: %.1f%% of total", percentage)
	}
}
