package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type GraphValidator struct {
	reference *Graph
	generated *Graph
}

type NormalizedNode struct {
	ModuleDir       string
	ModuleLang      string
	ModuleType      string
	DepCount        int
	OutputPaths     []string
	InputPaths      []string
	CommandCount    int
}

func NewGraphValidator(referencePath string, generated *Graph) *GraphValidator {
	referenceData := Throw2(os.ReadFile(referencePath))
	var reference Graph

	Throw(json.Unmarshal(referenceData, &reference))

	return &GraphValidator{
		reference: &reference,
		generated: generated,
	}
}

func (gv *GraphValidator) Validate() error {
	refNodeCount := len(gv.reference.Nodes)
	genNodeCount := len(gv.generated.Nodes)

	if refNodeCount != genNodeCount {
		return Fmt("node count mismatch: reference=%d, generated=%d", refNodeCount, genNodeCount)
	}

	refNormalized := gv.normalizeGraph(gv.reference)
	genNormalized := gv.normalizeGraph(gv.generated)

	if err := gv.compareNormalizedNodes(refNormalized, genNormalized); err != nil {
		return err
	}

	if err := gv.compareDepEdges(); err != nil {
		return err
	}

	return nil
}

func (gv *GraphValidator) normalizeGraph(graph *Graph) map[string]*NormalizedNode {
	normalized := make(map[string]*NormalizedNode)

	for _, node := range graph.Nodes {
		normNode := &NormalizedNode{
			ModuleDir:    node.TargetProperties.ModuleDir,
			ModuleLang:   node.TargetProperties.ModuleLang,
			ModuleType:   node.TargetProperties.ModuleType,
			DepCount:     len(node.Deps),
			OutputPaths:  make([]string, len(node.Outputs)),
			InputPaths:   make([]string, len(node.Inputs)),
			CommandCount: len(node.Cmds),
		}

		copy(normNode.OutputPaths, node.Outputs)
		copy(normNode.InputPaths, node.Inputs)

		sort.Strings(normNode.OutputPaths)
		sort.Strings(normNode.InputPaths)

		sequentialID := gv.generateSequentialID(node.TargetProperties.ModuleDir)
		normalized[sequentialID] = normNode
	}

	return normalized
}

func (gv *GraphValidator) generateSequentialID(moduleDir string) string {
	modules := make([]string, 0)
	seen := make(map[string]bool)

	for _, node := range gv.reference.Nodes {
		if !seen[node.TargetProperties.ModuleDir] {
			seen[node.TargetProperties.ModuleDir] = true
			modules = append(modules, node.TargetProperties.ModuleDir)
		}
	}

	for _, node := range gv.generated.Nodes {
		if !seen[node.TargetProperties.ModuleDir] {
			seen[node.TargetProperties.ModuleDir] = true
			modules = append(modules, node.TargetProperties.ModuleDir)
		}
	}

	sort.Strings(modules)

	for i, mod := range modules {
		if mod == moduleDir {
			return fmt.Sprintf("NODE_%04d", i)
		}
	}

	return fmt.Sprintf("NODE_UNKNOWN_%s", moduleDir)
}

func (gv *GraphValidator) compareNormalizedNodes(ref, gen map[string]*NormalizedNode) error {
	for id, refNode := range ref {
		genNode, exists := gen[id]
		if !exists {
			return Fmt("generated graph missing node: %s (dir=%s)", id, refNode.ModuleDir)
		}

		if refNode.ModuleDir != genNode.ModuleDir {
			return Fmt("node %s: module_dir mismatch (ref=%s, gen=%s)", id, refNode.ModuleDir, genNode.ModuleDir)
		}

		if refNode.ModuleLang != genNode.ModuleLang {
			return Fmt("node %s: module_lang mismatch (ref=%s, gen=%s)", id, refNode.ModuleLang, genNode.ModuleLang)
		}

		if refNode.ModuleType != genNode.ModuleType {
			return Fmt("node %s: module_type mismatch (ref=%s, gen=%s)", id, refNode.ModuleType, genNode.ModuleType)
		}

		if refNode.DepCount != genNode.DepCount {
			return Fmt("node %s: dependency count mismatch (ref=%d, gen=%d)", id, refNode.DepCount, genNode.DepCount)
		}

		if refNode.CommandCount != genNode.CommandCount {
			return Fmt("node %s: command count mismatch (ref=%d, gen=%d)", id, refNode.CommandCount, genNode.CommandCount)
		}
	}

	for id := range gen {
		if _, exists := ref[id]; !exists {
			return Fmt("generated graph has extra node: %s", id)
		}
	}

	return nil
}

func (gv *GraphValidator) compareDepEdges() error {
	refDeps := gv.extractDependencyMap(gv.reference)
	genDeps := gv.extractDependencyMap(gv.generated)

	for nodeID, refDepList := range refDeps {
		genDepList, exists := genDeps[nodeID]
		if !exists {
			return Fmt("generated graph missing node in dependency map: %s", nodeID)
		}

		if len(refDepList) != len(genDepList) {
			return Fmt("node %s: dependency list length mismatch (ref=%d, gen=%d)", nodeID, len(refDepList), len(genDepList))
		}

		sort.Strings(refDepList)
		sort.Strings(genDepList)

		for i, refDep := range refDepList {
			if refDep != genDepList[i] {
				return Fmt("node %s: dependency edge mismatch at index %d (ref=%s, gen=%s)", nodeID, i, refDep, genDepList[i])
			}
		}
	}

	return nil
}

func (gv *GraphValidator) extractDependencyMap(graph *Graph) map[string][]string {
	depMap := make(map[string][]string)

	for _, node := range graph.Nodes {
		nodeID := gv.generateSequentialID(node.TargetProperties.ModuleDir)

		depIDs := make([]string, 0, len(node.Deps))

		for _, depUID := range node.Deps {
			depNode := gv.findNodeByUID(graph, depUID)
			if depNode != nil {
				depID := gv.generateSequentialID(depNode.TargetProperties.ModuleDir)
				depIDs = append(depIDs, depID)
			}
		}

		depMap[nodeID] = depIDs
	}

	return depMap
}

func (gv *GraphValidator) findNodeByUID(graph *Graph, uid string) *GraphNode {
	for _, node := range graph.Nodes {
		if node.UID == uid {
			return node
		}
	}

	return nil
}
