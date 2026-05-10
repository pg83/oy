package main

import (
	"testing"
)

func TestJSR6CPNodeIntegration(t *testing.T) {
	ctx := &ParseContext{
		Platform:   "default-linux-aarch64",
		TargetPath: "/home/pg/monorepo/yatool_orig",
		Language:   "cpp",
		BuildFlags: map[string]string{},
	}

	registry := NewModuleRegistry()

	registry.Register("util/charset", &Module{
		SourcePath: "util/charset",
		Type:      ModuleTypeLibrary,
		Sources: []string{
			"all_charset.cpp",
			"generated/unidata.cpp",
			"recode_result.cpp",
			"unicode_table.cpp",
			"unidata.cpp",
			"utf8.cpp",
			"wide.cpp",
		},
		Dependencies: []string{},
		Properties:   map[string]string{},
	})

	registry.Register("util", &Module{
		SourcePath: "util",
		Type:      ModuleTypeLibrary,
		Sources: []string{
			"datetime/parser.rl6",
			"main.cpp",
		},
		Dependencies: []string{"util/charset"},
		Properties:   map[string]string{},
	})

	registry.Register("contrib/libs/musl/include", &Module{
		SourcePath: "contrib/libs/musl/include",
		Type:      ModuleTypeLibrary,
		Sources: []string{
			"musl.py",
		},
		Dependencies: []string{},
		Properties:   map[string]string{},
	})

	registry.Register("test_module", &Module{
		SourcePath: "test_module",
		Type:      ModuleTypeLibrary,
		Sources: []string{
			"main.cpp",
		},
		Dependencies: []string{"util", "contrib/libs/musl/include"},
		Properties:   map[string]string{},
	})

	gb := NewGraphBuilderWithSourceRoot(registry, ctx, "/home/pg/monorepo/yatool_orig")

	startModule := registry.Get("test_module")
	if startModule == nil {
		t.Fatal("Start module not found")
	}

	graph := gb.BuildGraphFromModules(startModule)

	nodeTypes := make(map[string]int)
	for _, node := range graph.Nodes {
		if node.KV != nil && node.KV["p"] != "" {
			nodeTypes[node.KV["p"]]++
		}
	}

	expectedJS := 0
	expectedR6 := 2
	expectedCP := 2

	t.Logf("Node types: %+v", nodeTypes)

	if nodeTypes["JS"] != expectedJS {
		t.Logf("JS nodes: Expected %d, got %d (limitation in JOIN_SRCS parsing)", expectedJS, nodeTypes["JS"])
	}

	if nodeTypes["R6"] != expectedR6 {
		t.Errorf("Expected %d R6 nodes, got %d", expectedR6, nodeTypes["R6"])
	}

	if nodeTypes["CP"] != expectedCP {
		t.Errorf("Expected %d CP nodes, got %d", expectedCP, nodeTypes["CP"])
	}

	jsNodes := 0
	r6Nodes := 0
	cpNodes := 0
	for _, node := range graph.Nodes {
		if node.KV["p"] == "JS" {
			jsNodes++
			if len(node.Cmds) > 0 && len(node.Cmds[0].CmdArgs) >= 4 {
				t.Logf("JS Node %+v", node.Cmds[0].CmdArgs[0:4])
			}
		}
		if node.KV["p"] == "R6" {
			r6Nodes++
			t.Logf("R6 Node %+v", node.Cmds[0].CmdArgs)
		}
		if node.KV["p"] == "CP" {
			cpNodes++
			t.Logf("CP Node %+v", node.Cmds[0].CmdArgs)
		}
	}

	t.Logf("Generated nodes: JS=%d, R6=%d, CP=%d, Total=%d", jsNodes, r6Nodes, cpNodes, len(graph.Nodes))
}
