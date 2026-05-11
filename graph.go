package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
)

type ParseContext struct {
	Platform       string
	ArchString     string
	TargetPath     string
	Language       string
	Musl           bool
	BuildFlags     map[string]string
	TargetPlatform string
}

type Graph struct {
	Nodes   []*GraphNode
	Inputs  []string
	Context ParseContext
	Result  []string `json:"-"`
	mu      sync.Mutex
}

type Command struct {
	CmdArgs []string          `json:"cmd_args"`
	Env     map[string]string `json:"env,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
}

type ReferenceGraph struct {
	Conf   *GraphConf        `json:"conf"`
	Nodes  []*GraphNode      `json:"graph"`
	Inputs map[string]string `json:"inputs"`
	Result []string          `json:"result"`
}

type GraphConf struct {
	GraphSize int `json:"graph_size"`
}

type CommandWithEnv struct {
	CmdArgs []string          `json:"cmd_args"`
	Env     map[string]string `json:"env"`
}

type GraphNode struct {
	UID              string                     `json:"uid"`
	SelfUID          string                     `json:"self_uid"`
	StatsUID         string                     `json:"stats_uid"`
	Cmds             []Command                  `json:"cmds"`
	Inputs           []string                   `json:"inputs"`
	Outputs          []string                   `json:"outputs"`
	Deps             []string                   `json:"deps"`
	KV               map[string]string          `json:"kv"`
	TargetProperties TargetProperties           `json:"target_properties"`
	Env              map[string]string          `json:"env"`
	Platform         string                     `json:"platform"`
	Requirements     Requirements               `json:"requirements"`
	Sandboxing       bool                       `json:"sandboxing"`
	Tags             []string                   `json:"tags"`
	ForeignDeps      ForeignDeps                `json:"foreign_deps"`
	HostPlatform     bool                       `json:"host_platform"`
	Extra            map[string]json.RawMessage `json:"-"`
}

type graphNodeJSON struct {
	UID              string            `json:"uid"`
	SelfUID          string            `json:"self_uid"`
	StatsUID         string            `json:"stats_uid"`
	Cmds             []Command         `json:"cmds"`
	Inputs           []string          `json:"inputs"`
	Outputs          []string          `json:"outputs"`
	Deps             []string          `json:"deps"`
	KV               map[string]string `json:"kv"`
	TargetProperties TargetProperties  `json:"target_properties"`
	Env              map[string]string `json:"env"`
	Platform         string            `json:"platform"`
	Requirements     Requirements      `json:"requirements"`
	Sandboxing       bool              `json:"sandboxing"`
	Tags             []string          `json:"tags"`
	ForeignDeps      ForeignDeps       `json:"foreign_deps"`
	HostPlatform     bool              `json:"host_platform"`
}

type TargetProperties struct {
	ModuleDir  string                     `json:"module_dir"`
	ModuleLang string                     `json:"module_lang"`
	ModuleType string                     `json:"module_type"`
	ModuleTag  string                     `json:"module_tag"`
	Fields     map[string]json.RawMessage `json:"-"`
}

func (node GraphNode) MarshalJSON() ([]byte, error) {
	fields := copyRawMap(node.Extra)
	knownFields, err := graphNodeKnownFields(node)
	if err != nil {
		return nil, err
	}

	for key, value := range knownFields {
		fields[key] = value
	}

	return json.Marshal(fields)
}

func (node *GraphNode) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	var decoded graphNodeJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	deleteGraphNodeKnownFields(fields)
	*node = graphNodeFromJSON(decoded)
	node.Extra = fields

	return nil
}

func (tp TargetProperties) MarshalJSON() ([]byte, error) {
	return json.Marshal(tp.rawFields())
}

func (tp *TargetProperties) UnmarshalJSON(data []byte) error {
	fields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	tp.Fields = fields
	var err error
	if tp.ModuleDir, err = targetPropertyString(fields, "module_dir"); err != nil {
		return err
	}
	if tp.ModuleLang, err = targetPropertyString(fields, "module_lang"); err != nil {
		return err
	}
	if tp.ModuleType, err = targetPropertyString(fields, "module_type"); err != nil {
		return err
	}
	if tp.ModuleTag, err = targetPropertyString(fields, "module_tag"); err != nil {
		return err
	}

	return nil
}

func (tp TargetProperties) rawFields() map[string]json.RawMessage {
	fields := copyRawMap(tp.Fields)

	delete(fields, "module_dir")
	delete(fields, "module_lang")
	delete(fields, "module_type")
	delete(fields, "module_tag")
	setTargetPropertyString(fields, "module_dir", tp.ModuleDir)
	setTargetPropertyString(fields, "module_lang", tp.ModuleLang)
	setTargetPropertyString(fields, "module_type", tp.ModuleType)
	setTargetPropertyString(fields, "module_tag", tp.ModuleTag)

	return fields
}

func graphNodeKnownFields(node GraphNode) (map[string]json.RawMessage, error) {
	data, err := json.Marshal(graphNodeToJSON(node))
	if err != nil {
		return nil, err
	}

	fields := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}

	return fields, nil
}

func graphNodeToJSON(node GraphNode) graphNodeJSON {
	return graphNodeJSON{
		UID:              node.UID,
		SelfUID:          node.SelfUID,
		StatsUID:         node.StatsUID,
		Cmds:             node.Cmds,
		Inputs:           node.Inputs,
		Outputs:          node.Outputs,
		Deps:             node.Deps,
		KV:               node.KV,
		TargetProperties: node.TargetProperties,
		Env:              node.Env,
		Platform:         node.Platform,
		Requirements:     node.Requirements,
		Sandboxing:       node.Sandboxing,
		Tags:             node.Tags,
		ForeignDeps:      node.ForeignDeps,
		HostPlatform:     node.HostPlatform,
	}
}

func graphNodeFromJSON(node graphNodeJSON) GraphNode {
	return GraphNode{
		UID:              node.UID,
		SelfUID:          node.SelfUID,
		StatsUID:         node.StatsUID,
		Cmds:             node.Cmds,
		Inputs:           node.Inputs,
		Outputs:          node.Outputs,
		Deps:             node.Deps,
		KV:               node.KV,
		TargetProperties: node.TargetProperties,
		Env:              node.Env,
		Platform:         node.Platform,
		Requirements:     node.Requirements,
		Sandboxing:       node.Sandboxing,
		Tags:             node.Tags,
		ForeignDeps:      node.ForeignDeps,
		HostPlatform:     node.HostPlatform,
	}
}

func deleteGraphNodeKnownFields(fields map[string]json.RawMessage) {
	for _, key := range []string{
		"uid",
		"self_uid",
		"stats_uid",
		"cmds",
		"inputs",
		"outputs",
		"deps",
		"kv",
		"target_properties",
		"env",
		"platform",
		"requirements",
		"sandboxing",
		"tags",
		"foreign_deps",
		"host_platform",
	} {
		delete(fields, key)
	}
}

type Requirements struct {
	CPU     int    `json:"cpu"`
	Network string `json:"network"`
	RAM     int    `json:"ram"`
}

type ForeignDeps map[string][]string

func NewGraph(ctx ParseContext) *Graph {
	return &Graph{
		Context: ctx,
		Nodes:   make([]*GraphNode, 0),
		Inputs:  make([]string, 0),
	}
}

func NewGraphNode(ctx ParseContext) *GraphNode {
	return &GraphNode{
		Env:          make(map[string]string),
		Platform:     ctx.Platform,
		Requirements: Requirements{CPU: 1, Network: "restricted", RAM: 32},
		Sandboxing:   true,
		Tags:         make([]string, 0),
		ForeignDeps:  nil,
		HostPlatform: false,
	}
}

func NewUID(moduleDir []byte) string {
	h := sha1.New()
	h.Write(moduleDir)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func (g *Graph) AddNode(node *GraphNode) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Nodes = append(g.Nodes, node)
}

func (g *Graph) AddInput(path string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Inputs = append(g.Inputs, path)
}

func (g *Graph) SetResult(uids ...string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Result = append([]string(nil), uids...)
}

func (g *Graph) ToOutput() *GraphOutput {
	output := &GraphOutput{}

	hostname := Throw2(os.Hostname())
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}
	sessionID := generateSessionID()

	output.Conf = &Conf{
		Cache:     true,
		Platform:  g.Context.Platform,
		GraphSize: len(g.Nodes),
		Gsid:      fmt.Sprintf("USER:%s YA:%s", username, sessionID),
		Description: &Description{
			Host:     hostname,
			Platform: runtime.GOOS + "-" + runtime.GOARCH,
			User:     username,
		},
		ExecutionCost: &ExecutionCost{
			CPU:              0,
			EvaluationErrors: 0,
		},
		DefaultNodeRequirements: map[string]interface{}{
			"network": "restricted",
		},
		ExplicitRemoteStoreUpload: true,
		Keepon:                    true,
		MinReqsErrors:             0,
		Resources: []Resource{
			{
				Pattern:  "OS_SDK_ROOT-sbr:243881345",
				Resource: "sbr:243881345",
			},
			{
				Pattern: "YMAKE_PYTHON3-1002064631",
				Resources: []ResourcePlatform{
					{Platform: "DARWIN", Resource: "sbr:11634932281"},
					{Platform: "DARWIN-ARM64", Resource: "sbr:11634870116"},
					{Platform: "LINUX", Resource: "sbr:11635938717"},
					{Platform: "LINUX-AARCH64", Resource: "sbr:11635443583"},
					{Platform: "WIN32", Resource: "sbr:11635994815"},
				},
			},
			{
				Pattern: "LLD_ROOT-3107549726",
				Resources: []ResourcePlatform{
					{Platform: "DARWIN", Resource: "sbr:9766080017"},
					{Platform: "DARWIN-ARM64", Resource: "sbr:9766092757"},
					{Platform: "LINUX", Resource: "sbr:9766798141"},
					{Platform: "LINUX-AARCH64", Resource: "sbr:9766596779"},
				},
			},
			{
				Pattern: "CLANG-2403293607",
				Resources: []ResourcePlatform{
					{Platform: "darwin", Resource: "sbr:10492078400"},
					{Platform: "darwin-arm64", Resource: "sbr:10492063598"},
					{Platform: "linux", Resource: "sbr:10492077879"},
					{Platform: "linux-aarch64", Resource: "sbr:10492240158"},
					{Platform: "win32", Resource: "sbr:10492219373"},
				},
			},
			{
				Pattern:  "OS_SDK_ROOT-sbr:309054781",
				Resource: "sbr:309054781",
			},
			{
				Name:     "vcs",
				Pattern:  "VCS",
				Resource: "base64:vcs.json:ewogICAgIkFSQ0FESUFfUEFUQ0hfTlVNQkVSIjogMTAwMzAsCiAgICAiQVJDQURJQV9TT1VSQ0VfSEdfSEFTSCI6ICJkOTU3MjgzYzgzNmNjNDhmOTEyNWZhMDc3Yjk0ZDdlZmE3OTgzN2NjIiwKICAgICJBUkNBRElBX1NPVVJDRV9MQVNUX0FVVEhPUiI6ICJhbGV2aXRza2lpIDxhbGV2aXRza2lpQHlhbmRleC10ZWFtLmNvbT4iLAogICAgIkFSQ0FESUFfU09VUkNFX0xBU1RfQ0hBTkdFIjogLTEsCiAgICAiQVJDQURJQV9TT1VSQ0VfUEFUSCI6ICIvaG9tZS9wZy9tb25vcmVwby95YXRvb2xfb3JpZyIsCiAgICAiQVJDQURJQV9TT1VSQ0VfUkVWSVNJT04iOiAtMSwKICAgICJBUkNBRElBX1NPVVJDRV9VUkwiOiAiIiwKICAgICJBUkNBRElBX1RBRyI6ICIiLAogICAgIkJSQU5DSCI6ICJoZWFkcy9tYWluIiwKICAgICJCVUlMRF9EQVRFIjogIjIwMjYtMDUtMDhUMTk6NDQ6NDQuMDAwMDAwWiIsCiAgICAiQlVJTERfSE9TVCI6ICJwZy52bGEueXAtYy55YW5kZXgubmV0IiwKICAgICJCVUlMRF9USU1FU1RBTVAiOiAxNzc4MjY5NDg0LAogICAgIkJVSUxEX1VTRVIiOiAicGciLAogICAgIkNVU1RPTV9WRVJTSU9OIjogIiIsCiAgICAiRElSVFkiOiAiIiwKICAgICJQUk9HUkFNX1ZFUlNJT04iOiAiR2l0IGluZm86XG4gICAgQ29tbWl0OiBkOTU3MjgzYzgzNmNjNDhmOTEyNWZhMDc3Yjk0ZDdlZmE3OTgzN2NjXG4gICAgQnJhbmNoOiBoZWFkcy9tYWluXG4gICAgQXV0aG9yOiBhbGV2aXRza2lpIDxhbGV2aXRza2lpQHlhbmRleC10ZWFtLmNvbT5cbiAgICBTdW1tYXJ5OiBFbmFibGUgeWZtIHN5bmMgdG8gb3BlbnNvdXJjZVxuT3RoZXIgaW5mbzpcbiAgICBCdWlsZCBieTogcGdcbiAgICBUb3Agc3JjIGRpcjogL2hvbWUvcGcvbW9ub3JlcG8veWF0b29sX29yaWdcbiAgICBUb3AgYnVpbGQgZGlyOiAvaG9tZS9wZy8ueWEvYnVpbGRcbiAgICBIb3N0bmFtZTogcGcudmxhLnlwLWMueWFuZGV4Lm5ldCAgICBIb3N0IGluZm9ybWF0aW9uOiBcbiAgICAgICAgTGludXggcGcudmxhLnlwLWMueWFuZGV4Lm5ldCA1LjQuMTYxLTI2LjMgIzEgU01QIE1vbiBGZWIgNyAxNDo0Nzo1OCBVVEMgMjAyMiB4ODZfNjRcblxuICAgICAiLAogICAgIlJFTEVBU0VfVkVSU0lPTiI6ICIiLAogICAgIlJFUE9TSVRPUlkiOiAiIiwKICAgICJTQ01fREFUQSI6ICJHaXQgaW5mbzpcbiAgICBDb21taXQ6IGQ5NTcyODNjODM2Y2M0OGY5MTI1ZmEwNzdiOTRkN2VmYTc5ODM3Y2NcbiAgICBCcmFuY2g6IGhlYWRzL21haW5cbiAgICBBdXRob3I6IGFsZXZpdHNraWkgPGFsZXZpdHNraWlAeWFuZGV4LXRlYW0uY29tPlxuICAgIFN1bW1hcnk6IEVuYWJsZSB5Zm0gc3luYyB0byBvcGVuc291cmNlIiwiVCZTIjogImdpdCJ9",
			},
		},
	}

	output.Graph = g.Nodes
	output.Inputs = make(GraphOutputInputs)

	if len(g.Result) > 0 {
		output.Result = g.Result
	} else {
		output.Result = g.calculateResult()
	}

	return output
}

func (g *Graph) calculateResult() []string {
	if len(g.Nodes) == 0 {
		return []string{}
	}

	rootNode := g.findRootNode()
	if rootNode != nil {
		return []string{rootNode.UID}
	}

	return []string{g.Nodes[0].UID}
}

func (g *Graph) findRootNode() *GraphNode {
	for _, node := range g.Nodes {
		if len(node.Deps) == 0 {
			return node
		}
	}
	return nil
}
