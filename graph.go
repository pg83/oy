package main

import (
	"crypto/sha1"
	"encoding/base64"
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
