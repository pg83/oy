# design_parser.md — `ya.make` parser (narrow scope)

This document specifies the Go-side `ya.make` parser at the **lexical and
flat-statement level only**. It is a *specification*: every "MUST" is binding on
the implementation ticket. No `.go` file ships with this ticket.

The parser takes one `ya.make` file's bytes and produces a typed
`ModuleAST` that names the module's opener (`PROGRAM` / `LIBRARY`), its
direct sources, its peers, and the rest of the file as opaque records. It
does **not** evaluate `IF`/`ELSEIF`/`ELSE`/`ENDIF`, expand `INCLUDE`, fan
out multimodules, interpolate `$(VAR)`, or know anything about flags or
target platforms. Those concerns are deferred to ticket 10 — see §11.

`CLAUDE.md`, `GOALS.md`, `STYLE.md` are referenced by name and not
duplicated.

---

## 1. Scope and non-goals

**In scope.**

- Reading the raw bytes of one `ya.make` from disk.
- Tokenising into `COMMENT` / `STRING_DQ` / `STRING_SQ` / `ATOM` /
  `COMMAND` / `LPAREN` / `RPAREN` / `EOF`.
- Grouping tokens into a flat `[]RawStmt` (one entry per
  `COMMAND ( args )`).
- Projecting that flat list into a typed `ModuleAST` whose opener is
  `PROGRAM` or `LIBRARY`, whose `Sources` are the concatenated args of
  every `SRCS(...)` call, whose `Peers` are the concatenated args of
  every `PEERDIR(...)` call, and whose `Tail []RawStmt` retains every
  other statement in source order.
- Comment handling (file scope, between args, inline).
- Single- and double-quoted string handling, with the upstream
  backslash-strip semantics.

**Out of scope.**

- `INCLUDE` resolution (no fs walk past the file we were asked to read).
- `IF` / `ELSEIF` / `ELSE` / `ENDIF` evaluation. Those are emitted as
  ordinary `RawStmt` entries inside `Tail` and never interpreted here.
- Multimodule expansion / facets (`PROTO_LIBRARY`, `DYNAMIC_LIBRARY`,
  `PACKAGE`, …).
- Any flag environment (`FlagEnv`, target platform, language axis).
- `$(VAR)` interpolation. Tokens of the form `$(NAME)` are preserved
  literally as part of the surrounding `ATOM`.
- `PEERDIR`-driven recursion into other `ya.make` files. The parser
  emits the literal `PEERDIR(...)` arguments as `Peers`; the graph
  builder traverses them.
- Caching. The parser is stateless; ticket 10 wraps a `(path, env)` cache
  around the flag-aware entry point it adds.

The current public entry point's signature is keyed by **path only** and
that is forward-compatible with ticket 10, which adds a *second* entry
point taking a `FlagEnv` — the path-only entry remains the IF-blind,
flag-blind base layer.

`GOALS.md` notes that "ya.make — это мультимодуль, его интерпретация
зависит … от входящих флагов". This document acknowledges that as the
sole reason the AST keeps a `Tail []RawStmt` field rather than throwing
on every unknown statement: `Tail` is the seam ticket 10 will widen.

**Authoritative reference.** The lexical grammar source of truth is
`/home/pg/monorepo/yatool_orig/devtools/ymake/lang/makelists/makefile_lang.rl6`.
The visitor interface that confirms the upstream "stream of (command,
args) records" model is
`/home/pg/monorepo/yatool_orig/devtools/ymake/lang/makelists/makefile_lang.h`.
The include / IF / multimodule machinery in
`/home/pg/monorepo/yatool_orig/devtools/ymake/lang/makefile_reader.cpp`
sits *above* the lexer and is **not** transcribed into this document —
ticket 10 owns that layer. Any deviation from upstream MUST be called
out here. None is asserted at this scope.

---

## 2. Lexical grammar (verbatim from the Ragel source)

The Go lexer MUST produce the same token kinds as the upstream Ragel
machine. The relevant fragment of `makefile_lang.rl6` (lines 61–86) is
reproduced so the spec is self-contained:

```ragel
spaces = space**;

comment = '#' (any - '\n')* '\n';

posentry = ('\n' ${ numcol = 1; numline++; }) | ((any - '\n') ${ numcol++; }) | (any ${ pos++; });
poscount = posentry*;

# string starts with quote and can have escaped quotes and spaces inside
str_dq_char = any - '"' - '\\';
str_dq = '"' str_dq_char* ('\\' any str_dq_char*)* '"';
str_sq_char = any - '\'' - '\\';
str_sq = '\'' str_sq_char* ('\\' any str_sq_char*)* '\'';
str = (str_dq | str_sq) >start_quoted_str %end_quoted_str;

escaped = ('\\' any);
allowed_sym = any - space - '(' - ')' - '#' - '\\';
atom = ( ( escaped | (allowed_sym - '"' - '\'') | ('$(' allowed_sym+ ')') )
        (allowed_sym | escaped)* ) >start_str %end_str;

command = (/[a-z_]/i /[\-a-z0-9_]/i*) >start_str %end_str;

list_elem = (str | atom) %push_list_elem;
list = spaces ((list_elem | comment) spaces)** >clear_list;
statement = (command >start_cmd %end_cmd) spaces '(' list ')' $end_statement;
statements = spaces ((statement | comment) spaces)* $^a_error;

main := statements | poscount;
```

The token kinds the Go lexer emits:

| Kind        | Production                                                                                                  | Notes                                                                                                                              |
|-------------|-------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------|
| `COMMENT`   | `# ... \n`                                                                                                  | Skipped except for line counting.                                                                                                  |
| `STRING_DQ` | `"..."` with `\\<any>` escapes                                                                              | Backslashes stripped on emit, matching `SubstGlobal(curstr, "\\", "", *pool)` at rl6:31.                                           |
| `STRING_SQ` | `'...'` with `\\<any>` escapes                                                                              | Same backslash strip (rl6:31 applies to both `str_dq` and `str_sq`).                                                               |
| `ATOM`      | `(escaped \| (allowed_sym - '"' - "'") \| '$(' allowed_sym+ ')') (allowed_sym \| escaped)*` (rl6:75–77)     | Backslashes preserved (no `SubstGlobal` applied to `atom` upstream — only `str` triggers `end_quoted_str`). `$(VAR)` only legal as the leading segment. |
| `COMMAND`   | `[A-Za-z_][A-Za-z0-9_\-]*` immediately followed by `(`                                                      | Macro / statement name; case-insensitive at the lexer (rl6:79 uses `/[a-z_]/i`).                                                   |
| `LPAREN`    | `(`                                                                                                         | Statement bracket open.                                                                                                            |
| `RPAREN`    | `)`                                                                                                         | Statement bracket close.                                                                                                           |
| `EOF`       | end of input                                                                                                |                                                                                                                                    |

**Implementation constraints (all MUST).**

- Hand-written byte scanner. The implementation MUST NOT use `regexp`,
  `text/scanner`, or `bufio.Scanner`. The rl6 char-classes MUST
  transcribe into Go helpers `isAtomByte`, `isCommandStartByte`,
  `isCommandTailByte`. Reason: a hand-written byte machine is the only
  way to stay inside the perf envelope (§8) and the rl6 grammar is
  small enough that transcription is cheaper than a generated DFA.
- Whitespace and `COMMENT` are dropped between tokens. Only line/column
  counters update. Line is 1-based, column is 1-based and resets to 1
  on each `\n`. This matches rl6:65 (`numcol = 1; numline++;` on
  newline; `numcol++` on every other byte).
- The lexer MUST track an absolute byte position alongside row/col, used
  for the error excerpt below.
- `COMMAND` requires lookahead. A name-shaped run of
  `[A-Za-z_][A-Za-z0-9_\-]*` is a `COMMAND` only when the first
  non-space byte after it is `(`; otherwise it is an `ATOM`. Upstream
  parses statements directly and never asks this question — we expose
  a token stream so the higher pass (§3) can consume without
  re-tokenising. The decision to expose a token kind here, rather than
  fold the parse into the lexer as upstream does, is a design choice:
  a typed token stream keeps the §3 pass and the §4 projection simple
  and unit-testable.
- The lexer emits `ATOM`, `STRING_DQ`, `STRING_SQ` tokens unrestricted
  by position — the higher pass enforces the "only inside `( ... )`"
  rule.

**Error contract.** Any malformed input throws via `Throw` / `ThrowFmt`
(`STYLE.md`'s rule). The thrown `*Exception` carries a typed payload:

```go
type LexError struct {
    Path    string
    Row     int    // 1-based
    Col     int    // 1-based
    Excerpt string // <prefix>[ <- HERE ]<suffix>, ~80 bytes either side
}
```

The `Excerpt` format mirrors upstream `TVisitorCtx::Here` at rl6:105–118
(`TString::Join(prefix, "[ <- HERE ]", suffix)`). Wrap into `*Exception`
per `throw.go`. STYLE.md §"Error handling" forbids the
`if err != nil { return err }` pass-through; lex errors `Throw` instead.

---

## 3. Statement-level intermediate form

After lexing, a single deterministic pass groups tokens into statements.
The intermediate form is a flat slice:

```go
type Loc struct {
    Path string
    Row  int // 1-based; row of the first byte of the command name
    Col  int // 1-based
}

type RawStmt struct {
    Cmd  string   // upper-cased command name
    Args []string // argument tokens in source order
    Loc  Loc
}
```

**Normalisation rules (all MUST).**

- `Cmd` is upper-cased on emit. Upstream matches case-insensitively
  (rl6:79 uses `/[a-z_]/i` and downstream string comparisons such as
  `command == TStringBuf("INCLUDE")` at `makefile_reader.cpp:49` rely on
  the canonicalised form); we collapse case so downstream switches in
  Go are stable.
- `Args` keeps tokens in source order. Quoted-string tokens
  (`STRING_DQ`, `STRING_SQ`) have their backslash escapes stripped on
  emit (matches rl6:31's `SubstGlobal(curstr, "\\", "", *pool)`).
  `ATOM` tokens keep their backslashes — that mirrors the rl6
  grammar, which only runs `SubstGlobal` on the quoted form
  (`%end_quoted_str` at rl6:29–32 vs. `%end_str` at rl6:21–23).
- No `$(VAR)` interpolation. Token text is preserved literally; ticket
  10 owns interpolation.
- `Loc` points at the first byte of the command name. This matches
  upstream `curProcRange.Line/Column` set in `start_cmd` at rl6:46–49.
- A bare `Cmd()` with no args is legal (`PROGRAM()`, `END()` are the
  canonical examples — see `tools/archiver/ya.make:1` and `:15`).
  `Args` is `[]string{}`, **not** `nil`, in that case. Reason: switch
  branches on `len(s.Args)` should be uniform; `nil` and empty mean
  the same thing semantically but produce different `reflect.DeepEqual`
  results against test fixtures.
- Comments inside the argument list (rl6:82 — `(list_elem | comment)
  spaces`) are skipped silently by the §3 pass; they do not appear in
  `Args`. State that explicitly because rl6 makes it a grammar-level
  alternative, not a whitespace rule.

The intermediate form is **not** part of the public API. It is the input
to §4. The implementation MAY hide it as an unexported `rawStmt`, but
this document names and shapes it because §7's tests assert against it.

---

## 4. Typed AST — the public output

The parser's public output is a single value per file:

```go
type ModuleKind int

const (
    ModuleProgram ModuleKind = iota + 1
    ModuleLibrary
)

type Source struct {
    Path string // exactly as written in SRCS(...), one entry per arg
    Loc  Loc
}

type Peer struct {
    Path string // exactly as written in PEERDIR(...), one entry per arg
    Loc  Loc
}

type ModuleAST struct {
    Path    string     // absolute path of the parsed ya.make
    Kind    ModuleKind // Program or Library; opener call must have zero args
    OpenLoc Loc        // location of the opener
    EndLoc  Loc        // location of the END() call
    Sources []Source   // concatenation of every SRCS(...) call's args, in file order
    Peers   []Peer     // concatenation of every PEERDIR(...) call's args, in file order
    Tail    []RawStmt  // every other statement, in file order — opaque, ticket 10 widens this
}
```

**Aggregation semantics (all MUST).**

- Multiple `SRCS(...)` calls in one file are concatenated in source
  order. Same for `PEERDIR(...)`. The closure contains both shapes:
  `library/cpp/digest/md5/ya.make:3-10` has one `SRCS` and one
  `PEERDIR`; `contrib/libs/nayuki_md5/ya.make:11-19` has *two* `SRCS`
  calls under conditional branches. **Under this narrow ticket the
  `IF` is not yet evaluated** (§1), so both branches' `SRCS` calls
  land in `Sources` — i.e. for `nayuki_md5` you will see both
  `md5-fast-x8664.S` (the `IF` branch) and `md5.c` (the `ELSE` branch)
  in `ModuleAST.Sources`. This is a **known limitation** and the
  precise reason ticket 10 must run before the parser output is fed
  into the graph builder; the graph builder's correctness depends on
  IF having been resolved against the actual flag environment.
- `PROGRAM` and `LIBRARY` openers MUST be called with zero arguments.
  Verified by reading every opener in the closure inventory (§6) —
  `tools/archiver/ya.make:1` is `PROGRAM()`, the four `LIBRARY()`
  openers in `library/cpp/{archive,digest/md5,getopt/small,colorizer}`,
  the one in `library/cpp/string_utils/base64`, and the one in
  `contrib/libs/nayuki_md5` are all bare. Non-zero args throw a
  `ParseError` (§9).
- Exactly one opener per file (the first `PROGRAM` or `LIBRARY` `Cmd`
  encountered). Exactly one `END` per file (the last `END` `RawStmt`
  encountered). Zero or multiple opener / `END` throws a `ParseError`.
- Statements **before** the opener and **after** the `END` are kept in
  `Tail` for ticket 10 to interpret — `RECURSE` and `RECURSE_FOR_TESTS`
  typically follow `END` (see `library/cpp/archive/ya.make:13-15`,
  `library/cpp/digest/md5/ya.make:14-18`,
  `library/cpp/string_utils/base64/ya.make:18-22`). They MUST NOT be
  silently dropped.
- Unrecognised commands (anything not in `{PROGRAM, LIBRARY, SRCS,
  PEERDIR, END}`) are kept verbatim in `Tail`. Logging is at most one
  line per unique command name **per process lifetime**, mirroring
  upstream `YConfWarn(ToDo) << "language - unprocessed statement: "
  << command` at `makefile_reader.cpp:75-77`. A `sync.Map` of
  seen-names is sufficient and is the only piece of process-global
  state this layer carries; it is logging-only, not semantic state.

---

## 5. Public API surface

```go
// parser.go
func ParseModule(path string) ModuleAST
```

Single entry point. Reads bytes via `os.ReadFile`, lexes, builds a flat
`[]RawStmt`, projects into `ModuleAST`. **Pure function**: no caches,
no goroutines, no shared state beyond the seen-unknown-names log
(§4). Concurrency-safe because it is otherwise stateless. Higher
layers (the graph builder, ticket 10's flag-aware parser) MAY wrap a
cache around it; this layer does not.

The signature deliberately omits a `FlagEnv` parameter. Ticket 10 will
add a *second* entry point — call it `ParseModuleWithEnv(path string,
env FlagEnv) ParsedModule` (final name TBD by that ticket) — that
returns a multi-facet shape. **This** entry point will continue to
exist as the IF-blind, flag-blind base layer. Both layers will share
the same `[]RawStmt` building blocks internally; ticket 10 specifies
the seam.

The parser MUST NOT walk the filesystem beyond the file it is asked to
read. `INCLUDE` arguments, if present, land in `Tail` as opaque
`RawStmt`s; nothing in this layer follows them.

---

## 6. Inventory — what the closure actually contains

Every command name appearing in the transitive `tools/archiver` closure,
verified by reading each `ya.make` directly. If the implementation
DIGGER finds a statement name in any of these files that is not in this
table, they MUST add it to `Tail`-only territory and update this
section — silent drops are forbidden.

| ya.make path                                 | Commands present                                                                          |
|----------------------------------------------|-------------------------------------------------------------------------------------------|
| `tools/archiver/ya.make`                     | `PROGRAM`, `PEERDIR`, `SRCS`, `SET`, `END`                                                |
| `library/cpp/archive/ya.make`                | `LIBRARY`, `SRCS`, `END`, `RECURSE_FOR_TESTS`                                             |
| `library/cpp/digest/md5/ya.make`             | `LIBRARY`, `SRCS`, `PEERDIR`, `END`, `RECURSE`                                            |
| `library/cpp/getopt/small/ya.make`           | `LIBRARY`, `PEERDIR`, `SRCS`, `END`                                                       |
| `contrib/libs/nayuki_md5/ya.make`            | `LIBRARY`, `LICENSE`, `LICENSE_TEXTS`, `VERSION`, `ORIGINAL_SOURCE`, `IF`, `SRCS`, `ELSE`, `ENDIF`, `END` |
| `library/cpp/string_utils/base64/ya.make`    | `LIBRARY`, `SRCS`, `PEERDIR`, `END`, `RECURSE`                                            |
| `library/cpp/colorizer/ya.make`              | `LIBRARY`, `SRCS`, `END`, `RECURSE_FOR_TESTS`                                             |

Of those, only `PROGRAM`, `LIBRARY`, `SRCS`, `PEERDIR`, `END` are
**understood** by the parser at this scope. Every other command lands
in `Tail` verbatim. `END` is special: it is recognised structurally
(matched against the opener and recorded as `EndLoc`) but its
`RawStmt.Args` is always empty in the closure — its semantic content
is empty here.

`IF` / `ELSEIF` / `ELSE` / `ENDIF` (occurring in
`contrib/libs/nayuki_md5/ya.make`) MUST be emitted as ordinary
`RawStmt` entries inside `Tail` and MUST NOT be evaluated. Ticket 10
walks `Tail` and elides taken/non-taken branches.

---

## 7. Test plan

The implementation ticket MUST land these tests in `parser_test.go`. The
present ticket only specifies them.

- `TestLexAtomsQuoted` — synthetic input mixing a double-quoted string
  with an embedded escaped quote, an atom with leading/trailing
  whitespace, a `$(VAR)` token, and a single-quoted string.
  Asserts: backslashes stripped from both quoted forms (rl6:31);
  `$(VAR)` survives intact inside the atom (rl6:77).
- `TestStmtFlatList` — golden test against
  `/home/pg/monorepo/yatool_orig/tools/archiver/ya.make`. Expects
  exactly five `RawStmt`s in order: `PROGRAM` (`Args: []`), `PEERDIR`
  (three args: `library/cpp/archive`, `library/cpp/digest/md5`,
  `library/cpp/getopt/small`), `SRCS` (one arg: `main.cpp`), `SET`
  (two args: `IDE_FOLDER`, `_Builders`), `END` (no args).
- `TestModuleASTArchiver` — same fixture, asserts: `Kind ==
  ModuleProgram`, `len(Sources) == 1` and `Sources[0].Path ==
  "main.cpp"`, `Peers == ["library/cpp/archive",
  "library/cpp/digest/md5", "library/cpp/getopt/small"]`,
  `len(Tail) == 1` (the `SET` call), `OpenLoc.Row == 1`, `EndLoc.Row
  == 15`.
- `TestModuleASTMd5` — `library/cpp/digest/md5/ya.make`, asserts `Kind
  == ModuleLibrary`, `Sources == ["md5.cpp"]`, `Peers ==
  ["contrib/libs/nayuki_md5", "library/cpp/string_utils/base64"]`,
  `Tail` contains the trailing `RECURSE(bench medium_ut ut)` call.
- `TestNayukiBothBranchesKept` — `contrib/libs/nayuki_md5/ya.make`.
  Asserts that **both** `md5-fast-x8664.S` (IF branch) and `md5.c`
  (ELSE branch) land in `Sources` because the `IF` is unevaluated.
  Documents the §1 limitation in the test name.
- `TestOpenerArgsRejected` — synthetic `PROGRAM(foo)\nEND()` — throws
  `*Exception` whose `Unwrap()` is `*ParseError`.
- `TestMissingEndRejected` — synthetic file with `PROGRAM()` and no
  `END()` — throws `*ParseError`.
- `TestUnknownStmtKeptOnceLogged` — synthetic input contains a bogus
  command (e.g. `WIBBLE(x y)`). Asserts: the `RawStmt` survives in
  `Tail`; the per-process log fires exactly once across two `ParseModule`
  invocations on inputs that both contain `WIBBLE`.
- `TestCommentsAndWhitespace` — synthetic input with `#` comments at
  file scope, between args inside `(...)` (covering rl6:82's
  `(list_elem | comment) spaces`), and inline at end of line.
  Asserts the comments are dropped and the surviving statements
  match expectation. Asserts `Loc.Row` advances correctly across
  comment lines.
- `TestQuotedAtomMix` — single statement with mixed quoted and atom
  args, e.g. `SET("a b" 'c\'d' $(VAR) plain)`. Verifies argument
  order, escape strip on both quoted forms, and `$(VAR)` preservation
  inside the atom slot.

No `t.Parallel` test is required at this ticket. `ParseModule` is
stateless (modulo the seen-unknown-names log, which is `sync.Map` and
safe by construction); parallelism is the caller's concern, not a
contract this layer defends.

---

## 8. Performance budget (pointer)

The parser falls under the project-wide 1 s budget set by `GOALS.md`.
Micro-targets are `design_perf.md`'s job (a future ticket). This
document fixes only the design choices that bear on perf:

- One `os.ReadFile` per file. Whole content held as one `[]byte` for
  the lifetime of the parse.
- Token text is a subslice into that `[]byte` during lexing; no string
  copying inside the lex loop. The conversion `string(buf[start:end])`
  happens **exactly once** per emitted argument, when the §3 pass
  appends to `RawStmt.Args`.
- `[]RawStmt` is allocated with `make([]RawStmt, 0, 32)`. The largest
  closure `ya.make` is `library/cpp/getopt/small/ya.make` at ~22
  statements (§6); 32 is safely above that and avoids re-grow during
  the §3 build.
- No `regexp`, no `bufio.Scanner`, no `text/scanner`. Hand-written byte
  machine.
- No caching at this layer. A parse cache is the caller's
  responsibility; ticket 10 will add one keyed by `(path, env)`.

---

## 9. Failure modes

The parser throws via `Throw` / `ThrowFmt` per `STYLE.md`'s rule (the
forbidden shape there is the pure pass-through `if err != nil { return
err }`; lex / parse errors use the `Throw` channel). Two typed
payloads:

- `LexError` (§2): malformed token, unbalanced quote, unbalanced
  escape at the lexical level. Carries `Path`, `Row`, `Col`,
  `Excerpt` — format matches upstream `TVisitorCtx::Here` at
  rl6:105–118.
- `ParseError`: structural violation at the statement level. Cases:
  - opener absent (no `PROGRAM`/`LIBRARY` in file);
  - opener has args (`PROGRAM(x)` or `LIBRARY(x)`);
  - multiple openers in one file;
  - no `END`;
  - multiple `END`s;
  - unbalanced `(`/`)` at the statement level.
  Carries `Path`, `Loc`, and a short message.

File-not-found / read failure: the implementation MUST `Throw2(os.ReadFile(path))`
per STYLE.md, letting `throw.go`'s `*Exception` wrapper carry the
underlying `os` error verbatim. Wrapping it in a synthetic `ParseError`
with `Row=0, Col=0` is acceptable but not required; the test plan does
not assert on read-failure shape.

Both `LexError` and `ParseError` MUST be reachable by
`errors.As(err, &target)` after passing through `(*Exception).AsError()`
or `(*Exception).Unwrap()`. This matches `STYLE.md`'s section on when an
`error` is part of a function's contract: tests in §7
(`TestOpenerArgsRejected`, `TestMissingEndRejected`) discriminate on
the typed payload, so the typed payload **is** part of the contract.

---

## 10. File layout (normative, narrow)

The implementation ticket lands these files in the workspace root, per
`CLAUDE.md`'s "All `.go` files live flat in the workspace root" rule:

- `parser.go` — public types (`Loc`, `RawStmt`, `Source`, `Peer`,
  `ModuleKind`, `ModuleAST`, typed errors `LexError`, `ParseError`)
  and the public `ParseModule` entry point.
- `parser_lex.go` — byte lexer; `Token` enum; the `isAtomByte` /
  `isCommandStartByte` / `isCommandTailByte` helpers transcribed from
  rl6:75–79.
- `parser_stmt.go` — `[]RawStmt` builder from the token stream
  (§3).
- `parser_ast.go` — typed AST projection from `[]RawStmt` to
  `ModuleAST` (§4).
- `parser_test.go` — the tests from §7.

This list deliberately does **not** include `parser_if.go`,
`parser_include.go`, or `parser_flags.go`. Ticket 10 adds those; the
present file boundaries must accommodate them without churn — i.e.
`parser_lex.go` and `parser_stmt.go` MUST stay reusable by ticket 10
without modification.

---

## 11. What ticket 10 owns

This section is a contract: the present design must not preclude any of
these. Each bullet names the upstream reference so ticket 10 can pick
up without re-discovery.

- `INCLUDE` expansion with cycle detection. Reference:
  `makefile_reader.cpp:97-109` (path resolution, abs vs. relative);
  `makefile_reader.cpp:66-67` (`TIncludeLoopException` re-raise across
  multimodule re-parse).
- `IF` / `ELSEIF` / `ELSE` / `ENDIF` evaluation against a `FlagEnv`.
  Reference: `eval_context.h:140-189` (`BranchTaken`, `OnStatement`,
  `ShouldSkip` interplay); `eval_context.cpp:267` (`ShouldSkip`
  body).
- Multimodule (`PROTO_LIBRARY`, `DYNAMIC_LIBRARY`, `PACKAGE`, …) facet
  expansion. Reference: `build/ymake.core.conf:2374-2398`
  (`DYNAMIC_LIBRARY`); `build/ymake.core.conf:2495` (`PACKAGE`); and
  the upstream "lex once, replay per facet" TODO at
  `makefile_reader.cpp:46-48`.
- `$(VAR)` interpolation, notably for `INCLUDE` arguments. Reference:
  `makefile_reader.cpp:58` (`EvalExpr(Context->Vars(), args[0])`).
- The flag environment translation
  (`--target-platform=default-linux-aarch64`, `--musl`,
  `--sandboxing`, `--host-platform-flag=MUSL=yes`) into `FlagEnv.Vars`
  bindings. Reference: `srun.sh` and the matching ymake
  command-line plumbing in `devtools/ya/`.
- Caching keyed by `(path, env)`. Reference: the upstream lock-free
  read pattern around per-run module state.

The seam at this layer is `Tail []RawStmt`. Ticket 10 walks `Tail` in
flag-evaluated order, expanding `IF`/`ENDIF` blocks and resolving
`INCLUDE` against the same `[]RawStmt` builder this layer exposes
internally.
