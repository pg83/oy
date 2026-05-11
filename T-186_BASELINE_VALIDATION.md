# T-186 Baseline Validation Report

## Objective
Generate fresh archiver graph with `--musl --host-platform-flag=MUSL=yes` and document exact node count and type breakdown by platform, identifying remaining platform filtering issues.

## Command Used
```bash
go run . -G --graph-file=T-186_BASELINE.json --musl --host-platform-flag=MUSL=yes /home/pg/monorepo/yatool_orig/tools/archiver
```

## Summary Statistics

### Total Node Count
- **Reference:** 3730 nodes
- **Generated:** 3799 nodes
- **Delta:** +69 nodes (+1.8%)

### Node Type Distribution

| Type | Reference | Generated | Delta | Status |
|------|-----------|-----------|-------|--------|
| CC   | 3571      | 3672      | +101  | ❌ Over |
| AS   | 83        | 39        | -44   | ❌ Under |
| AR   | 48        | 71        | +23   | ❌ Over |
| JS   | 23        | 14        | -9    | ❌ Under |
| LD   | 3         | 2         | -1    | ❌ Under |
| R6   | 1         | 1         | 0     | ✅ Match |
| CP   | 1         | 0         | -1    | ❌ Under |

### Platform Distribution

| Platform                | Reference | Generated | Delta | Status |
|-------------------------|-----------|-----------|-------|--------|
| default-linux-aarch64   | 1933      | 1874      | -59   | ❌ Under |
| default-linux-x86_64    | 1797      | 1925      | +128  | ❌ Over |
| **Total**               | **3730**  | **3799**  | **+69**|        |

### Per-Platform Node Type Breakdown

#### aarch64 Platform
| Type | Reference | Generated | Delta | Status |
|------|-----------|-----------|-------|--------|
| AR   | 33        | 35        | +2    | ❌ Over |
| AS   | 21        | 2         | -19   | ❌ Under |
| CC   | 1853      | 1836      | -17   | ❌ Under |
| CP   | 1         | 0         | -1    | ❌ Under |
| JS   | 23        | 0         | -23   | ❌ Under |
| LD   | 1         | 1         | 0     | ✅ Match |
| R6   | 1         | 0         | -1    | ❌ Under |
| **Total** | **1933** | **1874** | **-59**| |

#### x86_64 Platform
| Type | Reference | Generated | Delta | Status |
|------|-----------|-----------|-------|--------|
| AR   | 15        | 36        | +21   | ❌ Over |
| AS   | 62        | 37        | -25   | ❌ Under |
| CC   | 1718      | 1836      | +118  | ❌ Over |
| JS   | 0         | 14        | +14   | ❌ Over |
| LD   | 2         | 1         | -1    | ❌ Under |
| R6   | 0         | 1         | +1    | ❌ Over |
| **Total** | **1797** | **1925** | **+128**| |

## Key Findings

### 1. NO_PLATFORM Module Investigation
**Finding:** ✅ NO_PLATFORM modules do NOT appear on both architectures

Both reference and generated graphs have **0 nodes with NO_PLATFORM indicators**. The `NO_PLATFORM()` directive in `contrib/libs/asmlib/ya.make` is parsed and handled, but it does not create NO_PLATFORM nodes in the execution graph. Instead, it affects dependency injection (e.g., excludes util/musl dependencies).

**Contradiction to Ticket Assumption:**
The ticket assumed NO_PLATFORM modules would appear on "both arches" as NO_PLATFORM nodes. This is incorrect. NO_PLATFORM is a build configuration directive that affects dependency resolution, not a node platform classification.

### 2. asmlib AS Node Analysis
**Finding:** asmlib generates 25 AS nodes on x86_64 only (reference behavior)

| Metric | Reference | Generated |
|--------|-----------|-----------|
| asmlib AS nodes | 25 | 0 |
| Platform distribution | 25 on x86_64 | 0 on all platforms |

**Why 25, not 50?**
The `contrib/libs/asmlib/ya.make` file contains:
- `NO_PLATFORM()` directive (line 17)
- `IF (ARCH_X86_64)` block (lines 21-66)
- 25 x86_64-specific `.asm` files (lines 28-62)
- `dummy.c` file (lines 106-108) - always compiled

**Reference Behavior:**
The reference ymake generates:
- 25 AS nodes (for .asm files) on x86_64 only
- 2 CC nodes (for dummy.c) on both platforms
- 0 NO_PLATFORM nodes

**Interpretation:**
`NO_PLATFORM()` does NOT mean "no platform" in the sense of platform-agnostic nodes. It means "no platform-specific runtime dependencies" (excluding util/musl). For assembly files, the reference implementation generates AS nodes on the build platform only (x86_64 in this case), not on both architectures. This is semantically: "build-time tool, run on build arch, cross-compile to target arch."

**Generated Behavior:**
The current Go implementation generates 0 AS nodes for asmlib. AS node generation for `.asm` and `.S` files is not yet implemented.

### 3. Platform Filtering Issues
**Finding: ❌ Platform imbalance in CC/AR nodes**

**CC Nodes:**
- x86_64 over: +118 nodes
- aarch64 under: -17 nodes

**AR Nodes:**
- x86_64 over: +21 nodes
- aarch64 over: +2 nodes

**Analysis:**
The symmetric distribution of CC nodes (1836 each platform) vs the asymmetric reference (1853 aarch64, 1718 x86_64) suggests:
1. Double-computation for x86_64-only modules
2. Missing platform filtering for modules that should only appear on one platform
3. Incorrect module-to-platform mapping in dependency resolution

**Hypothesis:**
The `determinePlatformContexts()` function may not correctly handle module boundaries for architecture-specific modules, or platform-aware dependency resolution may not respect conditional compilation blocks.

### 4. JS Node Platform Assignment
**Finding: ❌ JS nodes missing on aarch64, appearing on x86_64**

**Reference Distribution:**
- aarch64: 23 JS nodes
- x86_64: 0 JS nodes

**Generated Distribution:**
- aarch64: 0 JS nodes
- x86_64: 14 JS nodes

**Analysis:**
JS nodes are appearing on the wrong platform. The reference has JS nodes on aarch64 only (likely for tool modules), but the generated graph has them on x86_64. This suggests a platform assignment error in JS node generation or dependency resolution.

### 5. AS Node Under-Generation
**Finding: ❌ 44 AS nodes missing**

**Breakdown:**
- asmlib AS: 25 nodes missing (x86_64 only)
- Other AS: 19 nodes missing (mixed platforms)

**Root Cause:**
AS node generation for `.asm` and `.S` source files is not implemented in the Go codebase. The graph builder currently only generates CC nodes, ignoring assembly source files.

### 6. LD/CP Node Gaps
**Finding: ❌ Missing final tool chain nodes**

- LD: -1 node (reference 3, generated 2)
- CP: -1 node (reference 1, generated 0)

**Analysis:**
These are likely final linker and copy-to-output operations for host tools. Their absence suggests incomplete graph generation for the final build phase.

## Prioritized Issues

### P0 - AS Node Generation (Foundation)
**Impact:** -44 nodes, including all 25 asmlib AS nodes
**Action:** Implement AS node generation for `.asm` and `.S` files
- Add AS node type to graph builder
- Detect `.asm` and `.S` source files in modules
- Assign AS nodes to correct platforms based on ARCH conditionals
- Handle asmlib NO_PLATFORM semantics (x86_64-only AS)

### P1 - Platform Filtering (CC/AR Balance)
**Impact:** +128 total platform imbalance
**Action:** Fix platform boundary detection
- Audit CC/AR node generation for double-counting
- Verify `determinePlatformContexts()` handles module boundaries correctly
- Check if module-level conditionals (IF/ELSE) properly propagate to source nodes
- Investigate asymmetric reference distribution (why aarch64 has 1853 CC vs x86_64 1718)

### P2 - JS Node Platform Correction
**Impact:** -14 JS nodes on incorrect platform
**Action:** Fix JS node platform assignment
- Verify JS nodes inherit correct platform from parent module
- Check if platform-aware dependency resolution works for tool modules
- Ensure JS nodes for contrib/tools/ragel6 appear on aarch64 (reference matches this)

### P3 - LD/CP Node Restoration
**Impact:** -2 nodes
**Action:** Implement final build phase nodes
- Restore 1 missing LD node
- Restore 1 missing CP node
- These are likely final tool chain operations for host tools

## AS vs NO_PLATFORM Semantics Clarification

**Corrected Understanding:**

1. **NO_PLATFORM()** (ya.make directive):
   - Affects dependency injection (excludes util/musl runtime dependencies)
   - Does NOT create NO_PLATFORM nodes in the execution graph
   - Example: asmlib has NO_PLATFORM() but generates 25 AS nodes on x86_64 only

2. **AS Nodes** (execution graph nodes):
   - Generated for `.asm` and `.S` source files
   - Platform assigned based on ARCH conditionals in ya.make
   - Reference behavior: AS nodes appear on build platform only for NO_PLATFORM modules
   - Example: asmlib generates 25 AS nodes on x86_64 only (build arch)

3. **Node Platform Fields** (execution graph JSON):
   - `platform` field: actual target platform (default-linux-aarch64, default-linux-x86_64)
   - `target_properties.platform` field: same as platform field
   - No NO_PLATFORM value exists in the platform field

## Next Steps for Implementation

### Week 1: AS Node Implementation
1. Implement AS node detection in `determinePlatformContexts()`
2. Add AS node generation code in graph builder
3. Test asmlib AS node generation (expected: 25 nodes on x86_64)
4. Restore remaining 19 AS nodes from other modules

### Week 2: Platform Filtering Fixes
1. Audit module-to-platform mapping logic
2. Fix double-computation of x86_64-only modules
3. Restore correct CC/AR node balance
4. Validate platform distribution matches reference

### Week 3: Final Integration
1. Fix JS node platform assignment
2. Restore LD/CP nodes
3. Run strict graph validation
4. Update documentation

## Delivered Artifacts

1. **T-186_BASELINE.json** - Fresh archiver graph (3799 nodes)
2. **analyze_baseline.py** - Graph analysis tool using kv.p field
3. **T-186_BASELINE_VALIDATION.md** - This report
4. **TestFinalGraphValidationWithReport** - Ongoing validation test

## Validation Commands

```bash
# Run baseline validation
go test -v -run TestFinalGraphValidationWithReport -timeout 60s

# Analyze graphs manually
python3 analyze_baseline.py T-186_BASELINE.json

# Generate fresh graph
go run . -G --graph-file=T-186_BASELINE.json --musl --host-platform-flag=MUSL=yes /home/pg/monorepo/yatool_orig/tools/archiver

# Run full validation suite
./validate.sh
```
