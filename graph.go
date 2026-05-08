package main

import (
	"crypto/sha1"
	"encoding/base64"
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
}

type Command struct {
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
}

type TargetProperties struct {
	ModuleDir  string `json:"module_dir"`
	ModuleLang string `json:"module_lang"`
	ModuleType string `json:"module_type"`
}

func NewGraph(ctx ParseContext) *Graph {
	return &Graph{
		Context: ctx,
		Nodes:   make([]*GraphNode, 0),
		Inputs:  make([]string, 0),
	}
}

func NewUID(moduleDir []byte) string {
	h := sha1.New()
	h.Write(moduleDir)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func (g *Graph) AddNode(node *GraphNode) {
	g.Nodes = append(g.Nodes, node)
}

func (g *Graph) AddInput(path string) {
	g.Inputs = append(g.Inputs, path)
}
