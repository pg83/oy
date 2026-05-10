package main

// diagnostics.go provides diagnostic logging for ymake build process.
// Use --diag-peerdir flag to enable PEERDIR traversal diagnostics.
// Use --diag-eval flag to enable variable evaluation diagnostics in conditionals.
//
// Variable evaluation diagnostics (--diag-eval):
// - Logs all variable lookups during IF/ELSEIF/BUILD_ONLY_IF/WHEN evaluation
// - Shows variable name, value, boolean result, and original expression
// - Helps debug conditional logic and understands which variables are being tested
//
// Example output:
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

import (
	"fmt"
	"strings"
	"sync"
)

type TraversalLogger struct {
	mu                 sync.Mutex
	enabled            bool
	evalTracingEnabled bool
	peerdirResolves    []PeerdirResolution
	moduleLoads        []ModuleLoad
	registryLookups    []RegistryLookup
	unreachableModules map[string]bool
	evalVarTraces      []EvalVarTrace
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

func (tl *TraversalLogger) LogEvalVar(varName, varValue, context, modulePath, expression string, result bool) {
	if !tl.IsEnabled() || !tl.IsEvalTracingEnabled() {
		return
	}
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.evalVarTraces = append(tl.evalVarTraces, EvalVarTrace{
		VariableName:  varName,
		VariableValue: varValue,
		Result:        result,
		Context:       context,
		ModulePath:    modulePath,
		Expression:    expression,
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

	if tl.IsEvalTracingEnabled() && len(tl.evalVarTraces) > 0 {
		fmt.Println("\n======== VARIABLE EVALUATION DIAGNOSTIC SUMMARY ========")

		successCount := 0
		for _, ev := range tl.evalVarTraces {
			if ev.Result {
				successCount++
			}
		}
		fmt.Printf("Total variable evaluations: %d\n", len(tl.evalVarTraces))
		fmt.Printf("Evaluations resulting in true: %d (%.1f%%)\n", successCount, float64(successCount)*100/float64(len(tl.evalVarTraces)))

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
			fmt.Println()
		}

		fmt.Println("======================== END EVAL SUMMARY ========================")
	}

	fmt.Println("======================== END SUMMARY ========================")
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

func NewTraversalLogger(enabled bool, evalTracingEnabled bool) *TraversalLogger {
	return &TraversalLogger{
		enabled:            enabled,
		evalTracingEnabled: evalTracingEnabled,
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
