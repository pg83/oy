# Reference Node Catalog

This document catalogs all execution node types from the reference `sg.json` generated from `/home/pg/monorepo/yatool_orig/tools/archiver`.

## Overview

- **Total nodes**: 3730
- **Node types**: 7 (CC, AS, AR, JS, LD, R6, CP)
- **Platforms**: 2 (aarch64, x86_64)
- **Modules**: 67 unique modules

## Node Type Distribution

| Type  | Count | Percentage | Name              | Color          |
|-------|-------|------------|-------------------|----------------|
| CC    | 3571  | 95.7%      | Compile           | green          |
| AS    | 83    | 2.2%       | Assembly          | light-green    |
| AR    | 48    | 1.3%       | Archive           | light-red      |
| JS    | 23    | 0.6%       | JavaScript        | magenta        |
| LD    | 3     | 0.1%       | Link              | light-blue     |
| R6    | 1     | 0.0%       | Ragel6            | yellow         |
| CP    | 1     | 0.0%       | Copy              | light-cyan     |

## Platform Distribution

| Platform                    | Count | Percentage |
|-----------------------------|-------|------------|
| default-linux-aarch64       | 1933  | 51.8%      |
| default-linux-x86_64        | 1797  | 48.2%      |

**Note**: Many modules have both platform variants (dual compilation).

## Module Distribution (Top 10)

| Module                              | Count | Percentage |
|-------------------------------------|-------|------------|
| contrib/libs/musl                   | 2656  | 71.2%      |
| contrib/libs/cxxsupp/builtins       | 344   | 9.2%       |
| contrib/restricted/abseil-cpp       | 157   | 4.2%       |
| contrib/libs/cxxsupp/libcxx         | 118   | 3.2%       |
| contrib/tools/yasm                  | 80    | 2.1%       |
| contrib/libs/jemalloc               | 64    | 1.7%       |
| contrib/libs/tcmalloc/no_percpu_cache | 57    | 1.5%       |
| util                                | 39    | 1.0%       |
| contrib/libs/asmlib                 | 27    | 0.7%       |
| contrib/libs/libunwind              | 20    | 0.5%       |

## Command Inventory

| Command                           | Count |
|-----------------------------------|-------|
| clang                             | 3206  |
| clang++                           | 426   |
| python3                           | 81    |
| yasm                              | 25    |
| ragel6                            | 1     |

## Python Build Scripts

| Script            | Count | Purpose                      |
|-------------------|-------|------------------------------|
| link_lib.py       | 48    | Library archiving (AR nodes) |
| gen_join_srcs.py  | 23    | Source generation (JS nodes) |
| vcs_info.py       | 3     | Version injection (LD nodes) |
| link_exe.py       | 3     | Executable linking (LD nodes) |
| fs_tools.py       | 4     | File operations (CP + LD)   |

---

## Node Type Specifications

### 1. CC Node (Compile)

**KV.p**: `"CC"`  
**KV.pc**: `"green"`  
**Count**: 3571

**Characteristics**:
- Exactly 1 command per node
- 1 output: `.c.o` or `.cpp.o` in `$(BUILD_ROOT)/<module>/`
- Inputs: 1-1230 files (source + include paths), average 127
- Deps: 0-1 (rare direct compile deps)
- Platform: `default-linux-aarch64` or `default-linux-x86_64`

**Compiler pattern**:
```
$(CLANG-2403293607)/bin/clang --target=<arch>-linux-gnu -march=<arch> \
  --sysroot=/nowhere -B$(OS_SDK_ROOT-sbr:<id>)/usr/bin \
  -fdebug-prefix-map=$(BUILD_ROOT)=/-B \
  -fdebug-prefix-map=$(SOURCE_ROOT)=/-S \
  -fdebug-prefix-map=$(TOOL_ROOT)=/-T \
  -c <source> -o <object>
```

**Example structure**:
```json
{
  "uid": "base64-encoded-hash",
  "self_uid": "base64-encoded-hash",
  "stats_uid": "base64-encoded-hash",
  "cmds": [{
    "cmd_args": [
      "$(CLANG-2403293607)/bin/clang",
      "--target=aarch64-linux-gnu",
      "-march=armv8-a",
      "--sysroot=/nowhere",
      "-c",
      "source.c",
      "-o",
      "$(BUILD_ROOT)/module/source.c.o",
      "-I$(BUILD_ROOT)",
      "-I$(SOURCE_ROOT)"
    ],
    "env": {
      "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
      "CPATH": "",
      "DYLD_LIBRARY_PATH": "",
      "LIBRARY_PATH": "",
      "SDKROOT": ""
    }
  }],
  "inputs": [
    "$(SOURCE_ROOT)/module/source.c",
    "$(SOURCE_ROOT)/contrib/libs/linux-headers/...",
    "..."
  ],
  "outputs": ["$(BUILD_ROOT)/module/source.c.o"],
  "deps": [],
  "kv": {"p": "CC", "pc": "green"},
  "target_properties": {
    "module_dir": "module/path",
    "module_lang": "cpp",
    "module_type": "lib"
  },
  "env": {},
  "platform": "default-linux-aarch64",
  "requirements": {"cpu": 1, "network": "restricted", "ram": 32},
  "sandboxing": true,
  "tags": [],
  "foreign_deps": null,
  "host_platform": false
}
```

**High-input cases**: 362 nodes have >500 inputs (libcxx averaging 850+ includes)

---

### 2. AS Node (Assembly/Codegen)

**KV.p**: `"AS"`  
**KV.pc**: `"light-green"`  
**Count**: 83

**Sub-types**:
- yasm: 25 nodes (contrib/libs/asmlib)
- clang assembly: 58 nodes (musl, builtins)

**Characteristics**:
- 1 command per node
- 1 output: `.S.o` in `$(BUILD_ROOT)/<module>/`
- Inputs: 1-1175 files (include paths + source)

**Yasm pattern**:
```
$(BUILD_ROOT)/contrib/tools/yasm/yasm -felf64 <src.asm> -o <src.asm.o>
```

**Example structure**:
```json
{
  "cmds": [{
    "cmd_args": [
      "$(BUILD_ROOT)/contrib/tools/yasm/yasm",
      "-felf64",
      "src.asm",
      "-o",
      "$(BUILD_ROOT)/module/src.asm.o"
    ],
    "env": {
      "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
      "CPATH": "",
      "DYLD_LIBRARY_PATH": "",
      "LIBRARY_PATH": "",
      "SDKROOT": ""
    }
  }],
  "inputs": [
    "$(SOURCE_ROOT)/module/src.asm",
    "$(SOURCE_ROOT)/contrib/libs/linux-headers/...",
    "..."
  ],
  "outputs": ["$(BUILD_ROOT)/module/src.asm.o"],
  "kv": {"p": "AS", "pc": "light-green"},
  "target_properties": {
    "module_dir": "module/path",
    "module_lang": "cpp",
    "module_type": "lib"
  }
}
```

---

### 3. AR Node (Archive/Static Library)

**KV.p**: `"AR"`  
**KV.pc**: `"light-red"`  
**KV.show_out**: `"yes"`  
**Count**: 48

**Characteristics**:
- Exactly 1 python3 invocation (`link_lib.py`)
- 1 output: `lib<module>.a` in `$(BUILD_ROOT)/<module>/`
- Inputs: 3-2921 files (.o files + generated sources)
- Deps: 1-1327 (all compile node deps)

**Link command pattern**:
```
$(YMAKE_PYTHON3-1002064631)/bin/python3 \
  $(SOURCE_ROOT)/build/scripts/link_lib.py \
  $(CLANG-2403293607)/bin/llvm-ar LLVM_AR gnu \
  $(BUILD_ROOT) None -- -- <output.a> <input1.o> ...
```

**Environment vars**: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT

**Example structure**:
```json
{
  "cmds": [{
    "cmd_args": [
      "$(YMAKE_PYTHON3-1002064631)/bin/python3",
      "$(SOURCE_ROOT)/build/scripts/link_lib.py",
      "$(CLANG-2403293607)/bin/llvm-ar",
      "LLVM_AR",
      "gnu",
      "$(BUILD_ROOT)",
      "None",
      "--",
      "--",
      "$(BUILD_ROOT)/module/libmodule.a",
      "$(BUILD_ROOT)/module/source1.c.o",
      "$(BUILD_ROOT)/module/source2.c.o",
      "..."
    ],
    "env": {
      "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
      "CPATH": "",
      "DYLD_LIBRARY_PATH": "",
      "LIBRARY_PATH": "",
      "SDKROOT": ""
    }
  }],
  "inputs": [
    "$(BUILD_ROOT)/module/source1.c.o",
    "$(BUILD_ROOT)/module/source2.c.o",
    "..."
  ],
  "outputs": ["$(BUILD_ROOT)/module/libmodule.a"],
  "deps": ["uid1", "uid2", "..."],
  "kv": {"p": "AR", "pc": "light-red", "show_out": "yes"},
  "target_properties": {
    "module_dir": "module/path",
    "module_lang": "cpp",
    "module_type": "lib"
  }
}
```

**Special cases**:
- `contrib/libs/cxxsupp/libcxx`: 165 deps, 1293 inputs (largest)
- `contrib/libs/cxxsupp/builtins`: 165 deps, 605 inputs

---

### 4. JS Node (JavaScript Generation)

**KV.p**: `"JS"`  
**KV.pc**: `"magenta"`  
**Count**: 23

**Characteristics**:
- Exactly 1 python3 invocation (`gen_join_srcs.py`)
- 1 output: Generated source file in `$(BUILD_ROOT)/`
- Inputs: Source template files (941 max)

**Gen command pattern**:
```
$(YMAKE_PYTHON3-1002064631)/bin/python3 \
  $(SOURCE_ROOT)/build/scripts/gen_join_srcs.py \
  <output.cpp> --ya-start-command-file <input1.cpp> ...
```

**Example structure**:
```json
{
  "cmds": [{
    "cmd_args": [
      "$(YMAKE_PYTHON3-1002064631)/bin/python3",
      "$(SOURCE_ROOT)/build/scripts/gen_join_srcs.py",
      "$(BUILD_ROOT)/module/generated.cpp",
      "--ya-start-command-file",
      "$(SOURCE_ROOT)/module/template1.cpp",
      "$(SOURCE_ROOT)/module/template2.cpp",
      "..."
    ],
    "env": {}
  }],
  "inputs": [
    "$(SOURCE_ROOT)/module/template1.cpp",
    "$(SOURCE_ROOT)/module/template2.cpp",
    "..."
  ],
  "outputs": ["$(BUILD_ROOT)/module/generated.cpp"],
  "kv": {"p": "JS", "pc": "magenta"},
  "target_properties": {
    "module_dir": "module/path",
    "module_lang": "cpp",
    "module_type": "lib"
  }
}
```

---

### 5. LD Node (Link/Executable)

**KV.p**: `"LD"`  
**KV.pc**: `"light-blue"`  
**KV.show_out**: `"yes"`  
**Count**: 3

**4-Command sequence** (all LD nodes have exactly 4 commands):

**Command 1: vcs_info.py** - Generate version.c
```
$(YMAKE_PYTHON3-1002064631)/bin/python3 \
  $(SOURCE_ROOT)/build/scripts/vcs_info.py \
  $(VCS)/vcs.json <output-version.c> \
  $(SOURCE_ROOT)/build/scripts/c_templates/svn_interface.c
```

**Command 2: clang** - Compile version.c
```
$(CLANG-2403293607)/bin/clang --target=<arch>-linux-gnu -march=<arch> \
  --sysroot=/nowhere -B$(OS_SDK_ROOT-sbr:<id>)/usr/bin \
  -c <version.c> -o <version.c.o>
```

**Command 3: link_exe.py** - Link final executable
```
$(YMAKE_PYTHON3-1002064631)/bin/python3 \
  $(SOURCE_ROOT)/build/scripts/link_exe.py \
  --start-plugins <plugins> --end-plugins --clang-ver 20 --source-root <source-root>
```

**Command 4: fs_tools.py** - Copy/install executable
```
$(YMAKE_PYTHON3-1002064631)/bin/python3 \
  $(SOURCE_ROOT)/build/scripts/fs_tools.py \
  link_or_copy_to_dir --no-check <output-dir>
```

**Characteristics**:
- 1 output: Executable in `$(BUILD_ROOT)/<module>/`
- Inputs: 263-1054 files (object files + headers)
- Deps: 23-89 (all archive deps)
- Platform: 2 aarch64, 1 x86_64
- Environment vars: 5 per python command, 1 per clang command

**Example structure**:
```json
{
  "cmds": [
    {
      "cmd_args": [
        "$(YMAKE_PYTHON3-1002064631)/bin/python3",
        "$(SOURCE_ROOT)/build/scripts/vcs_info.py",
        "$(VCS)/vcs.json",
        "$(BUILD_ROOT)/module/_version.c",
        "$(SOURCE_ROOT)/build/scripts/c_templates/svn_interface.c"
      ],
      "env": {
        "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
        "CPATH": "",
        "DYLD_LIBRARY_PATH": "",
        "LIBRARY_PATH": "",
        "SDKROOT": ""
      }
    },
    {
      "cmd_args": [
        "$(CLANG-2403293607)/bin/clang",
        "--target=aarch64-linux-gnu",
        "-march=armv8-a",
        "-c",
        "$(BUILD_ROOT)/module/_version.c",
        "-o",
        "$(BUILD_ROOT)/module/_version.c.o"
      ],
      "env": {
        "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
        "CPATH": "",
        "DYLD_LIBRARY_PATH": "",
        "LIBRARY_PATH": "",
        "SDKROOT": ""
      }
    },
    {
      "cmd_args": [
        "$(YMAKE_PYTHON3-1002064631)/bin/python3",
        "$(SOURCE_ROOT)/build/scripts/link_exe.py",
        "--start-plugins",
        "<plugins>",
        "--end-plugins",
        "--clang-ver",
        "20",
        "--source-root",
        "$(SOURCE_ROOT)"
      ],
      "env": {
        "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
        "CPATH": "",
        "DYLD_LIBRARY_PATH": "",
        "LIBRARY_PATH": "",
        "SDKROOT": ""
      }
    },
    {
      "cmd_args": [
        "$(YMAKE_PYTHON3-1002064631)/bin/python3",
        "$(SOURCE_ROOT)/build/scripts/fs_tools.py",
        "link_or_copy_to_dir",
        "--no-check",
        "$(BUILD_ROOT)/module/"
      ],
      "env": {
        "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
        "CPATH": "",
        "DYLD_LIBRARY_PATH": "",
        "LIBRARY_PATH": "",
        "SDKROOT": ""
      }
    }
  ],
  "inputs": [
    "$(BUILD_ROOT)/module/_version.c.o",
    "$(BUILD_ROOT)/module/main.c.o",
    "..."
  ],
  "outputs": ["$(BUILD_ROOT)/module/executable"],
  "deps": ["uid1", "uid2", "..."],
  "kv": {"p": "LD", "pc": "light-blue", "show_out": "yes"},
  "target_properties": {
    "module_dir": "module/path",
    "module_lang": "cpp",
    "module_type": "bin"
  }
}
```

---

### 6. R6 Node (Ragel6 Codegen)

**KV.p**: `"R6"`  
**KV.pc**: `"yellow"`  
**Count**: 1

**Characteristics**:
- 1 ragel6 invocation
- 1 output: Generated source
- Single node in `util` module

**Example structure**:
```json
{
  "cmds": [{
    "cmd_args": [
      "$(BUILD_ROOT)/contrib/tools/ragel6/ragel6",
      "-C",
      "-o",
      "$(BUILD_ROOT)/util/generated.cpp",
      "$(SOURCE_ROOT)/util/input.rl6"
    ],
    "env": {}
  }],
  "inputs": ["$(SOURCE_ROOT)/util/input.rl6"],
  "outputs": ["$(BUILD_ROOT)/util/generated.cpp"],
  "kv": {"p": "R6", "pc": "yellow"},
  "target_properties": {
    "module_dir": "util",
    "module_lang": "cpp",
    "module_type": "lib"
  }
}
```

---

### 7. CP Node (Copy Operation)

**KV.p**: `"CP"`  
**KV.pc**: `"light-cyan"`  
**Count**: 1

**Characteristics**:
- 1 fs_tools.py copy invocation
- 1 output: Copied file
- Single node in `contrib/libs/musl/include`

**Copy command**:
```
$(YMAKE_PYTHON3-1002064631)/bin/python3 \
  $(SOURCE_ROOT)/build/scripts/fs_tools.py copy <src> <dst>
```

**Example structure**:
```json
{
  "cmds": [{
    "cmd_args": [
      "$(YMAKE_PYTHON3-1002064631)/bin/python3",
      "$(SOURCE_ROOT)/build/scripts/fs_tools.py",
      "copy",
      "$(SOURCE_ROOT)/contrib/libs/musl/include/header.h",
      "$(BUILD_ROOT)/contrib/libs/musl/include/header.h"
    ],
    "env": {}
  }],
  "inputs": ["$(SOURCE_ROOT)/contrib/libs/musl/include/header.h"],
  "outputs": ["$(BUILD_ROOT)/contrib/libs/musl/include/header.h"],
  "kv": {"p": "CP", "pc": "light-cyan"},
  "target_properties": {
    "module_dir": "contrib/libs/musl/include",
    "module_lang": "cpp",
    "module_type": "lib"
  }
}
```

---

## Common Node Structure

**Required fields present in ALL 3730 nodes**:

```json
{
  "uid": "base64-encoded-hash",
  "self_uid": "base64-encoded-hash",
  "stats_uid": "base64-encoded-hash",
  "cmds": [{"cmd_args": [...], "env": {...}}],
  "inputs": ["path1", "path2", ...],
  "outputs": ["path1"],
  "deps": ["uid1", "uid2", ...],
  "kv": {"p": "CC|AS|AR|JS|LD|R6|CP", "pc": "color"},
  "target_properties": {
    "module_dir": "module/path",
    "module_lang": "cpp|go|python",
    "module_type": "bin|lib|go_lib"
  },
  "env": {
    "ARCADIA_ROOT_DISTBUILD": "$(SOURCE_ROOT)",
    "CPATH": "",
    "DYLD_LIBRARY_PATH": "...",
    "LIBRARY_PATH": "",
    "SDKROOT": ""
  },
  "platform": "default-linux-aarch64|default-linux-x86_64",
  "requirements": {"cpu": 1, "network": "restricted", "ram": 32},
  "sandboxing": true,
  "tags": [],
  "foreign_deps": null,
  "host_platform": false
}
```

---

## Key Insights and Patterns

### Per-Source Node Generation

- Reference creates **ONE node per compilable source file** (not per module)
- Source type detection drives node type:
  - `.c/.cpp/.cc/.cxx` → CC (Compile)
  - `.S/.asm` → AS (Assembly)
  - `.rl6` → R6 (Ragel6)
- Platform duplication: Each source gets nodes for both aarch64 and x86_64 when applicable

### Header Include Inflation

- Inputs array intentionally tracks **ALL transitive includes** (100-1000 per compile node)
- libcxx nodes average 850+ inputs (extensive stdlib headers)
- This is **NOT an artifact**—required for build correctness and caching
- Header paths follow `$(SOURCE_ROOT)/contrib/libs/` hierarchy

### Command Sequencing

- **Only LD nodes have multi-command sequences** (4 commands per LD node)
- All other node types have **exactly 1 command**
- Commands use expanded `$(VARIABLE)` references for platform resolution
- Environment vars are set per-command (not per-node in all cases)

### Dependency Topology

| Node Type | Topology | Characteristics |
|-----------|----------|-----------------|
| CC, AS, R6, CP | Leaf | Minimal deps, no child nodes |
| JS | Intermediate | Tool outputs consumed later by CC nodes |
| AR | Aggregator | Depends on all compile nodes for a module |
| LD | Root | Depends on all archive deps for a program |

### Platform Variant Handling

- Variants are **EXPLICIT duplicates**, not parameterized strings
- Compiler flags change per platform (`--target`, `--march`, SDK ID)
- 1933 aarch64 + 1797 x86_64 = 3730 total nodes
- Not all modules have both variants (depends on module configuration)

---

## Implementation Status

| Node Type | Status | Notes |
|-----------|--------|-------|
| CC | ❌ Not implemented | Need per-source file node generation |
| AS | ❌ Not implemented | Need yasm/clang assembly support |
| AR | ❌ Not implemented | Current implementation has module-level nodes only |
| JS | ❌ Not implemented | Need gen_join_srcs.py generation |
| LD | ❌ Not implemented | Need 4-command sequence generation |
| R6 | ❌ Not implemented | Need ragel6 codegen support |
| CP | ❌ Not implemented | Need fs_tools.py copy support |

**Current implementation**: Module-level nodes (1 per module = 12 total)
**Target implementation**: Source-level nodes (1 per source file = 3730 total)

---

## Validation Criteria

To achieve structural graph equality against the reference `sg.json`:

1. **Node count**: Exactly 3730 nodes
   - CC: 3571 (95.7%)
   - AS: 83 (2.2%)
   - AR: 48 (1.3%)
   - JS: 23 (0.6%)
   - LD: 3 (0.1%)
   - R6: 1 (0.0%)
   - CP: 1 (0.0%)

2. **Platform distribution**:
   - aarch64: 1933 nodes
   - x86_64: 1797 nodes

3. **Module distribution**: Top 5 modules match reference percentages

4. **Command arguments**: Preserve exact ordering and variable references

5. **Input arrays**: Include all transitive headers (not just source files)

6. **UID generation**: Can differ (hash-based acceptance per ACCEPTANCE.md)

7. **Multi-command nodes**: LD must have exactly 4 commands in correct sequence

---

## Implementation Considerations

### Performance Impact

- **Current**: 12-node graph, 531μs average generation time
- **Target**: 3730-node graph, must retain <1s performance
- **Optimization strategy**:
  - Batch header scanning
  - Parallel node generation per module
  - Minimal memory allocations during graph construction

### Memory Impact

- 3730 nodes × 16 fields × 100 average inputs = **75-100MB** in memory
- Input arrays dominate memory (transitive includes)
- Acceptable given <1s target runtime

### Architecture Implications

- **Current**: Module-level nodes (1 per module)
- **Required**: Source-level nodes (1 per source file)
- **Change needed**: Replace `createFinalNode()` with per-source node dispatch
- **Strategy needed**: Interface for `ExecutionNodeBuilder` with type-specific implementations

---

## Python Build Scripts Documentation

### link_lib.py (48 AR nodes)

**Purpose**: Archive object files into static libraries (.a files)

**Usage pattern**:
```bash
python3 $(SOURCE_ROOT)/build/scripts/link_lib.py \
  $(CLANG)/bin/llvm-ar LLVM_AR gnu \
  $(BUILD_ROOT) None -- \
  -- $(BUILD_ROOT)/module/libmodule.a \
  $(BUILD_ROOT)/module/source1.c.o \
  $(BUILD_ROOT)/module/source2.c.o \
  ...
```

**Environment variables**: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT

---

### gen_join_srcs.py (23 JS nodes)

**Purpose**: Generate C++ source files from input templates

**Usage pattern**:
```bash
python3 $(SOURCE_ROOT)/build/scripts/gen_join_srcs.py \
  $(BUILD_ROOT)/module/generated.cpp \
  --ya-start-command-file \
  $(SOURCE_ROOT)/module/template1.cpp \
  $(SOURCE_ROOT)/module/template2.cpp \
  ...
```

**Characteristics**: Up to 941 input template files per invocation

---

### vcs_info.py (3 LD nodes)

**Purpose**: Generate version.c from VCS metadata

**Usage pattern**:
```bash
python3 $(SOURCE_ROOT)/build/scripts/vcs_info.py \
  $(VCS)/vcs.json \
  $(BUILD_ROOT)/module/_version.c \
  $(SOURCE_ROOT)/build/scripts/c_templates/svn_interface.c
```

**Environment variables**: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT

---

### link_exe.py (3 LD nodes)

**Purpose**: Link executable from object files and libraries

**Usage pattern**:
```bash
python3 $(SOURCE_ROOT)/build/scripts/link_exe.py \
  --start-plugins <plugins> \
  --end-plugins \
  --clang-ver 20 \
  --source-root $(SOURCE_ROOT)
```

**Environment variables**: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT

---

### fs_tools.py (1 CP + 3 LD nodes)

**Purpose**: Filesystem operations (copy, link, install)

**Usage patterns**:
```bash
# CP node: copy file
python3 $(SOURCE_ROOT)/build/scripts/fs_tools.py \
  copy <src> <dst>

# LD node (command 4): install executable
python3 $(SOURCE_ROOT)/build/scripts/fs_tools.py \
  link_or_copy_to_dir --no-check <output-dir>
```

**Environment variables**: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT (for LD only)

---

## Reference Patterns

### CC Node Reference Pattern

From `sg.json` CC nodes:
- KV.p = "CC"
- KV.pc = "green"
- 1 command: clang/clang++ with --target, --march, --sysroot flags
- Output: $(BUILD_ROOT)/<module>/<source>.c.o or <source>.cpp.o
- Input array includes all transitive header dependencies
- Environment: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT

### AR Node Reference Pattern

From `sg.json` AR nodes:
- KV.p = "AR"
- KV.pc = "light-red"
- KV.show_out = "yes"
- 1 command: python3 link_lib.py with llvm-ar invocation
- Output: $(BUILD_ROOT)/<module>/lib<module>.a
- Inputs: All .o files from compilation + generated sources
- Deps: All compile node UIDs for the module
- Environment: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT

### LD Node Reference Pattern

From `sg.json` LD nodes:
- KV.p = "LD"
- KV.pc = "light-blue"
- KV.show_out = "yes"
- 4 commands in sequence:
  1. python3 vcs_info.py (generate version.c)
  2. clang (compile version.c to .o)
  3. python3 link_exe.py (link executable)
  4. python3 fs_tools.py (copy/install)
- Output: Executable in $(BUILD_ROOT)/<module>/
- Inputs: Version.c.o + all object files + transitive headers
- Deps: All archive node UIDs from PEERDIR dependencies
- Environment: ARCADIA_ROOT_DISTBUILD, CPATH, DYLD_LIBRARY_PATH, LIBRARY_PATH, SDKROOT (per-command)
