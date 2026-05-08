# design_multimodule.md — flag-driven module selection

This document specifies the **multimodule resolution layer**: the layer that sits
between `design_parser.md`'s flag-blind `ModuleAST` and the graph builder in
`design_graph.md`. It takes a parsed `ModuleAST` plus a `FlagEnv` and produces
one or more **concrete modules** (facets) by evaluating `IF`/`ELSEIF`/`ELSE`/`ENDIF`,
expanding `INCLUDE`, interpolating `$(VAR)`, and fanning out multimodule openers
into per-tag concrete modules.

`CLAUDE.md`, `GOALS.md`, `STYLE.md` are referenced by name and not duplicated.
Types owned by `design_parser.md` (`Loc`, `RawStmt`, `ModuleAST`, `Source`, `Peer`,
`ModuleKind`, `LexError`, `ParseError`) and `design_scanner.md` (`LangID`, `FileID`,
`IncludeKind`) are cross-referenced, not redefined.

---

## 1. Scope and non-goals

**In scope.**

- `IF` / `ELSEIF` / `ELSE` / `ENDIF` evaluation against a `FlagEnv`.
- `INCLUDE` resolution against the source tree, with cycle detection.
- `$(VAR)` interpolation against `FlagEnv.Vars`.
- Fan-out of multimodule openers (`PROTO_LIBRARY`, `DYNAMIC_LIBRARY`, `PACKAGE`, …)
  into per-tag `ConcreteModule` facets.
- The "deferred-to-runtime" rule: selection happens when a `(path, env)` pair is
  resolved, not at file-parse time.
- `INCLUDE_TAG` / `INCLUDE_TAGS` / `EXCLUDE_TAGS` / `ONLY_TAGS` refinement of the
  default-active facet set.

**Out of scope.**

- Graph traversal of `PEERDIR` (`design_graph.md` owns that).
- UID computation (`design_uid.md`).
- CMD / SEM rendering (graph-time).
- The full DSL macro library — `SET_APPEND` and friends are evaluated only as
  needed for IF and multimodule-tag expressions; the rest stays opaque in `Tail`.
- `RECURSE` / `RECURSE_FOR_TESTS` traversal (the graph builder treats these as
  a different edge class).

The parser layer (`parser.go`, `parser_lex.go`, `parser_stmt.go`, `parser_ast.go`)
**MUST NOT** be modified by this ticket. The seam is `Tail []RawStmt` and
`ParseModule(path string) ModuleAST`.

---

## 2. The seam from ticket 9

`ParseModule(path string) ModuleAST` (specified in `design_parser.md §5`) is the
IF-blind, flag-blind base layer. It returns:

- `Sources` — concatenation of every `SRCS(...)` call's args across **all** branches
  of any `IF`/`ELSE` block (both branches, unevaluated).
- `Peers` — same aggregation across every `PEERDIR(...)` call.
- `Tail []RawStmt` — every other statement in source order, including `IF`,
  `ELSEIF`, `ELSE`, `ENDIF`, `INCLUDE`, `SET`, `LICENSE`, `RECURSE`, …

Ticket 10's job is to re-walk `Tail`, replacing the flag-blind aggregation with a
flag-aware one. Specifically, `design_parser.md §4` notes that
`contrib/libs/nayuki_md5/ya.make` ends up with **both** `md5-fast-x8664.S` and
`md5.c` in `ModuleAST.Sources` because the `IF (OS_LINUX AND ARCH_X86_64)` block
is unevaluated. This ticket elides one based on the actual env: for our golden
invocation (`OS_LINUX=yes`, `ARCH_X86_64=""`), the `AND` short-circuits false,
the `ELSE` branch is taken, and only `md5.c` survives in `ConcreteModule.Sources`.

---

## 3. `FlagEnv` — shape and provenance

```go
type FlagEnv struct {
    // Variable bindings consulted by IF/ELSEIF and $(VAR) interpolation.
    // Strings only — bool is encoded as "yes"/"no" per upstream NYMake::IsTrue
    // (eval_context.cpp:36-37 and :192).
    Vars map[string]string
    // 64-bit FNV-1a digest of sorted Vars, precomputed at construction.
    // Cache key alongside path; never recomputed.
    Digest uint64
}
```

`FlagEnv` is **immutable** after construction. Multiple goroutines hold references;
nothing writes through them.

For our golden invocation (`srun.sh`: `--target-platform=default-linux-aarch64
--musl --host-platform-flag=MUSL=yes --sandboxing`), the binding set is derived
from `build/ymake_conf.py:Platform.os_variables` (lines 207-230) and
`Platform.arch_variables` (lines 233-265) plus the CLI flags:

| Variable         | Value | Source                                          |
|------------------|-------|-------------------------------------------------|
| `LINUX`          | `yes` | `os_variables` yield `self.os.upper()` (line 209) |
| `OS_LINUX`       | `yes` | `os_variables` yield `'OS_{}'.format(…)` (line 212) |
| `ARCH_ARM64`     | `yes` | `arch_variables` `is_armv8=True` for `aarch64` (line 241) |
| `ARCH_ARM`       | `yes` | `arch_variables` `is_arm=True` (line 245)       |
| `ARCH_AARCH64`   | `yes` | `arch_variables` `is_linux_armv8=True` (line 248) |
| `ARCH_TYPE_64`   | `yes` | `arch_variables` `is_64_bit=True` (line 264)    |
| `MUSL`           | `yes` | `--musl` and `--host-platform-flag=MUSL=yes`    |
| `SANDBOXING`     | `yes` | `--sandboxing`                                  |

Every other `OS_*` and `ARCH_*` variable is **absent** (looked up as `""`). The
values `OPENSOURCE`, `WITH_VALGRIND`, `ARCH_X86_64`, `ARCH_I386`, `OS_WINDOWS`,
`OS_DARWIN`, etc. are all absent for this invocation.

**Reason for hard-coding this table.** The only invocation for acceptance is the
one in `srun.sh`. A hard-coded table is bit-identical-friendlier than re-deriving
it from a CLI parser. A future ticket can replace the table with a full CLI parser
when a second target is added.

---

## 4. Conditional evaluation (IF/ELSEIF/ELSE/ENDIF)

The Go implementation MUST transcribe the upstream state machine from
`devtools/ymake/lang/eval_context.cpp:267-319` and `eval_context.h:93-143`.

### 4.1 State values

Five `EIfState` values (from `eval_context.h:93-99`):

- `IfProcessing` — IF() is false and all previous ELSEIF() are false; searching
  for a true branch or ELSE.
- `IfProcessed` — IF() is true or some ELSEIF() was true; branch already taken.
- `IfProcessedByElse` — all IF/ELSEIF were false; ELSE branch now processing.
- `IfSkipedElseIf` — last was ELSEIF(), but a previous branch already ran; skipping.
- `IfSkipedElse` — ELSE() was skipped because a prior branch already ran.

`BranchTaken()` returns true iff `IfState` is empty OR the top entry is
`IfProcessed` or `IfProcessedByElse` (`eval_context.h:141-143`). Statements are
processed only when `BranchTaken()` is true.

### 4.2 Push/pop rules

- `IF(args)`: evaluate `BoolExpr(args)`; push `IfProcessed` if true,
  `IfProcessing` if false. (`eval_context.cpp:288-292`)
- `ELSEIF(args)`: assert stack non-empty, assert top ≠ `IfProcessedByElse`/
  `IfSkipedElse` (ELSEIF-after-ELSE → `*ParseError`). If top is `IfProcessing`,
  evaluate and update; if top is `IfProcessed`, parse-only and set `IfSkipedElseIf`.
  (`eval_context.cpp:293-304`)
- `ELSE()`: assert stack non-empty, assert top ≠ `IfProcessedByElse`/`IfSkipedElse`
  (ELSE-after-ELSE → `*ParseError`). Flip to `IfSkipedElse` or `IfProcessedByElse`.
  (`eval_context.cpp:305-311`)
- `ENDIF()`: assert stack non-empty; pop. (`eval_context.cpp:312-314`)
- Nested skipped IFs: a `SkipIfs` counter handles IF inside a non-taken branch
  (`eval_context.cpp:272-286`). The counter prevents premature ENDIF from closing
  an outer IF.

### 4.3 `BoolExpr` and expression operators

`BoolExpr` parses and evaluates `OrExpr → AndExpr → NotExpr → BinaryExpr → UnaryExpr
→ PrimExpr` (`eval_context.cpp:154-205`).

- **Truth function** `NYMake::IsTrue`: the strings `"yes"`, `"true"`, `"on"`, `"1"`
  are true (case-insensitive); anything else (including `""`) is false.
  (`eval_context.cpp:36-37, :192`)
- **Unary operators**: `DEFINED(NAME)` — true iff `FlagEnv.Vars` has the key.
  `ISNUM(NAME)` — true iff the value parses as a long integer.
  (`eval_context.cpp:53-69`)
- **Binary operators** on `(left, op, right)`: `==`, `!=`, `MATCHES` (case-insensitive
  substring), `STARTS_WITH`, `ENDS_WITH`, `VERSION_GT/GE/LT/LE`.
  Numeric comparisons: `>`, `>=`, `<`, `<=` (values parsed as `long`, default 0).
  (`eval_context.cpp:103-151`)
- **`AND`** short-circuits: if the left operand is false, the right sub-expression
  is still parsed but not evaluated (`AndExpr` passes `just_parse=true`).
  (`eval_context.cpp:165-174`)
- **`OR`** similarly short-circuits on the left being true. (`eval_context.cpp:176-185`)
- **`NOT`** negates the sub-expression. (`eval_context.cpp:154-163`)
- **Silent-false fallback**: upstream catches `TRuntimeAssertion` and returns `false`
  (`eval_context.cpp:200-204`). The Go implementation mirrors this as an opt-in
  `LenientBoolExpr` mode; **off by default** — a malformed condition throws
  `*ParseError` so typos are not silently swallowed.

**Acceptance case**: `IF (OS_LINUX AND ARCH_X86_64)` in
`contrib/libs/nayuki_md5/ya.make:11`. With golden env: `OS_LINUX="yes"` (true),
`ARCH_X86_64=""` (false). `AND` short-circuits: left=true, right evaluated:
`IsTrue("")=false`. Result=false → ELSE branch taken → `Sources = ["md5.c"]`.

---

## 5. INCLUDE expansion

`INCLUDE(path)` is processed at evaluation time, not parse time.

Resolution cascade (mirrors `makefile_reader.cpp:97-109`):

1. If `path` is absolute or starts with `$S/` (source root) or `$B/` (build root),
   resolve directly against the root.
2. Else resolve relative to the directory of the file currently being evaluated
   (the current frame's path).
3. Else resolve relative to the source root.

The resolved file is parsed via `ParseModule` (cached — §9), its `[]RawStmt` body
is **spliced** into the current evaluation stream at the INCLUDE point, and the
IF-state stack from the parent file **carries through** into the included content.

Cycle detection: a path stack tracks the open include chain. On entering a file,
check if it is already in the stack; if so, `Throw` with an `*EvalError` wrapping
the path stack (mirrors `TIncludeLoopException` at `eval_context.h:19-37` and
`makefile_reader.cpp:66-67`).

Out of scope for this ticket: the upstream "TODO(spreis): replay of recorded
statements" for multimodule re-parse (`makefile_reader.cpp:259-264`). The acceptance
closure has zero `INCLUDE` calls; this ticket ships the INCLUDE machinery and
exercises it only in the §12 fixture tests.

---

## 6. `$(VAR)` interpolation

`EvalExpr(vars, expr)` from `devtools/ymake/lang/eval.cpp:43-52`, re-encoded in Go.

Three substitution forms:
- `$(NAME)` — standard form.
- `${NAME}` — alias (same semantics).
- Bare `$NAME` in IF expressions (`eval_context.cpp:33-38`).

Recursive expansion with depth cap = 8 (matches upstream `MacroEvalDepth`). Past 8,
`Throw` an `*EvalError` — a self-referential `SET(X $X)` is a bug, not silent data.

`EvalExpr` is used by the INCLUDE resolver (to expand `$(VAR)` in INCLUDE argument
paths), by `BoolExpr` (right-hand side of comparisons), and by `Deref` (expanding
SET/SET_APPEND values). The new file `multimodule_eval.go` owns `EvalExpr`.

The parser layer (`parser.go`/`parser_lex.go`/`parser_stmt.go`) **MUST NOT** be
modified. The seam is `Tail []RawStmt` → ticket 10's `evalContext.run(tail)`.

---

## 7. Multimodule fan-out

### 7.1 Grammar (upstream DSL)

```
multimodule NAME {
    module TAG1: BASE_CLASS { module-body }
    module TAG2: BASE_CLASS { module-body }
    …
}
```

Source: `build/ymake.core.conf:2374-2398` (`DYNAMIC_LIBRARY`),
`build/conf/proto.conf:916-973` (`PROTO_LIBRARY`),
`build/ymake.core.conf:2495` (`PACKAGE`). The Go side does **not** parse these
files; it hard-codes the semantics per `CLAUDE.md` ("We do not interpret the DSL
build descriptions; their semantics are hard-coded").

### 7.2 Registry types

```go
type multimoduleSpec struct {
    Name       string         // "PROTO_LIBRARY"
    Submodules []submoduleSpec
}

type submoduleSpec struct {
    Tag            string     // "CPP_PROTO"
    BaseModuleKind ModuleKind // from design_parser.md
    IncludeTag     bool       // .INCLUDE_TAG (default true per config.h:105)
    PeerdirSelf    []string   // .PEERDIRSELF tags — intra-multimodule edges
    LangID         LangID     // from design_scanner.md
    PeerdirTags    []string   // what callers' PEERDIR_TAGS must contain to select
}
```

The registry is a package-level `var multimoduleSpecs = []multimoduleSpec{…}`
initialised at package load — no runtime construction (§13 perf note).

### 7.3 Registry entries (normative)

The ticket MUST include at minimum these four multimodules, sufficient for the
acceptance closure:

**`PROTO_LIBRARY`** (`build/conf/proto.conf:916-973`):

| Tag              | IncludeTag | PeerdirSelf      | PeerdirTags                           |
|------------------|------------|------------------|---------------------------------------|
| `CPP_PROTO`      | true       | —                | `["CPP_PROTO"]`                       |
| `JAVA_PROTO`     | true       | —                | `["JAVA_PROTO"]`                      |
| `PY_PROTO`       | true       | `["CPP_PROTO"]`  | `["PY_PROTO"]`                        |
| `PY3_PROTO`      | true       | `["CPP_PROTO"]`  | `["PY3_PROTO"]`                       |
| `GO_PROTO`       | true       | —                | `["GO_PROTO"]`                        |
| `DOCS_PROTO`     | false      | —                | `["DOCS_PROTO"]`                      |
| `TS_PROTO`       | false      | `["TS_PREPARE_DEPS"]` | `[]`                             |
| `TS_PREPARE_DEPS`| false      | —                | `[]`                                  |
| `DESC_PROTO`     | true       | —                | `["DESC_PROTO"]`                      |

**`PROTO_SCHEMA`** (`build/conf/proto.conf:1003-1050`): similar layout; all
submodules have `DISABLE(START_TARGET)` and tags carry `_FROM_SCHEMA` suffix.
Full entry in `multimodule_specs.go`.

**`DYNAMIC_LIBRARY`** (`build/ymake.core.conf:2374-2398`):

| Tag      | IncludeTag | PeerdirSelf   | PeerdirTags                                |
|----------|------------|---------------|--------------------------------------------|
| `DLL_BIN`| true       | —             | `["DLL"]` (MODULE_TAG=DLL)                 |
| `DLL_LIB`| true       | `["DLL_BIN"]` | `["DLL_LIB", "__EMPTY__", "RESOURCE_LIB"]` |

**`PACKAGE`** (`build/ymake.core.conf:2495-2510`):

| Tag             | IncludeTag | PeerdirSelf        | PeerdirTags (inherited from base class) |
|-----------------|------------|--------------------|-----------------------------------------|
| `PACKAGE_FINAL` | true       | `["PACKAGE_UNION"]`| broad set from `_PACKAGE_FINAL`         |
| `PACKAGE_UNION` | true       | —                  | same broad set                          |

Entries deliberately omitted (out of scope — if encountered, `*MultimoduleError` fires):
`JTEST`, `JUNIT`, `TS_LIBRARY` and any other multimodule not listed above.

### 7.4 Selection rule

When the graph builder traverses `PEERDIR(path)` from caller module C with
`C.Vars["PEERDIR_TAGS"] = "CPP_PROTO …"`, the resolver given `(path, env)`:

1. Parse `path` (cached).
2. If the opener is a known multimodule name → fan out into facets; one
   `ConcreteModule` per submodule. Each facet inherits the parent's IF-resolved
   `Sources`/`Peers`/`Tail` but **scopes** its `LangID`, `PeerdirSelf`, and
   `PeerdirTags` from `submoduleSpec`.
3. The graph builder picks the facet whose `PeerdirTags` intersects the caller's
   `PEERDIR_TAGS`, per `MatchPeer` (`module_state.cpp:407-423`).
4. Default-active facets at root (no caller) are those with `IncludeTag=true`,
   modulated by `INCLUDE_TAGS`/`EXCLUDE_TAGS`/`ONLY_TAGS` calls in the file
   (`makefile_loader.cpp:338-362`).

Verified: the acceptance closure (`tools/archiver` transitive) contains **zero**
multimodule openers. Multimodule fan-out is not on the bit-identical-acceptance
critical path; the table is specified here because `CLAUDE.md` requires readiness,
and ticket 11's `MatchPeer` consumes `PeerdirTags`.

---

## 8. Public API surface

New files (§15) export these symbols:

```go
// multimodule.go

// ResolveModule evaluates IF/INCLUDE/$(VAR) against env, fans out multimodule
// openers into facets, and returns the resulting concrete modules.
// Pure function modulo INCLUDE (reads filesystem via ParseModule, cached by
// ModuleResolver one layer up).
func ResolveModule(ast ModuleAST, env *FlagEnv) []ConcreteModule

// ModuleResolver is the (path, env) cache used by the graph builder.
// Keyed by (absolute path, env.Digest). First miss: ParseModule + ResolveModule.
// Subsequent lookups: lock-free sync.Map read.
type ModuleResolver struct { /* sync.Map + singleflight.Group */ }

func NewModuleResolver(env *FlagEnv) *ModuleResolver
func (r *ModuleResolver) Get(path string) []ConcreteModule
```

`ConcreteModule` is the post-fan-out shape:

```go
type ConcreteModule struct {
    Path        string            // absolute ya.make path
    Tag         string            // "" for non-multimodule; "CPP_PROTO" / "DLL_BIN" / … otherwise
    Kind        ModuleKind        // from design_parser.md
    LangID      LangID            // from design_scanner.md
    Sources     []Source          // IF-resolved; from design_parser.md
    Peers       []Peer            // IF-resolved; from design_parser.md
    PeerdirSelf []string          // intra-multimodule edges (other facets of same parent)
    Vars        map[string]string // SET/SET_APPEND results resolved during evaluation
    PeerdirTags []string          // tags this facet exposes; consumed by callers' MatchPeer
    IncludeTag  bool              // active by default at root?
    OpenLoc     Loc               // from design_parser.md
    EndLoc      Loc               // from design_parser.md
    Tail        []RawStmt         // stmts not consumed here — graph builder reads what it needs
}
```

`Callers MUST treat the returned `[]ConcreteModule` slice as immutable (§13).`

---

## 9. Cache keying

Two distinct caches with different keys and lifetimes — do not merge.

**Parser cache** (owned by `ModuleResolver`): `path → ModuleAST`. Keyed by
absolute path. One `ParseModule` call per file across the whole run. Backed by
`sync.Map` with `singleflight.Group` to coalesce concurrent misses on the same
path.

**Resolve cache**: `(path, env.Digest) → []ConcreteModule`. Keyed by the pair.
Different `FlagEnv`s for the same file produce different facet lists; both stay
cached. For the golden invocation only one `FlagEnv` exists, so this behaves as
`path → []ConcreteModule`. A future host/target split creates two `FlagEnv`s and
exercises the env axis.

`env.Digest` is FNV-1a over the sorted `(key, value)` pairs of `FlagEnv.Vars`,
computed once at construction.

**Keys deliberately excluded.** Sysincl rule set (process-global, not per-module);
the multimodule registry (process-global, hard-coded); active language registry.
If any of these change, restart the process.

---

## 10. INCLUDE_TAG / INCLUDE_TAGS / EXCLUDE_TAGS / ONLY_TAGS

These statements in the **ya.make file** (not in the conf) refine the default-active
facet set when the file is entered as a root target, per `makefile_loader.cpp:338-362`.

- Default active set = `{tag | submoduleSpec.IncludeTag == true}`.
- `INCLUDE_TAGS(t1 t2 …)` — adds tags to the set. (`makefile_loader.cpp:345, :356-359`)
- `EXCLUDE_TAGS(t1 …)` — removes tags. (`makefile_loader.cpp:355-356`)
- `ONLY_TAGS(t1 …)` — clears then adds. (`makefile_loader.cpp:345-347`)
- These calls are valid only inside a `multimodule`-shaped opener; elsewhere →
  `*ParseError` (mirrors `makefile_loader.cpp:456-461`).
- If `DefaultTags` becomes empty: emit a warning via the diag layer ("All available
  variants of multimodule discarded", per `makefile_loader.cpp:210-211`) and return
  `[]ConcreteModule{}`.

The acceptance closure does not exercise these statements — they are specified for
completeness and tested only via the §12 synthetic fixtures.

---

## 11. Closure inventory — expected output under golden `FlagEnv`

Verified by reading each `ya.make` directly. This table is the post-implementation
contract; the §12 `TestArchiverGoldenEnv` asserts it cell-by-cell.

| File | Multimodule? | IF resolves to | Final `Sources` | Final `Peers` |
|------|--------------|----------------|-----------------|---------------|
| `tools/archiver/ya.make` | no | (none) | `[main.cpp]` | `[library/cpp/archive, library/cpp/digest/md5, library/cpp/getopt/small]` |
| `library/cpp/archive/ya.make` | no | (none) | `[yarchive.cpp, yarchive.h, directory_models_archive_reader.cpp, directory_models_archive_reader.h, models_archive_reader.cpp]` | `[]` |
| `library/cpp/digest/md5/ya.make` | no | (none) | `[md5.cpp]` | `[contrib/libs/nayuki_md5, library/cpp/string_utils/base64]` |
| `library/cpp/getopt/small/ya.make` | no | (none) | `[completer.cpp, completer_command.cpp, completion_generator.cpp, formatted_output.cpp, last_getopt.cpp, last_getopt_easy_setup.cpp, last_getopt_opt.cpp, last_getopt_opts.cpp, last_getopt_parser.cpp, last_getopt_parse_result.cpp, modchooser.cpp, opt.cpp, opt2.cpp, posix_getopt.cpp, wrap.cpp, ygetopt.cpp]` | `[library/cpp/colorizer]` |
| `contrib/libs/nayuki_md5/ya.make` | no | `OS_LINUX=yes ∧ ARCH_X86_64="" → false; ELSE branch` | `[md5.c]` | `[]` |
| `library/cpp/string_utils/base64/ya.make` | no | (none) | `[base64.cpp]` | `[contrib/libs/base64/avx2, contrib/libs/base64/ssse3, contrib/libs/base64/neon32, contrib/libs/base64/neon64, contrib/libs/base64/plain32, contrib/libs/base64/plain64]` |
| `library/cpp/colorizer/ya.make` | no | (none) | `[colors.cpp, output.cpp]` | `[]` |

Note on `nayuki_md5`: the parser (ticket 9) emits **both** `md5-fast-x8664.S` and
`md5.c` in `ModuleAST.Sources` (the unevaluated form, per `design_parser.md §4`).
This ticket's `ResolveModule` elides the IF-branch sources and emits only `md5.c`.

---

## 12. Test plan

The implementation ticket MUST land these tests in `multimodule_test.go`.

- **`TestEvalIfElseSimple`** — synthetic input `IF (FOO == "yes") SRCS(a.cpp) ELSE
  SRCS(b.cpp) ENDIF`; run with `FOO=yes` and `FOO=no`; assert the right branch's
  SRCS land in `ConcreteModule.Sources`.

- **`TestEvalIfElseifChain`** — synthetic 3-way `IF / ELSEIF / ELSE / ENDIF` chain;
  all three branches exercised across three env instances.

- **`TestEvalNestedIf`** — `IF (A) IF (B) SRCS(inner) ENDIF SRCS(outer) ENDIF`;
  assert depth-2 scoping.

- **`TestEvalAndOr`** — `IF (A AND B)`, `IF (A OR B)`, `IF (NOT A)`, `DEFINED(X)`,
  `ISNUM(Y)`; each with appropriate env bindings; assert short-circuit behaviour.

- **`TestNayukiAarch64`** — `contrib/libs/nayuki_md5/ya.make` with the golden env;
  assert `Sources == ["md5.c"]`.

- **`TestNayukiX86_64`** — same file, env with `ARCH_X86_64=yes`; assert
  `Sources == ["md5-fast-x8664.S"]`.

- **`TestArchiverGoldenEnv`** — run `ResolveModule` over all seven files in the
  closure with the golden env; assert the §11 table cell-by-cell. **Gate condition
  before wiring into `BuildArchiverGraph`**: if this passes, the resolve layer is done.

- **`TestMultimoduleFanoutSynthetic`** — construct a synthetic `ModuleAST` with
  opener `"PROTO_LIBRARY"` and empty body; call `ResolveModule`; assert nine
  `ConcreteModule` values with correct `Tag`, `IncludeTag`, `PeerdirTags`, `LangID`.

- **`TestProtoLibraryFanout`** — asserts the full CPP_PROTO / JAVA_PROTO / PY_PROTO /
  PY3_PROTO / GO_PROTO / DOCS_PROTO / TS_PROTO / TS_PREPARE_DEPS / DESC_PROTO
  submodule layout matches `build/conf/proto.conf:916-973`.

- **`TestIncludeOnce`** — INCLUDE expansion: two files, each including the other;
  assert cycle detection throws `*EvalError` containing both paths.

- **`TestUnknownVarInIf`** — `IF (UNDEFINED == "yes")` with env missing `UNDEFINED`;
  assert the missing var evaluates as `""`, IsTrue="" is false, ELSE branch taken.

- **`TestIncludeTagDefault`** — multimodule with one submodule having `.INCLUDE_TAG=no`
  in the registry; assert `ConcreteModule.IncludeTag == false` and default-active
  set excludes it.

- **`TestEnvDigestStable`** — same `Vars` map (different map instances, same content)
  produces the same `Digest`; one bit changed in one value produces a different digest.

- **`TestResolveCacheHit`** — call `ModuleResolver.Get` twice on the same path; assert
  `ParseModule` is called exactly once (counter injected via the resolver's parse
  function field).

---

## 13. Performance budget pointer

The project-wide 1-second budget is specified in `acceptance.md`. This layer's piece:
parse 7 files once; resolve 7 `(path, env)` pairs; serve subsequent reads from cache.

Design choices that affect performance:

- `ParseModule` is sub-ms per file (`design_parser.md §8`: single `os.ReadFile`,
  hand lexer, ~22 statements max).
- `ResolveModule` is a single linear pass over `Tail` plus IF-eval (a few hundred
  string ops at worst). No allocations beyond `ConcreteModule.Sources` / `Peers`
  slices.
- Resolve cache value is the slice `[]ConcreteModule` itself — **no deep copy on read**.
  Callers MUST treat it as immutable. (`design_graph.md` reads `Sources/Peers/Tail`
  and never mutates.)
- `ConcreteModule.Vars` is owned by exactly one resolution and shared across readers.
- `FlagEnv.Vars` is allocated once at process start, never mutated.
- `multimoduleSpecs` registry: package-level `var`, initialised at package load,
  zero runtime construction.
- Cache reads after first miss: lock-free `sync.Map` load.

---

## 14. Failure modes

Three typed error values, all reachable via `errors.As` after `(*Exception).Unwrap()`:

**`*ParseError`** (re-used from `design_parser.md §9`, extended): malformed IF
condition, mismatched ENDIF (ENDIF without IF), ELSE-after-ELSE, ELSEIF-after-ELSE,
INCLUDE_TAGS / EXCLUDE_TAGS / ONLY_TAGS outside a multimodule opener, multimodule
with zero `IncludeTag` facets and no `INCLUDE_TAGS()` override.

**`*EvalError`** (new): `$(VAR)` recursive expansion past depth 8 (carries the
variable name); INCLUDE cycle (carries the path stack); INCLUDE of a missing file
(wraps the underlying `*os.PathError`).

**`*MultimoduleError`** (new): unknown multimodule name in opener — a multimodule
appears in the closure that is not in the hard-coded registry. `Throw` immediately
so failures are loud; never silently fall through as a plain module. Consistent with
`design_scanner.md`'s philosophy on out-of-scope language extensions.

---

## 15. File layout

Per `CLAUDE.md` "All `.go` files live flat in the workspace root":

- `multimodule.go` — public types (`FlagEnv`, `ConcreteModule`, `ModuleResolver`,
  typed errors `EvalError`, `MultimoduleError`); `ResolveModule`; `NewModuleResolver`;
  `(*ModuleResolver).Get`.
- `multimodule_eval.go` — IF/ELSEIF/ELSE/ENDIF state machine; `BoolExpr`; `EvalExpr`
  / `$(VAR)` interpolation.
- `multimodule_include.go` — INCLUDE expansion; cycle stack; path resolution.
- `multimodule_specs.go` — the `[]multimoduleSpec` registry hard-coded from
  `ymake.core.conf` / `proto.conf`. Contains exactly `PROTO_LIBRARY`, `PROTO_SCHEMA`,
  `DYNAMIC_LIBRARY`, `PACKAGE`. Other multimodules (JTEST, JUNIT, TS_* standalone, …)
  are explicitly absent; encountering one fires `*MultimoduleError`.
- `multimodule_env.go` — `NewSrunFlagEnv() *FlagEnv` returning the binding table
  from §3 for use by `main`; `NewFlagEnvFromMap(m map[string]string) *FlagEnv` for
  tests. Digest computed via FNV-1a over sorted pairs.
- `multimodule_test.go` — the §12 tests.

---

## 16. What ticket 11 owns

- **PEERDIR traversal**: walk `ConcreteModule.Peers`, call `ModuleResolver.Get` on
  each, pick the facet whose `PeerdirTags` intersects the caller's `PEERDIR_TAGS`.
- **`MatchPeer`**: the caller-side facet selection (`module_state.cpp:407-442`),
  including the `__EMPTY__` tag exception and PROXY library / static-library
  compatibility checks.
- **Graph builder wiring**: `ConcreteModule.Sources` → scanner → `ResolvedEdge`.
- **`RECURSE`/`RECURSE_FOR_TESTS` resolution**: separate edge class, no `MatchPeer`.
- **`ADDINCL` propagation** through PEERDIR: constructing `*ModuleIncDirs` for
  `design_scanner.md`'s consumer side.

The seam at this layer is **`[]ConcreteModule`**. Ticket 11 reads `Peers`, calls
`ModuleResolver.Get`, and recurses.

---

## 17. Authoritative references

Every MUST in this document is grounded in one of these upstream locations:

- `devtools/ymake/lang/eval_context.h:93-143` — `EIfState` enum; `BranchTaken()`.
- `devtools/ymake/lang/eval_context.cpp:33-205` — `VarExpr`, `UnaryExpr`,
  `BinaryExpr`, `NotExpr`, `AndExpr`, `OrExpr`, `BoolExpr`.
- `devtools/ymake/lang/eval_context.cpp:267-319` — `ShouldSkip`, `CondStatement`
  (IF/ELSEIF/ELSE/ENDIF state machine).
- `devtools/ymake/lang/eval.cpp:43-52` — `EvalExpr` / `$(VAR)` interpolation.
- `devtools/ymake/lang/makefile_reader.cpp:46-67` — INCLUDE handling, `TIncludeLoopException`.
- `devtools/ymake/lang/makefile_reader.cpp:97-109` — INCLUDE path resolution.
- `devtools/ymake/lang/makefile_reader.cpp:259-265` — multimodule re-parse TODO.
- `devtools/ymake/makefile_loader.cpp:170-275` — multimodule fan-out, NextSubModule
  loop, `NukeModule`.
- `devtools/ymake/makefile_loader.cpp:338-362` — `RefineSubModules`: INCLUDE_TAGS /
  ONLY_TAGS / EXCLUDE_TAGS.
- `devtools/ymake/makefile_loader.cpp:456-461` — error on tags outside multimodule.
- `devtools/ymake/makefile_loader.cpp:502-541` — epilogue processing.
- `devtools/ymake/module_state.h:65-92` — `FromMultimodule` flag, `TModuleAttrs`.
- `devtools/ymake/module_state.cpp:395-442` — `MatchPeer`, `ImportPeerdirTags`,
  `ImportPeerdirRules`, `ImportPeerdirPolicy`.
- `devtools/ymake/lang/properties.h:65-67` — `.INCLUDE_TAG` property.
- `devtools/ymake/lang/properties.h:78-80` — `.PEERDIRSELF` property.
- `devtools/ymake/config/config.h:105` — `IncludeTag = true` default.
- `build/ymake.core.conf:2374-2398` — `DYNAMIC_LIBRARY` (DLL_BIN / DLL_LIB).
- `build/ymake.core.conf:2495-2510` — `PACKAGE` (PACKAGE_FINAL / PACKAGE_UNION).
- `build/conf/proto.conf:916-973` — `PROTO_LIBRARY` submodule layout.
- `build/conf/proto.conf:1003-1050` — `PROTO_SCHEMA` submodule layout.
- `build/conf/sysincl.conf:51-94` — sysincl gating by MUSL/ARCH_AARCH64/OPENSOURCE.
- `build/ymake_conf.py:86-265` — `Platform` class; `os_variables`, `arch_variables`.
- `srun.sh` — the CLI invocation this layer must reproduce exactly.
