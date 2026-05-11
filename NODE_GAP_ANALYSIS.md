# Node Gap Analysis Post-T-131

## Executive Summary

After T-131's architecture-specific ASM file filtering changes, the current implementation generates **4131 nodes** for `tools/archiver` vs the reference **3730 nodes**, a remaining gap of **+401 nodes**.

This is **unchanged** from the pre-T131 baseline of 4131 nodes, indicating that T-131's ASM filtering did not significantly reduce the total node count. The gap remains substantial and requires multiple follow-up fixes.

## Validation Results

### Node Count Comparison

| Metric | Current | Reference | Gap |
|--------|---------|-----------|-----|
| **Total Nodes** | 4131 | 3730 | **+401** |
| CC | 4008 | 3571 | +437 |
| AS | 39 | 83 | -44 |
| AR | 67 | 48 | +19 |
| JS | 14 | 23 | -9 |
| LD | 2 | 3 | -1 |
| R6 | 1 | 1 | 0 |
| CP | 0 | 1 | -1 |

### Platform Distribution

| Platform | Current | Reference | Gap |
|----------|---------|-----------|-----|
| both | 15 | 0 | +15 |
| default-linux-aarch64 | 2040 | 1933 | +107 |
| default-linux-x86_64 | 2076 | 1797 | +279 |

## Detailed Gap Analysis

### 1. Dual-Platform Over-Generation (+307 extra nodes)

**Root Cause**: `NewPlatformContexts()` always generates for BOTH aarch64 and x86_64. The reference uses `--target-platform=default-linux-aarch64` so archiver deps only appear on aarch64.

**Impact**:
- **CC Over-Generation (+307 nodes)**: 307 CC nodes exist in current that shouldn't (x86_64 versions of archiver dependencies)
  - `contrib/restricted/abseil-cpp/default-linux-x86_64`: +156
  - `contrib/libs/jemalloc/default-linux-aarch64`: +63 (should be only aarch64)
  - `contrib/libs/mimalloc/default-linux-aarch64`: +16
  - `library/cpp/getopt/small/default-linux-x86_64`: +16
  - `contrib/libs/zlib/default-linux-x86_64`: +15
  - Plus 55 additional modules with 1-8 extra CC nodes each

- **AR Over-Generation (+19 nodes)**: 19 AR nodes exist in current that shouldn't (x86_64 versions):
  - 26 modules have AR on both platforms instead of just aarch64

**Fix Required**: Modify `NewPlatformContexts()` to respect `--target-platform` flag instead of always generating for both platforms.

### 2. Architecture-Specific CC Node Generation (+298 extra nodes)

**Root Cause**: Per-architecture SRCS conditional resolution gap. Modules with extensive ARCH conditionals (e.g., `contrib/libs/cxxsupp/builtins`) generate ALL source nodes instead of platform-specific subsets.

**Impact**:
- **builtins CC mismatch**:
  - `contrib/libs/cxxsupp/builtins/default-linux-aarch64`: curr=316, ref=161, **+155**
  - `contrib/libs/cxxsupp/builtins/default-linux-x86_64`: curr=316, ref=173, **+143**

**Expected Fix**: T-131's architecture-specific ASM filtering should have addressed this, but the node count shows no reduction, indicating the fix was incomplete or only applied to AS nodes, not CC nodes.

**Other CC count mismatches**:
- `contrib/libs/tcmalloc/no_percpu_cache/default-linux-aarch64`: curr=1, ref=54, **-53** (severe under-generation)
- `util/default-linux-aarch64`: curr=4, ref=21, **-17**
- `contrib/libs/musl/default-linux-aarch64`: curr=1297, ref=1314, **-17**
- `contrib/libs/cxxsupp/libcxx`: Both platforms +6 each
- `contrib/libs/libc_compat/default-linux-aarch64`: +2

### 3. Missing Host Tool Modules (-88 CC, -2 LD)

**Root Cause**: Host tool modules (`contrib/tools/ragel6`, `contrib/tools/yasm`) are not loaded as build dependencies. These tools generate JS nodes for .rl6 processing.

**Impact**:
- **contrib/tools/yasm**: -79 CC, -1 AS, -1 LD (all x86_64)
- **contrib/tools/ragel6**: -9 CC, -1 LD (all x86_64)
- **JS node impact**: These missing tools explain part of the JS node gap (-9 JS)

**Fix Required**: Implement host tool module loading and dependency chain traversal.

### 4. AS Node Gap (-44 AS nodes)

**Root Cause**: Architecture-specific .S files not generated correctly, or missing `musl/full` dependencies.

**Missing AS modules**:
| Module | Platform | Gap |
|--------|----------|-----|
| `contrib/libs/asmglibc` | x86_64 | -1 |
| `contrib/libs/asmlib` | x86_64 | -25 |
| `contrib/libs/cxxsupp/builtins` | aarch64 | -4 |
| `contrib/libs/musl` | aarch64 | -13 |
| `contrib/libs/tcmalloc/no_percpu_cache` | aarch64 | -1 |
| `util` | aarch64 | -1 |

**Extra AS node**: `contrib/libs/nayuki_md5/default-linux-x86_64`: +1 (should not exist)

**Root Causes**:
- **musl/full not loaded**: `asmlib` and `asmglibc` are PEERDIR dependencies of `contrib/libs/musl/full`. The `musl/full` module is not traversed despite `--musl` flag.
- **Builtins aarch64 gap**: Architecture-specific .S files in builtins for aarch64 not generated (T-131 may have partially fixed this).
- **tcmalloc and util aarch64**: Architecture-specific .S files missing.

### 5. JS Node Gap (-9 JS nodes)

**Root Cause**: Platform assignment bug. JS nodes get `platform=both` instead of being assigned to the target platform.

**Impact**:
- Current: 14 JS nodes with `platform=both`
- Reference: 23 JS nodes with `platform=default-linux-aarch64`
- Gap: -9 JS nodes

**Analysis**: The reference has 23 JS nodes on aarch64. Current has 14 JS nodes on `both`. This suggests:
1. Missing JS nodes from host tools (ragel6, yasm) not loaded
2. Platform assignment error: JS nodes should go on target platform, not `both`

**Fix Required**: Fix JS node platform assignment in graph builder to use target platform.

### 6. Missing Node Types

| Type | Missing |
|------|---------|
| CP | -1 |
| LD | -1 (additional, already missing 2) |

**CP**: Missing 1 CP node (likely file copy operation)
**LD**: Missing 1 additional LD node beyond the 2 we have (ragel6 or yasm LD)

### 7. AR Node Gap (+19 extra, -9 missing)

**AR Over-Generation (+19)**: Same dual-platform issue as CC nodes.

**AR Under-Generation (-9)**: Missing AR for:
- `build/cow/on` (aarch64, x86_64): -2
- `contrib/libs/linuxvdso` (aarch64, original/aarch64): -2
- `contrib/libs/musl/full` (aarch64, x86_64): -2
- `contrib/libs/asmglibc` (x86_64): -1
- `contrib/libs/musl_extra` (x86_64): -1
- `contrib/libs/nayuki_md5` (aarch64): -1

## Root Cause Summary Table

| Gap Category | Gap Count | Root Cause | Status |
|--------------|-----------|------------|--------|
| CC dual-platform over-generation | +307 | NewPlatformContexts always both platforms | Unfixed |
| CC ARCH conditional mismatch (builtins) | +298 | Per-platform SRCS resolution incomplete | Partial fix by T-131 |
| Missing host tools (ragel6, yasm) | -88 CC, -2 LD | Tools not loaded as build deps | Unfixed |
| Missing musl/full deps | -3 AR, -26 AS | musl/full not traversed with MUSL=yes | Unfixed |
| AS ARCH mismatch (builtins aarch64, musl aarch64) | -17 AS | Per-platform .S file resolution | Partial fix by T-131 |
| JS platform assignment bug | -9 JS | platform=both vs aarch64 | Unfixed |
| Missing CP/LD nodes | -2 | Missing host tools (ragel6, yasm) | Unfixed |

**Total Gap Calculation**:
+307 (CC dual-platform)
+298 (CC ARCH mismatch)
+19 (AR dual-platform)
+15 (platform=both)
-88 (missing host tools CC)
-44 (missing AS)
-9 (JS gap)
-2 (missing CP/LD)
= +401 (matches observed gap of +401)

## Recommended Follow-up Tickets

1. **Fix NewPlatformContexts() to respect --target-platform flag**
   - High priority - largest impact (+307 nodes)
   - Single target platform for archiver deps, host platform for tool modules
   - Estimated effort: medium

2. **Load musl/full and its dependencies when MUSL=yes**
   - Resolves missing asmlib, asmglibc AS nodes (-26 AS)
   - Resolves missing musl/full AR nodes (-2 AR)
   - Estimated effort: low-medium

3. **Fix JS node platform assignment**
   - Should assign to target platform, not `both`
   - Estimated effort: low

4. ** Implement host tool module loading**
   - Load ragel6 and yasm as build dependencies
   - Resolves missing -88 CC, -2 LD, -9 JS
   - Estimated effort: medium-high

5. **Complete per-platform SRCS resolution for CC nodes**
   - T-131 may have partially addressed this, but gap remains (+298 builtins CC)
   - Need to ensure ARCH conditionals are evaluated per-platform during module loading
   - Estimated effort: high

## Notes on T-131 Effectiveness

T-131 added architecture-specific ASM file filtering in commit 424c6c5. However, the analysis shows:

1. **Total node count unchanged**: 4131 nodes before and after T-131
2. **AS node gap persists**: Still 39 AS vs 83 reference (gap -44)
3. **Builtins aarch64 AS still missing**: 4 builtins AS nodes missing for aarch64

This suggests T-131's fix was either:
- Only applied to a subset of modules (not builtins)
- Only applied to AS nodes, not CC nodes (explaining why builtins CC gap persists)
- Incomplete in its implementation

The ASM filtering fix should reduce both CC and AS over-generation for architecture-specific modules, but the data shows no change in total node count.

## Conclusion

The current implementation has made significant progress (module-level working, 4131 nodes vs 3730 reference), but substantial gaps remain:

- **401 extra nodes** total (+13%
 over reference)
- **Dual-platform over-generation** is the largest single cause (+307 CC + 19 AR + 15 platform=both)
- **Missing host tools** and **musl/full** account for missing nodes (-88 CC, -26 AS, -9 JS)
- **Platform assignment bug** for JS nodes
- **Per-platform SRCS resolution** still incomplete for CC nodes

The next tickets should prioritize the dual-platform fix (NewPlatformContexts) and host tool loading, as these have the highest impact and clearest resolution path.
