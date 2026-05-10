package main

import (
	"path/filepath"
	"strings"
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

	modules := gb.getOrderedModuleList(startModule, transitiveDeps, moduleUIDMap)

	for _, module := range modules {
		if module == nil {
			continue
		}

		nodes := gb.createExecutionNodes(module, moduleUIDMap)
		for _, node := range nodes {
			graph.AddNode(node)
		}
	}

	for _, input := range gb.collectInputs(startModule, transitiveDeps) {
		graph.AddInput(input)
	}

	graph.SetResult(NewUID([]byte(startModule.SourcePath)))

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

func (gb *GraphBuilder) getOrderedModuleList(startModule *Module, transitiveDeps map[string]struct{}, moduleUIDMap map[string]*Module) []*Module {
	modules := []*Module{startModule}

	for depUID := range transitiveDeps {
		if depModule := moduleUIDMap[depUID]; depModule != nil {
			modules = append(modules, depModule)
		}
	}

	return modules
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

func isCSource(src string) bool {
	return strings.HasSuffix(src, ".c")
}

func isCXXSource(src string) bool {
	ext := strings.ToLower(filepath.Ext(src))
	return ext == ".cc" || ext == ".cpp" || ext == ".cxx" || ext == ".c++"
}

func isCompilableCSource(src string) bool {
	return isCSource(src) || isCXXSource(src)
}

func (gb *GraphBuilder) sourceInput(module *Module, src string) string {
	return "$(SOURCE_ROOT)/" + filepath.Join(module.SourcePath, src)
}

func (gb *GraphBuilder) objectOutput(module *Module, src string) string {
	return "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, src+".o")
}

func (gb *GraphBuilder) moduleOutput(module *Module) string {
	baseName := filepath.Base(module.SourcePath)

	switch module.Type {
	case ModuleTypeProgram:
		return "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, baseName)
	case ModuleTypeLibrary:
		return "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "lib"+baseName+".a")
	case ModuleTypeGoLibrary:
		return "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "lib"+baseName+".a")
	default:
		return ""
	}
}

func (gb *GraphBuilder) createExecutionNodes(module *Module, moduleUIDMap map[string]*Module) []*GraphNode {
	var nodes []*GraphNode

	switch module.Type {
	case ModuleTypeProgram, ModuleTypeLibrary:
		compilableSources := gb.filterCompilableSources(module.Sources)

		if len(compilableSources) > 0 {
			for _, src := range compilableSources {
				compileNode := gb.createCompileNode(module, src)
				nodes = append(nodes, compileNode)
			}

			finalNode := gb.createFinalNode(module, compilableSources, moduleUIDMap)
			nodes = append(nodes, finalNode)
		} else {
			fallbackNode := gb.createFallbackNode(module, moduleUIDMap)
			if fallbackNode != nil {
				nodes = append(nodes, fallbackNode)
			}
		}
	case ModuleTypeGoLibrary:
		fallbackNode := gb.createFallbackNode(module, moduleUIDMap)
		if fallbackNode != nil {
			nodes = append(nodes, fallbackNode)
		}
	default:
		fallbackNode := gb.createFallbackNode(module, moduleUIDMap)
		if fallbackNode != nil {
			nodes = append(nodes, fallbackNode)
		}
	}

	return nodes
}

func (gb *GraphBuilder) filterCompilableSources(sources []string) []string {
	var result []string
	for _, src := range sources {
		if isCompilableCSource(src) {
			result = append(result, src)
		}
	}
	return result
}

// Reference CC node pattern from sg.json:
// - KV.p = "CC", KV.pc = "green"
// - 1 command: clang/clang++ with --target, --march, --sysroot flags
// - Output: $(BUILD_ROOT)/<module>/<source>.c.o or <source>.cpp.o
// - Input array includes all transitive header dependencies
// - Environment: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT
// See REFERENCE_NODE_CATALOG.md for full specification
func (gb *GraphBuilder) createCompileNode(module *Module, src string) *GraphNode {
	compileUIDKey := module.SourcePath + ":compile:" + src
	node := NewGraphNode(*gb.ctx)

	node.UID = NewUID([]byte(compileUIDKey))
	node.SelfUID = NewUID([]byte(compileUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(compileUIDKey + "_stats"))

	node.TargetProperties = TargetProperties{
		ModuleDir: module.SourcePath,
	}

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateCompileCommand(module, src),
		},
	}

	node.Inputs = []string{gb.sourceInput(module, src)}
	node.Outputs = []string{gb.objectOutput(module, src)}
	node.Deps = []string{}

	node.KV = map[string]string{
		"uid": NewUID([]byte(compileUIDKey + "_kv")),
		"p":   "CC",
		"pc":  "green",
	}

	return node
}

func (gb *GraphBuilder) generateCompileCommand(module *Module, src string) []string {
	compiler := "clang"
	if isCXXSource(src) {
		compiler = "clang++"
	}

	return []string{
		compiler,
		"-c",
		gb.sourceInput(module, src),
		"-o",
		gb.objectOutput(module, src),
		"-I",
		"$(SOURCE_ROOT)/" + module.SourcePath,
	}
}

func (gb *GraphBuilder) createFinalNode(module *Module, compilableSources []string, moduleUIDMap map[string]*Module) *GraphNode {
	node := NewGraphNode(*gb.ctx)

	node.UID = NewUID([]byte(module.SourcePath))
	node.SelfUID = NewUID([]byte(module.SourcePath + "_self"))
	node.StatsUID = NewUID([]byte(module.SourcePath + "_stats"))

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: gb.determineModuleLanguage(module),
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	moduleDeps := gb.collectModuleDependencies(module)
	compileDeps := gb.getCompileDepUIDs(module, compilableSources)

	allDeps := append(compileDeps, moduleDeps...)
	node.Deps = allDeps

	var inputs []string
	for _, src := range module.Sources {
		if !isCompilableCSource(src) {
			inputs = append(inputs, gb.sourceInput(module, src))
		}
	}
	node.Inputs = inputs

	outputPath := gb.moduleOutput(module)
	if outputPath != "" {
		node.Outputs = []string{outputPath}
	}

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateFinalCommand(module, compilableSources, moduleUIDMap),
		},
	}

	node.KV = map[string]string{
		"uid": NewUID([]byte(module.SourcePath + "_kv")),
	}

	// Reference AR archive node pattern from sg.json:
	// - KV.p = "AR", KV.pc = "light-red", KV.show_out = "yes"
	// - 1 command: python3 link_lib.py with llvm-ar invocation
	// - Output: $(BUILD_ROOT)/<module>/lib<module>.a
	// - Inputs: All .o files from compilation + generated sources
	// - Deps: All compile node UIDs for the module
	// - Environment: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT
	// See REFERENCE_NODE_CATALOG.md for full specification
	switch module.Type {
	case ModuleTypeProgram:
		node.KV["p"] = "LD"
		node.KV["pc"] = "light-blue"
		node.KV["show_out"] = "yes"
	case ModuleTypeLibrary:
		node.KV["p"] = "AR"
		node.KV["pc"] = "light-red"
		node.KV["show_out"] = "yes"
	}

	return node
}

func (gb *GraphBuilder) getCompileDepUIDs(module *Module, compilableSources []string) []string {
	var deps []string
	for _, src := range compilableSources {
		compileUIDKey := module.SourcePath + ":compile:" + src
		compileUID := NewUID([]byte(compileUIDKey))
		deps = append(deps, compileUID)
	}
	return deps
}

func (gb *GraphBuilder) generateFinalCommand(module *Module, compilableSources []string, moduleUIDMap map[string]*Module) []string {
	outputPath := gb.moduleOutput(module)
	if outputPath == "" {
		return []string{}
	}

	var args []string
	var objectOutputs []string

	for _, src := range compilableSources {
		objectOutputs = append(objectOutputs, gb.objectOutput(module, src))
	}

	// Reference LD link node pattern from sg.json:
	// - KV.p = "LD", KV.pc = "light-blue", KV.show_out = "yes"
	// - 4 commands in sequence:
	//   1. python3 vcs_info.py (generate version.c)
	//   2. clang (compile version.c to .o)
	//   3. python3 link_exe.py (link executable)
	//   4. python3 fs_tools.py (copy/install)
	// - Cmds should be implemented to match reference structure
	// See REFERENCE_NODE_CATALOG.md for full specification
	switch module.Type {
	case ModuleTypeProgram:
		args = append(args, "clang++", "-o", outputPath)
		args = append(args, objectOutputs...)

		depOutputs := gb.getDependencyOutputs(module, moduleUIDMap)
		for _, depOutput := range depOutputs {
			if depOutput != "" {
				args = append(args, depOutput)
			}
		}
	case ModuleTypeLibrary:
		args = append(args, "ar", "rcs", outputPath)
		args = append(args, objectOutputs...)
	}

	return args
}

func (gb *GraphBuilder) getDependencyOutputs(module *Module, moduleUIDMap map[string]*Module) []string {
	var outputs []string

	for _, depPath := range module.Dependencies {
		depModule := gb.resolveDependencyModule(depPath, moduleUIDMap)
		if depModule != nil {
			outputPath := gb.moduleOutput(depModule)
			if outputPath != "" {
				outputs = append(outputs, outputPath)
			}
		}
	}

	return outputs
}

func (gb *GraphBuilder) resolveDependencyModule(depPath string, moduleUIDMap map[string]*Module) *Module {
	allModules := gb.registry.AllModules()

	for _, module := range allModules {
		if gb.normalizeDepPath(depPath) == module.SourcePath {
			return module
		}
	}

	return nil
}

func (gb *GraphBuilder) normalizeDepPath(depPath string) string {
	depPath = filepath.Clean(depPath)
	if strings.HasPrefix(depPath, "./") {
		depPath = depPath[2:]
	}
	return depPath
}

func (gb *GraphBuilder) createFallbackNode(module *Module, moduleUIDMap map[string]*Module) *GraphNode {
	if len(module.Sources) == 0 && len(module.Dependencies) == 0 {
		return nil
	}

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

	node.KV = copyStringMap(module.Properties)
	if node.KV == nil {
		node.KV = map[string]string{}
	}
	node.KV["uid"] = NewUID([]byte(module.SourcePath + "_kv"))

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
