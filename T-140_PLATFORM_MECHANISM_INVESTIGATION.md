# T-140: Reference Platform Selection Mechanism Investigation

## Executive Summary

**Key Finding**: The reference ya/ymake build system uses a **module filtering** mechanism, not a dual-platform generation strategy. Reference generates 3730 nodes split across two platforms (1933 aarch64 + 1797 x86_64), but **each module is assigned to exactly ONE platform** based on its role in the build.

**Root Cause of Current Implementation Gap**: The current Go implementation uses `NewPlatformContexts()` which hardcodes generation of **both platforms for all modules** (495 aarch64 + 495 x86_64), plus 14 nodes with `platform="both"`. This creates unnecessary duplication for modules that should only build on one platform.

**Critical Discovery**: Reference uses `--target-platform=default-linux-aarch64` but **still generates x86_64 nodes** for host tools (yasm, ragel6). The `--target-platform` flag controls **target platform modules**, but host tools always build on build machine architecture (x86_64).

## Reference Graph Platform Distribution

### Overall Statistics
```
Reference graph: 3730 total nodes
├─ default-linux-aarch64: 1933 nodes (52%)
└─ default-linux-x86_64: 1797 nodes (48%)

Current graph: 1004 total nodes
├─ default-linux-aarch64: 495 nodes
├─ default-linux-x86_64: 495 nodes
└─ "both": 14 nodes (BUG - reference has 0)
```

### Platform Assignment by Module Type

#### x86_64-Only Modules (Host Tools)
All 1797 x86_64 nodes in reference are tagged with `"tags": ["tool"]`. These are:
- **contrib/tools/yasm** - x86_64 assembler (build-time tool)
- **contrib/tools/ragel6** - code generator (build-time tool)

Key pattern: Build-time tools run on build machine, so they build for x86_64 regardless of `--target-platform`.

#### aarch64-Only Modules (Target Dependencies)
The 1933 aarch64 nodes are primarily:
- **tools/archiver** - target application (specified on command line)
- **contrib/libs/musl/*** - C library for target (2660+ transitive nodes)
- **contrib/libs/base64/*** - target library dependency
- **library/cpp/*** - target C++ libraries

Key pattern: Target application and its dependencies build for `--target-platform`.

### NO_PLATFORM() Directive

Found in `/home/pg/monorepo/yatool_orig/contrib/tools/yasm/ya.make`:
```make
PROGRAM(yasm)

IF (MUSL)
    PEERDIR(
        contrib/libs/musl_extra
        contrib/libs/jemalloc
    )
    DISABLE(USE_ASMLIB)
    ENABLE(MUSL_LITE)
ELSE()
    NO_PLATFORM()    # Line 28 - critical directive
ENDIF()
```

The `NO_PLATFORM()` macro definition from `build/ymake.core.conf`:
```make
macro NO_PLATFORM() {
    NO_LIBC()
    ENABLE(NOPLATFORM)
}
```

**Interpretation**:
- `NO_PLATFORM()` excludes C/C++ runtime dependencies (util, musl, libcxx)
- Sets `NOPLATFORM` flag for special processing
- Used when module doesn't need runtime libraries (e.g., pure tools)
- Likely signals platform assignment decision to ymake graph builder

## Platform Selection Algorithm (Reconstructed)

Based on reference graph analysis and ya.make directives:

```
For each module during PEERDIR traversal:

if module has NO_PLATFORM() directive:
    if module is build-time tool (in contrib/tools/):
        platform = host_platform_arch  # x86_64 on build machine
        tags.append("tool")
    else:
        platform = --target-platform flag value
elif module dependency of tagged tools:
    platform = host_platform_arch  # x86_64 for tool dependencies
elif module path pattern matches target deps:
    platform = --target-platform flag value  # aarch64 for archiver
else:
    platform = --target-platform flag value
```

## Current Implementation Issues

### Issue 1: Dual-Platform Generation for All Modules
**Location**: `graph_builder.go:58-63:NewPlatformContexts()`
```go
func NewPlatformContexts(ctx *ParseContext) []PlatformAwareContext {
    return []PlatformAwareContext{
        {ctx: ctx, arch: PlatformAARCH64},
        {ctx: ctx, arch: PlatformX86_64},
    }
}
```
**Problem**: Always generates both platforms, ignoring module role.
**Impact**: Unnecessary duplication, incorrect node count.

### Issue 2: platform="both" Assignments
**Location**: `graph_builder.go:650` (JS nodes), `graph_builder.go:780` (R6 nodes)
```go
node.Platform = "both"  // Line 650, 780
```
**Problem**: Reference has 0 nodes with `platform="both"`.
**Impact**: Graph structure mismatch, incorrect resource generation.

### Issue 3: NO_PLATFORM() Directive Not Parsed
**Problem**: `NO_PLATFORM()` directive is parsed as macro but not used for platform decisions.
**Impact**: Cannot distinguish tools vs target modules.

### Issue 4: Tag Assignment Missing
**Problem**: No mechanism to assign `"tags": ["tool"]` to host tool modules.
**Impact**: Cannot identify tools for platform filtering.

## Required Implementation Changes

### Change 1: Detect Tool Modules
**Approach 1**: Path-based detection
```go
func isToolModule(modulePath string) bool {
    return strings.HasPrefix(modulePath, "contrib/tools/")
}
```

**Approach 2**: NO_PLATFORM() flag detection
```go
module.Properties["NOPLATFORM"] = "true"  // Set during macro expansion
```

### Change 2: Platform-Aware Module Selection
```go
func (gb *GraphBuilder) determinePlatformContexts(module *Module) []PlatformAwareContext {
    isTool := isToolModule(module.SourcePath) || module.Properties["NOPLATFORM"] == "true"

    if isTool {
        // Tools always build on host platform (x86_64)
        return []PlatformAwareContext{
            {ctx: gb.ctx, arch: PlatformX86_64},
        }
    }

    // Target modules build on specified target platform
    targetArch := gb.parseTargetPlatform()
    return []PlatformAwareContext{
        {ctx: gb.ctx, arch: targetArch},
    }
}
```

### Change 3: Remove platform="both" Assignments
**Replace**: `node.Platform = "both"` with specific platform assignments based on node type.

**For JS nodes** (code generation, architecture-independent):
```go
node.Platform = platformCtx.getPlatformString()  // Matches generating module
```

**For R6 nodes** (ragel6 is a tool):
```go
node.Platform = "default-linux-x86_64"  // Tools are x86_64
```

### Change 4: Tag Assignment for Tools
```go
func (gb *GraphBuilder) assignModuleTags(module *Module, node *GraphNode) {
    if isToolModule(module.SourcePath) {
        node.Tags = []string{"tool"}
    }
}
```

## Validation Strategy

### Test Case 1: Single Platform Build
```bash
# Generate graph with --target-platform=default-linux-aarch64
go run . --target-platform=default-linux-aarch64 /path/to/module

# Expected:
# - Target modules: aarch64 only
# - Tool modules: x86_64 only
# - NO platform="both" nodes
```

### Test Case 2: Tool Dependency Chain
```bash
# Build contrib/tools/yasm (tool node count verification)
go run . contrib/tools/yasm

# Expected:
# - All nodes on x86_64 platform
# - Tagged with "tool"
# - No aarch64 nodes
```

### Test Case 3: Target Platform with Tools
```bash
# Build tools/archiver (has tool dependencies)
go run . --target-platform=default-linux-aarch64 tools/archiver

# Expected:
# - Archiver nodes: aarch64
# - Tool chains (yasm, ragel6): x86_64
# - Exact platform split matching reference
```

## Edge Cases and Open Questions

### Question 1: Cross-Compilation Tools
**Scenario**: Tool runs on host but generates code for target.
**Resolution**: Build tool on host (x86_64), but it generates aarch64 binaries.

### Question 2: Dual-Architecture Libraries
**Scenario**: Library must exist in both architectures.
**Resolution**: Reference shows **no modules exist on both platforms** - each module is single-architecture. Dual-arch requirements are handled via separate builds, not single nodes.

### Question 3: Test Modules
**Scenario**: Tests may need to run on multiple platforms.
**Investigation Needed**: Reference graph doesn't include test modules (archiver build uses -k flag to skip tests).

### Question 4: Platform-Specific Source Files
**Scenario**: .S files for assembly, architecture-specific C files.
**Current Handling**: File discovery includes all files.
**Reference Behavior**: Individual CC nodes are per-platform.

## Dependencies

### T-134: ARCH Conditional Evaluation
- Platform selection uses ARCH conditionals conceptually
- Implementation may require ARCH conditional support
- However, NO_PLATFORM() flag is primary signal for tools

### T-61: Build Configuration Injection
- MUSL injection uses platform-aware logic
- Allocator selection varies by platform
- Platform selection must respect MUSL configuration

## Success Criteria

1. **Graph Equality**: Generated graph platform distribution matches reference (1933 aarch64 + 1797 x86_64)
2. **No "both" Nodes**: Eliminate all `platform="both"` assignments
3. **Tool Identification**: Correctly identify and tag tool modules
4. **Target Platform Respect**: Honor `--target-platform` flag for non-tool modules

## Deliverables

1. ✅ Platform distribution analysis (done)
2. ✅ NO_PLATFORM() directive documentation (done)
3. ⏳ Platform selection algorithm implementation
4. ⏳ Tool module detection logic
5. ⏳ Validation test suite
6. ⏳ Reference graph match verification

## Timeline Estimate

- Platform filtering implementation: 2 hours
- Tool detection logic: 1 hour
- Tag assignment: 30 minutes
- Remove "both" assignments: 30 minutes
- Validation tests: 1 hour
- **Total**: 5 hours

---

**Investigator**: T-140 digger
**Date**: 2026-05-11
**Status**: Analysis complete, implementation guidance provided
