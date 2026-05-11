package main

// diagnostics.go provides diagnostic logging for ymake build process.
// Use --diag-peerdir flag to enable PEERDIR traversal diagnostics.
// Use --diag-eval flag to enable variable evaluation diagnostics in conditionals.
// Use --diag-tool-modules flag to enable tool module loading and node creation diagnostics.
//
// Variable evaluation diagnostics (--diag-eval):
// - Logs all variable lookups during IF/ELSEIF/BUILD_ONLY_IF/WHEN evaluation
// - Shows variable name, value, boolean result, and original expression
// - Helps debug conditional logic and understands which variables are being tested
//
// Tool module diagnostics (--diag-tool-modules):
// - Logs tool module discovery and loading (e.g., contrib/tools/ragel6)
// - Tracks execution node creation by module and platform
// - Shows platform-based filtering decisions and conditional evaluation
// - Helps understand node count discrepancies between reference and generated graphs
//
// Example output (--diag-eval):
// ======== VARIABLE EVALUATION DIAGNOSTIC SUMMARY ========
// Total variable evaluations: 78
// Evaluations resulting in true: 18 (23.1%)
//
// --- Variable Evaluation Details ---
//   [IF] tools/archiver: MSVC -> "false" => false (expr: MSVC)
//   [IF] tools/archiver: MUSL -> "yes" => true (expr: MUSL)
//   [BUILD_ONLY_IF] util/charset: OS_LINUX -> "true" => true (expr: OS_LINUX)
//   [WHEN] util: PREBUILT -> "no" => false (expr: PREBUILT)
// ========================= END EVAL SUMMARY ========================
//
// Example output (--diag-tool-modules):
// ======== TOOL MODULE LOADING DIAGNOSTIC SUMMARY ========
// Tool modules discovered: 1
//   - contrib/tools/ragel6 (loaded from util)
//
// ======== EXECUTION NODE CREATION DIAGNOSTIC SUMMARY ========
// Total nodes created: 3730
// Nodes by platform:
//   - default-linux-aarch64: 1865 (50.0%)
//   - default-linux-x86_64: 1865 (50.0%)
//
// --- Tool Module Node Details ---
// util module R6 nodes:
//   - default-linux-aarch64: datetime/parser.rl6
//   - default-linux-x86_64: datetime/parser.rl6
//
// ========================= END TOOL MODULE SUMMARY ========================

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type TraversalLogger struct {
	mu                 sync.Mutex
	enabled            bool
	evalTracingEnabled bool
	toolDiagEnabled    bool
	peerdirResolves    []PeerdirResolution
	moduleLoads        []ModuleLoad
	registryLookups    []RegistryLookup
	unreachableModules map[string]bool
	evalVarTraces      []EvalVarTrace
	toolModuleLoads    []ToolModuleLoad
	nodeCreationLogs   []NodeCreationLog
	platformFilterLogs []PlatformFilterLog
}

type PeerdirResolution struct {
	FromModule  string
	ToPath      string
	Resolved    bool
	FoundModule string
}

type ModuleLoad struct {
	ModulePath string
	FromParent string
	Success    bool
	Reason     string
}

type RegistryLookup struct {
	Path     string
	Found    bool
	InModule string
}

type EvalVarTrace struct {
	VariableName  string
	VariableValue string
	Result        bool
	Context       string
	ModulePath    string
	Expression    string

	ArchValue      string
	MuslValue      bool
	OSValue        string
	NoPlatformValue bool
}

type ToolModuleLoad struct {
	ToolPath   string
	ParentPath string
	Loaded     bool
	NodeType   string
	SourceFile string
	Success    bool
	Reason     string
}

type NodeCreationLog struct {
	ModulePath string
	Platform   string
	NodeType   string
	SourceFile string
	UID        string
}

type PlatformFilterLog struct {
	ModulePath string
	Platform   string
	NodeType   string
	SourceFile string
	Filtered   bool
	Reason     string
}

var globalLogger *TraversalLogger

func SetGlobalTraversalLogger(logger *TraversalLogger) {
	globalLogger = logger
}

func GetGlobalTraversalLogger() *TraversalLogger {
	return globalLogger
}

func (tl *TraversalLogger) IsEnabled() bool {
	if tl == nil {
		return false
	}
	return tl.enabled
}

func (tl *TraversalLogger) IsEvalTracingEnabled() bool {
	if tl == nil {
		return false
	}
	return tl.evalTracingEnabled
}

func (tl *TraversalLogger) IsToolDiagEnabled() bool {
	if tl == nil {
		return false
	}
	return tl.toolDiagEnabled
}

func (tl *TraversalLogger) LogPeerdirResolution(fromModule, toPath string, resolved bool, foundModule string) {
	if !tl.IsEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.peerdirResolves = append(tl.peerdirResolves, PeerdirResolution{
		FromModule:  fromModule,
		ToPath:      toPath,
		Resolved:    resolved,
		FoundModule: foundModule,
	})

	if !resolved && foundModule == "" {
		tl.unreachableModules[toPath] = true
	}
}

func (tl *TraversalLogger) LogEvalVar(varName, varValue, context, modulePath, expression string,
	result bool, archValue string, muslValue bool, osValue string, noPlatformValue bool) {
	if !tl.IsEnabled() || !tl.IsEvalTracingEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.evalVarTraces = append(tl.evalVarTraces, EvalVarTrace{
		VariableName:    varName,
		VariableValue:  varValue,
		Result:          result,
		Context:         context,
		ModulePath:      modulePath,
		Expression:      expression,
		ArchValue:       archValue,
		MuslValue:       muslValue,
		OSValue:         osValue,
		NoPlatformValue: noPlatformValue,
	})
}

func (tl *TraversalLogger) LogModuleLoad(modulePath, fromParent string, success bool, reason string) {
	if !tl.IsEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.moduleLoads = append(tl.moduleLoads, ModuleLoad{
		ModulePath: modulePath,
		FromParent: fromParent,
		Success:    success,
		Reason:     reason,
	})
}

func (tl *TraversalLogger) LogToolModuleLoad(toolPath, parentPath string, loaded bool, nodeType, sourceFile string) {
	if !tl.IsEnabled() || !tl.IsToolDiagEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.toolModuleLoads = append(tl.toolModuleLoads, ToolModuleLoad{
		ToolPath:   toolPath,
		ParentPath: parentPath,
		Loaded:     loaded,
		NodeType:   nodeType,
		SourceFile: sourceFile,
		Success:    loaded,
	})
}

func (tl *TraversalLogger) LogNodeCreation(modulePath, platform, nodeType, sourceFile, uid string) {
	if !tl.IsEnabled() || !tl.IsToolDiagEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.nodeCreationLogs = append(tl.nodeCreationLogs, NodeCreationLog{
		ModulePath: modulePath,
		Platform:   platform,
		NodeType:   nodeType,
		SourceFile: sourceFile,
		UID:        uid,
	})
}

func (tl *TraversalLogger) LogPlatformFilter(modulePath, platform, nodeType, sourceFile string, filtered bool, reason string) {
	if !tl.IsEnabled() || !tl.IsToolDiagEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.platformFilterLogs = append(tl.platformFilterLogs, PlatformFilterLog{
		ModulePath: modulePath,
		Platform:   platform,
		NodeType:   nodeType,
		SourceFile: sourceFile,
		Filtered:   filtered,
		Reason:     reason,
	})
}

func (tl *TraversalLogger) OutputSummary() {
	if !tl.IsEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	fmt.Println("\n======== PEERDIR TRAVERSAL DIAGNOSTIC SUMMARY ========")

	resolvedCount := 0
	for _, pr := range tl.peerdirResolves {
		if pr.Resolved {
			resolvedCount++
		}
	}
	fmt.Printf("Total PEERDIRs processed: %d\n", len(tl.peerdirResolves))
	fmt.Printf("Successfully resolved: %d (%.1f%%)\n", resolvedCount, float64(resolvedCount)*100/float64(len(tl.peerdirResolves)))

	fmt.Println("\n--- PEERDIR Resolution Details ---")
	for _, pr := range tl.peerdirResolves {
		status := "RESOLVED"
		if !pr.Resolved {
			status = "UNRESOLVED"
		}
		fmt.Printf("  %s → %s [%s]", pr.FromModule, pr.ToPath, status)
		if pr.FoundModule != "" {
			fmt.Printf(" → %s", pr.FoundModule)
		}
		fmt.Println()
	}

	loadedCount := 0
	for _, ml := range tl.moduleLoads {
		if ml.Success {
			loadedCount++
		}
	}
	fmt.Printf("\nTotal module load attempts: %d\n", len(tl.moduleLoads))
	fmt.Printf("Successfully loaded: %d (%.1f%%)\n", loadedCount, float64(loadedCount)*100/float64(len(tl.moduleLoads)))

	fmt.Println("\n--- Module Load Details ---")
	for _, ml := range tl.moduleLoads {
		status := "SUCCESS"
		if !ml.Success {
			status = "FAIL"
		}
		fmt.Printf("  %s from %s (%s)", ml.ModulePath, ml.FromParent, status)
		if ml.Reason != "" {
			fmt.Printf(": %s", ml.Reason)
		}
		fmt.Println()
	}

	if tl.IsToolDiagEnabled() {
		tl.outputToolModuleSummary()
	}

	if tl.IsEvalTracingEnabled() && len(tl.evalVarTraces) > 0 {
		tl.outputEvalSummary()
	}

	fmt.Println("======================== END SUMMARY ========================")
}

func (tl *TraversalLogger) outputEvalSummary() {
	fmt.Println("\n======== VARIABLE EVALUATION DIAGNOSTIC SUMMARY ========")

	successCount := 0
	for _, ev := range tl.evalVarTraces {
		if ev.Result {
			successCount++
		}
	}
	fmt.Printf("Total variable evaluations: %d\n", len(tl.evalVarTraces))
	fmt.Printf("Evaluations resulting in true: %d (%.1f%%)\n", successCount, float64(successCount)*100/float64(len(tl.evalVarTraces)))

	archEvalCount := 0
	muslEvalCount := 0
	osEvalCount := 0
	noPlatformEvalCount := 0
	archTrueCount := 0
	muslTrueCount := 0
	osTrueCount := 0
	noPlatformTrueCount := 0

	for _, ev := range tl.evalVarTraces {
		if strings.HasPrefix(ev.VariableName, "ARCH_") || strings.HasPrefix(ev.VariableName, "ARCH_") || strings.HasPrefix(ev.VariableName, "ARCH_") || ev.VariableName == "ARCH" {
			archEvalCount++
			if ev.Result {
				archTrueCount++
			}
		}
		if ev.VariableName == "MUSL" {
			muslEvalCount++
			if ev.Result {
				muslTrueCount++
			}
		}
		if strings.HasPrefix(ev.VariableName, "OS_") {
			osEvalCount++
			if ev.Result {
				osTrueCount++
			}
		}
		if ev.VariableName == "NO_PLATFORM" {
			noPlatformEvalCount++
			if ev.Result {
				noPlatformTrueCount++
			}
		}
	}

	fmt.Println("\n--- Platform Variable Distribution ---")
	fmt.Printf("  ARCH-related evaluations: %d\n", archEvalCount)
	fmt.Printf("  MUSL-related evaluations: %d\n", muslEvalCount)
	fmt.Printf("  OS-related evaluations: %d\n", osEvalCount)
	fmt.Printf("  NO_PLATFORM-related evaluations: %d\n", noPlatformEvalCount)

	fmt.Println("\n--- Platform variable evaluations by result ---")
	if archEvalCount > 0 {
		fmt.Printf("  ARCH: true=%d, false=%d\n", archTrueCount, archEvalCount-archTrueCount)
	} else {
		fmt.Printf("  ARCH: N/A (0 evaluations)\n")
	}
	if muslEvalCount > 0 {
		fmt.Printf("  MUSL: true=%d, false=%d\n", muslTrueCount, muslEvalCount-muslTrueCount)
	} else {
		fmt.Printf("  MUSL: N/A (0 evaluations)\n")
	}
	if osEvalCount > 0 {
		fmt.Printf("  OS:  true=%d, false=%d\n", osTrueCount, osEvalCount-osTrueCount)
	} else {
		fmt.Printf("  OS:  N/A (0 evaluations)\n")
	}
	if noPlatformEvalCount > 0 {
		fmt.Printf("  NO_PLATFORM: true=%d, false=%d\n", noPlatformTrueCount, noPlatformEvalCount-noPlatformTrueCount)
	} else {
		fmt.Printf("  NO_PLATFORM: N/A (0 evaluations)\n")
	}

	fmt.Println("\n--- Variable Evaluation Details ---")
	for _, ev := range tl.evalVarTraces {
		valueStr := ev.VariableValue
		if valueStr == "" {
			valueStr = "(unset)"
		}
		fmt.Printf("  [%s] %s: %s -> \"%s\" => %v", ev.Context, ev.ModulePath, ev.VariableName, valueStr, ev.Result)
		if ev.Expression != "" {
			fmt.Printf(" (expr: %s)", ev.Expression)
		}
		if ev.ArchValue != "" || ev.MuslValue || ev.OSValue != "" {
			contextParts := []string{}
			if ev.ArchValue != "" {
				contextParts = append(contextParts, fmt.Sprintf("arch=%s", ev.ArchValue))
			}
			contextParts = append(contextParts, fmt.Sprintf("musl=%v", ev.MuslValue))
			if ev.OSValue != "" {
				contextParts = append(contextParts, fmt.Sprintf("os=%s", ev.OSValue))
			}
			if ev.NoPlatformValue {
				contextParts = append(contextParts, "noplatform=true")
			}
			fmt.Printf(" [%s]", strings.Join(contextParts, " "))
		}
		fmt.Println()
	}

	fmt.Println("======================== END EVAL SUMMARY ========================")
}

func (tl *TraversalLogger) outputToolModuleSummary() {

	if len(tl.toolModuleLoads) == 0 {
		fmt.Println("No tool modules loaded.")
	} else {
		loadedTools := make(map[string]bool)
		for _, tml := range tl.toolModuleLoads {
			if tml.Loaded {
				loadedTools[tml.ToolPath] = true
			}
		}

		fmt.Printf("Tool modules discovered: %d\n", len(loadedTools))
		for toolPath := range loadedTools {
			fmt.Printf("  - %s\n", toolPath)
		}

		platformNodes := make(map[string]map[string]int)
		moduleNodes := make(map[string]map[string]int)

		for _, ncl := range tl.nodeCreationLogs {
			if platformNodes[ncl.NodeType] == nil {
				platformNodes[ncl.NodeType] = make(map[string]int)
			}
			platformNodes[ncl.NodeType][ncl.Platform]++

			key := ncl.ModulePath + ":" + ncl.NodeType
			if moduleNodes[key] == nil {
				moduleNodes[key] = make(map[string]int)
			}
			moduleNodes[key][ncl.Platform]++
		}

		if len(tl.nodeCreationLogs) > 0 {
			fmt.Println("\n======== EXECUTION NODE CREATION DIAGNOSTIC SUMMARY ========")
			fmt.Printf("Total nodes created: %d\n", len(tl.nodeCreationLogs))

			fmt.Println("\n--- Nodes by Node Type ---")
			for nodeType := range platformNodes {
				total := 0
				for _, count := range platformNodes[nodeType] {
					total += count
				}
				fmt.Printf("  %s: %d\n", nodeType, total)
			}

			fmt.Println("\n--- Nodes by Platform ---")
			platformTotals := make(map[string]int)
			for _, ncl := range tl.nodeCreationLogs {
				platformTotals[ncl.Platform]++
			}

			totalNodes := len(tl.nodeCreationLogs)
			for platform, count := range platformTotals {
				percent := float64(count) * 100 / float64(totalNodes)
				fmt.Printf("  - %s: %d (%.1f%%)\n", platform, count, percent)
			}

			fmt.Println("\n--- Tool Module Node Details ---")
			sortedKeys := make([]string, 0, len(moduleNodes))
			for key := range moduleNodes {
				if strings.Contains(key, "R6") || strings.Contains(key, "JS") {
					sortedKeys = append(sortedKeys, key)
				}
			}

			sort.Strings(sortedKeys)
			for _, key := range sortedKeys {
				platforms := moduleNodes[key]
				parts := strings.Split(key, ":")
				modulePath := parts[0]
				nodeType := parts[1]

				fmt.Printf("%s module %s nodes:\n", modulePath, nodeType)
				for platform, count := range platforms {
					fmt.Printf("  - %s: %d nodes\n", platform, count)
				}
			}

			fmt.Println("======================== END TOOL MODULE SUMMARY ========================")
		}
	}
}

func (tl *TraversalLogger) GetUnreachableModules() []string {
	if !tl.IsEnabled() {
		return nil
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	var modules []string
	for path := range tl.unreachableModules {
		modules = append(modules, path)
	}
	return modules
}

func NewTraversalLogger(enabled bool, evalTracingEnabled bool, toolDiagEnabled bool) *TraversalLogger {
	return &TraversalLogger{
		enabled:            enabled,
		evalTracingEnabled: evalTracingEnabled,
		toolDiagEnabled:    toolDiagEnabled,
		unreachableModules: make(map[string]bool),
	}
}

func (tl *TraversalLogger) LogProgress(msg string) {
	if tl.IsEnabled() {
		prefix := "[DIAG PROGRESS] "
		if strings.HasPrefix(msg, "[DIAG]") {
			prefix = ""
		}
		fmt.Println(prefix + msg)
	}
}
