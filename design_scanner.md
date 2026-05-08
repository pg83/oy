# design_scanner.md — C/C++/proto include scanner

This is the design for the **include scanner** layer of the Go reimplementation.
The scanner takes a translation-unit-like file (a `.cpp/.h/.proto/...`),
extracts its dependency edges (`#include`, `import`), resolves each one to a
concrete file in the source tree, and feeds the resolved set to the graph
builder. It is the inner loop of acceptance: every reachable source under
`tools/archiver` and its three `PEERDIR`s passes through it.

This document specifies the **scanner only** — the parser is
`design_parser.md`, the graph is `design_graph.md`, the in-memory
representation is `design_repr.md`. Sysincl rule loading and `ADDINCL` data
flow are read from project config; their construction lives outside this file
but their consumption is described here.

## Acceptance constraints that shape the design

Two from `GOALS.md`:

- bit-identical to `sg.json` modulo uid renumbering — so resolution order
  and tie-breaks are not a free parameter; we mirror ymake's behaviour;
- ≤ 1 second wall time — so the scanner must be (a) parallel across
  translation units, (b) cheap per-file (single linear pass, no regexes
  inside the hot loop), (c) deduplicated (each file scanned once even
  though many translation units include it).

Two further structural constraints from `CLAUDE.md`:

- `ya.make` is a multimodule, so the scanner's input context (ADDINCLs,
  language) depends on (module, target platform, language axis); the cache
  key must include that axis;
- the system must be ready for direct in-memory execution — so the scanner
  produces in-memory `ResolvedInclude` records, not JSON, and the graph
  builder consumes them in place.

## Reference behaviour (what we are reproducing)

`devtools/ymake/include_parsers/cpp_parser.cpp` (lexer) and
`devtools/ymake/include_processors/cpp_processor.cpp` (classifier) define
the C/C++ surface; `proto_parser.cpp` / `proto_processor.cpp` define the
proto surface. `module_resolver.cpp::ResolveSingleInclude` defines
resolution; `sysincl_resolver.cpp::Resolve` defines sysincl. Read those
four files before changing this design.

The contract is:

1. **Lex**. Read the file as bytes. Walk lines up to
   `INCLUDE_LINES_LIMIT=60000` (`devtools/ymake/options/static_options.h`).
   For C/C++: skip leading whitespace; if the first non-whitespace character
   is `#`, try to parse `#include <…>` or `#include "…"` (and the bracketed
   `Y_UCRT_INCLUDE_NEXT(…)` / `Y_MSVC_INCLUDE_NEXT(…)` macro forms — these
   are skipped, see cpp_processor.cpp:35-38). Strip a trailing line comment
   (`// …`). For proto: same scaffolding but the prefix is `import` and the
   accepted quoted forms are `"…"` and `'…'`. Trailing `;` is stripped.
2. **Classify**. Each parsed string becomes an `Include{Kind,Path}` where
   `Kind` is `Local` (quoted), `System` (angle-bracket), or `Macro`
   (anything that didn't match either, used downstream by sysincl macro
   resolution — `BOOST_PP_ITERATE()` and friends). `Path` is the literal
   text between the delimiters; no normalisation here.
3. **Resolve**. For each include in source order, run the cascade:
   1. If the include path is a "known root" (starts with `$S/`, `$B/`,
      `arcadia/`, etc.) — resolve directly against build/source roots.
   2. Else, if `Kind == Local`, try the source file's own directory
      (`srcDir`) first. On hit — done.
   3. Else, walk the module's `IncDirs.Get(langId)` collection — that is
      the language-specific stack: own ADDINCLs first, then peerdir-induced
      ADDINCLs (see addincls.h:127-150 for the priority order). On hit —
      candidate is the addincl result.
   4. Run sysincl in parallel with step 3. Sysincl matches lower-cased
      include path against the global rule table (`build/sysincl/*.yml`),
      filtered by the source-file-path regex on the rule. A sysincl match
      can produce zero (system include — no edge) or one or many target
      files (each becomes an edge).
   5. **Sysincl is authoritative when it matches**: if both addincl and
      sysincl resolve, the addincl result must equal one of the sysincl
      targets, otherwise warn. If only sysincl matches, drop addincl.
      If neither matches, emit an "unresolved" stub so the graph still has
      the edge. See module_resolver.cpp:381-417 for the exact merge rule.
4. **Hand off**. The resolved set is appended (as a sequence of file IDs
   plus a small `Kind` tag for "induced/h+cpp" propagation) to the
   in-memory node for the source file under construction.

There is no preprocessor evaluation. `#if`/`#ifdef`/`#define` are ignored
entirely — the scanner over-approximates includes. This matches ymake.

## Scope of this scanner — languages we cover for tools/archiver

The acceptance target `tools/archiver` plus the transitive closure of its
three `PEERDIR`s (`library/cpp/archive`, `library/cpp/digest/md5`,
`library/cpp/getopt/small`, plus `contrib/libs/nayuki_md5`,
`library/cpp/string_utils/base64`, `library/cpp/colorizer`) contains only
C and C++ sources — `.cpp`, `.h`, `.hpp`, `.cc`, `.c`. Proto / asm / flatc
/ ragel / cython / fortran / swig / etc. do **not** appear.

We therefore implement two parsers in this ticket — **cpp** and **proto** —
and stub all other extensions to "no parser". The proto parser is in scope
even though no proto file appears in the closure: the resolver/sysincl
pipeline is shared, the parser is trivially cheap, and exercising it
de-risks the multimodule axis (proto produces both a native edge and an
induced `.pb.h` edge — that exact two-output shape is what makes the data
model worth getting right early). All other parsers from
`devtools/ymake/include_parsers/` are explicitly out of scope and the code
must `Throw` if a file with one of those extensions appears in the
archiver closure (it shouldn't, ever — fail loud).

## Data shape (Go)

All types live flat in the workspace root next to `throw.go`. New files:
`scanner.go` (entry points), `scan_cpp.go` (cpp lexer), `scan_proto.go`
(proto lexer), `sysincl.go` (sysincl rule loader + resolver), `resolver.go`
(addincl + sysincl merge), `inccache.go` (the parse cache).

```go
type IncludeKind uint8

const (
    IncludeLocal  IncludeKind = iota // "..."
    IncludeSystem                    // <...>
    IncludeMacro                     // BOOST_PP_ITERATE()
)

type RawInclude struct {
    Kind IncludeKind
    Path string // exactly the bytes between delimiters
}

// FileID is a stable interned ID for a path. The interner lives in the
// symbol table (see design_repr.md). Scanner returns FileIDs; never raw
// strings in the hot path.
type FileID uint32

const FileIDUnresolved FileID = 0 // sentinel — emit an "unresolved" graph stub

type ResolvedEdge struct {
    Target FileID
    // Source for diagnostics only. Not part of the graph hash.
    Origin EdgeOrigin
}

type EdgeOrigin uint8

const (
    OriginLocal EdgeOrigin = iota // resolved against srcDir
    OriginAddincl                 // resolved against an ADDINCL stack entry
    OriginSysincl                 // resolved by sysincl rule
    OriginUnresolved              // neither matched — placeholder
    OriginKnownRoot               // started with $S/, $B/, etc.
)

// Output of one scan-and-resolve pass over one file.
type ScanResult struct {
    Native []ResolvedEdge // direct edges (the file's own includes)
    // Induced is the per-language extra props the processor emits — for proto,
    // this is the .pb.h "h+cpp" set; for cpp it is empty.
    Induced []ResolvedEdge
}
```

Two notes:

- `RawInclude.Path` is intentionally a `string`, not `[]byte`. Most
  includes hit the cache after the first occurrence; the few that miss
  pay one allocation each — fine inside the 1 s budget given the closure
  is ~hundreds of files. Don't try to make the lexer zero-alloc until the
  perf doc says it's needed.
- `FileIDUnresolved` (= 0) lets the graph builder distinguish "we tried
  and failed" from "we never tried". Reserve ID 0 in the symbol table.

## Cache keying

Two distinct caches, with distinct keys, distinct lifetimes. Don't merge.

### 1. Parse cache (scanner-internal)

Maps **content of one file** → its `[]RawInclude`. Pure lex, no resolution.

```go
type parseKey struct {
    file FileID    // the source file being scanned
    parser ParserID // cpp / proto / ...
}

type parseCache struct {
    mu sync.Mutex
    m  map[parseKey][]RawInclude
}
```

Keying is `(FileID, ParserID)`, **not** content hash. Files don't change
during a single ymake run — see ymake's parsers_cache.h:79-95 for the same
choice. ParserID is in the key so two languages over the same file (rare:
the same `.h` reused under a non-cpp processor) produce different cached
results. ParserID encoding follows
`devtools/ymake/include_processors/parser_id.h`: 16 low bits = type, the
rest = version. Bumping a parser's version invalidates its cached entries
across runs (we don't yet persist this cache; bumping is for future
on-disk reuse).

A `sync.Map` looks tempting but the cache value is only ever written once
per key — a plain `map` guarded by `sync.Mutex` with a per-key
`sync.Once`-style filling protocol is simpler and just as fast for our
scale. Use `singleflight.Group` keyed by `parseKey` to coalesce concurrent
misses on the same file.

### 2. Resolve cache (per module)

Maps **(include path, resolution-plan key)** → resolved file ID. This
mirrors `TResolveCacheKey` in module_resolver.cpp:301:

```go
type resolveKey struct {
    path        string // RawInclude.Path verbatim
    planKey     uint64 // see below
}

type resolveResult struct {
    target  FileID
    origin  EdgeOrigin
    status  resolveStatus // Success / Missing / Absolute / NotFound
}
```

`planKey` encodes "which list of directories did we search":

- `0` — roots only (build root + source root). For known-root and
  `Kind==System` paths.
- `((srcDirElemID << 32) | withRootsBit)` — src-dir-relative search,
  optionally with roots appended. For local quoted includes.
- `((numIncDirs << 32) | (langID << 0) | (loadedBit << 31) | 2)` —
  ADDINCL search; `numIncDirs` plus `langID` is enough because `IncDirs`
  is append-only during module construction (module_resolver.cpp:284-294
  spells out the same trick). The `+2` keeps this disjoint from the
  `(srcDir, withRoots)` family.

The resolve cache lives on the **module** (one module = one combination of
target platform × language axis). Different modules built from the same
`ya.make` get different caches because their ADDINCL stack and language
differ; that's the multimodule axis from CLAUDE.md spelled out in cache
shape. Mutex, not sync.Map — same reasoning as the parse cache.

### What the keys deliberately do NOT include

- File mtime / content hash. Inputs are immutable for the run.
- Sysincl rules. Rules are loaded once per run; they are part of the
  per-process state, not the cache key.
- Compiler flags / defines. We don't preprocess.

If any of these change, the run is re-invoked from the top — ymake takes
the same view.

## Concurrency model

The scanner is consumed by the graph builder, which walks modules.
Concurrency lives at two levels:

1. **Across files within a module**. A module's source set is fanned out
   to a worker pool of `runtime.GOMAXPROCS(0)` goroutines. Each worker
   pulls a `(FileID, langID)` job, lexes if needed, resolves against the
   module's `IncDirs`, and writes the `ScanResult` into a per-job slot the
   builder reads back. Workers share the parse cache (process-global) and
   the resolve cache (module-scoped, passed as a pointer).

2. **Across modules**. Each module gets its own goroutine in the graph
   builder. Modules share the parse cache and the sysincl rules. They do
   **not** share the resolve cache — see above.

Goroutine-safety contract:

- `parseCache` — `sync.Mutex` + `singleflight`. Reads after fill are
  lock-free reads of an immutable slice (`[]RawInclude` is never mutated
  after insert).
- `resolveCache` — single mutex per module. Contention is bounded because
  each module's worker pool size is small (≤ GOMAXPROCS) and resolves
  are mostly cache hits after the first translation unit.
- `sysinclResolver` — built once at startup, then immutable. `Resolve`
  takes only the input strings; no shared state. This differs from the
  C++ implementation, which has mutable scratch buffers
  (sysincl_resolver.h:58-59) — we avoid that by allocating per call (the
  result is small) or by using a per-goroutine `*sysinclScratch`.

Per STYLE.md, every goroutine entry function wraps its body in `Try` and
either logs via `Catch` or fails the whole run; no panic crosses a
goroutine boundary.

## Sysincl handling — concrete design

Sysincl rules come from the YAML files under `build/sysincl/`. We load
them up-front, in Go, into a single immutable table.

```go
type sysinclRule struct {
    include       string   // header path (lowercased for the lookup key)
    caseSensitive bool     // when true, include must equal rule.include exactly
    srcFilter     *regexp.Regexp // nil ⇒ matches any source
    targets       []string // 0..N replacement paths (0 ⇒ ignore include)
}

type sysinclTable struct {
    // Multimap: lowercased include → all rules with that include.
    byInclude map[string][]sysinclRule
}

func (t *sysinclTable) Resolve(srcPath, include string) (targets []string, matched bool)
```

Steps:

- Strip type prefixes from `include` (the C++ does
  `NPath::ResolveLink` then `NPath::CutType` —
  sysincl_resolver.cpp:9-11). Lowercase. Look up in `byInclude`. Miss ⇒
  return `nil, false`.
- For each rule: if `caseSensitive` and the rule's stored mixed-case form
  differs from the input, skip; if `srcFilter` is non-nil and does not
  match `srcPath`, skip; else append `rule.targets` to the result and set
  `matched=true`.
- Return. `matched && len(targets)==0` is the "system include — no edge"
  case.

Why YAML if STYLE.md says "JSON only"? STYLE.md applies to **our** config.
The sysincl files are reference inputs from the upstream tree — we read
them as-is (one-shot, at startup). Pulling in `gopkg.in/yaml.v3` for this
single boundary is cheaper than maintaining a converted shadow tree; the
parsing is paid once per process (~55 small files, < 5 ms total). If even
that proves too costly we pre-convert to JSON in a build step, but not
yet.

The 55 files in `build/sysincl/` are not all loaded — the active set is a
function of target platform. For our golden invocation
(`--target-platform=default-linux-aarch64 --musl
--host-platform-flag=MUSL=yes --sandboxing`) the active set is determined
by `build/conf/sysincl/*` config; `design_graph.md` specifies the platform
flag mapping. The scanner just consumes whatever list the configuration
layer hands it.

### Macro resolves recursively

When `Kind == Macro` (e.g. `BOOST_PP_ITERATE()`), sysincl is the only
source of truth. Looking up the macro form in `byInclude` returns one or
more **regular include paths**, and each of those is then **re-resolved
through the full pipeline** (cpp_processor → resolver, including a fresh
sysincl lookup if it's a system header). This recursion has no preset
depth limit in ymake; in practice it is at most two hops. Track depth and
`Throw` past 8 to catch loops in misconfigured rule sets.

## Peerdir interaction

The scanner does not walk `PEERDIR` itself — that is the graph builder's
job (`design_graph.md`). It does, however, **consume** the union of
ADDINCLs that flow up through PEERDIRs.

The contract from the graph builder to the scanner is one input value:
`*ModuleIncDirs` — already populated with the module's own + peerdir-
propagated ADDINCLs, in priority order, language-keyed. The scanner reads
it; it does not mutate it.

ADDINCL priority (mirrors addincls.h:140-148):

1. own ADDINCLs for this language;
2. peerdir-induced ADDINCLs for this language;
3. own ADDINCLs for the default language (`C_LANG`);
4. peerdir-induced ADDINCLs for default.

`ModuleIncDirs.Get(langID)` returns these four lists chained, in that
order. The resolver walks the chain and stops at the first hit. The chain
type is the Go equivalent of `TIterableCollections<TDirs>` from
addincls.h:25-125 — a thin slice-of-slices with a forward iterator;
nothing exotic.

Languages we encounter in the closure are only `C_LANG` (cpp) — but the
data path supports multiple. When `langID == BY_SRC` (the default
sentinel), the scanner picks the language by source extension (cpp for
`.cpp/.cc/.h/.hpp/.c`); ditto module_resolver.cpp:267-277.

Out of scope for this ticket: the GLOBAL/UserGlobal/Local distinction
inside `TLanguageIncDirs` (addincls.h:152-168). For the archiver target,
all directories on the chain are treated as a single ordered list; the
finer-grained UserGlobal-vs-Global distinction matters only for command
rendering, not scanning. If a future target needs it, extend
`ModuleIncDirs.Get` rather than the scanner.

## Resolution algorithm — final concrete form

```
resolve(src, inc, langID, mod, sysincl, cache) → []ResolvedEdge:
    if mod.DontResolveIncludes:
        return [unresolved(inc.Path)]

    if knownRoot(inc.Path):
        return [resolveAgainstRoots(inc.Path)]

    switch inc.Kind {

    case Local:
        // 1. srcDir first, with optional roots fall-through.
        hit := cache.lookupOrFill(planSrcDir(srcDir(src), withRoots(langID)),
                                   inc.Path, () → searchSrcDir(src, inc.Path))
        if hit.success { return [{hit.target, OriginLocal}] }
        // fall through to ADDINCL+sysincl

    case System:
        // 2. roots only.
        hit := cache.lookupOrFill(planRoots, inc.Path,
                                   () → searchRoots(inc.Path))
        if hit.success { return [{hit.target, OriginKnownRoot}] }
        // fall through to ADDINCL+sysincl

    case Macro:
        // 3. sysincl only — and recurse on each target.
        targets, matched := sysincl.Resolve(srcPath(src), inc.Path)
        if !matched { error: "Can't resolve macro target"; return [] }
        out := []
        for t in targets {
            out = append(out, resolve(src, {System, t}, langID, mod, sysincl, cache)...)
        }
        return out
    }

    // ADDINCL + sysincl merge (shared by Local and System).
    incDirs := mod.IncDirs.Get(langID)
    addinclHit := nil
    if len(incDirs) > 0 {
        addinclHit = cache.lookupOrFill(planAddincl(len(incDirs), langID, mod.Loaded),
                                         inc.Path, () → searchAddincl(inc.Path, incDirs))
    }
    sysTargets, sysMatched := sysincl.Resolve(srcPath(src), inc.Path)

    if !sysMatched {
        if addinclHit != nil && addinclHit.success {
            return [{addinclHit.target, OriginAddincl}]
        }
        return [unresolved(inc.Path)]
    }

    out := []
    addinclInsideSysincl := false
    for t in sysTargets {
        r := resolveAgainstRoots(t)
        if r.success {
            if addinclHit != nil && addinclHit.success && r.target == addinclHit.target {
                addinclInsideSysincl = true
            }
            out = append(out, {r.target, OriginSysincl})
        } else {
            out = append(out, {unresolved(t), OriginUnresolved})
        }
    }
    if addinclHit != nil && addinclHit.success && !addinclInsideSysincl {
        // ymake warns and still emits the addincl edge — we mirror it.
        out = append(out, {addinclHit.target, OriginAddincl})
    }
    return out
```

This is module_resolver.cpp:247-418 transliterated. Keep the structure
and the order verbatim — every reorder risks a bit-flip in the golden
graph.

## API the rest of the system sees

```go
// scanner.go

// Scanner holds process-global state: parse cache + sysincl table +
// language registry. One Scanner per ymake run.
type Scanner struct {
    parseCache *parseCache
    sysincl    *sysinclTable
    langs      *languageRegistry
}

func NewScanner(sysinclConfigs []string) *Scanner

// ScanContext is what the graph builder hands in for each file.
type ScanContext struct {
    File     FileID
    LangID   LangID            // BY_SRC means "infer from extension"
    Module   *Module           // gives us IncDirs, attrs, resolveCache
    SrcRoot  string            // arcadia root abs path
    BldRoot  string            // build root abs path
}

func (s *Scanner) Scan(ctx ScanContext) ScanResult
```

Implementation flow:

```
Scan(ctx):
    parser := s.langs.parserFor(ctx.File, ctx.LangID)
    if parser == nil:
        return ScanResult{} // no parser → no edges (e.g. binary blob)

    raws := s.parseCache.lookupOrFill(parseKey{ctx.File, parser.ID},
                                       () → parser.Lex(read(ctx.File)))

    edges := []ResolvedEdge{}
    induced := []ResolvedEdge{}
    for r in raws:
        edges = append(edges, resolve(ctx.File, r, ctx.LangID, ctx.Module,
                                       s.sysincl, ctx.Module.resolveCache)...)
    induced = parser.Induced(raws, ctx.Module) // proto: emits .pb.h ; cpp: nil

    return ScanResult{Native: edges, Induced: induced}
```

`parser.Lex` and `parser.Induced` are the two language hooks. For cpp,
`Induced` returns nil. For proto, `Induced` walks the raw includes and
emits, for each, an unresolved-by-design edge naming `<basename>.pb.h`
(see proto_processor.cpp:43-56 — the `MakeUnresolved` is intentional;
the `.pb.h` is a generated file, resolved later by the graph builder when
the proto codegen edge is materialised).

## Failure modes

- **Read failure**. `os.ReadFile` panics into `Throw2` at the lexer
  boundary; the `Try` at the worker entry catches and aborts the module.
  Don't paper over: a missing source file is a real bug at this stage.
- **Bad include syntax**. `parseLine` returns nothing — same behaviour as
  ymake's `YConfWarn(Incl)` (base.cpp:57). Log via the diag layer
  (`design_graph.md` defines the diag sink); do not throw.
- **Sysincl misconfiguration** (unresolved sysincl target). Mirror ymake:
  emit `unresolved(target)` plus a logged `YConfErr`. The graph still
  has an edge so the diff against `sg.json` works.
- **Macro recursion past depth 8**. `Throw` — this is a config-corruption
  bug, not a per-include problem.

## Testing strategy (concrete)

The test budget here is small — the bigger acceptance harness lives in
`acceptance.md`. Scanner-local tests:

1. **Lexer table test** (`scan_cpp_test.go`): a slice of (input bytes,
   expected `[]RawInclude`) pairs covering each branch of base.cpp:22-66
   — quoted, angle-bracketed, with trailing comment, with trailing `;`,
   the macro forms, mid-line whitespace, the line-limit cutoff at 60000.
2. **Sysincl loader test**: load `linux-musl-aarch64.yml` and assert a
   handful of expected mappings (`bits/alltypes.h →
   contrib/libs/musl/arch/aarch64/bits/alltypes.h`, etc.).
3. **End-to-end on `library/cpp/digest/md5/md5.cpp`**: run the full
   `Scan` against a fixture module pre-populated with the right ADDINCLs;
   assert the resolved set matches the includes that show up under that
   file in `sg.json`. This is the gold standard — if this passes for the
   three peerdir libraries plus `tools/archiver/main.cpp`, the scanner
   is done.

## What to write, in order

For the DIGGER:

1. `scanner.go` — types (`RawInclude`, `IncludeKind`, `ResolvedEdge`,
   `EdgeOrigin`, `ScanResult`, `ScanContext`, `Scanner`, `NewScanner`,
   `Scan`). Stub `resolve` and `parseCache` to "panics with TODO".
2. `scan_cpp.go` — the cpp lexer. Single linear pass, no allocations
   beyond the resulting `[]RawInclude`. Test against the table in §Testing.
3. `inccache.go` — `parseCache` with `sync.Mutex` + `singleflight`. No
   eviction. Test under `t.Parallel`.
4. `sysincl.go` — YAML loader (use `gopkg.in/yaml.v3`) and `sysinclTable`.
   `Resolve` exactly per the C++ contract.
5. `resolver.go` — the algorithm in §Resolution algorithm.
   `resolveCache` keyed as in §Cache keying.
6. `scan_proto.go` — the proto lexer + the `.pb.h` induced emitter.
7. Wire `Scanner.Scan` to call into the right per-extension parser via
   the language registry; languageRegistry is a simple
   `map[string]parserKind` extension table populated in `NewScanner`.
8. Run the end-to-end fixture test from §Testing.

Each step compiles and tests on its own. Don't move to the next without
green tests on the previous.

## Out of scope

- Persistent on-disk parse cache (versioned by `ParserID`). Useful later;
  not needed to hit the 1 s budget on a cold run because the closure is
  small and reads come straight from the page cache after the first sweep.
- Languages other than cpp+proto (asm, ragel, cython, fortran, etc.).
- The GLOBAL/UserGlobal scope distinction inside ADDINCLs.
- `ProcessOutputIncludes` and `MapProps` — those are graph-time concerns
  (`design_graph.md`).
- UID computation for source nodes (`design_uid.md`).
