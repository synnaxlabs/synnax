# PR Separation Plan: sy-3498-oracle

## Overview

This document outlines the plan to split the `sy-3498-oracle` branch (~150K LOC) into 9
manageable PRs for review.

**Key Constraints:**

- Breaking release — all components upgrade together
- One reviewer for all PRs
- Generated code needs full review (schema correctness, output correctness, style)
- Strict linear merge sequence
- Generated code bundled with schemas (not split across consumers)

---

## PR Summary

| #   | PR Name                 | Est. Size | Status  |
| --- | ----------------------- | --------- | ------- |
| 1   | Oracle Tool             | ~47K      | Pending |
| 2   | x/* Foundation          | ~12K      | Pending |
| 3   | Arc System              | ~20K      | Pending |
| 4   | API Types               | ~20K      | Pending |
| 5   | Server Implementation   | ~12K      | Pending |
| 6   | TypeScript Client       | ~4K       | Pending |
| 7   | Python Client           | ~3K       | Pending |
| 8   | C++ Client              | ~5K       | Pending |
| 9   | Driver                  | ~8K       | Pending |

**Total:** ~131K LOC across 9 PRs

---

## Merge Sequence

```
PR 1: Oracle Tool
    ↓
PR 2: x/* Foundation (x schemas + x generated + x non-generated)
    ↓
PR 3: Arc System (arc schemas + arc generated + arc non-generated)
    ↓
PR 4: API Types (service schemas + ALL generated for server + clients)
    ↓
PR 5: Server Implementation (server non-generated only)
    ↓
PR 6: TypeScript Client (TS non-generated only)
    ↓
PR 7: Python Client (Py non-generated only)
    ↓
PR 8: C++ Client (C++ non-generated only)
    ↓
PR 9: Driver
```

---

## Schema → Generated Code Strategy

**Key insight:** A single schema generates code for multiple languages/components.

For example, `channel.oracle` generates:
- `core/pkg/distribution/channel/types.gen.go` (server)
- `client/ts/src/channel/types.gen.ts` (TS client)
- `client/py/synnax/channel/types_gen.py` (Py client)
- `client/cpp/channel/types.gen.h` (C++ client)

**Strategy:** Schemas and ALL their generated outputs land together in one PR.
Implementation code (non-generated) lands in subsequent PRs.

This ensures:
- Type definitions are consistent across languages
- Generated types exist before implementation code needs them
- Reviewers can verify schema→output correctness in one place

---

## PR 1: Oracle Tool

**Purpose:** Introduce the Oracle code generation framework.

**Review Focus:**
- Generator architecture and plugin design
- Template quality and correctness
- LSP implementation
- CLI interface

### Files Included

```
oracle/
├── analyzer/
├── cli/
├── cmd/
├── domain/
├── exec/
├── formatter/
├── lsp/
├── output/
├── parser/
├── paths/
├── plugin/
│   ├── cpp/
│   ├── enum/
│   ├── framework/
│   ├── go/
│   ├── pb/
│   ├── primitives/
│   ├── py/
│   ├── resolver/
│   └── ts/
├── resolution/
├── testutil/
├── go.mod
├── go.sum
└── oracle.go
```

**File count:** ~127 files

---

## PR 2: x/* Foundation

**Purpose:** Foundational utilities with their schemas and generated types.

**Review Focus:**
- Namespace consistency (`xerrors` → `x::errors`, etc.)
- Schema correctness for primitive types (telem, color, spatial, control, status, label)
- Generated code style across Go, TypeScript, C++

### Schemas Included

```
schemas/
├── color.oracle
├── control.oracle
├── label.oracle
├── spatial.oracle
├── status.oracle
└── telem.oracle
```

### Generated Code Included

**x/cpp generated:**
```
x/cpp/color/{types,json,proto}.gen.h
x/cpp/control/{types,json,proto}.gen.h
x/cpp/label/{types,json,proto}.gen.h
x/cpp/spatial/{types,json,proto}.gen.h
x/cpp/status/{types,json,proto}.gen.h
x/cpp/telem/{types,json,proto}.gen.h
```

**x/go generated:**
```
x/go/color/{types.gen.go, pb/translator.gen.go}
x/go/control/{types.gen.go, pb/translator.gen.go}
x/go/label/{types.gen.go, pb/translator.gen.go}
x/go/spatial/{types.gen.go, pb/translator.gen.go}
x/go/status/{types.gen.go, pb/translator.gen.go}
x/go/telem/{types.gen.go, pb/translator.gen.go}
```

**x/ts generated:**
```
x/ts/src/control/types.gen.ts
x/ts/src/label/types.gen.ts
x/ts/src/spatial/types.gen.ts
x/ts/src/status/types.gen.ts
```

### Non-Generated Code Included

**x/cpp (namespace refactor + new utilities):**
```
x/cpp/
├── args/           (renamed from xargs/)
├── binary/
├── breaker/
├── caseconv/
├── cli/
├── color/color.h
├── control/control.h
├── defer/
├── env/
├── errors/         (renamed from xerrors/)
├── fs/
├── json/           (enhanced with any.h, struct.h, value.h)
├── kv/
├── label/label.h
├── lib/            (renamed from xlib/)
├── log/            (renamed from xlog/)
├── loop/
├── lua/            (renamed from xlua/)
├── mem/            (new - indirect.h)
├── notify/
├── os/             (renamed from xos/)
├── path/           (renamed from xpath/)
├── pb/             (new)
├── queue/
├── shutdown/       (renamed from xshutdown/)
├── spatial/spatial.h
├── status/status.h
├── telem/
├── test/           (renamed from xtest/)
├── thread/         (renamed from xthread/)
└── uuid/           (new)
```

**x/go and x/ts non-generated files**

**File count:** ~150 files

---

## PR 3: Arc System

**Purpose:** Arc compiler, runtime, and related schemas.

**Review Focus:**
- Arc schema correctness (ir, graph, text, module, compiler, types)
- Generated code for Arc across Go, C++, TypeScript
- Compiler and runtime implementation
- Protobuf structure reorganization

### Schemas Included

```
schemas/
├── arc.oracle
└── arc/
    ├── compiler.oracle
    ├── graph.oracle
    ├── ir.oracle
    ├── module.oracle
    ├── text.oracle
    └── types.oracle
```

### Generated Code Included

**arc/go generated:**
```
arc/go/compiler/{types.gen.go, pb/translator.gen.go}
arc/go/graph/{types.gen.go, pb/translator.gen.go}
arc/go/ir/{types.gen.go, pb/translator.gen.go}
arc/go/module/{types.gen.go, pb/translator.gen.go}
arc/go/text/{types.gen.go, pb/translator.gen.go}
arc/go/types/{types.gen.go, pb/translator.gen.go}
```

**arc/cpp generated:**
```
arc/cpp/compiler/{types,json,proto}.gen.h
arc/cpp/graph/{types,json,proto}.gen.h
arc/cpp/ir/{types,json,proto}.gen.h
arc/cpp/module/{types,json,proto}.gen.h
arc/cpp/text/{types,json,proto}.gen.h
arc/cpp/types/{types,json,proto}.gen.h
```

**client/ts arc generated:**
```
client/ts/src/arc/types.gen.ts
client/ts/src/arc/compiler/types.gen.ts
client/ts/src/arc/graph/types.gen.ts
client/ts/src/arc/ir/types.gen.ts
client/ts/src/arc/module/types.gen.ts
client/ts/src/arc/text/types.gen.ts
client/ts/src/arc/types/types.gen.ts
```

**core/pkg/service/arc generated:**
```
core/pkg/service/arc/{types.gen.go, pb/translator.gen.go}
```

### Non-Generated Code Included

**arc/go (compiler, analyzer, lsp, runtime, symbol):**
```
arc/go/
├── analyzer/
├── compiler/
├── graph/
├── ir/
├── lsp/
├── module/
├── runtime/
├── symbol/
├── text/
└── types/
```

**arc/cpp (runtime, ir, module, etc.):**
```
arc/cpp/
├── compiler/
├── errors/
├── graph/
├── ir/
├── module/
├── runtime/
├── text/
└── types/
```

**core/pkg/service/arc (non-generated):**
```
core/pkg/service/arc/
├── runtime/
├── task.go
└── task_test.go
```

**x/go/diagnostics (moved from arc)**

**File count:** ~200 files

---

## PR 4: API Types

**Purpose:** Service schemas and ALL generated code for server + clients.

**Review Focus:**
- Schema correctness for API types (channel, device, rack, task, ranger, workspace, etc.)
- Generated code consistency across all languages
- Protobuf definitions

### Schemas Included

```
schemas/
├── access.oracle
├── alias.oracle
├── channel.oracle
├── cluster.oracle
├── device.oracle
├── framer.oracle
├── group.oracle
├── kv.oracle
├── lineplot.oracle
├── log.oracle
├── ontology.oracle
├── rack.oracle
├── ranger.oracle
├── role.oracle
├── schematic.oracle
├── symbol.oracle
├── table.oracle
├── task.oracle
├── user.oracle
├── view.oracle
└── workspace.oracle
```

### Generated Code Included

**core/ generated (server):**
```
core/pkg/api/channel/{types.gen.go, pb/translator.gen.go}
core/pkg/api/ranger/{types.gen.go, pb/translator.gen.go}
core/pkg/distribution/channel/{types.gen.go, pb/translator.gen.go}
core/pkg/distribution/group/{types.gen.go, pb/translator.gen.go}
core/pkg/service/access/types.gen.go
core/pkg/service/device/{types.gen.go, pb/translator.gen.go}
core/pkg/service/label/types.gen.go
core/pkg/service/rack/{types.gen.go, pb/translator.gen.go}
core/pkg/service/ranger/{types.gen.go, pb/translator.gen.go}
core/pkg/service/ranger/alias/{types.gen.go, pb/translator.gen.go}
core/pkg/service/ranger/kv/{types.gen.go, pb/translator.gen.go}
core/pkg/service/status/types.gen.go
core/pkg/service/task/{types.gen.go, pb/translator.gen.go}
core/pkg/service/user/{types.gen.go, pb/translator.gen.go}
core/pkg/service/workspace/{types.gen.go, pb/translator.gen.go}
core/pkg/service/workspace/lineplot/{types.gen.go, pb/translator.gen.go}
core/pkg/service/workspace/log/{types.gen.go, pb/translator.gen.go}
core/pkg/service/workspace/schematic/{types.gen.go, pb/translator.gen.go}
core/pkg/service/workspace/table/{types.gen.go, pb/translator.gen.go}
```

**client/ts generated:**
```
client/ts/src/access/{types.gen.ts, role/types.gen.ts}
client/ts/src/channel/types.gen.ts
client/ts/src/cluster/types.gen.ts
client/ts/src/device/types.gen.ts
client/ts/src/framer/types.gen.ts
client/ts/src/group/types.gen.ts
client/ts/src/ontology/{types.gen.ts, group/types.gen.ts}
client/ts/src/rack/types.gen.ts
client/ts/src/range/{types.gen.ts, alias/types.gen.ts, kv/types.gen.ts}
client/ts/src/task/types.gen.ts
client/ts/src/user/types.gen.ts
client/ts/src/view/types.gen.ts
client/ts/src/workspace/{types.gen.ts, lineplot/types.gen.ts, log/types.gen.ts}
client/ts/src/workspace/schematic/{types.gen.ts, symbol/types.gen.ts}
client/ts/src/workspace/table/types.gen.ts
```

**client/py generated:**
```
client/py/synnax/access/{types_gen.py, role/types_gen.py}
client/py/synnax/channel/types_gen.py
client/py/synnax/cluster/types_gen.py
client/py/synnax/color/types_gen.py
client/py/synnax/device/types_gen.py
client/py/synnax/group/types_gen.py
client/py/synnax/label/types_gen.py
client/py/synnax/rack/types_gen.py
client/py/synnax/ranger/types_gen.py
client/py/synnax/status/types_gen.py
client/py/synnax/task/types_gen.py
client/py/synnax/telem/types_gen.py
client/py/synnax/user/types_gen.py
client/py/synnax/x/control/types_gen.py
```

**client/cpp generated:**
```
client/cpp/arc/{types,json,proto}.gen.h
client/cpp/channel/{types,json,proto}.gen.h
client/cpp/cluster/{types,json}.gen.h
client/cpp/device/{types,json,proto}.gen.h
client/cpp/rack/{types,json,proto}.gen.h
client/cpp/ranger/{types,json,proto}.gen.h
client/cpp/task/{types,json,proto}.gen.h
```

**File count:** ~120 files (mostly generated)

---

## PR 5: Server Implementation

**Purpose:** Server API layer implementation (non-generated code only).

**Review Focus:**
- API layer separation (transport-agnostic services)
- Handler correctness
- Service organization
- Ranger/workspace component splits

### Files Included

**API layer:**
```
core/pkg/api/
├── access/access.go
├── arc/arc.go
├── auth/{auth.go, middleware.go}
├── channel/channel.go
├── config/config.go
├── connectivity/connectivity.go
├── device/device.go
├── framer/framer.go
├── group/group.go
├── grpc/
│   ├── arc/handler.go
│   ├── auth/handler.go
│   ├── channel/handler.go
│   ├── connectivity/handler.go
│   ├── device/handler.go
│   ├── framer/handler.go
│   ├── rack/handler.go
│   ├── ranger/{handler.go, alias/, kv/, status/}
│   ├── status/handler.go
│   └── task/handler.go
├── http/
├── ontology/
├── rack/rack.go
├── ranger/ranger.go
├── status/status.go
├── task/task.go
├── user/user.go
└── workspace/workspace.go
```

**Service layer:**
```
core/pkg/service/
├── ranger/
│   ├── alias/{alias.go, service.go, reader.go, writer.go}
│   └── kv/{kv.go, service.go, reader.go, writer.go}
└── workspace/
    ├── lineplot/{writer.go, retrieve.go, ontology.go}
    ├── log/{writer.go, retrieve.go, ontology.go}
    ├── schematic/{writer.go, retrieve.go, ontology.go}
    └── table/{writer.go, retrieve.go, ontology.go}
```

**Distribution layer:**
```
core/pkg/distribution/
├── channel/
├── framer/
└── group/
```

**Protobuf definitions (.proto files, .pb.go files)**

**File count:** ~180 files (excluding generated types)

---

## PR 6: TypeScript Client

**Purpose:** TypeScript client implementation (non-generated code only).

**Review Focus:**
- API call correctness
- Type usage (imports from types.gen.ts)
- Error handling
- Module organization (`ranger/` → `range/`)

### Files Included

```
client/ts/src/
├── access/client.ts
├── arc/client.ts
├── channel/{client.ts, retriever.ts, writer.ts}
├── cluster/client.ts
├── connection/
├── control/
├── device/client.ts
├── framer/client.ts
├── group/client.ts
├── hardware/
├── label/client.ts
├── ontology/
├── range/{client.ts, alias/client.ts, kv/client.ts}
├── rack/client.ts
├── status/client.ts
├── task/client.ts
├── user/client.ts
└── workspace/{client.ts, lineplot/, log/, schematic/, table/}
```

**Also includes:**
- Test files (`*.spec.ts`)
- Package configuration updates
- Payload files where they still exist

**File count:** ~80 files (excluding generated)

---

## PR 7: Python Client

**Purpose:** Python client implementation (non-generated code only).

**Review Focus:**
- Pydantic model usage
- API correctness
- Pythonic style
- Type hints

### Files Included

```
client/py/synnax/
├── access/client.py
├── channel/{client.py, retriever.py, writer.py}
├── device/client.py
├── framer/client.py
├── group/client.py
├── hardware/
├── label/client.py
├── ontology/
├── rack/client.py
├── ranger/{client.py, alias/client.py, kv/client.py}
├── status/client.py
├── task/client.py
├── user/client.py
└── workspace/
```

**Also includes:**
- Test files in `tests/`
- Package configuration (`pyproject.toml`, etc.)

**File count:** ~65 files (excluding generated)

---

## PR 8: C++ Client

**Purpose:** C++ client and freighter transport (non-generated code only).

**Review Focus:**
- Memory safety
- Namespace usage (`x::errors::`, etc.)
- RAII patterns
- Protobuf integration

### Files Included

**client/cpp:**
```
client/cpp/
├── arc/{arc.h, arc.cpp, arc_test.cpp}
├── channel/{channel.h, channel.cpp, channel_test.cpp}
├── device/{device.h, device.cpp}
├── framer/{framer.h, framer.cpp}
├── group/{group.h, group.cpp}
├── rack/{rack.h, rack.cpp}
├── ranger/{ranger.h, ranger.cpp}
├── status/status.h
├── task/{task.h, task.cpp}
└── BUILD.bazel files
```

**freighter/cpp:**
```
freighter/cpp/
├── errors/errors.h     (ERR_* constant renames)
├── grpc/               (renamed from fgrpc/)
│   ├── grpc.h
│   ├── grpc_test.cpp
│   └── mock/
├── freighter.h
└── BUILD.bazel files
```

**File count:** ~50 files (excluding generated)

---

## PR 9: Driver

**Purpose:** Hardware driver updates for new namespaces and Arc integration.

**Review Focus:**
- Import updates (x/cpp namespaces)
- Arc task integration
- Hardware task correctness
- Platform-specific code

### Files Included

```
driver/
├── arc/{arc.h, arc.cpp, arc_test.cpp}
├── cmd/{clear/, exec/, login/, service/, start/, version/, main.cpp}
├── daemon/{daemon.h, daemon_linux.cpp, daemon_nilinuxrt.cpp, daemon_noop.cpp}
├── errors/errors.h
├── labjack/{device/, ljm/, read_task.h, write_task.h, scan_task.h}
├── modbus/{device/, read_task.h, write_task.h, scan_task.h}
├── ni/{channel/, daqmx/, hardware/, syscfg/, analog_read_task.h, ...}
├── opc/{util/, read_task.h, write_task.h, scan_task.h}
├── pipeline/{acquisition.h, control.h}
├── rack/{config.h, factories.cpp, rack.h}
└── task/common/{read_task.h, write_task.h, scan_task.h, sample_clock.h, status.h}
```

**File count:** ~184 files

---

## Dependency Graph

```
┌─────────────┐
│  PR 1       │
│  Oracle     │
│  Tool       │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 2       │
│  x/*        │
│  Foundation │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 3       │
│  Arc        │
│  System     │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 4       │
│  API        │
│  Types      │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 5       │
│  Server     │
│  Impl       │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 6       │
│  TS Client  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 7       │
│  Py Client  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 8       │
│  C++ Client │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  PR 9       │
│  Driver     │
└─────────────┘
```

---

## File Listing Commands

```bash
# PR 1: Oracle files
git diff --name-only rc...HEAD -- oracle/

# PR 2: x/* files (schemas + generated + non-generated)
git diff --name-only rc...HEAD -- x/cpp/ x/go/ x/ts/
git diff --name-only rc...HEAD -- schemas/color.oracle schemas/control.oracle \
    schemas/label.oracle schemas/spatial.oracle schemas/status.oracle schemas/telem.oracle

# PR 3: Arc files (schemas + generated + non-generated)
git diff --name-only rc...HEAD -- arc/ schemas/arc.oracle schemas/arc/
git diff --name-only rc...HEAD -- core/pkg/service/arc/

# PR 4: API schemas + ALL generated
git diff --name-only rc...HEAD -- schemas/ | grep -v -E '(color|control|label|spatial|status|telem|arc)'
git diff --name-only rc...HEAD | grep -E '\.(gen\.ts|gen\.go|_gen\.py)$' | grep -v -E '(x/|arc/)'
git diff --name-only rc...HEAD -- 'client/cpp/**/*.gen.h' | grep -v arc

# PR 5: Server non-generated
git diff --name-only rc...HEAD -- core/pkg/api/ core/pkg/service/ core/pkg/distribution/ \
    | grep -v -E '\.(gen\.go|gen\.ts)$'

# PR 6: TS client non-generated
git diff --name-only rc...HEAD -- client/ts/ | grep -v '\.gen\.ts$'

# PR 7: Python client non-generated
git diff --name-only rc...HEAD -- client/py/ | grep -v '_gen\.py$'

# PR 8: C++ client non-generated
git diff --name-only rc...HEAD -- client/cpp/ freighter/cpp/ | grep -v '\.gen\.h$'

# PR 9: Driver files
git diff --name-only rc...HEAD -- driver/
```

---

## Verification Checklist

Before creating each PR, verify:

- [ ] All files for the PR are included
- [ ] Schema dependencies are satisfied (Tier 0 before Tier 1, etc.)
- [ ] Generated files match their schemas
- [ ] Non-generated code can import from generated types
- [ ] CI passes on the branch
- [ ] PR description references this plan

---

## Merge Strategy

**Integration branch:** `sy-3498-oracle-v2`

**Process:**

1. Create `sy-3498-oracle-v2` from `rc` (or current main)
2. For each PR (1-9):
   - Create feature branch: `pr/1-oracle-tool`, `pr/2-x-foundation`, etc.
   - Open PR targeting `sy-3498-oracle-v2`
   - Review and approve
   - **Squash merge** into `sy-3498-oracle-v2`
3. After all 9 PRs are squash-merged:
   - Open PR from `sy-3498-oracle-v2` → `main`/`rc`
   - **Merge (no squash)** to preserve the 9 commits

**Result:** Main branch receives 9 clean, atomic commits with descriptive messages.

```
main/rc
    │
    └──► sy-3498-oracle-v2
              ├── [squash] PR 1: Oracle Tool
              ├── [squash] PR 2: x/* Foundation
              ├── [squash] PR 3: Arc System
              ├── [squash] PR 4: API Types
              ├── [squash] PR 5: Server Implementation
              ├── [squash] PR 6: TypeScript Client
              ├── [squash] PR 7: Python Client
              ├── [squash] PR 8: C++ Client
              └── [squash] PR 9: Driver
              │
              ▼
         [merge no-squash] → main/rc
         (9 commits preserved)
```

---

## Notes

- All PRs must pass CI before merge
- PRs must merge in strict sequence (1 → 2 → 3 → ... → 9)
- PR 4 contains generated code for ALL languages; subsequent PRs only add implementation
- After PR 9 merges, the breaking release is complete
