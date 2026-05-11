package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GraphBuilder struct {
	registry   *ModuleRegistry
	ctx        *ParseContext
	sourceRoot string
	config     *BuildConfig
	diag       *TraversalLogger
}

func NewGraphBuilder(registry *ModuleRegistry, ctx *ParseContext) *GraphBuilder {
	return &GraphBuilder{
		registry:   registry,
		ctx:        ctx,
		sourceRoot: "",
		config:     NewBuildConfig(ctx),
	}
}

func NewGraphBuilderWithSourceRoot(registry *ModuleRegistry, ctx *ParseContext, sourceRoot string) *GraphBuilder {
	return &GraphBuilder{
		registry:   registry,
		ctx:        ctx,
		sourceRoot: sourceRoot,
		config:     NewBuildConfig(ctx),
	}
}

func NewGraphBuilderWithDiag(registry *ModuleRegistry, ctx *ParseContext, sourceRoot string, diag *TraversalLogger) *GraphBuilder {
	return &GraphBuilder{
		registry:   registry,
		ctx:        ctx,
		sourceRoot: sourceRoot,
		config:     NewBuildConfig(ctx),
		diag:       diag,
	}
}

type PlatformArch string

const (
	PlatformAARCH64 PlatformArch = "default-linux-aarch64"
	PlatformX86_64  PlatformArch = "default-linux-x86_64"
)

type PlatformAwareContext struct {
	ctx  *ParseContext
	arch PlatformArch
}

func NewPlatformContexts(ctx *ParseContext, module *Module) []PlatformAwareContext {
	if module != nil && module.NoPlatform {
		return []PlatformAwareContext{
			{ctx: ctx, arch: PlatformX86_64},
		}
	}

	return []PlatformAwareContext{
		{ctx: ctx, arch: PlatformAARCH64},
		{ctx: ctx, arch: PlatformX86_64},
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

		gb.injectAllocatorDependencies(module)

		nodes := gb.createExecutionNodes(module, moduleUIDMap)
		for _, node := range nodes {
			graph.AddNode(node)
		}
	}

	for _, input := range gb.collectInputs(startModule, transitiveDeps) {
		graph.AddInput(input)
	}

	if startModule.Type == ModuleTypeProgram {
		resultUIDs := []string{}
		targetArches := []PlatformArch{PlatformX86_64}
		if !startModule.NoPlatform {
			targetArches = []PlatformArch{PlatformAARCH64, PlatformX86_64}
		}
		for _, arch := range targetArches {
			ldUIDKey := startModule.SourcePath + ":LD:" + string(arch)
			resultUIDs = append(resultUIDs, NewUID([]byte(ldUIDKey)))
		}
		graph.SetResult(resultUIDs...)
	} else {
		graph.SetResult(NewUID([]byte(startModule.SourcePath)))
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

func isRagel6Source(src string) bool {
	return strings.HasSuffix(src, ".rl6")
}

func isCopyRequiredSource(src string) bool {
	return strings.Contains(src, "musl.py") || strings.Contains(src, ".pyplugin")
}

func isJSGenSource(modulePath string) bool {
	jsGenModules := map[string]bool{
		"util/charset":         true,
		"util":                 true,
		"contrib/tools/ragel6": true,
	}
	return jsGenModules[modulePath]
}

func isJSOutputSource(src string) bool {
	jsOutputs := map[string]bool{
		"all_charset.cpp": true,
		"main.cpp":        true,
	}
	return jsOutputs[strings.TrimSuffix(src, filepath.Ext(src))]
}

func (gb *GraphBuilder) isJoinSrcsOutput(module *Module, src string) bool {
	for _, jsd := range module.JoinSrcsDirectives {
		if jsd.OutputFile == src {
			return true
		}
	}
	return false
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

	platformContexts := NewPlatformContexts(gb.ctx, module)

	if module.NoPlatform && len(platformContexts) == 1 {
		fmt.Printf("NO_PLATFORM module %s: building for target %s only\n",
			module.SourcePath, platformContexts[0].arch)
	}

	r6NodesCreated := make(map[string]bool)
	for _, platformCtx := range platformContexts {

		if !gb.isTargetPlatform(platformCtx) {
			phaseNodes := gb.createPlatformExecutionNodes(module, moduleUIDMap, platformCtx)
			nodes = append(nodes, phaseNodes...)
			continue
		}

		for _, src := range module.Sources {
			if isRagel6Source(src) && !r6NodesCreated[src] {
				r6Node := gb.createR6Node(module, src, platformCtx)
				if r6Node != nil {
					nodes = append(nodes, r6Node)
				}
				r6NodesCreated[src] = true
			}
		}

		phaseNodes := gb.createPlatformExecutionNodes(module, moduleUIDMap, platformCtx)
		nodes = append(nodes, phaseNodes...)
	}

	return nodes
}

func (gb *GraphBuilder) isTargetPlatform(platformCtx PlatformAwareContext) bool {
	if gb.ctx.TargetPlatform != "" {
		if strings.Contains(gb.ctx.TargetPlatform, "aarch64") {
			return platformCtx.arch == PlatformAARCH64
		}
		if strings.Contains(gb.ctx.TargetPlatform, "x86_64") {
			return platformCtx.arch == PlatformX86_64
		}
	}

	return platformCtx.arch == PlatformX86_64
}

func (gb *GraphBuilder) createPlatformExecutionNodes(
	module *Module,
	moduleUIDMap map[string]*Module,
	platformCtx PlatformAwareContext,
) []*GraphNode {
	var nodes []*GraphNode

	compileNodes, objectOutputs := gb.createCompilePhaseNodes(module, platformCtx)
	nodes = append(nodes, compileNodes...)

	if len(objectOutputs) > 0 && (module.Type == ModuleTypeProgram || module.Type == ModuleTypeLibrary) {
		finalNode := gb.createFinalPhaseNode(module, objectOutputs, moduleUIDMap, platformCtx)
		if finalNode != nil {
			if gb.diag != nil && gb.diag.IsToolDiagEnabled() {
				gb.diag.LogNodeCreation(module.SourcePath, string(platformCtx.arch),
					finalNode.KV["p"], "", finalNode.UID)
			}
			nodes = append(nodes, finalNode)
		}
	}

	return nodes
}

func (gb *GraphBuilder) createCompilePhaseNodes(
	module *Module,
	platformCtx PlatformAwareContext,
) ([]*GraphNode, []string) {
	var nodes []*GraphNode
	var objectOutputs []string

	isTarget := gb.isTargetPlatform(platformCtx)

	for _, jsd := range module.JoinSrcsDirectives {
		if isTarget {
			jsNode := gb.createJoinSrcsNode(module, jsd, platformCtx)
			gb.logNodeCreation(module, platformCtx.arch, "JS", jsd.OutputFile, jsNode.UID)
			nodes = append(nodes, jsNode)
		}
		objOutput := gb.platformObjectOutput(module, jsd.OutputFile, platformCtx.arch)
		objectOutputs = append(objectOutputs, objOutput)
	}

	for _, src := range module.Sources {
		var compileSrc string = src

		if isRagel6Source(src) {
			compileSrc = "_" + src + ".cpp"
		}

		nodeType := gb.determineCompileNodeTypeWithArch(module, src, platformCtx.arch)

		if nodeType == "" {
			continue
		}

		switch nodeType {
		case "CC":
			ccNode := gb.createCCNode(module, compileSrc, platformCtx)
			gb.logNodeCreation(module, platformCtx.arch, "CC", src, ccNode.UID)
			nodes = append(nodes, ccNode)
			objOutput := gb.platformObjectOutput(module, compileSrc, platformCtx.arch)
			objectOutputs = append(objectOutputs, objOutput)
		case "AS":
			asNode := gb.createASNode(module, src, platformCtx)
			gb.logNodeCreation(module, platformCtx.arch, "AS", src, asNode.UID)
			nodes = append(nodes, asNode)
			objOutput := gb.platformObjectOutput(module, src, platformCtx.arch)
			objectOutputs = append(objectOutputs, objOutput)
		case "JS":
			if isTarget {
				jsNode := gb.createJSNode(module, src, platformCtx)
				gb.logNodeCreation(module, platformCtx.arch, "JS", src, jsNode.UID)
				nodes = append(nodes, jsNode)
				objOutput := gb.platformObjectOutput(module, src, platformCtx.arch)
				objectOutputs = append(objectOutputs, objOutput)
			}
		case "CP":
			if isTarget {
				cpNode := gb.createCPNode(module, src, platformCtx)
				gb.logNodeCreation(module, platformCtx.arch, "CP", src, cpNode.UID)
				nodes = append(nodes, cpNode)
			}
		}
	}

	return nodes, objectOutputs
}

func (gb *GraphBuilder) logNodeCreation(module *Module, arch PlatformArch, nodeType, src, uid string) {
	if gb.diag == nil || !gb.diag.IsToolDiagEnabled() {
		return
	}
	gb.diag.LogNodeCreation(module.SourcePath, string(arch), nodeType, src, uid)
}

func (gb *GraphBuilder) determineCompileNodeType(module *Module, src string) string {
	if gb.isJoinSrcsOutput(module, src) {
		return "JS"
	}
	if isRagel6Source(src) {
		return "R6"
	}
	if isCopyRequiredSource(src) {
		return "CP"
	}
	if isJSGenSource(module.SourcePath) && isJSOutputSource(src) {
		return "JS"
	}
	if strings.HasSuffix(strings.ToLower(src), ".s") || strings.HasSuffix(strings.ToLower(src), ".S") || strings.HasSuffix(strings.ToLower(src), ".asm") {
		return "AS"
	}
	if isCompilableCSource(src) {
		return "CC"
	}
	return ""
}

func (gb *GraphBuilder) determineCompileNodeTypeWithArch(module *Module, src string, arch PlatformArch) string {
	nodeType := gb.determineCompileNodeType(module, src)

	if nodeType == "AS" {
		if !isArchSpecificASM(src, arch) {
			return ""
		}
	}

	return nodeType
}

func isArchSpecificASM(src string, arch PlatformArch) bool {
	srcLower := strings.ToLower(src)

	isASMFile := strings.HasSuffix(srcLower, ".s") || strings.HasSuffix(srcLower, ".S") || strings.HasSuffix(srcLower, ".asm")
	if !isASMFile {
		return true
	}

	containsAarch64 := strings.Contains(srcLower, "aarch64/") || strings.Contains(srcLower, "arm64/")
	containsX86_64 := strings.Contains(srcLower, "x86_64/") || strings.Contains(srcLower, "x8664") || strings.Contains(srcLower, "x86-64")
	containsX86 := strings.Contains(srcLower, "/x86/") || strings.Contains(srcLower, "/i386/")

	if containsAarch64 && (containsX86_64 || containsX86) {
		return true
	}

	if containsAarch64 {
		return arch == PlatformAARCH64
	}

	if containsX86_64 || containsX86 {
		return arch == PlatformX86_64
	}

	if strings.HasSuffix(srcLower, "64.asm") || strings.HasSuffix(srcLower, "64.S") {
		return arch == PlatformX86_64
	}

	if strings.HasSuffix(srcLower, "32.asm") || strings.HasSuffix(srcLower, "32.S") {
		return arch == PlatformX86_64
	}

	return true
}

func (gb *GraphBuilder) platformObjectOutput(module *Module, src string, arch PlatformArch) string {
	return "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, src+".o")
}

func (gb *GraphBuilder) createCCNode(
	module *Module,
	src string,
	platformCtx PlatformAwareContext,
) *GraphNode {
	compileUIDKey := fmt.Sprintf("%s:CC:%s:%s", module.SourcePath, platformCtx.arch, src)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(compileUIDKey))
	node.SelfUID = NewUID([]byte(compileUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(compileUIDKey + "_stats"))

	node.Platform = string(platformCtx.arch)

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: gb.determineCCSourceLanguage(src),
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateCCCommand(module, src, platformCtx.arch),
			Env:     gb.generateCCEnvironment(module, platformCtx.arch),
		},
	}

	node.Inputs = []string{gb.sourceInput(module, src)}
	node.Outputs = []string{gb.platformObjectOutput(module, src, platformCtx.arch)}
	node.Deps = []string{}

	node.KV = map[string]string{
		"uid": NewUID([]byte(compileUIDKey + "_kv")),
		"p":   "CC",
		"pc":  "green",
	}

	return node
}

func (gb *GraphBuilder) determineCCSourceLanguage(src string) string {
	if isCSource(src) {
		return "c"
	}
	return "cpp"
}

func (gb *GraphBuilder) generateCCCommand(
	module *Module,
	src string,
	arch PlatformArch,
) []string {
	compiler := "clang"
	if isCXXSource(src) {
		compiler = "clang++"
	}

	target := "aarch64-linux-gnu"
	march := "armv8-a"
	if arch == PlatformX86_64 {
		target = "x86_64-linux-gnu"
		march = "x86-64"
	}

	return []string{
		compiler,
		"--target=" + target,
		"-march=" + march,
		"--sysroot=/nowhere",
		"-B$(OS_SDK_ROOT-sbr:309054781)/usr/bin",
		"-c",
		gb.sourceInput(module, src),
		"-o",
		gb.platformObjectOutput(module, src, arch),
		"-I$(SOURCE_ROOT)",
		"-I$(SOURCE_ROOT)/" + module.SourcePath,
		"-fdebug-prefix-map=$(BUILD_ROOT)=/-B",
		"-fdebug-prefix-map=$(SOURCE_ROOT)=/-S",
		"-pipe",
		"-g",
		"-fsigned-char",
	}
}

func (gb *GraphBuilder) generateCCEnvironment(
	module *Module,
	arch PlatformArch,
) map[string]string {
	dyldPath := "$(CLANG-2403293607)/lib:$(OS_SDK_ROOT-sbr:309054781)/usr/lib/x86_64-linux-gnu"
	if arch == PlatformAARCH64 {
		dyldPath = "$(CLANG-2403293607)/lib:$(OS_SDK_ROOT-sbr:309054781)/usr/lib/aarch64-linux-gnu"
	}

	return map[string]string{
		"ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
		"CPATH":                  "",
		"DYLD_LIBRARY_PATH":      dyldPath,
		"LIBRARY_PATH":           "",
		"SDKROOT":                "",
	}
}

func (gb *GraphBuilder) createASNode(
	module *Module,
	src string,
	platformCtx PlatformAwareContext,
) *GraphNode {
	asUIDKey := fmt.Sprintf("%s:AS:%s:%s", module.SourcePath, platformCtx.arch, src)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(asUIDKey))
	node.SelfUID = NewUID([]byte(asUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(asUIDKey + "_stats"))

	node.Platform = string(platformCtx.arch)

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: "asm",
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateASCommand(module, src, platformCtx.arch),
			Env:     gb.generateCCEnvironment(module, platformCtx.arch),
		},
	}

	node.Inputs = []string{gb.sourceInput(module, src)}
	node.Outputs = []string{gb.platformObjectOutput(module, src, platformCtx.arch)}
	node.Deps = []string{}

	node.KV = map[string]string{
		"uid": NewUID([]byte(asUIDKey + "_kv")),
		"p":   "AS",
		"pc":  "light-green",
	}

	return node
}

func (gb *GraphBuilder) generateASCommand(
	module *Module,
	src string,
	arch PlatformArch,
) []string {
	compiler := "clang"

	target := "aarch64-linux-gnu"
	march := "armv8-a"
	if arch == PlatformX86_64 {
		target = "x86_64-linux-gnu"
		march = "x86-64"
	}

	return []string{
		compiler,
		"--target=" + target,
		"-march=" + march,
		"--sysroot=/nowhere",
		"-B$(OS_SDK_ROOT-sbr:309054781)/usr/bin",
		"-c",
		gb.sourceInput(module, src),
		"-o",
		gb.platformObjectOutput(module, src, arch),
	}
}

func (gb *GraphBuilder) createJSNode(
	module *Module,
	src string,
	platformCtx PlatformAwareContext,
) *GraphNode {
	jsUIDKey := fmt.Sprintf("%s:JS:%s:%s", module.SourcePath, platformCtx.arch, src)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(jsUIDKey))
	node.SelfUID = NewUID([]byte(jsUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(jsUIDKey + "_stats"))

	if gb.ctx.TargetPlatform != "" {
		node.Platform = gb.ctx.TargetPlatform
	} else {
		node.Platform = string(platformCtx.arch)
	}

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: gb.determineCCSourceLanguage(src),
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	outputPath := "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, src)

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateJSCommand(module, src),
			Env:     map[string]string{},
		},
	}

	seenInputs := make(map[string]bool)
	var allInputs []string
	for _, s := range module.Sources {
		if strings.HasSuffix(s, ".cpp") {
			inputPath := "$(SOURCE_ROOT)/" + filepath.Join(module.SourcePath, s)
			if !seenInputs[inputPath] {
				seenInputs[inputPath] = true
				allInputs = append(allInputs, inputPath)
			}
		}
	}
	node.Inputs = allInputs
	node.Outputs = []string{outputPath}
	node.Deps = []string{}

	node.KV = map[string]string{
		"uid": NewUID([]byte(jsUIDKey + "_kv")),
		"p":   "JS",
		"pc":  "magenta",
	}

	return node
}

func (gb *GraphBuilder) generateJSCommand(
	module *Module,
	src string,
) []string {
	outputPath := "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, src)

	args := []string{
		"$(YMAKE_PYTHON3-1002064631)/bin/python3",
		"$(SOURCE_ROOT)/build/scripts/gen_join_srcs.py",
		outputPath,
		"--ya-start-command-file",
	}

	for _, s := range module.Sources {
		if strings.HasSuffix(s, ".cpp") && s != src {
			args = append(args, filepath.Join(module.SourcePath, s))
		}
	}

	return args
}

func (gb *GraphBuilder) createJoinSrcsNode(
	module *Module,
	jsd *JoinSrcsDirective,
	platformCtx PlatformAwareContext,
) *GraphNode {
	jsUIDKey := fmt.Sprintf("%s:JS:%s:%s", module.SourcePath, platformCtx.arch, jsd.OutputFile)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(jsUIDKey))
	node.SelfUID = NewUID([]byte(jsUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(jsUIDKey + "_stats"))
	node.Platform = string(platformCtx.arch)

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: gb.determineCCSourceLanguage(jsd.OutputFile),
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	outputPath := "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, jsd.OutputFile)

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateJoinSrcsCommand(module, jsd),
			Env:     map[string]string{},
		},
	}

	seenInputs := make(map[string]bool)
	var allInputs []string
	for _, inputFile := range jsd.InputFiles {
		inputPath := "$(SOURCE_ROOT)/" + filepath.Join(module.SourcePath, inputFile)
		if !seenInputs[inputPath] {
			seenInputs[inputPath] = true
			allInputs = append(allInputs, inputPath)
		}
	}
	node.Inputs = allInputs
	node.Outputs = []string{outputPath}
	node.Deps = []string{}

	node.KV = map[string]string{
		"uid": NewUID([]byte(jsUIDKey + "_kv")),
		"p":   "JS",
		"pc":  "magenta",
	}

	return node
}

func (gb *GraphBuilder) generateJoinSrcsCommand(
	module *Module,
	jsd *JoinSrcsDirective,
) []string {
	outputPath := "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, jsd.OutputFile)

	args := []string{
		"$(YMAKE_PYTHON3-1002064631)/bin/python3",
		"$(SOURCE_ROOT)/build/scripts/gen_join_srcs.py",
		outputPath,
		"--ya-start-command-file",
	}

	for _, inputFile := range jsd.InputFiles {
		args = append(args, filepath.Join(module.SourcePath, inputFile))
	}

	return args
}

func (gb *GraphBuilder) createR6Node(
	module *Module,
	src string,
	platformCtx PlatformAwareContext,
) *GraphNode {
	r6UIDKey := fmt.Sprintf("%s:R6:%s:%s", module.SourcePath, platformCtx.arch, src)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(r6UIDKey))
	node.SelfUID = NewUID([]byte(r6UIDKey + "_self"))
	node.StatsUID = NewUID([]byte(r6UIDKey + "_stats"))

	node.Platform = string(platformCtx.arch)

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: "cpp",
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	outputPath := "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "_", src+".cpp")

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateR6Command(module, src),
			Env:     map[string]string{},
		},
	}

	node.Inputs = []string{
		"$(BUILD_ROOT)/contrib/tools/ragel6/ragel6",
		gb.sourceInput(module, src),
	}
	node.Outputs = []string{outputPath}
	node.Deps = []string{}

	node.KV = map[string]string{
		"uid": NewUID([]byte(r6UIDKey + "_kv")),
		"p":   "R6",
		"pc":  "yellow",
	}

	return node
}

func (gb *GraphBuilder) generateR6Command(
	module *Module,
	src string,
) []string {
	outputPath := "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "_", src+".cpp")

	return []string{
		"$(BUILD_ROOT)/contrib/tools/ragel6/ragel6",
		"-CT0",
		"-L",
		"-I$(SOURCE_ROOT)",
		"-o",
		outputPath,
		gb.sourceInput(module, src),
	}
}

func (gb *GraphBuilder) createCPNode(
	module *Module,
	src string,
	platformCtx PlatformAwareContext,
) *GraphNode {
	cpUIDKey := fmt.Sprintf("%s:CP:%s:%s", module.SourcePath, platformCtx.arch, src)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(cpUIDKey))
	node.SelfUID = NewUID([]byte(cpUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(cpUIDKey + "_stats"))

	node.Platform = string(platformCtx.arch)

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: "cpp",
		ModuleType: gb.mapModuleTypeToString(module.Type),
	}

	srcPath := "$(SOURCE_ROOT)/" + filepath.Join(module.SourcePath, src)
	var dstPath string
	if strings.Contains(src, "musl.py") {
		dstPath = "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, src+".pyplugin")
	} else {
		dstPath = "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, src)
	}

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateCPCommand(module, srcPath, dstPath),
			Env:     map[string]string{},
		},
	}

	node.Inputs = []string{
		"$(SOURCE_ROOT)/build/scripts/fs_tools.py",
		"$(SOURCE_ROOT)/build/scripts/process_command_files.py",
		srcPath,
	}
	node.Outputs = []string{dstPath}
	node.Deps = []string{}

	node.KV = map[string]string{
		"uid": NewUID([]byte(cpUIDKey + "_kv")),
		"p":   "CP",
		"pc":  "light-cyan",
	}

	return node
}

func (gb *GraphBuilder) generateCPCommand(
	module *Module,
	src string,
	dst string,
) []string {
	return []string{
		"$(YMAKE_PYTHON3-1002064631)/bin/python3",
		"$(SOURCE_ROOT)/build/scripts/fs_tools.py",
		"copy",
		src,
		dst,
	}
}

func (gb *GraphBuilder) createFinalPhaseNode(
	module *Module,
	objectOutputs []string,
	moduleUIDMap map[string]*Module,
	platformCtx PlatformAwareContext,
) *GraphNode {
	switch module.Type {
	case ModuleTypeLibrary:
		return gb.createARNode(module, objectOutputs, moduleUIDMap, platformCtx)
	case ModuleTypeProgram:
		return gb.createLDNode(module, objectOutputs, moduleUIDMap, platformCtx)
	default:
		return nil
	}
}

func (gb *GraphBuilder) createARNode(
	module *Module,
	objectOutputs []string,
	moduleUIDMap map[string]*Module,
	platformCtx PlatformAwareContext,
) *GraphNode {
	finalUIDKey := fmt.Sprintf("%s:AR:%s", module.SourcePath, platformCtx.arch)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(finalUIDKey))
	node.SelfUID = NewUID([]byte(finalUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(finalUIDKey + "_stats"))

	node.Platform = string(platformCtx.arch)

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: gb.determineModuleLanguage(module),
		ModuleType: "lib",
	}

	compileDeps := gb.getCompileDepUIDs(module, platformCtx.arch)
	node.Deps = compileDeps

	transitiveHeaders := gb.collectTransitiveHeaders(module, platformCtx.arch)

	node.Cmds = []Command{
		{
			CmdArgs: gb.generateARCommand(module, objectOutputs, platformCtx.arch),
			Env:     gb.generateCCEnvironment(module, platformCtx.arch),
		},
	}

	var allInputs []string
	allInputs = append(allInputs, objectOutputs...)
	allInputs = append(allInputs, "$(SOURCE_ROOT)/build/scripts/link_lib.py")
	allInputs = append(allInputs, transitiveHeaders...)

	for _, src := range module.Sources {
		allInputs = append(allInputs, gb.sourceInput(module, src))
	}

	node.Inputs = allInputs
	node.Outputs = []string{gb.platformModuleOutput(module, platformCtx.arch)}

	node.KV = map[string]string{
		"uid":      NewUID([]byte(finalUIDKey + "_kv")),
		"p":        "AR",
		"pc":       "light-red",
		"show_out": "yes",
	}

	return node
}

func (gb *GraphBuilder) platformObjectOutputs(module *Module, arch PlatformArch) []string {
	var outputs []string
	for _, src := range module.Sources {
		if isCompilableCSource(src) || strings.HasSuffix(strings.ToLower(src), ".s") || strings.HasSuffix(strings.ToLower(src), ".S") {
			outputs = append(outputs, gb.platformObjectOutput(module, src, arch))
		}
	}
	return outputs
}

func (gb *GraphBuilder) collectTransitiveHeaders(module *Module, arch PlatformArch) []string {
	headers := make(map[string]bool)

	for _, src := range module.Sources {
		srcPath := filepath.Join(gb.sourceRoot, module.SourcePath, src)
		content, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "#include") {
				continue
			}

			var header string
			if strings.Contains(trimmed, "<") && strings.Contains(trimmed, ">") {
				start := strings.Index(trimmed, "<") + 1
				end := strings.LastIndex(trimmed, ">")
				if start > 0 && end > start {
					header = trimmed[start:end]
				}
			} else if strings.Contains(trimmed, "\"") {
				start := strings.Index(trimmed, "\"") + 1
				end := strings.LastIndex(trimmed, "\"")
				if start > 0 && end > start {
					header = trimmed[start:end]
				}
			}

			if header != "" {
				headerPath := gb.resolveHeaderPath(header, module.SourcePath)
				if headerPath != "" {
					headers["$(SOURCE_ROOT)/"+headerPath] = true
				}
			}
		}
	}

	result := make([]string, 0, len(headers))
	for h := range headers {
		result = append(result, h)
	}
	return result
}

func (gb *GraphBuilder) resolveHeaderPath(header, moduleSourcePath string) string {
	paths := []string{
		filepath.Join(moduleSourcePath, header),
		filepath.Join("contrib/libs/musl/include", header),
		filepath.Join("contrib/libs/musl/include/arpa", header),
		filepath.Join("contrib/libs/musl/include/net", header),
		filepath.Join("contrib/libs/musl/include/netinet", header),
		filepath.Join("contrib/libs/musl/include/sys", header),
		filepath.Join("contrib/libs/linux-headers/include", header),
		filepath.Join("contrib/libs/linux-headers/include/asm", header),
		filepath.Join("contrib/libs/linux-headers/include/asm-generic", header),
		filepath.Join("contrib/libs/linux-headers/include/linux", header),
		filepath.Join("contrib/libs/linux-headers/include/net", header),
		filepath.Join("contrib/libs/linux-headers/include/uapi", header),
		filepath.Join("contrib/libs/linux-headers/include/uapi/asm", header),
		filepath.Join("contrib/libs/linux-headers/include/uapi/asm-generic", header),
		filepath.Join("contrib/libs/linux-headers/include/uapi/linux", header),
		filepath.Join("contrib/libs/cxxsupp/libcxx/include", header),
		filepath.Join("contrib/libs/cxxsupp/libcxx/include/__support", header),
		filepath.Join("contrib/libs/cxxsupp/libcxx/include/__support/ibm", header),
		filepath.Join("contrib/libs/cxxsupp/libcxx/include/__support/win32", header),
		filepath.Join("contrib/libs/cxxsupp/libcxx/include/__filesystem", header),
		filepath.Join("contrib/libs/cxxsupp/builtins", header),
	}

	for _, path := range paths {
		fullPath := filepath.Join(gb.sourceRoot, path)
		if _, err := os.Stat(fullPath); err == nil {
			return path
		}
	}

	return ""
}

func (gb *GraphBuilder) getCompileDepUIDs(module *Module, arch PlatformArch) []string {
	var deps []string
	for _, src := range module.Sources {
		if !isCompilableCSource(src) && !(strings.HasSuffix(strings.ToLower(src), ".s") || strings.HasSuffix(strings.ToLower(src), ".S")) {
			continue
		}
		compileUIDKey := fmt.Sprintf("%s:CC:%s:%s", module.SourcePath, arch, src)
		if strings.HasSuffix(strings.ToLower(src), ".s") || strings.HasSuffix(strings.ToLower(src), ".S") {
			compileUIDKey = fmt.Sprintf("%s:AS:%s:%s", module.SourcePath, arch, src)
		}
		compileUID := NewUID([]byte(compileUIDKey))
		deps = append(deps, compileUID)
	}
	return deps
}

func (gb *GraphBuilder) platformModuleOutput(module *Module, arch PlatformArch) string {
	baseName := filepath.Base(module.SourcePath)
	switch module.Type {
	case ModuleTypeLibrary:
		return "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "lib"+baseName+".a")
	case ModuleTypeProgram:
		return "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, baseName)
	default:
		return ""
	}
}

func (gb *GraphBuilder) generateARCommand(
	module *Module,
	objectOutputs []string,
	arch PlatformArch,
) []string {
	args := []string{
		"$(YMAKE_PYTHON3-1002064631)/bin/python3",
		"$(SOURCE_ROOT)/build/scripts/link_lib.py",
		"$(CLANG-2403293607)/bin/llvm-ar",
		"LLVM_AR",
		"gnu",
		"$(BUILD_ROOT)",
		"None",
		"--",
		"--",
		gb.platformModuleOutput(module, arch),
	}
	args = append(args, objectOutputs...)
	return args
}

func (gb *GraphBuilder) createLDNode(
	module *Module,
	objectOutputs []string,
	moduleUIDMap map[string]*Module,
	platformCtx PlatformAwareContext,
) *GraphNode {
	finalUIDKey := fmt.Sprintf("%s:LD:%s", module.SourcePath, platformCtx.arch)

	node := NewGraphNode(*platformCtx.ctx)

	node.UID = NewUID([]byte(finalUIDKey))
	node.SelfUID = NewUID([]byte(finalUIDKey + "_self"))
	node.StatsUID = NewUID([]byte(finalUIDKey + "_stats"))

	node.Platform = string(platformCtx.arch)

	node.TargetProperties = TargetProperties{
		ModuleDir:  module.SourcePath,
		ModuleLang: gb.determineModuleLanguage(module),
		ModuleType: "bin",
	}

	compileDeps := gb.getCompileDepUIDs(module, platformCtx.arch)
	archiveDeps := gb.getArchiveDepUIDs(module, moduleUIDMap, platformCtx.arch)
	node.Deps = append(compileDeps, archiveDeps...)

	archiveFiles := gb.getArchiveFiles(module, moduleUIDMap, platformCtx.arch)

	transitiveHeaders := gb.collectTransitiveHeaders(module, platformCtx.arch)

	var allInputs []string
	versionO := "$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "_version.c.o")
	allInputs = append(allInputs, versionO)
	allInputs = append(allInputs, objectOutputs...)
	allInputs = append(allInputs, archiveFiles...)
	allInputs = append(allInputs, transitiveHeaders...)
	allInputs = append(allInputs, "$(SOURCE_ROOT)/build/scripts/vcs_info.py")
	allInputs = append(allInputs, "$(SOURCE_ROOT)/build/scripts/c_templates/svn_interface.c")
	allInputs = append(allInputs, "$(SOURCE_ROOT)/build/scripts/link_exe.py")
	allInputs = append(allInputs, "$(SOURCE_ROOT)/build/scripts/fs_tools.py")

	node.Cmds = gb.generateLDCommands(module, objectOutputs, platformCtx.arch)
	node.Inputs = allInputs
	node.Outputs = []string{gb.platformModuleOutput(module, platformCtx.arch)}

	node.KV = map[string]string{
		"uid":      NewUID([]byte(finalUIDKey + "_kv")),
		"p":        "LD",
		"pc":       "light-blue",
		"show_out": "yes",
	}

	return node
}

func (gb *GraphBuilder) getArchiveFiles(module *Module, moduleUIDMap map[string]*Module, arch PlatformArch) []string {
	var files []string
	seen := make(map[string]bool)

	gb.collectTransitiveArchiveFiles(module, moduleUIDMap, arch, seen)

	for file := range seen {
		files = append(files, file)
	}
	return files
}

func (gb *GraphBuilder) collectTransitiveArchiveFiles(
	module *Module,
	moduleUIDMap map[string]*Module,
	arch PlatformArch,
	seen map[string]bool,
) {
	for _, depPath := range module.Dependencies {
		depModule := gb.resolveDependencyModule(depPath, moduleUIDMap)
		if depModule == nil || depModule.Type != ModuleTypeLibrary {
			continue
		}

		archiveFile := gb.platformModuleOutput(depModule, arch)
		if !seen[archiveFile] {
			seen[archiveFile] = true
			gb.collectTransitiveArchiveFiles(depModule, moduleUIDMap, arch, seen)
		}
	}
}

func (gb *GraphBuilder) getArchiveDepUIDs(module *Module, moduleUIDMap map[string]*Module, arch PlatformArch) []string {
	var deps []string
	seen := make(map[string]bool)

	gb.collectTransitiveArchiveDeps(module, moduleUIDMap, arch, seen)

	for uid := range seen {
		deps = append(deps, uid)
	}
	return deps
}

func (gb *GraphBuilder) collectTransitiveArchiveDeps(
	module *Module,
	moduleUIDMap map[string]*Module,
	arch PlatformArch,
	seen map[string]bool,
) {
	for _, depPath := range module.Dependencies {
		depModule := gb.resolveDependencyModule(depPath, moduleUIDMap)
		if depModule == nil || depModule.Type != ModuleTypeLibrary {
			continue
		}

		archiveUIDKey := fmt.Sprintf("%s:AR:%s", depModule.SourcePath, arch)
		archiveUID := NewUID([]byte(archiveUIDKey))

		if !seen[archiveUID] {
			seen[archiveUID] = true
			gb.collectTransitiveArchiveDeps(depModule, moduleUIDMap, arch, seen)
		}
	}
}

func (gb *GraphBuilder) generateLDCommands(
	module *Module,
	objectOutputs []string,
	arch PlatformArch,
) []Command {
	target := "aarch64-linux-gnu"
	march := "armv8-a"
	if arch == PlatformX86_64 {
		target = "x86_64-linux-gnu"
		march = "x86-64"
	}

	cmd1 := Command{
		CmdArgs: []string{
			"$(YMAKE_PYTHON3-1002064631)/bin/python3",
			"$(SOURCE_ROOT)/build/scripts/vcs_info.py",
			"$(VCS)/vcs.json",
			"$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "_version.c"),
			"$(SOURCE_ROOT)/build/scripts/c_templates/svn_interface.c",
		},
		Env: gb.generateCCEnvironment(module, arch),
	}

	cmd2 := Command{
		CmdArgs: []string{
			"$(CLANG-2403293607)/bin/clang",
			"--target=" + target,
			"-march=" + march,
			"--sysroot=/nowhere",
			"-B$(OS_SDK_ROOT-sbr:309054781)/usr/bin",
			"-c",
			"-o",
			"$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "_version.c.o"),
			"$(BUILD_ROOT)/" + filepath.Join(module.SourcePath, "_version.c"),
			"-I$(SOURCE_ROOT)",
			"-fdebug-prefix-map=$(BUILD_ROOT)=/-B",
			"-fdebug-prefix-map=$(SOURCE_ROOT)=/-S",
			"-fdebug-prefix-map=$(TOOL_ROOT)=/-T",
			"-pipe",
			"-g",
			"-fsigned-char",
		},
		Env: gb.generateCCEnvironment(module, arch),
	}

	cmd3Args := []string{
		"$(YMAKE_PYTHON3-1002064631)/bin/python3",
		"$(SOURCE_ROOT)/build/scripts/link_exe.py",
		"--start-plugins",
		"$(BUILD_ROOT)/contrib/libs/musl/include/musl.py.pyplugin",
		"--end-plugins",
		"--clang-ver",
		"20",
		"--source-root",
		"$(SOURCE_ROOT)",
		"--build-root",
		"$(BUILD_ROOT)",
		"--arch=LINUX",
		"--objcopy-exe",
		"$(CLANG-2403293607)/bin/llvm-objcopy",
		"$(CLANG-2403293607)/bin/clang++",
		"-Wl,--whole-archive",
		"--ya-start-command-file",
		"--ya-end-command-file",
		"-Wl,--no-whole-archive",
	}

	cmd3Args = append(cmd3Args, "$(BUILD_ROOT)/"+filepath.Join(module.SourcePath, "_version.c.o"))
	cmd3Args = append(cmd3Args, objectOutputs...)

	cmd3Args = append(cmd3Args,
		"-o",
		"$(BUILD_ROOT)/"+filepath.Join(module.SourcePath, filepath.Base(module.SourcePath)),
		"--target="+target,
		"-march="+march,
		"--sysroot=/nowhere",
		"-B$(OS_SDK_ROOT-sbr:309054781)/usr/bin",
		"-Wl,--start-group",
		"-Wl,--end-group",
		"-Wl,-rpath=@loader_path/.",
		"-Wl,-rpath,@loader_path/../lib",
	)

	cmd3 := Command{
		CmdArgs: cmd3Args,
		Env:     gb.generateCCEnvironment(module, arch),
	}

	cmd4 := Command{
		CmdArgs: []string{
			"$(YMAKE_PYTHON3-1002064631)/bin/python3",
			"$(SOURCE_ROOT)/build/scripts/fs_tools.py",
			"link_or_copy_to_dir",
			"--no-check",
			"$(BUILD_ROOT)/" + module.SourcePath + "/",
		},
		Env: gb.generateCCEnvironment(module, arch),
	}

	return []Command{cmd1, cmd2, cmd3, cmd4}
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
	compileDeps := gb.getCompileDepUIDsLegacy(module, compilableSources)

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

func (gb *GraphBuilder) getCompileDepUIDsLegacy(module *Module, compilableSources []string) []string {
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

func (gb *GraphBuilder) injectAllocatorDependencies(module *Module) {
	if module == nil || module.Type != ModuleTypeProgram {
		return
	}

	allocType := gb.config.ParseAllocatorFromModule(module)

	if allocType == "FAKE" {
		return
	}

	peerdirs := gb.config.resolveAllocatorPEERDIRs(allocType)

	if len(peerdirs) == 0 {
		return
	}

	existing := make(map[string]bool)
	for _, dep := range module.Dependencies {
		existing[dep] = true
	}

	for _, peerdir := range peerdirs {
		if !existing[peerdir] {
			module.Dependencies = append(module.Dependencies, peerdir)
		}
	}
}
