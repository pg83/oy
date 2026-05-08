package main

import (
	"path/filepath"
)

type GraphBuilder struct {
	registry *ModuleRegistry
	ctx      *ParseContext
}

func NewGraphBuilder(registry *ModuleRegistry, ctx *ParseContext) *GraphBuilder {
	return &GraphBuilder{
		registry: registry,
		ctx:      ctx,
	}
}

func (gb *GraphBuilder) BuildGraphFromModules(startModule *Module) *Graph {
	if startModule == nil {
		ThrowFmt("cannot build graph from nil module")
	}

	graph := NewGraph(*gb.ctx)

	transitiveResolver := NewTransitiveResolver(gb.registry)
	transitiveDeps := transitiveResolver.ComputeTransitiveDeps(startModule)

	moduleUIDMap := gb.buildModuleUIDMap(startModule, transitiveDeps)

	for uid := range transitiveDeps {
		module := moduleUIDMap[uid]
		if module == nil {
			continue
		}

		node := gb.createGraphNode(module, transitiveDeps)
		graph.AddNode(node)
	}

	for _, input := range gb.collectInputs(startModule, transitiveDeps) {
		graph.AddInput(input, "")
	}

	return graph
}

func (gb *GraphBuilder) buildModuleUIDMap(startModule *Module, transitiveDeps map[string]struct{}) map[string]*Module {
	moduleMap := make(map[string]*Module)

	startUID := NewUID([]byte(startModule.SourcePath))
	moduleMap[startUID] = startModule

	for depUID := range transitiveDeps {
		depModule := gb.findModuleByUID(depUID)
		if depModule != nil {
			moduleMap[depUID] = depModule
		}
	}

	return moduleMap
}

func (gb *GraphBuilder) findModuleByUID(uid string) *Module {
	allModules := gb.registry.AllModules()

	for _, module := range allModules {
		moduleUID := NewUID([]byte(module.SourcePath))
		if moduleUID == uid {
			return module
		}
	}

	return nil
}

func (gb *GraphBuilder) createGraphNode(module *Module, transitiveDeps map[string]struct{}) *GraphNode {
	node := NewGraphNode(*gb.ctx)

	node.UID = NewUID([]byte(module.SourcePath))
	node.SelfUID = NewUID([]byte(module.SourcePath + "_self"))
	node.StatsUID = NewUID([]byte(module.SourcePath + "_stats"))

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: gb.determineModuleLanguage(module),
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	node.Cmds = gb.generateCommands(module)

	node.Inputs = gb.collectModuleInputs(module)

	node.Outputs = gb.collectModuleOutputs(module)

	node.Deps = gb.collectModuleDependencies(module)

	node.KV = module.Properties

	kvString := NewUID([]byte(module.SourcePath + "_kv"))
	node.KV["uid"] = kvString

	return node
}

func (gb *GraphBuilder) determineModuleLanguage(module *Module) string {
	if module == nil {
		return ""
	}

	if module.Type == ModuleTypeGoLibrary {
		return "go"
	}

	if module.Type == ModuleTypePyLibrary || module.Type == ModuleTypePy23Library {
		return "python"
	}

	for _, src := range module.Sources {
		ext := filepath.Ext(src)
		switch ext {
		case ".go":
			return "go"
		case ".py":
			return "python"
		case ".cpp", ".cc", ".cxx":
			return "cpp"
		case ".c":
			return "c"
		}
	}

	return ""
}

func (gb *GraphBuilder) mapModuleTypeToString(moduleType ModuleType) string {
	switch moduleType {
	case ModuleTypeProgram:
		return "bin"
	case ModuleTypeLibrary:
		return "lib"
	case ModuleTypeGoLibrary:
		return "go_lib"
	case ModuleTypeDLL:
		return "dll"
	case ModuleTypePyLibrary, ModuleTypePy23Library:
		return "py_lib"
	default:
		return "unknown"
	}
}

func (gb *GraphBuilder) generateCommands(module *Module) []Command {
	if module == nil {
		return []Command{}
	}

	commands := make([]Command, 0)

	switch module.Type {
	case ModuleTypeProgram:
		commands = gb.generateProgramCommands(module)
	case ModuleTypeLibrary:
		commands = gb.generateLibraryCommands(module)
	case ModuleTypeGoLibrary:
		commands = gb.generateGoLibraryCommands(module)
	}

	return commands
}

func (gb *GraphBuilder) generateProgramCommands(module *Module) []Command {
	commands := make([]Command, 0)

	if len(module.Sources) == 0 {
		return commands
	}

	for _, src := range module.Sources {
		compileCmd := Command{
			CmdArgs: []string{
				"clang++",
				"-c",
				src,
				"-o",
				src + ".o",
				"-I",
				filepath.Join("$(SOURCE_ROOT)", module.SourcePath),
			},
		}
		commands = append(commands, compileCmd)
	}

	linkCmd := Command{
		CmdArgs: []string{
			"clang++",
			"-o",
			filepath.Join("$(BUILD_ROOT)", module.SourcePath, filepath.Base(module.SourcePath)),
		},
	}

	for _, src := range module.Sources {
		linkCmd.CmdArgs = append(linkCmd.CmdArgs, src+".o")
	}

	commands = append(commands, linkCmd)

	return commands
}

func (gb *GraphBuilder) generateLibraryCommands(module *Module) []Command {
	commands := make([]Command, 0)

	if len(module.Sources) == 0 {
		return commands
	}

	for _, src := range module.Sources {
		compileCmd := Command{
			CmdArgs: []string{
				"clang++",
				"-c",
				src,
				"-o",
				src + ".o",
				"-I",
				filepath.Join("$(SOURCE_ROOT)", module.SourcePath),
			},
		}
		commands = append(commands, compileCmd)
	}

	arCmd := Command{
		CmdArgs: []string{
			"ar",
			"rcs",
			filepath.Join("$(BUILD_ROOT)", module.SourcePath, "lib"+filepath.Base(module.SourcePath)+".a"),
		},
	}

	for _, src := range module.Sources {
		arCmd.CmdArgs = append(arCmd.CmdArgs, src+".o")
	}

	commands = append(commands, arCmd)

	return commands
}

func (gb *GraphBuilder) generateGoLibraryCommands(module *Module) []Command {
	commands := make([]Command, 0)

	if len(module.Sources) == 0 {
		return commands
	}

	compileCmd := Command{
		CmdArgs: []string{
			"go",
			"build",
			"-buildmode=archive",
			"-o",
			filepath.Join("$(BUILD_ROOT)", module.SourcePath, "lib"+filepath.Base(module.SourcePath)+".a"),
		},
	}

	for _, src := range module.Sources {
		compileCmd.CmdArgs = append(compileCmd.CmdArgs, src)
	}

	commands = append(commands, compileCmd)

	return commands
}

func (gb *GraphBuilder) collectModuleInputs(module *Module) []string {
	inputs := make([]string, 0)

	for _, src := range module.Sources {
		inputPath := filepath.Join("$(SOURCE_ROOT)", module.SourcePath, src)
		inputs = append(inputs, inputPath)
	}

	return inputs
}

func (gb *GraphBuilder) collectModuleOutputs(module *Module) []string {
	outputs := make([]string, 0)

	if module.Type == ModuleTypeProgram {
		outputPath := filepath.Join("$(BUILD_ROOT)", module.SourcePath, filepath.Base(module.SourcePath))
		outputs = append(outputs, outputPath)
	} else if module.Type == ModuleTypeLibrary {
		outputPath := filepath.Join("$(BUILD_ROOT)", module.SourcePath, "lib"+filepath.Base(module.SourcePath)+".a")
		outputs = append(outputs, outputPath)
	} else if module.Type == ModuleTypeGoLibrary {
		outputPath := filepath.Join("$(BUILD_ROOT)", module.SourcePath, "lib"+filepath.Base(module.SourcePath)+".a")
		outputs = append(outputs, outputPath)
	}

	return outputs
}

func (gb *GraphBuilder) collectModuleDependencies(module *Module) []string {
	resolver := NewResolver(gb.registry, gb.ctx)
	resolvedDeps := Throw2(resolver.ResolveDependencies(module))

	return resolvedDeps
}

func (gb *GraphBuilder) collectInputs(startModule *Module, transitiveDeps map[string]struct{}) []string {
	allInputs := make([]string, 0)
	seen := make(map[string]bool)

	moduleUIDMap := gb.buildModuleUIDMap(startModule, transitiveDeps)

	for _, module := range moduleUIDMap {
		for _, src := range module.Sources {
			inputPath := filepath.Join("$(SOURCE_ROOT)", module.SourcePath, src)
			if !seen[inputPath] {
				seen[inputPath] = true
				allInputs = append(allInputs, inputPath)
			}
		}
	}

	return allInputs
}
