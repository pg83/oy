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
	diag        *TraversalLogger
	config      *ConfigParser
}

func NewBuildEngine(ctx *ParseContext, sourceRoot string) *BuildEngine {
	registry := NewModuleRegistry()
	recurser := NewRecurseProcessor(registry, ctx)

	configPath := sourceRoot + "/build/ymake.core.conf"
	var config *ConfigParser
	if _, err := os.Stat(configPath); err == nil {
		config = Throw2(ParseYmakeCoreConfig(configPath))
	}

	return &BuildEngine{
		registry:    registry,
		recurser:    recurser,
		ctx:         ctx,
		sourceRoot:  sourceRoot,
		visitedDeps: make(map[string]bool),
		config:      config,
	}
}

func NewBuildEngineWithDiag(ctx *ParseContext, sourceRoot string, diag *TraversalLogger) *BuildEngine {
	registry := NewModuleRegistry()
	recurser := NewRecurseProcessor(registry, ctx)

	configPath := sourceRoot + "/build/ymake.core.conf"
	var config *ConfigParser
	if _, err := os.Stat(configPath); err == nil {
		config = Throw2(ParseYmakeCoreConfig(configPath))
	}

	return &BuildEngine{
		registry:    registry,
		recurser:    recurser,
		ctx:         ctx,
		sourceRoot:  sourceRoot,
		visitedDeps: make(map[string]bool),
		diag:        diag,
		config:      config,
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
	vars := NewMemoryVariableSet()
	bcVars := NewBuildContextVariableSet(buildContext, vars)

	resolvedModule := ResolveConditionals(mainModule, buildContext, bcVars)
	ResolveWhenBlocks(resolvedModule, buildContext, bcVars)

	// Process INCLUDE directives for the main module
	for _, include := range resolvedModule.IncludeDirectives {
		be.processInclude(include, resolvedModule, filepath.Dir(filepath.ToSlash(absTargetPath)), buildContext, bcVars)
	}

	if !EvaluateBuildCondition(resolvedModule, buildContext, bcVars) {
		ThrowFmt("module excluded by BUILD_ONLY_IF condition: %s", mainModule.SourcePath)
	}

	be.registry.Register(mainModule.SourcePath, resolvedModule)

	be.processPeerDependencies(resolvedModule)

	var graphBuilder *GraphBuilder
	if be.diag != nil && be.diag.IsToolDiagEnabled() {
		graphBuilder = NewGraphBuilderWithDiag(be.registry, be.ctx, be.sourceRoot, be.diag)
	} else {
		graphBuilder = NewGraphBuilderWithSourceRoot(be.registry, be.ctx, be.sourceRoot)
	}
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

	if be.ctx.ArchString != "" {
		ctx.ArchString = be.ctx.ArchString
		switch be.ctx.ArchString {
		case "x86_64":
			ctx.Arch = Arch64
		case "x86_32":
			ctx.Arch = Arch32
		case "aarch64":
			ctx.Arch = Arch64
		}
	}

	for flag, value := range be.ctx.BuildFlags {
		ctx.Flags[flag] = (value == "true" || value == "1" || value == "yes")
	}

	ctx.Musl = be.ctx.Musl

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

		if module.Type == ModuleTypeProgram && be.shouldInjectConfigPEERDIRs() {
			configVars := be.buildConfigVariables()
			injected := be.config.ResolveConfigPEERDIRS(configVars)
			for _, peerdir := range injected {
				module.Dependencies = append(module.Dependencies, peerdir)
			}
		}
	}

	for _, dep := range module.Dependencies {
		be.loadModuleAndDependencies(dep, module.SourcePath)
	}

	be.logToolModuleDiscovery(module)
}

func (be *BuildEngine) logToolModuleDiscovery(module *Module) {
	if be.diag == nil || !be.diag.IsToolDiagEnabled() {
		return
	}

	for _, src := range module.Sources {
		if strings.HasSuffix(src, ".rl6") {
			be.diag.LogToolModuleLoad("contrib/tools/ragel6", module.SourcePath, true, "R6", src)
		}
	}
}

func (be *BuildEngine) loadModuleAndDependencies(depPath string, parentPath string) {
	normDep := NormalizedPath(depPath)

	if be.registry.Has(depPath) {
		return
	}

	moduleYaMakePath := be.findModuleYaMakeFile(depPath)
	if moduleYaMakePath == "" {
		if be.diag != nil {
			be.diag.LogModuleLoad(normDep, parentPath, false, "file not found")
		}
		return
	}

	var file *File
	defer func() {
		if r := recover(); r != nil {
			if be.diag != nil {
				be.diag.LogModuleLoad(normDep, parentPath, false, "parse error")
			}
			return
		}
	}()
	file = ParseYaMakeFile(moduleYaMakePath)

	if file == nil {
		if be.diag != nil {
			be.diag.LogModuleLoad(normDep, parentPath, false, "no modules in file")
		}
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
		vars := NewMemoryVariableSet()
		bcVars := NewBuildContextVariableSet(buildContext, vars)

		resolvedModule := ResolveConditionals(module, buildContext, bcVars)
		ResolveWhenBlocks(resolvedModule, buildContext, bcVars)

		for _, include := range resolvedModule.IncludeDirectives {
			be.processInclude(include, resolvedModule, moduleDir, buildContext, bcVars)
		}

		normModulePath := NormalizedPath(module.SourcePath)

		if EvaluateBuildCondition(resolvedModule, buildContext, bcVars) {
			be.registry.Register(normModulePath, resolvedModule)
			if be.diag != nil {
				be.diag.LogModuleLoad(normModulePath, parentPath, true, "")
			}
			be.diag.LogPeerdirResolution(parentPath, normDep, true, normModulePath)
			be.processPeerDependencies(resolvedModule)
		} else {
			if be.diag != nil {
				be.diag.LogModuleLoad(normModulePath, parentPath, false, "BUILD_ONLY_IF condition failed")
			}
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

func (be *BuildEngine) processInclude(include *IncludeDirective, module *Module, baseDir string, buildContext *BuildContext, vars VariableSet) {
	resolvedPath := be.resolveIncludePath(include.FilePath, baseDir)
	content := Throw2(os.ReadFile(resolvedPath))

	includeFile := ParseYaMakeString(string(content), resolvedPath)

	if len(includeFile.Modules) > 0 {
		for _, inclModule := range includeFile.Modules {
			resolvedInclModule := ResolveConditionals(inclModule, buildContext, vars)
			ResolveWhenBlocks(resolvedInclModule, buildContext, vars)

			if !EvaluateBuildCondition(resolvedInclModule, buildContext, vars) {
				continue
			}

			MergeModule(module, resolvedInclModule)
		}
	} else {
		inclModule := ParseModuleFragment(string(content), resolvedPath)
		resolvedInclModule := ResolveConditionals(inclModule, buildContext, vars)
		ResolveWhenBlocks(resolvedInclModule, buildContext, vars)

		if !EvaluateBuildCondition(resolvedInclModule, buildContext, vars) {
			return
		}

		MergeModule(module, resolvedInclModule)
	}
}

func (be *BuildEngine) resolveIncludePath(includePath, containingDir string) string {
	if filepath.IsAbs(includePath) {
		return includePath
	}
	return filepath.Join(containingDir, includePath)
}

func (be *BuildEngine) buildConfigVariables() map[string]string {
	vars := make(map[string]string)
	vars["MUSL"] = "no"
	if be.ctx.Musl {
		vars["MUSL"] = "yes"
	}
	vars["OS_LINUX"] = "no"
	if be.ctx.Platform == "linux" || be.ctx.Platform == "" {
		vars["OS_LINUX"] = "yes"
	}
	vars["OS_WINDOWS"] = "no"
	if be.ctx.Platform == "windows" {
		vars["OS_WINDOWS"] = "yes"
	}
	vars["ARCH_X86_64"] = "no"
	if be.ctx.ArchString == "x86_64" {
		vars["ARCH_X86_64"] = "yes"
	}
	vars["ARCH_AARCH64"] = "no"
	if be.ctx.ArchString == "aarch64" {
		vars["ARCH_AARCH64"] = "yes"
	}
	return vars
}

func (be *BuildEngine) shouldInjectConfigPEERDIRs() bool {
	if be.config == nil {
		return false
	}
	return be.ctx.Musl
}

func BuildDependencyGraph(targetPath string, ctx ParseContext, sourceRoot string, diag *TraversalLogger) (*Graph, error) {
	var engine *BuildEngine
	if diag != nil {
		engine = NewBuildEngineWithDiag(&ctx, sourceRoot, diag)
	} else {
		engine = NewBuildEngine(&ctx, sourceRoot)
	}

	graph := engine.BuildDependencyGraph(targetPath)

	return graph, nil
}
