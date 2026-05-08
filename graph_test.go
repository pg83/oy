package main

import (
	"encoding/json"
	"testing"
)

func TestGraphNodeFields(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node := NewGraphNode(ctx)
	node.UID = "test-uid"
	node.SelfUID = "test-self-uid"
	node.StatsUID = "test-stats-uid"
	node.Cmds = []Command{
		{
			CmdArgs: []string{"clang++", "-o", "output"},
		},
	}
	node.Env = map[string]string{
		"PATH": "/usr/bin",
	}
	node.Platform = "default-linux-x86_64"
	node.Requirements = Requirements{CPU: 1, Network: "restricted", RAM: 32}
	node.Sandboxing = true
	node.Tags = []string{"tool"}
	node.ForeignDeps = ForeignDeps{"pip": []string{"requests==2.28.0"}}
	node.HostPlatform = true
	node.Inputs = []string{"$(SOURCE_ROOT)/main.cpp"}
	node.Outputs = []string{"$(BUILD_ROOT)/main"}
	node.Deps = []string{"dep1", "dep2"}
	node.KV = map[string]string{"key": "value"}
	node.TargetProperties = TargetProperties{
		ModuleDir:  "tools/test",
		ModuleLang: "cpp",
		ModuleType: "bin",
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

	if node.Platform != "default-linux-x86_64" {
		t.Errorf("Expected Platform 'default-linux-x86_64', got '%s'", node.Platform)
	}

	if node.Requirements.CPU != 1 {
		t.Errorf("Expected Requirements.CPU 1, got %d", node.Requirements.CPU)
	}

	if node.Requirements.Network != "restricted" {
		t.Errorf("Expected Requirements.Network 'restricted', got '%s'", node.Requirements.Network)
	}

	if node.Requirements.RAM != 32 {
		t.Errorf("Expected Requirements.RAM 32, got %d", node.Requirements.RAM)
	}

	if !node.Sandboxing {
		t.Errorf("Expected Sandboxing true, got false")
	}

	if len(node.Tags) != 1 || node.Tags[0] != "tool" {
		t.Errorf("Expected Tags ['tool'], got %v", node.Tags)
	}

	if node.ForeignDeps["pip"][0] != "requests==2.28.0" {
		t.Errorf("Expected ForeignDeps, got %v", node.ForeignDeps)
	}

	if !node.HostPlatform {
		t.Errorf("Expected HostPlatform true, got false")
	}
}

func TestGraphAddNode(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/archiver",
		Language:   "cpp",
	}

	graph := NewGraph(ctx)

	node := NewGraphNode(ctx)
	node.UID = "test-uid"
	node.SelfUID = "test-self-uid"
	node.Cmds = []Command{{}}
	node.Inputs = []string{}
	node.Outputs = []string{}
	node.Deps = []string{}
	node.KV = map[string]string{}
	node.TargetProperties = TargetProperties{
		ModuleDir:  "test",
		ModuleLang: "cpp",
		ModuleType: "bin",
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
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/archiver",
		Language:   "cpp",
	}

	graph := NewGraph(ctx)

	uid := NewUID([]byte("tools/archiver"))

	node := NewGraphNode(ctx)
	node.UID = uid
	node.SelfUID = "D29NTmzE_1tS8GbzG5m7Qw"
	node.StatsUID = "916007d422fd2aeff8a8d863e6bd3ced"
	node.Cmds = []Command{
		{
			CmdArgs: []string{"clang++", "-o", "output"},
		},
	}
	node.Env = map[string]string{
		"ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
	}
	node.Inputs = []string{"$(SOURCE_ROOT)/main.cpp"}
	node.Outputs = []string{"$(BUILD_ROOT)/archiver"}
	node.Deps = []string{"dep1", "dep2"}
	node.KV = map[string]string{
		"p":        "AR",
		"pc":       "light-red",
		"show_out": "yes",
	}
	node.TargetProperties = TargetProperties{
		ModuleDir:  "tools/archiver",
		ModuleLang: "cpp",
		ModuleType: "bin",
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

	if graphNode.Env["ARCADIA_ROOT_DISTBUILD"] != "$(SOURCE_ROOT)" {
		t.Errorf("Expected env ARCADIA_ROOT_DISTBUILD, got '%s'", graphNode.Env["ARCADIA_ROOT_DISTBUILD"])
	}
}

func TestNodeEnvironment(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node := NewGraphNode(ctx)
	node.Env = map[string]string{
		"PATH":       "/usr/bin",
		"CPATH":      "/usr/include",
		"ARCADIA":    "/arcadia",
		"BUILD_ROOT": "/build",
	}

	if len(node.Env) != 4 {
		t.Errorf("Expected 4 env vars, got %d", len(node.Env))
	}

	if node.Env["PATH"] != "/usr/bin" {
		t.Errorf("Expected PATH '/usr/bin', got '%s'", node.Env["PATH"])
	}

	if node.Env["CPATH"] != "/usr/include" {
		t.Errorf("Expected CPATH '/usr/include', got '%s'", node.Env["CPATH"])
	}
}

func TestNodePlatform(t *testing.T) {
	ctx1 := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node1 := NewGraphNode(ctx1)
	if node1.Platform != "default-linux-x86_64" {
		t.Errorf("Expected Platform 'default-linux-x86_64', got '%s'", node1.Platform)
	}

	ctx2 := ParseContext{
		Platform:   "default-linux-aarch64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node2 := NewGraphNode(ctx2)
	if node2.Platform != "default-linux-aarch64" {
		t.Errorf("Expected Platform 'default-linux-aarch64', got '%s'", node2.Platform)
	}
}

func TestNodeRequirements(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node := NewGraphNode(ctx)
	node.Requirements = Requirements{
		CPU:     2,
		Network: "full",
		RAM:     64,
	}

	if node.Requirements.CPU != 2 {
		t.Errorf("Expected CPU 2, got %d", node.Requirements.CPU)
	}

	if node.Requirements.Network != "full" {
		t.Errorf("Expected Network 'full', got '%s'", node.Requirements.Network)
	}

	if node.Requirements.RAM != 64 {
		t.Errorf("Expected RAM 64, got %d", node.Requirements.RAM)
	}
}

func TestNodeSandboxing(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node1 := NewGraphNode(ctx)
	if !node1.Sandboxing {
		t.Errorf("Expected default Sandboxing true, got false")
	}

	node2 := NewGraphNode(ctx)
	node2.Sandboxing = false
	if node2.Sandboxing {
		t.Errorf("Expected Sandboxing false, got true")
	}
}

func TestNodeTags(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node1 := NewGraphNode(ctx)
	if len(node1.Tags) != 0 {
		t.Errorf("Expected empty tags, got %v", node1.Tags)
	}

	node2 := NewGraphNode(ctx)
	node2.Tags = []string{"tool"}
	if len(node2.Tags) != 1 || node2.Tags[0] != "tool" {
		t.Errorf("Expected Tags ['tool'], got %v", node2.Tags)
	}

	node3 := NewGraphNode(ctx)
	node3.Tags = []string{"tool", "compiler", "binary"}
	if len(node3.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(node3.Tags))
	}
}

func TestNodeForeignDeps(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node1 := NewGraphNode(ctx)
	if node1.ForeignDeps != nil {
		t.Errorf("Expected nil ForeignDeps, got %v", node1.ForeignDeps)
	}

	node2 := NewGraphNode(ctx)
	node2.ForeignDeps = ForeignDeps{
		"pip": []string{"requests==2.28.0", "numpy==1.24.0"},
		"npm": []string{"lodash"},
	}

	if len(node2.ForeignDeps) != 2 {
		t.Errorf("Expected 2 foreign dep types, got %d", len(node2.ForeignDeps))
	}

	if len(node2.ForeignDeps["pip"]) != 2 {
		t.Errorf("Expected 2 pip packages, got %d", len(node2.ForeignDeps["pip"]))
	}

	if node2.ForeignDeps["pip"][0] != "requests==2.28.0" {
		t.Errorf("Expected pip package 'requests==2.28.0', got '%s'", node2.ForeignDeps["pip"][0])
	}

	if node2.ForeignDeps["npm"][0] != "lodash" {
		t.Errorf("Expected npm package 'lodash', got '%s'", node2.ForeignDeps["npm"][0])
	}
}

func TestNodeHostPlatform(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node1 := NewGraphNode(ctx)
	if node1.HostPlatform {
		t.Errorf("Expected default HostPlatform false, got true")
	}

	node2 := NewGraphNode(ctx)
	node2.HostPlatform = true
	if !node2.HostPlatform {
		t.Errorf("Expected HostPlatform true, got false")
	}
}

func TestGraphJsonSerialization(t *testing.T) {
	ctx := ParseContext{
		Platform:   "default-linux-x86_64",
		TargetPath: "tools/test",
		Language:   "cpp",
	}

	node := NewGraphNode(ctx)
	node.UID = "test-uid"
	node.SelfUID = "test-self-uid"
	node.StatsUID = "test-stats-uid"
	node.Cmds = []Command{
		{
			CmdArgs: []string{"clang++", "-o", "output", "src.cpp"},
		},
	}
	node.Env = map[string]string{
		"PATH":       "/usr/bin",
		"CPATH":      "/usr/include",
		"ARCADIA":    "/arcadia",
		"BUILD_ROOT": "/build",
	}
	node.Inputs = []string{"$(SOURCE_ROOT)/main.cpp", "$(SOURCE_ROOT)/lib.cpp"}
	node.Outputs = []string{"$(BUILD_ROOT)/main"}
	node.Deps = []string{"dep1-uid", "dep2-uid"}
	node.KV = map[string]string{
		"p":        "AR",
		"pc":       "light-red",
		"show_out": "yes",
	}
	node.TargetProperties = TargetProperties{
		ModuleDir:  "tools/test",
		ModuleLang: "cpp",
		ModuleType: "bin",
	}
	node.Platform = "default-linux-x86_64"
	node.Requirements = Requirements{
		CPU:     2,
		Network: "full",
		RAM:     64,
	}
	node.Sandboxing = true
	node.Tags = []string{"tool", "compiler"}
	node.ForeignDeps = ForeignDeps{
		"pip": []string{"requests==2.28.0"},
	}
	node.HostPlatform = true

	data, err := json.MarshalIndent(node, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	var decoded GraphNode
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal node: %v", err)
	}

	if decoded.UID != node.UID {
		t.Errorf("Round-trip failed: UID mismatch")
	}

	if decoded.SelfUID != node.SelfUID {
		t.Errorf("Round-trip failed: SelfUID mismatch")
	}

	if decoded.Platform != node.Platform {
		t.Errorf("Round-trip failed: Platform mismatch")
	}

	if len(decoded.Env) != len(node.Env) {
		t.Errorf("Round-trip failed: Env size mismatch")
	}

	if decoded.Env["PATH"] != node.Env["PATH"] {
		t.Errorf("Round-trip failed: Env PATH mismatch")
	}

	if decoded.Requirements.CPU != node.Requirements.CPU {
		t.Errorf("Round-trip failed: Requirements.CPU mismatch")
	}

	if len(decoded.Tags) != len(node.Tags) {
		t.Errorf("Round-trip failed: Tags size mismatch")
	}

	if len(decoded.ForeignDeps) != len(node.ForeignDeps) {
		t.Errorf("Round-trip failed: ForeignDeps size mismatch")
	}

	if decoded.HostPlatform != node.HostPlatform {
		t.Errorf("Round-trip failed: HostPlatform mismatch")
	}
}
