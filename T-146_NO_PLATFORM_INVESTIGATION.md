# NO_PLATFORM() Directive Investigation Report

## Executive Summary

The `NO_PLATFORM()` directive is a ya.make macro defined in `/home/pg/monorepo/yatool_orig/build/ymake.core.conf` that controls platform-specific dependencies for modules. This report documents the mechanism, semantics, and provides a concrete implementation specification for the Go reimplementation.

## Reference Implementation Analysis

### Macro Definition

Location: `/home/pg/monorepo/yatool_orig/build/ymake.core.conf` (line ~4385)

```conf
### @usage: NO_PLATFORM()
###
### Exclude dependencies on C++ and C runtimes (including util, musl and libeatmydata) and set NO_PLATFORM variable for special processing.
### Note: use this with care. libc most likely will be linked into executable anyway,
### so using libc headers/functions may not be detected at build time and may lead to unpredictable behavors at configure time.
macro NO_PLATFORM() {
    NO_LIBC()
    ENABLE(NOPLATFORM)
}
```

### Macro Composition

`NO_PLATFORM()` is a compound macro that calls:

1. **`NO_LIBC()`**:
   ```conf
   macro NO_LIBC() {
       NO_RUNTIME()
       DISABLE(MUSL)
       ENABLE(NOLIBC)
   }
   ```

2. **`NO_RUNTIME()`**:
   ```conf
   macro NO_RUNTIME() {
       DISABLE(NORUNTIME)
   }
   ```

3. **`ENABLE(NOPLATFORM)`**: Sets the `NOPLATFORM` flag for conditional processing

### Semantic Effect

When `NO_PLATFORM()` is called in a ya.make file:

1. **Dependency Exclusion** (via `NO_LIBC()`):
   - Excludes dependencies on C++ runtime (contrib/libs/cxxsupp)
   - Excludes dependencies on C runtime (contrib/libs/musl)
   - Excludes dependencies on libeatmydata
   - Excludes dependencies on util library

2. **Flag Setting** (via `ENABLE(DEFAULT Allocator)`):
   - Sets `NOPLATFORM = yes` in module variables
   - This flag is checked in `ymake.core.conf` conditional blocks

3. **Platform Peerdirs** (interaction with `NEED_PLATFORM_PEERDIRS`):
   - Sets `NEED_PLATFORM_PEERDIRS = no` (via `_BARE_MODULE()` macro which also calls `NO_PLATFORM()`)
   - Prevents automatic platform dependency injection

### Conditional Processing in Reference

The `NOPLATFORM` flag is checked in `ymake.core.conf`:

```conf
# Allocator linking
when ($DARWIN == "yes" && $NOPLATFORM != "yes") {
    PEERDIR += contrib/libs/cxxsupp
}

# Assembly library injection
when ($MSVC != "yes" && $NOPLATFORM != "yes" && $WITH_VALGRIND != "yes" && ...) {
    PEERDIR+=contrib/libs/glibcasm
    # or contrib/libs/asmlib
}
```

### Related Macros

#### `NO_PLATFORM_RESOURCES()`

```conf
### @usage: NO_PLATFORM_RESOURCES() # internal
### Exclude dependency on platform resources libraries.
macro NO_PLATFORM_RESOURCES() {
    ENABLE(NOPLATFORM_RESOURCES)
}
```

Used by `build/platform/linux_sdk/ya.make` to exclude platform resource dependencies.

#### Other Related Macros

- `NO_COMPILER_WARNINGS()`: Disables compiler warnings
- `NO_RUNTIME()`: Excludes runtime dependencies
- `NO_UTIL()`: Excludes util library
- `NO_LTO()`: Disables Link-Time Optimization
- `NO_CODENAVIGATION()`: Disables code navigation

## Usage Pattern Analysis

### Common Users

1. **`contrib/libs/musl/ya.make`**:
   - `NO_PLATFORM()` + `NO_RUNTIME()` + `NO_COMPILER_WARNINGS()`
   - Purpose: C library, no platform-specific dependencies needed

2. **`contrib/libs/cxxsupp/ya.make`**:
   - `NO_PLATFORM()` + conditional STL selection
   - Purpose: C++ standard library support

3. **`contrib/libs/cxxsupp/builtins/ya.make`**:
   - `NO_PLATFORM()` + `NO_COMPILER_WARNINGS()` + `NO_RUNTIME()`
   - Purpose: Compiler builtins

4. **`contrib/libs/asmglibc/ya.make`**:
   - `NO_PLATFORM()`
   - Purpose: Assembly library

5. **`contrib/libs/linux-headers/ya.make`**:
   - `NO_PLATFORM()`
   - Purpose: Kernel headers

### Usage Pattern

Modules that:
- Provide platform-agnostic functionality (libc, compiler support, headers)
- Are themselves platform dependencies
- Need explicit control over dependency injection

## Implementation Specification for Go

### 1. Lexer Enhancement

**File**: `lexer.go`

**Action**: Ensure `NO_PLATFORM`, `NO_PLATFORM_RESOURCES`, `NO_COMPILER_WARNINGS`, `NO_RUNTIME`, `NO_UTIL`, `NO_LTO` are recognized keywords.

These are likely already recognized since they appear as bare function calls (like `PROGRAM()`, `LIBRARY()`).

### 2. Parser Enhancement

**File**: `parser_yamake.go`

**Action**: Add handlers for these directives in the parse loop (around line 139-246).

Implementation:

```go
case "NO_PLATFORM":
    if p.tok.Token != IDENTIFIER || p.tok.Val != "(" {
        return nil, p.errorExpected("NO_PLATFORM()")
    }
    p.next()
    if p.tok.Token != IDENTIFIER || p.tok.Val != ")" {
        return nil, p.errorExpected("NO_PLATFORM()")
    }
    p.next()
    if module != nil {
        module.AddProperty("NOPLATFORM", "yes")
        module.AddProperty("NEED_PLATFORM_PEERDIRS", "no")
        module.AddProperty("NORUNTIME", "yes")
        module.AddProperty("MUSL", "no")
        module.AddProperty("NOLIBC", "yes")
    }

case "NO_PLATFORM_RESOURCES":
    // Similar handler
    module.AddProperty("NOPLATFORM_RESOURCES", "yes")

case "NO_COMPILER_WARNINGS":
    module.AddProperty("NOCOMPILERWARNINGS", "yes")

case "NO_RUNTIME":
    module.AddProperty("NORUNTIME", "yes")

case "NO_UTIL":
    module.AddProperty("NOUTIL", "yes")

case "NO_LTO":
    module.AddProperty("NOLTO", "yes")
```

### 3. AST Enhancement

**File**: `ast.go`

**Action**: Properties are already stored in `Module.Properties map[string]string`, no structural changes needed.

### 4. Build Configuration Enhancement

**File**: `build_config.go`

**Action**: No changes needed - properties are stored in module-level properties, not build config.

### 5. Dependency Resolution Enhancement

**File**: `transitive.go` or `graph_builder.go`

**Action**: Implement conditional PEERDIR injection based on module properties.

```go
func (gb *GraphBuilder) injectPlatformDependencies(module *Module, ctx *BuildConfig) {
    if module.HasProperty("NOPLATFORM") {
        return
    }

    // Inject asmlib
    if shouldInjectAsmlib(module, ctx) {
        module.AddDependency("contrib/libs/asmlib")
    }

    // Inject linux-headers
    if shouldInjectLinuxHeaders(module, ctx) {
        module.AddDependency("contrib/libs/linux-headers")
    }

    // Inject cxxsupp
    if shouldInjectCxxsupp(module, ctx) {
        module.AddDependency("contrib/libs/cxxsupp")
    }

    // Inject util
    if shouldInjectUtil(module, ctx) {
        module.AddDependency("util")
    }
}

func shouldInjectAsmlib(module *Module, ctx *BuildConfig) bool {
    // Implement condition: MSVC != yes && NOPLATFORM != yes && WITH_VALGRIND != yes && USE_ASMLIB != no && MIC_ARCH != yes && PIC != yes && PIE != yes
    // Plus platform-specific checks for glibcasm vs asmlib
    return false // Placeholder
}

func shouldInjectLinuxHeaders(module *Module, ctx *BuildConfig) bool {
    // Implement condition: OS_LINUX && NEED_PLATFORM_PEERDIRS == yes
    return ctx.OS == "linux" && module.GetProperty("NEED_PLATFORM_PEERDIRS") == "yes"
}

func shouldInjectCxxsupp(module *Module, ctx *BuildConfig) bool {
    // Implement condition: NORUNTIME != yes
    return module.GetProperty("NORUNTIME") != "yes"
}

func shouldInjectUtil(module *Module, ctx *BuildConfig) bool {
    // Implement condition: NOUTIL != yes
    return module.GetProperty("NOUTIL") != "yes"
}
```

### 6. PEERDIR Filtering Enhancement

**Reference**: `module_builder.cpp:589` - `ExcludedPeerdirs` check

**Implementation**: The Go reimplementation uses a different model (module-level dependency injection), but similar filtering may be needed:

```go
func (gb *GraphBuilder) shouldIncludePeerdir(module *Module, peerdirPath string, ctx *BuildConfig) bool {
    // Check for excluded peerdirs if needed
    // For now, this is handled by conditional injection
    return true
}
```

## Test Strategy

### Unit Tests

1. **Parser Tests** (`parser_yamake_test.go`):
   ```go
   TestModuleNoPlatform,
   TestModuleNoPlatformResources,
   TestModuleNoCompilerWarnings,
   TestModuleNoRuntime,
   TestModuleNoUtil,
   TestModuleNoLTO,
   ```

2. **Property Tests**:
   ```go
   func TestNoPlatformProperties(t *testing.T) {
       // Verify NO_PLATFORM sets all expected properties
   }
   ```

### Integration Tests

1. **Musl Module Test** (`musl_standalone_test.go`):
   ```go
   func TestMuslNoPlatformDependencies(t *testing.T) {
       // Build contrib/libs/musl and verify no platform deps
   }
   ```

2. **Cxxsupp Module Test** (`musl_standalone_test.go`):
   ```go
   func TestCxxsuppNoPlatformDependencies(t *testing.T) {
       // Build contrib/libs/cxxsupp and verify behavior
   }
   ```

### Comparison Tests

1. **Graph Comparison**:
   ```bash
   # Build reference with NO_PLATFORM modules
   cd /home/pg/monorepo/yatool_orig
   ./srun.sh

   # Build Go implementation
   cd /home/pg/monorepo/oy/w/workspaces/ws-2026-05-11-004854-0037
   go run . --graph-file=go_sg.json /home/pg/monorepo/yatool_orig/tools/archiver

   # Compare graphs
   python3 compare_graphs.py sg.json go_sg.json
   ```

## Implementation Checklist

- [ ] Parser handles `NO_PLATFORM()` directive
- [ ] Parser handles related directives (NO_PLATFORM_RESOURCES, NO_COMPILER_WARNINGS, etc.)
- [ ] Module properties correctly set (NOPLATFORM, NORUNTIME, etc.)
- [ ] NEED_PLATFORM_PEERDIRS handling
- [ ] Platform dependency injection based on properties
- [ ] Unit tests for parsing
- [ ] Unit tests for dependency injection
- [ ] Integration tests with musl module
- [ ] Integration tests with cxxsupp module
- [ ] Graph comparison with reference
- [ ] Documentation updates

## Priority Modules

For initial validation, focus on:

1. `contrib/libs/musl/ya.make` - Complex case with NO_PLATFORM + NO_RUNTIME + conditional architecture includes
2. `contrib/libs/cxxsupp/ya.make` - STL selection logic
3. `contrib/libs/cxxsupp/builtins/ya.make` - Compiler builtins
4. `contrib/libs/linux-headers/ya.make` - Simple NO_PLATFORM case

## Interactions with Other Features

### MUSL Flag

The `--musl` / `--host-platform-flag=MUSL=yes` flag interacts with NO_PLATFORM:

- `NO_PLATFORM()` includes `DISABLE(MUSL)`, which sets MUSL property to "no"
- However, MUSL can still be explicitly added via:
  ```conf
  IF (MUSL)
      ADDINCL(contrib/libs/musl/arch/x86_64)
  ENDIF()
  ```
  (as seen in `contrib/libs/cxxsupp/builtins/ya.make`)

### Platform Flags

NO_PLATFORM affects:
- Compiler platform injection (when: `COMPILER_PLATFORM && NEED_PLATFORM_PEERDIRS == yes`)
- Linker selection (when: `... && NOPLATFORM != "yes"`)

### Allocator

Allocator selection is independent of NO_PLATFORM but respects need for platform dependencies.

## Known Gaps and Open Questions

1. **ExcludedPeerdirs Mechanism**: Reference uses global `ExcludedPeerdirs` hash set to filter PEERDIR statements. Current Go implementation uses conditional injection model. May need clarification on exact semantics.

2. **Platform Resources**: NO_PLATFORM_RESOURCES() usage in linux_sdk needs investigation.

3. **Foreign Platform**: KeepTargetPlatform flag in reference for foreign platform builds - not yet implemented in Go.

4. **Dependency Management**: Reference has complex DEPENDENCY_MANAGEMENT system - not yet clear if NO_PLATFORM interacts with this.

## Next Steps for Implementation

1. Implement parser handlers for NO_PLATFORM() and related directives
2. Implement property setting in Module struct
3. Implement conditional dependency injection in GraphBuilder
4. Write unit tests
5. Test against musl module
6. Compare generated graphs with reference
7. Iterate to achieve graph equality

## References

- Reference implementation: `/home/pg/monorepo/yatool_orig`
- Core configuration: `/home/pg/monorepo/yatool_orig/build/ymake.core.conf`
- Module attributes: `/home/pg/monorepo/yatool_orig/devtools/ymake/module_state.h`
- PEERDIR filtering: `/home/pg/monorepo/yatool_orig/devtools/ymake/module_builder.cpp:576-614`
