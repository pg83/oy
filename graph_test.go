package main

import (
	"testing"
)

func TestGraphNodeFields(t *testing.T) {
	node := &GraphNode{
		UID:      "test-uid",
		SelfUID:  "test-self-uid",
		StatsUID: "test-stats-uid",
		Cmds: []Command{
			{
				CmdArgs: []string{"clang++", "-o", "output"},
				Env: map[string]string{
					"PATH": "/usr/bin",
				},
			},
		},
		Inputs:  []string{"$(SOURCE_ROOT)/main.cpp"},
		Outputs: []string{"$(BUILD_ROOT)/main"},
		Deps:    []string{"dep1", "dep2"},
		KV:      map[string]string{"key": "value"},
		TargetProperties: TargetProperties{
			ModuleDir:  "tools/test",
			ModuleLang: "cpp",
			ModuleType: "bin",
		},
	}

	if node.UID != "test-uid" {
		t.Errorf("Expected UID 'test-uid', got '%s'", node.UID)
	}

	if node.SelfUID != "test-self-uid" {
		t.Errorf("Expected SelfUID 'test-self-uid', got '%s'", node.SelfUID)
	}

	if node.StatsUID != "test-stats-uid" {
		t.Errorf("Expected StatsUID 'test-stats-uid', got '%s'", node.StatsUID)
	}

	if len(node.Cmds) != 1 {
		t.Errorf("Expected 1 command, got %d", len(node.Cmds))
	}

	if len(node.Cmds[0].CmdArgs) != 3 {
		t.Errorf("Expected 3 command args, got %d", len(node.Cmds[0].CmdArgs))
	}

	if node.Cmds[0].CmdArgs[0] != "clang++" {
		t.Errorf("Expected first arg 'clang++', got '%s'", node.Cmds[0].CmdArgs[0])
	}

	if len(node.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(node.Inputs))
	}

	if len(node.Outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(node.Outputs))
	}

	if len(node.Deps) != 2 {
		t.Errorf("Expected 2 deps, got %d", len(node.Deps))
	}

	if node.KV["key"] != "value" {
		t.Errorf("Expected KV 'key' to be 'value', got '%s'", node.KV["key"])
	}

	if node.TargetProperties.ModuleDir != "tools/test" {
		t.Errorf("Expected ModuleDir 'tools/test', got '%s'", node.TargetProperties.ModuleDir)
	}

	if node.TargetProperties.ModuleLang != "cpp" {
		t.Errorf("Expected ModuleLang 'cpp', got '%s'", node.TargetProperties.ModuleLang)
	}

	if node.TargetProperties.ModuleType != "bin" {
		t.Errorf("Expected ModuleType 'bin', got '%s'", node.TargetProperties.ModuleType)
	}
}

func TestGraphAddNode(t *testing.T) {
	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: "tools/archiver",
		Language:   "cpp",
	}

	graph := NewGraph(ctx)

	node := &GraphNode{
		UID:     "test-uid",
		SelfUID: "test-self-uid",
		Cmds:    []Command{{}},
		Inputs:  []string{},
		Outputs: []string{},
		Deps:    []string{},
		KV:      map[string]string{},
		TargetProperties: TargetProperties{
			ModuleDir:  "test",
			ModuleLang: "cpp",
			ModuleType: "bin",
		},
	}

	graph.AddNode(node)

	if len(graph.Nodes) != 1 {
		t.Errorf("Expected 1 node in graph, got %d", len(graph.Nodes))
	}

	if graph.Nodes[0].UID != "test-uid" {
		t.Errorf("Expected node UID 'test-uid', got '%s'", graph.Nodes[0].UID)
	}
}

func TestGenerateUID(t *testing.T) {
	uid1 := NewUID([]byte("test/path"))
	uid2 := NewUID([]byte("test/path"))
	uid3 := NewUID([]byte("different/path"))

	if uid1 != uid2 {
		t.Errorf("Expected same UIDs for same input, got '%s' and '%s'", uid1, uid2)
	}

	if uid1 == uid3 {
		t.Errorf("Expected different UIDs for different inputs")
	}

	if uid1 == "" {
		t.Errorf("Expected non-empty UID, got empty string")
	}

	if len(uid1) < 20 {
		t.Errorf("Expected UID length > 20, got %d", len(uid1))
	}
}

func TestGenerateStatsUID(t *testing.T) {
	uid := NewUID([]byte("test"))

	if len(uid) == 0 {
		t.Errorf("Expected non-empty UID")
	}
}

func TestGraphMatchesSgJsonFormat(t *testing.T) {
	ctx := ParseContext{
		Platform:   "linux",
		TargetPath: "tools/archiver",
		Language:   "cpp",
	}

	graph := NewGraph(ctx)

	uid := NewUID([]byte("tools/archiver"))

	node := &GraphNode{
		UID:      uid,
		SelfUID:  "D29NTmzE_1tS8GbzG5m7Qw",
		StatsUID: "916007d422fd2aeff8a8d863e6bd3ced",
		Cmds: []Command{
			{
				CmdArgs: []string{"clang++", "-o", "output"},
				Env: map[string]string{
					"ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
				},
			},
		},
		Inputs:  []string{"$(SOURCE_ROOT)/main.cpp"},
		Outputs: []string{"$(BUILD_ROOT)/archiver"},
		Deps:    []string{"dep1", "dep2"},
		KV: map[string]string{
			"p":        "AR",
			"pc":       "light-red",
			"show_out": "yes",
		},
		TargetProperties: TargetProperties{
			ModuleDir:  "tools/archiver",
			ModuleLang: "cpp",
			ModuleType: "bin",
		},
	}

	graph.AddNode(node)

	if len(graph.Nodes) != 1 {
		t.Errorf("Expected 1 node in graph, got %d", len(graph.Nodes))
	}

	graphNode := graph.Nodes[0]

	if graphNode.UID != uid {
		t.Errorf("Expected UID '%s', got '%s'", uid, graphNode.UID)
	}

	if graphNode.KV["p"] != "AR" {
		t.Errorf("Expected KV[p]='AR', got '%s'", graphNode.KV["p"])
	}

	if graphNode.TargetProperties.ModuleType != "bin" {
		t.Errorf("Expected ModuleType 'bin', got '%s'", graphNode.TargetProperties.ModuleType)
	}

	if len(graphNode.Cmds) != 1 {
		t.Errorf("Expected 1 command, got %d", len(graphNode.Cmds))
	}

	if graphNode.Cmds[0].Env["ARCADIA_ROOT_DISTBUILD"] != "$(SOURCE_ROOT)" {
		t.Errorf("Expected env ARCADIA_ROOT_DISTBUILD, got '%s'", graphNode.Cmds[0].Env["ARCADIA_ROOT_DISTBUILD"])
	}
}
