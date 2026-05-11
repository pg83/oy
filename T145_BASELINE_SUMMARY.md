# T-145 Baseline Verification

## Summary

Successfully verified trunk baseline for `tools/archiver` graph generation with MUSL flags.

**Results:**
- ✅ Node count confirmed: **4131 nodes** (matches NODE_GAP_ANALYSIS.md)
- ✅ Performance confirmed: **383.85ms median** (92.9µs/node, well under 1s target)
- ✅ Node type distribution: CC=4008, AS=39, AR=67, JS=14, LD=2, R6=1, CP=0
- ✅ Platform distribution: both=1, aarch64=2040, x86_64=2076, linux[JS]=14
- ✅ Gap vs reference: **+401 nodes** (current 4131 vs reference 3730)

## Deliverables

1. **`baseline_sg.json`**: Full graph JSON (8.5MB) for 4131-node baseline
2. **`BASELINE_VERIFICATION_T145.md`**: Complete baseline verification report
3. **`PERFORMANCE_BASELINE.md`**: Updated with 4131-node baseline section

## Documentation

- `BASELINE_VERIFICATION_T145.md`: Trunk commit 999ab98, full node distribution, performance metrics, known gaps
- `PERFORMANCE_BASELINE.md`: Added T-145 baseline section, confirmed linear scaling (92.9µs/node)

## Validation

- ✅ Go tests pass
- ✅ Go vet passes
- ✅ Formatting correct
- ✅ Graph generation completes without errors
- ✅ Baseline node count matches NODE_GAP_ANALYSIS.md

## Next Steps

Follow-on tickets should prioritize:
1. Fix `NewPlatformContexts()` dual-platform over-generation (+307 CC, +19 AR)
2. Implement host tool module loading (-88 CC, -2 LD, -9 JS)
3. Load `musl/full` and dependencies with MUSL=yes (-26 AS)
4. Complete per-platform SRCS resolution for CC nodes (+298 CC builtins)

---

**Trunk Commit:** 999ab98d4137325bc980df76f41ba772e12f3c56
**Verification Date:** 2026-05-11
**Ticket:** T-145
