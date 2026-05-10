package main

import (
	"os"
	"path/filepath"
	"strings"
)

type BuildEngine struct {
	registry    *ModuleRegistry
	recurser    *RecurseProcessor
	ctx         *ParseContext
	sourceRoot  string
	visitedDeps map[string]bool
}

func NewBuildEngine(ctx *ParseContext, sourceRoot string) *BuildEngine {
	registry := NewModuleRegistry()
	recurser := NewRecurseProcessor(registry, ctx)

	return &BuildEngine{
		registry:    registry,
		recurser:    recurser,
		ctx:         ctx,
		sourceRoot:  sourceRoot,
		visitedDeps: make(map[string]bool),
	}
}

func (be *BuildEngine) BuildDependencyGraph(targetPath string) *Graph {
	absTargetPath := be.resolveTargetPath(targetPath)

	file := ParseYaMakeFile(absTargetPath)

	targetDir := filepath.Dir(filepath.ToSlash(absTargetPath))
	be.recurser.ProcessFileImports(file, targetDir, be.sourceRoot)

	if len(file.Modules) == 0 {
		ThrowFmt("no modules found in ya.make file: %s", absTargetPath)
	}

	mainModule := file.Modules[0]
	mainModule.SourcePath = filepath.Dir(filepath.ToSlash(absTargetPath))
	mainModule.SourcePath = strings.TrimPrefix(mainModule.SourcePath, be.sourceRoot)
	mainModule.SourcePath = strings.TrimPrefix(mainModule.SourcePath, "/")

	be.recurser.ProcessRecurse(mainModule, be.sourceRoot)

	buildContext := be.createBuildContext()

	resolvedModule := ResolveConditionals(mainModule, buildContext, NewMemoryVariableSet())
	ResolveWhenBlocks(resolvedModule, buildContext, NewMemoryVariableSet())

	if !EvaluateBuildCondition(resolvedModule, buildContext, NewMemoryVariableSet()) {
		ThrowFmt("module excluded by BUILD_ONLY_IF condition: %s", mainModule.SourcePath)
	}

	be.registry.Register(mainModule.SourcePath, resolvedModule)

	be.processPeerDependencies(resolvedModule)

	graphBuilder := NewGraphBuilderWithSourceRoot(be.registry, be.ctx, be.sourceRoot)
	graph := graphBuilder.BuildGraphFromModules(resolvedModule)

	return graph
}

func (be *BuildEngine) resolveTargetPath(targetPath string) string {
	if filepath.IsAbs(targetPath) {
		return targetPath
	}

	sourcePath := filepath.Join(be.sourceRoot, targetPath, "ya.make")

	if _, err := os.Stat(sourcePath); err == nil {
		return sourcePath
	}

	sourcePath = filepath.Join(be.sourceRoot, targetPath)

	if _, err := os.Stat(sourcePath); err == nil {
		return sourcePath
	}

	ThrowFmt("target path not found: %s", targetPath)

	return ""
}

func (be *BuildEngine) createBuildContext() *BuildContext {
	ctx := NewBuildContext()

	if be.ctx.Platform != "" {
		switch be.ctx.Platform {
		case "linux":
			ctx.Platform = PlatformLinux
		case "windows":
			ctx.Platform = PlatformWindows
		case "darwin", "mac":
			ctx.Platform = PlatformDarwin
		}
	}

	for flag, value := range be.ctx.BuildFlags {
		ctx.Flags[flag] = (value == "true" || value == "1" || value == "yes")
	}

	return ctx
}

func (be *BuildEngine) processPeerDependencies(module *Module) {
	if module == nil {
		return
	}

	moduleKey := NormalizedPath(module.SourcePath)
	if be.visitedDeps[moduleKey] {
		return
	}
	be.visitedDeps[moduleKey] = true

	if module.Type == ModuleTypeProgram || module.Type == ModuleTypeLibrary {
		hasUtil := false
		hasLibcxx := false

		for _, dep := range module.Dependencies {
			normDep := NormalizedPath(dep)
			if normDep == "util" {
				hasUtil = true
			} else if normDep == "contrib/libs/cxxsupp/libcxx" {
				hasLibcxx = true
			}
		}

		if !hasUtil && moduleKey != "util" {
			module.Dependencies = append(module.Dependencies, "util")
		}
		if !hasLibcxx && moduleKey != "contrib/libs/cxxsupp/libcxx" {
			module.Dependencies = append(module.Dependencies, "contrib/libs/cxxsupp/libcxx")
		}
	}

	for _, depPath := range module.Dependencies {
		be.loadModuleAndDependencies(depPath)
	}
}

func (be *BuildEngine) loadModuleAndDependencies(depPath string) {
	if be.registry.Has(depPath) {
		return
	}

	moduleYaMakePath := be.findModuleYaMakeFile(depPath)
	if moduleYaMakePath == "" {
		return
	}

	var file *File
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	file = ParseYaMakeFile(moduleYaMakePath)

	if file == nil {
		return
	}

	moduleDir := filepath.Dir(filepath.ToSlash(moduleYaMakePath))
	relModuleDir := strings.TrimPrefix(moduleDir, be.sourceRoot)
	relModuleDir = strings.TrimPrefix(relModuleDir, "/")

	be.recurser.ProcessFileImports(file, relModuleDir, be.sourceRoot)

	for _, module := range file.Modules {
		module.SourcePath = strings.TrimPrefix(module.SourcePath, be.sourceRoot)
		module.SourcePath = strings.TrimPrefix(module.SourcePath, "/")

		buildContext := be.createBuildContext()
		resolvedModule := ResolveConditionals(module, buildContext, NewMemoryVariableSet())
		ResolveWhenBlocks(resolvedModule, buildContext, NewMemoryVariableSet())

		if EvaluateBuildCondition(resolvedModule, buildContext, NewMemoryVariableSet()) {
			be.registry.Register(module.SourcePath, resolvedModule)
			be.processPeerDependencies(resolvedModule)
		}
	}
}

func (be *BuildEngine) findModuleYaMakeFile(modulePath string) string {
	fullPath := filepath.Join(be.sourceRoot, modulePath, "ya.make")

	if _, err := os.Stat(fullPath); err == nil {
		return fullPath
	}

	return ""
}

func BuildDependencyGraph(targetPath string, ctx ParseContext, sourceRoot string) (*Graph, error) {
	engine := NewBuildEngine(&ctx, sourceRoot)

	graph := engine.BuildDependencyGraph(targetPath)

	return graph, nil
}
