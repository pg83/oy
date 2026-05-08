package main

import (
	"os"
	"path/filepath"
	"strings"
)

type BuildEngine struct {
	registry   *ModuleRegistry
	recurser   *RecurseProcessor
	ctx        *ParseContext
	sourceRoot string
}

func NewBuildEngine(ctx *ParseContext, sourceRoot string) *BuildEngine {
	registry := NewModuleRegistry()
	recurser := NewRecurseProcessor(registry, ctx)

	return &BuildEngine{
		registry:   registry,
		recurser:   recurser,
		ctx:        ctx,
		sourceRoot: sourceRoot,
	}
}

func (be *BuildEngine) BuildDependencyGraph(targetPath string) *Graph {
	absTargetPath := be.resolveTargetPath(targetPath)

	parser := NewParser(be.ctx)
	file := Throw2(parser.ParseFile(absTargetPath))

	targetDir := filepath.Dir(filepath.ToSlash(absTargetPath))
	be.recurser.ProcessFileImports(file, targetDir, be.sourceRoot)

	if len(file.Modules) == 0 {
		ThrowFmt("no modules found in ya.make file: %s", absTargetPath)
	}

	mainModule := file.Modules[0]
	mainModule.SourcePath = filepath.Join(filepath.Dir(absTargetPath), mainModule.SourcePath)

	mainModule.SourcePath = filepath.Clean(mainModule.SourcePath)
	mainModule.SourcePath = strings.TrimPrefix(mainModule.SourcePath, be.sourceRoot)
	mainModule.SourcePath = strings.TrimPrefix(mainModule.SourcePath, "/")

	be.recurser.ProcessRecurse(mainModule, be.sourceRoot)

	buildContext := be.createBuildContext()

	resolvedModule := ResolveConditionals(mainModule, buildContext, NewMemoryVariableSet())

	if !EvaluateBuildCondition(resolvedModule, buildContext, NewMemoryVariableSet()) {
		ThrowFmt("module excluded by BUILD_ONLY_IF condition: %s", mainModule.SourcePath)
	}

	be.registry.Register(mainModule.SourcePath, resolvedModule)

	graphBuilder := NewGraphBuilder(be.registry, be.ctx)
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

func BuildDependencyGraph(targetPath string, ctx ParseContext, sourceRoot string) (*Graph, error) {
	engine := NewBuildEngine(&ctx, sourceRoot)

	graph := engine.BuildDependencyGraph(targetPath)

	return graph, nil
}
