package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompareGraphFilesIgnoresUIDRenumbering(t *testing.T) {
	ref := syntheticGraph([]ValidationNode{
		syntheticNode("ref-root", "tools/archiver", []string{"ref-lib"}, []string{"archiver"}),
		syntheticNode("ref-lib", "library/cpp/archive", nil, []string{"libarchive.a"}),
	}, []string{"ref-root"})
	gen := syntheticGraph([]ValidationNode{
		syntheticNode("gen-lib", "library/cpp/archive", nil, []string{"libarchive.a"}),
		syntheticNode("gen-root", "tools/archiver", []string{"gen-lib"}, []string{"archiver"}),
	}, []string{"gen-root"})

	assertGraphsMatch(t, ref, gen)
}

func TestCompareGraphFilesComparesResultAfterUIDMapping(t *testing.T) {
	ref := syntheticGraph([]ValidationNode{
		syntheticNode("ref-root", "tools/archiver", []string{"ref-lib"}, []string{"archiver"}),
		syntheticNode("ref-lib", "library/cpp/archive", nil, []string{"libarchive.a"}),
	}, []string{"ref-root"})
	gen := syntheticGraph([]ValidationNode{
		syntheticNode("gen-root", "tools/archiver", []string{"gen-lib"}, []string{"archiver"}),
		syntheticNode("gen-lib", "library/cpp/archive", nil, []string{"libarchive.a"}),
	}, []string{"gen-root"})

	assertGraphsMatch(t, ref, gen)

	gen.Result = []string{"gen-lib"}
	result := CompareGraphs(ref, gen, GraphComparisonOptions{})
	if result.Err() == nil {
		t.Fatal("expected result mismatch")
	}
	if !strings.Contains(result.Err().Error(), "result") {
		t.Fatalf("expected result mismatch context, got %v", result.Err())
	}
}

func TestCompareGraphFilesDoesNotCollapseSameModuleDir(t *testing.T) {
	ref := syntheticGraph([]ValidationNode{
		syntheticNode("ref-a", "same/module", nil, []string{"a.o"}),
		syntheticNode("ref-b", "same/module", nil, []string{"b.o"}),
	}, []string{"ref-a"})
	gen := syntheticGraph([]ValidationNode{
		syntheticNode("gen-b", "same/module", nil, []string{"b.o"}),
		syntheticNode("gen-a", "same/module", nil, []string{"a.o"}),
	}, []string{"gen-a"})

	assertGraphsMatch(t, ref, gen)

	missing := syntheticGraph([]ValidationNode{
		syntheticNode("gen-a", "same/module", nil, []string{"a.o"}),
	}, []string{"gen-a"})
	result := CompareGraphs(ref, missing, GraphComparisonOptions{})
	if result.Err() == nil {
		t.Fatal("expected missing same-module node mismatch")
	}
}

func TestCompareGraphFilesReportsFirstNMismatches(t *testing.T) {
	ref := syntheticGraph([]ValidationNode{syntheticNode("ref", "m", nil, []string{"out"})}, []string{"ref"})
	gen := syntheticGraph([]ValidationNode{syntheticNode("gen", "m", nil, []string{"out"})}, []string{"gen"})
	gen.Conf.Cache = false
	gen.Conf.GraphSize = 2
	gen.Conf.Platform = "darwin"
	gen.Inputs.Values = []string{"different"}
	gen.Nodes[0].Platform = "darwin"
	gen.Graph = gen.Nodes
	gen.finalize()

	result := CompareGraphs(ref, gen, GraphComparisonOptions{MaxMismatches: 2})
	err := result.Err()
	if err == nil {
		t.Fatal("expected mismatches")
	}
	if len(result.Mismatches) != 2 {
		t.Fatalf("expected 2 shown mismatches, got %d", len(result.Mismatches))
	}
	if result.TotalMismatches <= len(result.Mismatches) {
		t.Fatalf("expected hidden mismatches, got total=%d shown=%d", result.TotalMismatches, len(result.Mismatches))
	}
	if !strings.Contains(err.Error(), "showing first 2") {
		t.Fatalf("expected capped error header, got %v", err)
	}
}

func TestCompareGraphFilesComparesCommandEnvAndCWD(t *testing.T) {
	refNode := syntheticNode("ref", "m", nil, []string{"out"})
	refNode.Cmds = []ValidationCommand{{CmdArgs: []string{"compile", "a.cc"}, Env: map[string]string{"A": "1"}, Cwd: "$(BUILD_ROOT)/m"}}
	genNode := syntheticNode("gen", "m", nil, []string{"out"})
	genNode.Cmds = []ValidationCommand{{CmdArgs: []string{"compile", "a.cc"}, Env: map[string]string{"A": "2"}, Cwd: "$(BUILD_ROOT)/other"}}

	result := CompareGraphs(syntheticGraph([]ValidationNode{refNode}, []string{"ref"}), syntheticGraph([]ValidationNode{genNode}, []string{"gen"}), GraphComparisonOptions{})
	if result.Err() == nil {
		t.Fatal("expected command env/cwd mismatch")
	}
	if !strings.Contains(result.Err().Error(), "cmds[0].env") && !strings.Contains(result.Err().Error(), "cmds[0].cwd") {
		t.Fatalf("expected command env or cwd context, got %v", result.Err())
	}
}

func TestCompareGraphFilesComparesKVEnvRequirementsTagsForeignDepsHostPlatform(t *testing.T) {
	refNode := syntheticNode("ref", "m", nil, []string{"out"})
	refNode.KV = map[string]string{"k": "v"}
	refNode.Env = map[string]string{"ENV": "v"}
	refNode.Requirements = ValidationRequirements{CPU: 2, Network: "restricted", RAM: 64}
	refNode.Tags = []string{"tag"}
	refNode.ForeignDeps = map[string][]string{"linux": []string{"dep"}}
	refNode.HostPlatform = true

	genNode := refNode
	genNode.UID = "gen"
	genNode.SelfUID = "gen-self"
	genNode.StatsUID = "gen-stats"
	genNode.KV = map[string]string{"k": "other"}

	result := CompareGraphs(syntheticGraph([]ValidationNode{refNode}, []string{"ref"}), syntheticGraph([]ValidationNode{genNode}, []string{"gen"}), GraphComparisonOptions{})
	if result.Err() == nil {
		t.Fatal("expected kv mismatch")
	}
	if !strings.Contains(result.Err().Error(), "field=kv") {
		t.Fatalf("expected kv context, got %v", result.Err())
	}
}

func TestCompareGraphFilesComparesAllTargetProperties(t *testing.T) {
	refNode := syntheticNode("ref", "m", nil, []string{"out"})
	refNode.TargetProperties.ModuleTag = "tag-a"
	genNode := syntheticNode("gen", "m", nil, []string{"out"})
	genNode.TargetProperties.ModuleTag = "tag-b"

	result := CompareGraphs(syntheticGraph([]ValidationNode{refNode}, []string{"ref"}), syntheticGraph([]ValidationNode{genNode}, []string{"gen"}), GraphComparisonOptions{})
	if result.Err() == nil {
		t.Fatal("expected target_properties.module_tag mismatch")
	}
	if !strings.Contains(result.Err().Error(), "target_properties") || !strings.Contains(result.Err().Error(), "module_tag") {
		t.Fatalf("expected module_tag context, got %v", result.Err())
	}
}

func TestCompareGraphFilesComparesTopLevelInputs(t *testing.T) {
	ref := syntheticGraph([]ValidationNode{syntheticNode("ref", "m", nil, []string{"out"})}, []string{"ref"})
	gen := syntheticGraph([]ValidationNode{syntheticNode("gen", "m", nil, []string{"out"})}, []string{"gen"})

	refPath := writeRawGraph(t, `{"conf":{"graph_size":1},"graph":[{"uid":"ref","self_uid":"ref-self","stats_uid":"ref-stats","outputs":["out"],"target_properties":{"module_dir":"m"},"requirements":{"cpu":1,"network":"restricted","ram":32},"sandboxing":true}],"inputs":{"a":"a","b":"b"},"result":["ref"]}`)
	genPath := writeRawGraph(t, `{"conf":{"graph_size":1},"graph":[{"uid":"gen","self_uid":"gen-self","stats_uid":"gen-stats","outputs":["out"],"target_properties":{"module_dir":"m"},"requirements":{"cpu":1,"network":"restricted","ram":32},"sandboxing":true}],"inputs":["b","a"],"result":["gen"]}`)

	result := CompareGraphFiles(refPath, genPath, GraphComparisonOptions{})
	if result.Err() != nil {
		t.Fatalf("expected object and array inputs to match, got %v", result.Err())
	}

	gen.Inputs.Values = []string{"c"}
	result = CompareGraphs(ref, gen, GraphComparisonOptions{})
	if result.Err() == nil {
		t.Fatal("expected top-level input mismatch")
	}
}

func TestCompareGraphFilesPreservesSourceAndBuildPrefixes(t *testing.T) {
	refNode := syntheticNode("ref", "m", nil, []string{"$(BUILD_ROOT)/m/out"})
	refNode.Inputs = []string{"$(SOURCE_ROOT)/m/in.cc"}
	genNode := syntheticNode("gen", "m", nil, []string{"/tmp/build/m/out"})
	genNode.Inputs = []string{"/home/pg/monorepo/yatool_orig/m/in.cc"}

	result := CompareGraphs(syntheticGraph([]ValidationNode{refNode}, []string{"ref"}), syntheticGraph([]ValidationNode{genNode}, []string{"gen"}), GraphComparisonOptions{})
	if result.Err() == nil {
		t.Fatal("expected prefix preservation mismatch")
	}
}

func TestCompareGraphFilesParsesReferenceShape(t *testing.T) {
	if _, err := os.Stat(REFERENCE_GRAPH_PATH); err != nil {
		t.Skipf("reference graph unavailable: %v", err)
	}

	graph := loadValidationGraph(REFERENCE_GRAPH_PATH)
	if len(graph.Nodes) == 0 {
		t.Fatal("expected reference graph nodes")
	}
	if len(graph.Result) == 0 {
		t.Fatal("expected reference result")
	}

	foundCommandFields := false
	for _, node := range graph.Nodes {
		for _, cmd := range node.Cmds {
			if cmd.Env != nil || cmd.Cwd != "" {
				foundCommandFields = true
			}
		}
	}
	if !foundCommandFields {
		t.Log("reference graph did not contain command env/cwd fields in this checkout")
	}
}

func TestMeasureGraphGenerationMedian(t *testing.T) {
	durations := []time.Duration{30 * time.Millisecond, 10 * time.Millisecond, 20 * time.Millisecond}
	if got := medianDuration(durations); got != 20*time.Millisecond {
		t.Fatalf("expected median 20ms, got %v", got)
	}

	even := []time.Duration{40 * time.Millisecond, 10 * time.Millisecond, 30 * time.Millisecond, 20 * time.Millisecond}
	if got := medianDuration(even); got != 25*time.Millisecond {
		t.Fatalf("expected median 25ms, got %v", got)
	}
}

func TestPerformanceMetadataBestEffort(t *testing.T) {
	if bestEffortCommand("definitely-not-a-command") != "unknown" {
		t.Fatal("expected unknown for missing command")
	}
	if bestEffortCPU() == "" {
		t.Fatal("expected CPU metadata fallback")
	}
	if bestEffortRAM() == "" {
		t.Fatal("expected RAM metadata fallback")
	}
}

func BenchmarkValidation3730Nodes(b *testing.B) {
	if _, err := os.Stat(REFERENCE_GRAPH_PATH); err != nil {
		b.Skipf("reference graph unavailable: %v", err)
	}

	reference := loadValidationGraph(REFERENCE_GRAPH_PATH)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := CompareGraphs(reference, reference, GraphComparisonOptions{})
		if result.Err() != nil {
			b.Fatalf("validation failed: %v", result.Err())
		}
	}
}

func assertGraphsMatch(t *testing.T, ref, gen *ValidationGraph) {
	t.Helper()

	result := CompareGraphs(ref, gen, GraphComparisonOptions{})
	if result.Err() != nil {
		t.Fatalf("expected graphs to match, got %v", result.Err())
	}
}

func syntheticGraph(nodes []ValidationNode, result []string) *ValidationGraph {
	graph := &ValidationGraph{
		Conf: &ValidationConf{
			Cache:                     true,
			DefaultNodeRequirements:   json.RawMessage(`{"network":"restricted"}`),
			ExecutionCost:             json.RawMessage(`{"cpu":0,"evaluation_errors":0}`),
			ExplicitRemoteStoreUpload: true,
			GraphSize:                 len(nodes),
			Keepon:                    true,
			Platform:                  "linux",
			Resources:                 json.RawMessage(`[]`),
		},
		Graph:  nodes,
		Inputs: GraphInputs{Values: []string{"ya.make"}},
		Result: result,
	}
	graph.finalize()

	return graph
}

func syntheticNode(uid, moduleDir string, deps []string, outputs []string) ValidationNode {
	return ValidationNode{
		UID:      uid,
		SelfUID:  uid + "-self",
		StatsUID: uid + "-stats",
		Cmds: []ValidationCommand{{
			CmdArgs: []string{"cmd", moduleDir},
		}},
		Inputs:  []string{"$(SOURCE_ROOT)/" + moduleDir + "/ya.make"},
		Outputs: outputs,
		Deps:    deps,
		KV: map[string]string{
			"uid": uid,
		},
		TargetProperties: ValidationTargetProperties{ModuleDir: moduleDir, ModuleLang: "cpp", ModuleType: "library"},
		Env:              map[string]string{},
		Platform:         "linux",
		Requirements:     ValidationRequirements{CPU: 1, Network: "restricted", RAM: 32},
		Sandboxing:       true,
		Tags:             []string{},
		ForeignDeps:      map[string][]string{},
	}
}

func writeRawGraph(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "graph.json")
	Throw(os.WriteFile(path, []byte(content), 0644))

	return path
}
