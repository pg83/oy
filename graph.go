package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
	"sync"
)

type ParseContext struct {
	Platform   string
	TargetPath string
	Language   string
	Musl       bool
	BuildFlags map[string]string
}

type Graph struct {
	Nodes   []*GraphNode
	Inputs  []string
	Context ParseContext
	mu      sync.Mutex
}

type Command struct {
	CmdArgs []string `json:"cmd_args"`
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
	ModuleDir  string `json:"module_dir"`
	ModuleLang string `json:"module_lang"`
	ModuleType string `json:"module_type"`
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
		Resources:                 []Resource{},
	}

	output.Graph = g.Nodes
	output.Inputs = g.Inputs
	output.Result = g.calculateResult()

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
