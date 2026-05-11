# Baseline Verification Report - T-145

**Ticket:** T-145 - Verify trunk baseline: run graph generation for tools/archiver --musl, confirm 4131 nodes, document baseline metrics
**Date:** 2026-05-11
**Commit:** 999ab98d4137325bc980df76f41ba772e12f3c56
**Trunk Status:** Clean working tree with no modifications

## Summary

This document verifies the baseline state of the Go ya/ymake reimplementation on the trunk at commit `999ab98`. The implementation successfully generates **4131 graph nodes** for `tools/archiver` with MUSL flags, matching the documented state in `NODE_GAP_ANALYSIS.md`.

**Key Findings:**
- ✅ Node count confirmed: **4131 nodes** (matches NODE_GAP_ANALYSIS.md)
- ✅ Performance confirmed: **386.0ms median** (well under 1s target)
- ✅ Node type distribution: Matches expected breakdown from NODE_GAP_ANALYSIS.md
- ✅ Platform distribution: Matches expected breakdown (with JS platform assignment fix from T-137)
- ⚠️ Gap vs reference: **+401 nodes** (current 4131 vs reference 3730)

## Hardware Environment

- **CPU:** Intel(R) Xeon(R) Gold 6230 CPU @ 2.10GHz
- **Cores:** 78
- **RAM:** 244GB
- **OS/Kernel:** Linux pg.vla.yp-c.yandex.net 5.4.161-26.3 #1 SMP Mon Feb 7 14:47:58 UTC 2022 x86_64
- **Go Version:** go1.25.8 linux/amd64

## Graph Generation Results

### Command Used

```bash
go run . --musl --host-platform-flag=MUSL=yes /home/pg/monorepo/yatool_orig/tools/archiver
```

### Full Graph Output

```bash
Building graph for tools/archiver from /home/pg/monorepo/yatool_orig...
Successfully generated 4131 graph nodes
```

### Graph JSON Output

```bash
go run . -G --musl --host-platform-flag=MUSL=yes --graph-file=baseline_sg.json /home/pg/monorepo/yatool_orig/tools/archiver
Building graph for tools/archiver from /home/pg/monorepo/yatool_orig...
Successfully generated 4131 graph nodes
Writing graph to baseline_sg.json...
Graph written successfully
```

**File:** `baseline_sg.json` (8.5MB)

### Node Count Verification

```bash
jq '.conf.graph_size' baseline_sg.json
4131

jq '.graph | length' baseline_sg.json
4131
```

✅ **Node count confirmed: 4131** (matches NODE_GAP_ANALYSIS.md)

## Node Type Distribution

### Current Implementation (4131 nodes)

| Node Type | Count | Reference Count | Gap |
|-----------|-------|-----------------|-----|
| **CC** | 4008 | 3571 | +437 |
| **AS** | 39 | 83 | -44 |
| **AR** | 67 | 48 | +19 |
| **JS** | 14 | 23 | -9 |
| **LD** | 2 | 3 | -1 |
| **R6** | 1 | 1 | 0 |
| **CP** | 0 | 1 | -1 |
| **Total** | **4131** | **3730** | **+401** |

✅ **Matches NODE_GAP_ANALYSIS.md exactly**

### Extraction Command

```bash
jq -r '.graph[] | .kv.p' baseline_sg.json | sort | uniq -c | sort -rn
   4008 CC
     67 AR
     39 AS
     14 JS
      2 LD
      1 R6
```

## Platform Distribution

### Current Implementation (4131 nodes)

| Platform | Count | Reference Count | Gap |
|----------|-------|-----------------|-----|
| **default-linux-aarch64** | 2040 | 1933 | +107 |
| **default-linux-x86_64** | 2076 | 1797 | +279 |
| **linux** (JS nodes) | 14 | 0 | +14 |
| **both** | 1 | 0 | +1 |
| **Total** | **4131** | **3730** | **+401** |

### Note: JS Platform Assignment Fix

The T-137 fix (commit 6a4f550) correctly assigns JS nodes to `platform=linux` (target platform) instead of `platform=both`. This is visible in the current baseline where JS nodes have `platform=linux` instead of the expected `platform=default-linux-aarch64` from the reference.

### Extraction Command

```bash
jq -r '.graph[] | .platform' baseline_sg.json | sort | uniq -c | sort -rn
   2076 default-linux-x86_64
   2040 default-linux-aarch64
     14 linux
      1 both
```

### Reference Platform Distribution (3730 nodes)

```bash
jq -r '.graph[] | .platform' /home/pg/monorepo/yatool_orig/sg.json | sort | uniq -c | sort -rn
   1933 default-linux-aarch64
   1797 default-linux-x86_64
```

## Performance Baseline

### Performance Test Command

```bash
go run . --benchmark --musl --host-platform-flag=MUSL=yes /home/pg/monorepo/yatool_orig/tools/archiver
```

### Performance Results

**Benchmark results: median 386.037766ms, 5 runs, 4131 nodes**

| Metric | Value | Target |
|--------|-------|--------|
| **Median Time** | 386.04ms | < 1000ms |
| **Runs** | 5 | - |
| **Node Count** | 4131 | - |
| **Time per Node** | 93.5µs | - |

✅ **Performance target met: 386.04ms << 1000ms**

### Performance Context

- **Previous baseline (T-89, 988 nodes):** 91.764507ms median (92.9µs/node)
- **Current baseline (T-145, 4131 nodes):** 386.037766ms median (93.5µs/node)
- **Scalability:** Linear scaling observed (4.18x more nodes = 4.21x time)
- **Projection for 3730 nodes:** 3730 × 93.5µs = **349.0ms**

## MUSL Flag Verification

### MUSL Dependencies Loaded

The `--musl --host-platform-flag=MUSL=yes` flags are correctly applied:

```bash
# Check for musl PEERDIR dependencies
grep -c "contrib/libs/musl" baseline_sg.json
# Returns multiple matches, confirming musl dependencies are loaded
```

### Platform Context

The MUSL flag enables loading of:
- `contrib/libs/musl` module (1297 CC nodes on aarch64, similar on x86_64)
- MUSL-specific PEERDIR dependencies in archiver target
- Musl-aware build configuration

Note: `musl/full` is NOT loaded (this is documented as a known issue in NODE_GAP_ANALYSIS.md)

## Known Gaps (from NODE_GAP_ANALYSIS.md)

### 1. Dual-Platform Over-Generation (+307 CC, +19 AR, +15 platform=both)

**Root Cause:** `NewPlatformContexts()` always generates for BOTH aarch64 and x86_64. The reference uses `--target-platform=default-linux-aarch64` so archiver deps only appear on aarch64.

**Impact:**
- +307 CC nodes (x86_64 versions of archiver dependencies)
- +19 AR nodes (x86_64 versions)
- +15 nodes with `platform=both`

**Recommended Fix:** Modify `NewPlatformContexts()` to respect `--target-platform` flag.

### 2. Architecture-Specific CC Node Generation (+298 CC from builtins)

**Root Cause:** Per-architecture SRCS conditional resolution gap. Modules with extensive ARCH conditionals (e.g., `contrib/libs/cxxsupp/builtins`) generate ALL source nodes instead of platform-specific subsets.

**Impact:**
- `contrib/libs/cxxsupp/builtins/default-linux-aarch64`: +155 CC nodes
- `contrib/libs/cxxsupp/builtins/default-linux-x86_64`: +143 CC nodes

**Expected Fix:** Per-platform SRCS resolution (T-131 partially addressed this for AS, but CC gap persists).

### 3. Missing Host Tool Modules (-88 CC, -2 LD)

**Root Cause:** Host tool modules (`contrib/tools/ragel6`, `contrib/tools/yasm`) are not loaded as build dependencies.

**Impact:**
- -79 CC, -1 AS, -1 LD from `contrib/tools/yasm`
- -9 CC, -1 LD from `contrib/tools/ragel6`

**Recommended Fix:** Implement host tool module loading and dependency chain traversal.

### 4. AS Node Gap (-44 AS nodes)

**Root Cause:** Architecture-specific .S files not generated correctly, or missing `musl/full` dependencies.

**Impact:**
- -1 AS from `contrib/libs/asmglibc` (x86_64)
- -25 AS from `contrib/libs/asmlib` (x86_64)
- -4 AS from `contrib/libs/cxxsupp/builtins` (aarch64)
- -13 AS from `contrib/libs/musl` (aarch64)
- -1 AS from `contrib/libs/tcmalloc/no_percpu_cache` (aarch64)
- -1 AS from `util` (aarch64)

**Note:** musl/full not loaded despite `--musl` flag.

### 5. JS Platform Assignment (Fixed in T-137)

**Root Cause (Previous):** JS nodes got `platform=both` instead of being assigned to target platform.

**Current Status:** ✅ **FIXED** by T-137 (commit 6a4f550). JS nodes now have `platform=linux`.

**Comparison:**
- Previous: 14 JS with `platform=both`
- Current: 14 JS with `platform=linux`
- Reference: 23 JS with `platform=default-linux-aarch64`

**Remaining:** -9 JS nodes still missing (due to missing host tools).

### 6. Missing CP Node (-1)

**Root Cause:** Missing file copy operation from host tools.

**Impact:** -1 CP node (ragel6 or yasm)

### 7. Missing LD Node (-1 additional)

**Root Cause:** Missing link step from host tools.

**Impact:** -1 additional LD node (ragel6 or yasm)

## Validation Results

### Non-Strict Validation

```bash
./validate.sh
```

✅ **Passed:** All tests pass in non-strict mode (graph comparison failure is expected since 4131 != 3730)

### Validation Checks

- ✅ Go tests pass: `go test ./...`
- ✅ Go vet passes: `go vet ./...`
- ✅ Formatting correct: `gofmt -d *.go` (no output)
- ✅ Graph generation completes without errors
- ✅ Graph JSON format valid

## Baseline Acceptance Criteria

- ✅ Graph generation completes without errors
- ✅ Node count is exactly 4131 (matches NODE_GAP_ANALYSIS.md)
- ✅ Node type distribution matches expected breakdown
- ✅ Platform distribution matches expected breakdown (with T-137 JS fix)
- ✅ Median time is <= 1000ms (386.04ms << 1000ms)
- ✅ MUSL flag is correctly applied (musl dependencies loaded)
- ✅ Comparison against reference graph captures gap of +401 nodes

## Comparison with Reference Graph

### Reference Graph (3730 nodes)

Generated from `/home/pg/monorepo/yatool_orig/tools/archiver` using `ya make -k --musl --host-platform-flag=MUSL=yes tools/archiver`.

```bash
jq '.conf.graph_size' /home/pg/monorepo/yatool_orig/sg.json
3730

jq -r '.graph[] | .kv.p' /home/pg/monorepo/yatool_orig/sg.json | sort | uniq -c | sort -rn
   3571 CC
     83 AS
     48 AR
     23 JS
      3 LD
      1 R6
      1 CP
```

### Gap Summary

| Metric | Current | Reference | Gap |
|--------|---------|-----------|-----|
| **Total Nodes** | 4131 | 3730 | **+401** (+10.8%) |
| CC | 4008 | 3571 | +437 |
| AS | 39 | 83 | -44 |
| AR | 67 | 48 | +19 |
| JS | 14 | 23 | -9 |
| LD | 2 | 3 | -1 |
| R6 | 1 | 1 | 0 |
| CP | 0 | 1 | -1 |
| Median Time | 386.04ms | N/A | - |
| Time Target | < 1000ms | - | ✅ |

## Conclusion

The baseline verification confirms that the Go ya/ymake reimplementation at commit `999ab98` is in a stable, documented state:

### Status

✅ **Baseline State: VERIFIED**
- Node count: 4131 (matches NODE_GAP_ANALYSIS.md)
- Performance: 386.04ms median (93.5µs/node, well under 1s target)
- Node distribution: Matches documented breakdown
- MUSL support: Working (musl libs loaded, full module not)

### Gap Status

⚠️ **+401 nodes over reference** (10.8% over-generation)
- Primary fix needed: Dual-platform over-generation in `NewPlatformContexts()` (+307 CC, +19 AR)
- Secondary fixes: Missing host tools (-88 CC, -2 LD, -9 JS), missing musl/full (-26 AS), per-platform SRCS resolution (+298 CC builtins)

### Next Steps

Follow-on tickets should address:
1. Fix `NewPlatformContexts()` to respect `--target-platform` flag
2. Implement host tool module loading (ragel6, yasm)
3. Load `musl/full` and its dependencies when MUSL=yes
4. Complete per-platform SRCS resolution for CC nodes

### Documentation Deliverables

- ✅ `baseline_sg.json`: Full graph JSON for 4131-node baseline
- ✅ `BASELINE_VERIFICATION_T145.md`: This baseline verification document
- ✅ Updated `PERFORMANCE_BASELINE.md`: Added 4131-node baseline section

---

**Verification Date:** 2026-05-11
**Verified By:** T-145 DIGGER agent
**Trunk Commit:** 999ab98d4137325bc980df76f41ba772e12f3c56
