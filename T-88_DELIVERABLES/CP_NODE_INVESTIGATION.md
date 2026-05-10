# CP Node Generation Investigation

## Current Status

- **Expected**: 1 CP node
- **Generated**: 0 CP nodes
- **Gap**: 1 node (100% missing)

## Implementation Status

### Completed Components

1. **createCPNode() Implementation**
   - Location: `graph_builder.go:604`
   - Called at: `graph_builder.go:267` for copy operations
   - Status: Code exists, generates 0 nodes

2. **PEERDIR Traversal** (T-75 MERGED)
   - Confirmed working 100%
   - Not the root cause

## Root Cause Analysis

### Blocker Location

The CP generation path is blocked at the **module loading stage**, not at the graph builder implementation.

### Primary Blocker: MUSL Module Not Loading

The CP node should be generated for copying `musl.py` from `contrib/libs/musl`, but:
- **MUSL modules are not loading** into the module registry
- Build config injection (T-61) does not trigger MUSL inclusion
- contrib/libs/musl directory not parsed/processed

## Expected CP Node Generation

Based on reference analysis:
- **Source**: musl.py copy operation
- **Origin**: contrib/libs/musl module
- **Target**: build output
- **Count**: 1 node (single copy operation)

## MUSL Loading Investigation

### Current State

```bash
# Check if MUSL modules are in registry
ls -la modules/contrib/libs/musl/* 2>/dev/null
# Expected: "No such file or directory" - MUSL not loaded

# Verify build config exists
cat /home/pg/monorepo/yatool_orig/build/ymake.core.conf | grep -i musl
```

### Why MUSL Not Loading

1. **Build Config Injection** (T-61 DONE)
   - Code exists to inject build/ymake.core.conf
   - Config should trigger MUSL module inclusion
   - **Issue**: Config evaluation not triggering module loading

2. **INCLUDE Parsing** (T-77 NEEDED)
   - MUSL modules require INCLUDE directives to resolve
   - Build config uses INCLUDE to reference MUSL paths
   - Missing INCLUDE parsing prevents MUSL resolution

3. **Conditional Evaluation**
   - MUSL inclusion may have conditional guards
   - Platform flags (OS_LINUX, etc.) may not match
   - Build context may not have required macros

## Dependencies and Blockers

### Blocking Components

1. **INCLUDE Parsing** (T-77 NEEDED)
   - Build config references MUSL via INCLUDE
   - Without INCLUDE parsing, paths not resolved
   - MUSL ya.make files never loaded

2. **Build Config Conditional Evaluation**
   - Config may have conditionals around MUSL inclusion
   - Requires platform flags, macro evaluation
   - May be evaluating to false in current context

3. **Conditional PEERDIR Resolution**
   - Even if MUSL loads, conditional PEERDIRs may exclude it
   - Platform-specific MUSL variants may not select

### Depends On

- **T-61**: Build config injection (DONE, but not triggering MUSL)
- **T-77**: INCLUDE parsing (CRITICAL - not implemented)
- **T-53**: MUSL analysis ticket (identifies 2400+ missing nodes)

## Priority Assessment

**LOW PRIORITY** (1 node, 0.03% of gap)

This is a symptom of the larger MUSL loading issue:
- Module loading is the real problem
- CP node will appear when MUSL loads
- Fixing MUSL loading (2400+ nodes) will naturally fix CP node
- Not worth independent debugging effort

## Estimated Impact

- **MUSL Stack**: ~2400 nodes (CC: 2000+, AS: 60+, CP: 1)
- **Priority**: HIGH for MUSL stack overall
- **Priority**: LOW for CP node alone

## Recommendation

DO NOT debug CP node independently. Fix MUSL loading first:
1. Implement INCLUDE parsing (T-77)
2. Debug build config conditional evaluation
3. Verify MUSL modules load into registry
4. CP node will appear automatically as side effect

The CP node is a **canary** for MUSL loading, not a standalone issue to fix.
