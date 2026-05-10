package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sync/errgroup"
)

type FileParser struct {
	registry     *ModuleRegistry
	graph        *Graph
	buildCtx     *BuildContext
	vars         VariableSet
	visited      map[string]bool
	visitedMutex sync.Mutex
}

func NewFileParser(registry *ModuleRegistry, graph *Graph, buildCtx *BuildContext, vars VariableSet) *FileParser {
	return &FileParser{
		registry:     registry,
		graph:        graph,
		buildCtx:     buildCtx,
		vars:         vars,
		visited:      make(map[string]bool),
		visitedMutex: sync.Mutex{},
	}
}

func (fp *FileParser) Parse(rootPath string) {
	absPath := Throw2(filepath.Abs(rootPath))

	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	fp.parseFile(absPath, g, ctx)

	Throw(g.Wait())
}

func (fp *FileParser) markVisited(path string) bool {
	fp.visitedMutex.Lock()
	defer fp.visitedMutex.Unlock()

	if fp.visited[path] {
		return false
	}

	fp.visited[path] = true
	return true
}

func (fp *FileParser) parseFile(path string, g *errgroup.Group, ctx context.Context) {
	if !fp.markVisited(path) {
		return
	}

	file := ParseYaMakeFile(path)

	for _, module := range file.Modules {
		fp.processModule(module, filepath.Dir(path), g, ctx)
	}

	for _, dir := range file.Imports {
		fp.handleRecurse(dir, filepath.Dir(path), g, ctx)
	}
}

func (fp *FileParser) processModule(module *Module, moduleDir string, g *errgroup.Group, ctx context.Context) {
	resolvedModule := ResolveConditionals(module, fp.buildCtx, fp.vars)
	ResolveWhenBlocks(resolvedModule, fp.buildCtx, fp.vars)

	if !EvaluateBuildCondition(resolvedModule, fp.buildCtx, fp.vars) {
		return
	}

	modulePath := filepath.Join(moduleDir, "")
	fp.registry.Register(modulePath, resolvedModule)

	node := fp.moduleToGraphNode(resolvedModule, moduleDir)
	fp.graph.AddNode(node)

	for _, recurse := range resolvedModule.Recursions {
		fp.handleRecurse(recurse, moduleDir, g, ctx)
	}
}

func (fp *FileParser) moduleToGraphNode(module *Module, moduleDir string) *GraphNode {
	node := NewGraphNode(fp.graph.Context)

	moduleType := "lib"
	switch module.Type {
	case ModuleTypeProgram:
		moduleType = "bin"
	case ModuleTypeGoLibrary:
		moduleType = "go"
	case ModuleTypeDLL:
		moduleType = "dll"
	case ModuleTypePy23Library, ModuleTypePyLibrary:
		moduleType = "python"
	}

	node.TargetProperties = TargetProperties{
		ModuleDir:  moduleDir,
		ModuleLang: fp.inferLanguage(module),
		ModuleType: moduleType,
	}

	defaultPlatform := "linux"
	if fp.buildCtx.Platform == PlatformWindows {
		defaultPlatform = "windows"
	} else if fp.buildCtx.Platform == PlatformDarwin {
		defaultPlatform = "darwin"
	}
	node.Platform = defaultPlatform

	node.UID = NewUID([]byte(moduleDir + module.Type.String()))
	node.SelfUID = NewUID([]byte(moduleDir + "_self"))
	node.StatsUID = NewUID([]byte(moduleDir + "_stats"))

	return node
}

func (fp *FileParser) inferLanguage(module *Module) string {
	switch module.Type {
	case ModuleTypeProgram, ModuleTypeLibrary:
		return "cpp"
	case ModuleTypeGoLibrary:
		return "go"
	case ModuleTypePy23Library, ModuleTypePyLibrary:
		return "python"
	default:
		return "unknown"
	}
}

func (fp *FileParser) handleRecurse(dir *RecurseDirective, basePath string, g *errgroup.Group, ctx context.Context) {
	for _, relPath := range dir.Paths {
		subdirPath := filepath.Join(basePath, relPath)
		yaMakePath := filepath.Join(subdirPath, "ya.make")

		if _, err := os.Stat(yaMakePath); os.IsNotExist(err) {
			continue
		}

		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			fp.parseFile(yaMakePath, g, ctx)
			return nil
		})
	}
}
