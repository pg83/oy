# acceptance.md — bit-identical graph on `tools/archiver`

## 1. Scope

This document defines the binary acceptance test for our Go reimplementation
of the ya/ymake graph generator. Success is measured against the committed
reference graph `/home/pg/monorepo/yatool_orig/sg.json`, produced by
`srun.sh` for the target `tools/archiver` under
`--target-platform=default-linux-aarch64 --musl
--host-platform-flag=MUSL=yes --sandboxing -G -j0`. Other targets, other
flag axes, and partial-graph scenarios are out of scope of this acceptance
test (and may live in their own follow-up tests later).

## 2. Reference fixture

- Path: `/home/pg/monorepo/yatool_orig/sg.json` (~71 MB, committed).
- Regeneration: `cd /home/pg/monorepo/yatool_orig && bash srun.sh`. The
  committed fixture is authoritative — only regenerate if the reference tree
  itself changes. Verbatim from `srun.sh`:

  ```
  PATH=$PATH:$PWD/exp:/ix/realm/boot/bin python3 ./ya.py make -k --musl --host-platform-flag=MUSL=yes \
      --sandboxing \
      --target-platform=default-linux-aarch64 -G -j0 tools/archiver > sg.json
  ```

- Top-level shape:
  - `conf` (object) — graph-level metadata.
  - `graph` (array of node objects, length 3730 in the current fixture).
  - `inputs` (object, currently empty).
  - `result` (array of `uid` strings; currently length 1, the `LD` node for
    `tools/archiver/archiver`).
- Per-node fields seen across the fixture: `cmds`, `deps`, `env`,
  `foreign_deps`, `host_platform`, `inputs`, `kv`, `outputs`, `platform`,
  `requirements`, `sandboxing`, `self_uid`, `stats_uid`, `tags`,
  `target_properties`, `uid`. Not all keys are present on every node
  (`host_platform` and `foreign_deps` are optional).
- uid encodings:
  - `uid`, `self_uid`: 22-char URL-safe base64 of 16 raw bytes (no padding).
  - `stats_uid`: 32 lowercase hex chars (16 bytes — md5-shaped).
  - `deps[*]`, `foreign_deps.<bucket>[*]`, top-level `result[*]`: the same
    22-char `uid` form, referencing the `uid` of another node.

## 3. Success — what "match modulo uid renumbering" means

Two graphs (ours and the reference) are accepted as equivalent iff, after
applying the normalisation rules in §4 to both, the canonical-JSON encodings
are byte-identical.

The harness fails the test on any byte difference, prints a unified diff
(truncated to a few hundred lines), and reports counts: nodes only in ours,
nodes only in the reference, nodes whose canonical form differs.

## 4. Normalisation rules

Numbered, executable. Each rule applies to *both* sides of the diff:

1. **Strip volatile `conf` fields** that depend on host / run / VCS state
   and cannot reasonably match bit-for-bit:
   - `conf.gsid`
   - `conf.description.host`
   - `conf.description.platform`
   - `conf.description.user`
   - `conf.resources` — replace each entry's `resource` value (a base64
     blob embedding `BUILD_DATE`, `BUILD_TIMESTAMP`, `BUILD_HOST`, etc.)
     with the placeholder string `"<vcs-resource>"`. Keep `name` and
     `pattern`.

   These keys are deleted from the canonical form (or, for `resources`,
   masked). Rationale: they cannot match across runs; the rest of the
   graph is otherwise deterministic.

2. **Renumber uids consistently across both graphs.** For each node `n`,
   compute a *shape signature* `S(n)` via DFS:

   ```
   S(n) = sha256( contentBytes(n) || joined(sorted(S(d) for d in deps(n))) )
   ```

   where `contentBytes(n)` is the canonical-JSON encoding of `n` with
   `uid`, `self_uid`, `stats_uid`, `deps`, and `foreign_deps` removed
   (those are the uid-bearing fields, masked out so the shape depends only
   on intrinsic content plus dep shapes).

   - Cycles must not occur in a build graph; if they do, the harness fails
     with `cycle detected at uid <u>`.
   - A dep that does not point at a known node fails with
     `dangling dep uid <u>`.
   - Build a per-graph map `M: original_uid -> "U:" + hex(S(n))[:16]`. The
     16-hex prefix is short, stable, and human-greppable in diff output.
   - Apply `M` everywhere a uid string appears: each node's `uid`,
     `self_uid`, `stats_uid`, every entry of `deps`, every entry of any
     `foreign_deps.<bucket>` array, every entry of top-level `result`.

   Note: `self_uid` and `stats_uid` are rewritten to `M[node.uid]` —
   i.e. a single canonical id per node. The harness does not separately
   verify that the reference's `self_uid` / `stats_uid` algorithms are
   reproduced; that is `design_uid.md`'s job and a separate ticket's
   test. This intentionally hides correct-vs-incorrect uid algorithm
   differences inside the acceptance test, in exchange for a stable
   shape-only equivalence. The trade-off is explicit so reviewers see it.

3. **Sort the `graph` array** by canonical `uid` (post-renumbering),
   ascending lexicographic.

4. **Sort dependency arrays.** Each node's `deps` and any
   `foreign_deps.<bucket>` array are sorted lexicographic post-renumbering.

5. **Sort top-level `result`** lexicographic post-renumbering.

6. **Drop `null` and absent fields.** A key whose value is `null` or that
   is missing on one side is treated uniformly as absent. Optional fields
   (`host_platform`, `foreign_deps`, `env` when empty, etc.) are emitted
   only when present and non-empty, otherwise dropped — avoids cosmetic
   diffs from optional-key asymmetry.

7. **Canonical JSON encoding.** Sorted object keys, two-space indent,
   trailing newline, no extraneous whitespace. Implemented via an explicit
   recursive marshaller (so `map[string]any` keys are sorted).

These rules are *the* contract. The harness enforces them; any future
extension (e.g. a different target) extends or overrides this list.

## 5. Wall-time budget

- The builder must produce the graph for `tools/archiver` in **≤ 1.0 s** of
  wall-clock time, measured from immediately before the call into
  `BuildArchiverGraph` to the moment it returns.
- The measurement excludes test-harness setup (loading the reference, doing
  the diff). It includes everything the builder does internally — first
  read of any cache, all file IO, etc.
- The harness fails the test (separately from the bit-identical assertion)
  if wall time exceeds the budget. The two assertions are independent: a
  diff-clean run that takes 2 s still fails.
- For local development the budget can be relaxed by setting environment
  variable `ACCEPTANCE_TIME_BUDGET_MS=<integer>`. Default is `1000`. The
  harness logs the actual elapsed time always.

## 6. Locating the reference

- Default path: `/home/pg/monorepo/yatool_orig`.
- Override: env var `YATOOL_ORIG=/path/to/yatool_orig`. The harness reads
  the reference graph from `$YATOOL_ORIG/sg.json`.
- If the path is missing or unreadable, the test calls `t.Skip` with a
  clear message — `go test` on a fresh checkout without the reference tree
  still passes (the harness runs but skips). Acceptance must not break
  unrelated CI when the fixture is absent.

## 7. Builder integration

The harness calls a single Go function:

```go
var BuildArchiverGraph func(yatoolOrig string) ([]byte, error)
```

- Declared in `builder_iface.go` (initially nil). A future ticket (T-3+)
  ships the actual builder and assigns the function in an `init()`.
- If `BuildArchiverGraph == nil` when the test runs, the harness `t.Skip`s
  with `"BuildArchiverGraph not implemented yet"`. The harness ships green
  ahead of the builder and turns red as soon as a builder is wired up but
  does not yet match.
- The argument is the absolute path to the yatool_orig source tree. The
  return is the serialised JSON graph as raw bytes — so the harness can
  normalise both sides identically without round-tripping our internal
  representation.

## 8. Test layout — what `go test` runs

Three `*_test.go` functions, all in the workspace root, package `main`:

1. `TestNormalizeIdempotent` — load reference, normalise, normalise again,
   assert byte equality. A pure sanity check that normalisation is a fixed
   point. Skips if reference is missing.
2. `TestNormalizeReferenceSelfDiff` — load reference, normalise, normalise
   a fresh copy, assert byte equality. Catches non-determinism in the
   normaliser. Skips if reference is missing.
3. `TestAcceptanceArchiver` — the headline test. Skips if reference missing
   *or* `BuildArchiverGraph` nil. Otherwise:
   - times the call to `BuildArchiverGraph`,
   - normalises both sides,
   - asserts byte equality of canonical JSON,
   - asserts wall time ≤ budget.
   - On failure dumps the unified diff (truncated) into `t.Logf` plus
     summary counts.

## 9. Out of scope

- Verifying `self_uid` or `stats_uid` algorithm correctness — owned by
  `design_uid.md` and a separate test (a follow-up ticket can re-check that
  the reference uids match what our uid implementation produces, given
  known inputs).
- Multi-target tests, multiple platform axes, partial-graph diffs.
- Performance profiling beyond the single ≤ 1 s assertion.
