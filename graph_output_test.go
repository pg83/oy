package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestGraphOutputConfStructure(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	node := NewGraphNode(ctx)
	node.UID = "test-uid"
	node.TargetProperties = TargetProperties{
		ModuleDir:  "test/dir",
		ModuleLang: "cpp",
		ModuleType: "bin",
	}
	graph.AddNode(node)

	output := graph.ToOutput()

	if output.Conf == nil {
		t.Fatal("expected conf section")
	}

	if output.Conf.Platform != "linux" {
		t.Errorf("expected platform=linux, got %s", output.Conf.Platform)
	}

	if output.Conf.GraphSize != 1 {
		t.Errorf("expected graph_size=1, got %d", output.Conf.GraphSize)
	}

	if output.Conf.Cache != true {
		t.Errorf("expected cache=true, got %v", output.Conf.Cache)
	}

	if output.Conf.Keepon != true {
		t.Errorf("expected keepon=true, got %v", output.Conf.Keepon)
	}

	if output.Conf.ExplicitRemoteStoreUpload != true {
		t.Errorf("expected explicit_remote_store_upload=true, got %v", output.Conf.ExplicitRemoteStoreUpload)
	}

	gsid := output.Conf.Gsid
	if output.Conf.Gsid == "" {
		t.Fatal("expected non-empty gsid")
	}

	if !matchesGsidFormat(gsid) {
		t.Errorf("expected gsid format 'USER:* YA:*', got %s", gsid)
	}

	if output.Conf.Description == nil {
		t.Fatal("expected description section")
	}

	if output.Conf.Description.Host == "" {
		t.Error("expected non-empty host in description")
	}

	if output.Conf.Description.User == "" {
		t.Error("expected non-empty user in description")
	}

	if output.Conf.ExecutionCost == nil {
		t.Fatal("expected execution_cost section")
	}

	if output.Conf.ExecutionCost.CPU != 0 {
		t.Errorf("expected execution_cost.cpu=0, got %d", output.Conf.ExecutionCost.CPU)
	}

	if output.Conf.ExecutionCost.EvaluationErrors != 0 {
		t.Errorf("expected execution_cost.evaluation_errors=0, got %d", output.Conf.ExecutionCost.EvaluationErrors)
	}

	if output.Conf.DefaultNodeRequirements == nil {
		t.Fatal("expected default_node_requirements")
	}

	if net, ok := output.Conf.DefaultNodeRequirements["network"]; !ok || net != "restricted" {
		t.Errorf("expected default_node_requirements.network=restricted, got %v", net)
	}

	if output.Conf.MinReqsErrors != 0 {
		t.Errorf("expected min_reqs_errors=0, got %d", output.Conf.MinReqsErrors)
	}

	if output.Conf.Resources == nil {
		t.Fatal("expected resources array")
	}

	if len(output.Conf.Resources) != 6 {
		t.Errorf("expected 6 resources, got %d", len(output.Conf.Resources))
	}

	if output.Conf.Resources[0].Pattern != "OS_SDK_ROOT-sbr:243881345" {
		t.Errorf("expected first resource pattern=OS_SDK_ROOT-sbr:243881345, got %s", output.Conf.Resources[0].Pattern)
	}

	if output.Conf.Resources[0].Resource != "sbr:243881345" {
		t.Errorf("expected first resource resource=sbr:243881345, got %s", output.Conf.Resources[0].Resource)
	}

	vcsIndex := -1
	for i, res := range output.Conf.Resources {
		if res.Pattern == "VCS" {
			vcsIndex = i
			break
		}
	}

	if vcsIndex == -1 {
		t.Fatal("expected VCS resource")
	}

	if output.Conf.Resources[vcsIndex].Name != "vcs" {
		t.Errorf("expected VCS resource name=vcs, got %s", output.Conf.Resources[vcsIndex].Name)
	}

	pythonIndex := -1
	for i, res := range output.Conf.Resources {
		if res.Pattern == "YMAKE_PYTHON3-1002064631" {
			pythonIndex = i
			break
		}
	}

	if pythonIndex == -1 {
		t.Fatal("expected YMAKE_PYTHON3 resource")
	}

	if len(output.Conf.Resources[pythonIndex].Resources) != 5 {
		t.Errorf("expected 5 platform resources for YMAKE_PYTHON3, got %d", len(output.Conf.Resources[pythonIndex].Resources))
	}
}

func matchesGsidFormat(gsid string) bool {
	prefix := "USER:"
	yapos := " YA:"
	userIdx := -1
	yaIdx := -1

	for i := 0; i < len(gsid)-len(" YA:"); i++ {
		if gsid[i:i+len(prefix)] == prefix && userIdx == -1 {
			userIdx = i
		}
		if gsid[i:i+len(yapos)] == yapos && userIdx != -1 {
			yaIdx = i
			break
		}
	}

	return userIdx == 0 && yaIdx > len(prefix) && yaIdx+4 < len(gsid)
}

func TestGraphOutputBasicGraph(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	node1 := NewGraphNode(ctx)
	node1.UID = "test-uid-1"
	node1.TargetProperties = TargetProperties{
		ModuleDir:  "test/dir1",
		ModuleLang: "cpp",
		ModuleType: "bin",
	}
	graph.AddNode(node1)

	node2 := NewGraphNode(ctx)
	node2.UID = "test-uid-2"
	node2.TargetProperties = TargetProperties{
		ModuleDir:  "test/dir2",
		ModuleLang: "cpp",
		ModuleType: "lib",
	}
	node2.Deps = []string{"test-uid-1"}
	graph.AddNode(node2)

	output := graph.ToOutput()

	if output.Conf == nil {
		t.Fatal("expected conf section")
	}

	if len(output.Graph) != 2 {
		t.Errorf("expected 2 graph nodes, got %d", len(output.Graph))
	}

	if output.Graph[0].UID != "test-uid-1" {
		t.Errorf("expected first node uid=test-uid-1, got %s", output.Graph[0].UID)
	}

	if output.Graph[1].UID != "test-uid-2" {
		t.Errorf("expected second node uid=test-uid-2, got %s", output.Graph[1].UID)
	}
}

func TestGraphOutputToFile(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	node := NewGraphNode(ctx)
	node.UID = "test-uid"
	node.TargetProperties = TargetProperties{
		ModuleDir:  "test/dir",
		ModuleLang: "cpp",
		ModuleType: "bin",
	}
	graph.AddNode(node)

	tempFile := t.TempDir() + "/test_graph.json"
	WriteGraphToFile(graph, tempFile)

	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("expected non-empty file content")
	}

	if string(content)[:1] != "{" {
		t.Errorf("expected JSON object start, got %s", string(content)[:1])
	}
}

func TestGraphOutputResultCalculation(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	node1 := NewGraphNode(ctx)
	node1.UID = "root-node"
	node1.TargetProperties = TargetProperties{
		ModuleDir:  "test/root",
		ModuleLang: "cpp",
		ModuleType: "bin",
	}
	graph.AddNode(node1)

	node2 := NewGraphNode(ctx)
	node2.UID = "dep-node"
	node2.TargetProperties = TargetProperties{
		ModuleDir:  "test/dep",
		ModuleLang: "cpp",
		ModuleType: "lib",
	}
	node2.Deps = []string{"root-node"}
	graph.AddNode(node2)

	output := graph.ToOutput()

	if len(output.Result) != 1 {
		t.Fatalf("expected 1 result node, got %d", len(output.Result))
	}

	if output.Result[0] != "root-node" {
		t.Errorf("expected result=root-node, got %s", output.Result[0])
	}
}

func TestGraphOutputEmptyGraph(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	output := graph.ToOutput()

	if output.Conf == nil {
		t.Fatal("expected conf section even for empty graph")
	}

	if output.Conf.GraphSize != 0 {
		t.Errorf("expected graph_size=0 for empty graph, got %d", output.Conf.GraphSize)
	}

	if len(output.Graph) != 0 {
		t.Errorf("expected 0 graph nodes, got %d", len(output.Graph))
	}

	if len(output.Result) != 0 {
		t.Errorf("expected empty result for empty graph, got %v", output.Result)
	}
}

func TestGraphOutputInputs(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	node := NewGraphNode(ctx)
	node.UID = "test-uid"
	node.TargetProperties = TargetProperties{
		ModuleDir:  "test/dir",
		ModuleLang: "cpp",
		ModuleType: "bin",
	}
	graph.AddNode(node)

	graph.AddInput("/path/to/input1.cpp")
	graph.AddInput("/path/to/input2.cpp")

	output := graph.ToOutput()

	if len(output.Inputs) != 0 {
		t.Fatalf("expected 0 top-level inputs (empty object), got %d", len(output.Inputs))
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal output: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, `"inputs":{}`) {
		t.Errorf("expected inputs to be empty object {}, got %s", jsonStr)
	}
}

func TestGraphOutputResultOverride(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	node1 := NewGraphNode(ctx)
	node1.UID = "root-node"
	node1.TargetProperties = TargetProperties{
		ModuleDir:  "test/root",
		ModuleLang: "cpp",
		ModuleType: "bin",
	}
	graph.AddNode(node1)

	node2 := NewGraphNode(ctx)
	node2.UID = "dep-node"
	node2.TargetProperties = TargetProperties{
		ModuleDir:  "test/dep",
		ModuleLang: "cpp",
		ModuleType: "lib",
	}
	node2.Deps = []string{"root-node"}
	graph.AddNode(node2)

	graph.SetResult("wanted")

	output := graph.ToOutput()

	if len(output.Result) != 1 {
		t.Fatalf("expected 1 result node, got %d", len(output.Result))
	}

	if output.Result[0] != "wanted" {
		t.Errorf("expected result=wanted, got %s", output.Result[0])
	}
}

func TestGraphOutputResourceJSONShape(t *testing.T) {
	ctx := ParseContext{Platform: "linux"}
	graph := NewGraph(ctx)

	output := graph.ToOutput()

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("failed to marshal output: %v", err)
	}

	jsonStr := string(data)

	if contains(jsonStr, `"resource":""`) {
		t.Errorf("should not have empty resource strings in JSON: %s", jsonStr)
	}

	if contains(jsonStr, `"resources":null`) {
		t.Errorf("should not have null resources in JSON: %s", jsonStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOfSubstring(s, substr))
}

func indexOfSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == "" || len(id1) != 32 {
		t.Errorf("expected 32-char session ID, got %s (len=%d)", id1, len(id1))
	}

	if id2 == "" || len(id2) != 32 {
		t.Errorf("expected 32-char session ID, got %s (len=%d)", id2, len(id2))
	}

	if id1 == id2 {
		t.Error("expected different session IDs on multiple calls")
	}
}
