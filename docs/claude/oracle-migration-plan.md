# Oracle Migration Plan: Centralized to Layer-Specific Proto Organization

This document provides a migration plan for reorganizing Oracle-generated code from the
current centralized proto structure to the layer-specific `pb/` subdirectory pattern
described in `docs/claude/oracle-organization.md`.

## Executive Summary

**Current State**: All protos centralized in `api/grpc/v1/`, translators in `api/grpc/`
**Target State**: Layer-specific `pb/` subdirectories with protos and translators per domain

**Estimated Scope**: ~15-20 domains across x/go, service, distribution, and api layers

---

## Current vs Target Architecture

### Current Structure

```
core/pkg/
├── api/
│   └── grpc/
│       ├── v1/
│       │   ├── rack.proto          # ALL protos here
│       │   ├── rack.pb.go
│       │   ├── channel.proto
│       │   ├── channel.pb.go
│       │   └── ... (34 files)
│       ├── rack.gen.go             # Translators here
│       ├── rack.go                 # gRPC handlers
│       └── ...
├── service/
│   └── rack/
│       ├── rack.go
│       └── types.gen.go            # No pb/ subdirectory
└── distribution/
    └── channel/
        └── channel.go              # No pb/ subdirectory

x/go/
└── status/
    ├── status.go
    ├── types.gen.go
    ├── status.proto                # Proto in root, not pb/
    └── grpc/
        └── status.gen.go           # Translator in grpc/, not pb/
```

### Target Structure

```
core/pkg/
├── api/
│   ├── rack/                       # Per-domain API package
│   │   ├── rack.go                 # Transport-agnostic handlers
│   │   └── pb/                     # Only if API type differs
│   │       ├── rack.proto
│   │       ├── rack.pb.go
│   │       └── translator.gen.go
│   ├── grpc/
│   │   └── rack/
│   │       └── handler.go          # gRPC-specific binding
│   └── http/
│       └── rack/
│           └── handler.go          # HTTP-specific binding
├── service/
│   └── rack/
│       ├── rack.go
│       ├── types.gen.go
│       └── pb/
│           ├── rack.proto
│           ├── rack.pb.go
│           └── translator.gen.go
└── distribution/
    └── channel/
        ├── channel.go
        └── pb/
            ├── channel.proto
            ├── channel.pb.go
            └── translator.gen.go

x/go/
└── status/
    ├── status.go
    ├── types.gen.go
    └── pb/
        ├── status.proto
        ├── status.pb.go
        └── translator.gen.go
```

---

## Migration Phases

### Dependency Order

```
Phase 1: Oracle Tool Updates
    ↓
Phase 2: x/go Layer (no dependencies)
    ↓
Phase 3: Service Layer (depends on x/go)
    ↓
Phase 4: Distribution Layer (depends on service)
    ↓
Phase 5: API Layer Refactoring (depends on all above)
    ↓
Phase 6: Cleanup & Deprecation
```

---

## Phase 1: Oracle Tool Updates

**Status**: Can be done first, no dependencies
**Parallelizable**: No (foundational)

### Objective

Update the Oracle tool to support generating:
1. Proto files to `{domain}/pb/{domain}.proto`
2. Translator files to `{domain}/pb/translator.gen.go`
3. Buf configuration for new proto locations

### Current Oracle Plugin Structure

```
oracle/
├── cmd/oracle/main.go
├── plugin/
│   ├── go/
│   │   ├── types/types.go          # Generates types.gen.go
│   │   └── api/api.go              # Generates API translators
│   ├── pb/
│   │   └── types/types.go          # Generates .proto files
│   ├── ts/types/types.go
│   ├── py/types/types.go
│   └── cpp/types/types.go
└── schema/                         # Schema parsing
```

### Required Changes

#### 1.1 New `go/pb` Plugin

Create a new plugin that generates `translator.gen.go` files to `pb/` subdirectories.

**File**: `oracle/plugin/go/pb/pb.go`

**Behavior**:
- For each struct with `@go output "path/to/domain"` and `@pb output "..."`
- Generate `path/to/domain/pb/translator.gen.go`
- Import the parent package for domain types
- Import the pb package for proto types
- Generate `ToPB()` and `FromPB()` functions

**Template Structure**:
```go
package pb

import (
    "path/to/domain"  // Parent package
)

func ToPB(d domain.Type) *PBType {
    return &PBType{
        Field: d.Field,
    }
}

func FromPB(pb *PBType) domain.Type {
    return domain.Type{
        Field: pb.Field,
    }
}
```

#### 1.2 Update `pb/types` Plugin

Modify to generate protos to `{go_output}/pb/{domain}.proto` instead of separate `@pb output`.

**Current behavior**: Uses `@pb output` directive for proto location
**New behavior**: Derive from `@go output` + `/pb/` suffix

#### 1.3 Schema Directive Updates

Add support for controlling translator generation:

```oracle
Rack struct {
    key   uint32
    name  string

    @go output "core/pkg/service/rack"
    @go pb                              # Enable pb/ generation
    @ts output "client/ts/src/rack"
}
```

#### 1.4 Buf Configuration Generation

Generate `buf.yaml` and `buf.gen.yaml` for each `pb/` directory, or update root config.

### Acceptance Criteria

- [ ] `oracle sync` generates `pb/translator.gen.go` when `@go pb` directive present
- [ ] Proto files generated to `{go_output}/pb/{domain}.proto`
- [ ] Generated translators compile without errors
- [ ] Existing generation still works (backward compatible)

---

## Phase 2: x/go Layer Migration

**Status**: Can start after Phase 1
**Parallelizable**: Each domain can be done independently

### Domains to Migrate

| Domain | Current Location | Has Proto | Priority |
|--------|-----------------|-----------|----------|
| `status` | `x/go/status/` | Yes (`status.proto` in root) | High |
| `telem` | `x/go/telem/` | Yes (in api/grpc/v1/) | High |
| `spatial` | `x/go/spatial/` | No | Low |

### 2.1 Migrate `x/go/status`

**Current**:
```
x/go/status/
├── status.go
├── types.gen.go
├── status.proto           # In root
├── status.pb.go           # In root
└── grpc/
    └── status.gen.go      # Translator here
```

**Target**:
```
x/go/status/
├── status.go
├── types.gen.go
└── pb/
    ├── status.proto
    ├── status.pb.go
    └── translator.gen.go
```

**Steps**:
1. Create `x/go/status/pb/` directory
2. Move `status.proto` to `pb/status.proto`
3. Update proto `go_package` option
4. Regenerate `status.pb.go` with buf
5. Update schema `status.oracle` with `@go pb` directive
6. Run `oracle sync` to generate `translator.gen.go`
7. Update all imports from `x/go/status/grpc` to `x/go/status/pb`
8. Delete old `x/go/status/grpc/` directory
9. Update Bazel BUILD files

**Import Changes**:
```go
// Before
import statuspb "github.com/synnaxlabs/x/status/grpc"

// After
import statuspb "github.com/synnaxlabs/x/status/pb"
```

### 2.2 Migrate `x/go/telem`

Similar process. Note: telem has more complex types (Series, Frame, TimeRange).

**Current proto location**: `core/pkg/api/grpc/v1/telem.proto`
**Target**: `x/go/telem/pb/telem.proto`

---

## Phase 3: Service Layer Migration

**Status**: Can start after Phase 2 (depends on x/go types)
**Parallelizable**: Each domain can be done independently

### Domains to Migrate

| Domain | Schema File | Has Types | Notes |
|--------|------------|-----------|-------|
| `rack` | `schemas/rack.oracle` | Yes | Simple, good first candidate |
| `task` | `schemas/task.oracle` | Yes | Depends on rack |
| `device` | `schemas/device.oracle` | Yes | Depends on rack |
| `user` | `schemas/user.oracle` | Yes | Independent |
| `group` | `schemas/group.oracle` | Yes | Depends on user |
| `access` | `schemas/access.oracle` | Yes | Independent |
| `workspace` | `schemas/workspace.oracle` | Yes | Independent |
| `label` | `schemas/label.oracle` | Yes | Independent |
| `lineplot` | `schemas/lineplot.oracle` | Yes | Depends on workspace |
| `schematic` | `schemas/schematic.oracle` | Yes | Depends on workspace |
| `table` | `schemas/table.oracle` | Yes | Depends on workspace |
| `log` | `schemas/log.oracle` | Yes | Depends on workspace |

### 3.1 Template: Service Domain Migration

For each service domain (e.g., `rack`):

**Current**:
```
core/pkg/service/rack/
├── rack.go
├── service.go
├── writer.go
├── retrieve.go
└── types.gen.go

core/pkg/api/grpc/v1/
├── rack.proto              # Proto here
└── rack.pb.go

core/pkg/api/grpc/
└── rack.gen.go             # Translator here
```

**Target**:
```
core/pkg/service/rack/
├── rack.go
├── service.go
├── writer.go
├── retrieve.go
├── types.gen.go
└── pb/
    ├── rack.proto
    ├── rack.pb.go
    └── translator.gen.go
```

**Steps**:
1. Create `core/pkg/service/rack/pb/` directory
2. Copy `rack.proto` from `api/grpc/v1/` to `service/rack/pb/`
3. Update proto package: `package synnax.rack;`
4. Update `go_package`: `github.com/synnaxlabs/synnax/pkg/service/rack/pb`
5. Run buf to generate `rack.pb.go`
6. Update `schemas/rack.oracle` with `@go pb` directive
7. Run `oracle sync` to generate `translator.gen.go`
8. Update imports in API layer to use new location
9. Create Bazel BUILD file for `pb/` directory

**Proto Package Naming**:
```protobuf
// Before
syntax = "proto3";
package api.v1;
option go_package = "github.com/synnaxlabs/synnax/pkg/api/grpc/v1";

// After
syntax = "proto3";
package synnax.rack;
option go_package = "github.com/synnaxlabs/synnax/pkg/service/rack/pb";
```

### 3.2 Recommended Migration Order

Based on dependencies:

1. **Independent domains first** (can parallelize):
   - `user`
   - `access`
   - `workspace`
   - `label`

2. **First-level dependencies**:
   - `rack` (after independent)
   - `group` (after user)

3. **Second-level dependencies**:
   - `task` (after rack)
   - `device` (after rack)
   - `lineplot`, `schematic`, `table`, `log` (after workspace)

---

## Phase 4: Distribution Layer Migration

**Status**: Can start after Phase 3
**Parallelizable**: Limited (channel and framer are interconnected)

### Domains to Migrate

| Domain | Current Proto Location | Notes |
|--------|----------------------|-------|
| `channel` | `api/grpc/v1/` + `distribution/transport/grpc/channel/v1/` | Complex, has two protos |
| `framer` | `api/grpc/v1/` + `distribution/transport/grpc/framer/v1/` | Complex, streaming |

### 4.1 Special Considerations

Distribution layer types may need **different protos** than service layer:
- Internal fields for cluster communication
- Routing information
- Node identifiers

**Decision Point**: Should distribution have its own protos, or share with service?

**Recommended**: Distribution layer should have its own `pb/` with protos that include
internal fields not exposed at API layer.

### 4.2 Migrate `distribution/channel`

**Current**:
```
core/pkg/distribution/channel/
├── channel.go
├── service.go
└── transport.go

core/pkg/distribution/transport/grpc/channel/v1/
├── channel.proto          # Distribution-specific proto
└── channel.pb.go
```

**Target**:
```
core/pkg/distribution/channel/
├── channel.go
├── service.go
├── transport.go
└── pb/
    ├── channel.proto      # Distribution-specific serialization
    ├── channel.pb.go
    └── translator.gen.go
```

---

## Phase 5: API Layer Refactoring

**Status**: Can start after Phases 3-4
**Parallelizable**: Per-domain after structure is established

### Objective

Transform API layer from transport-bound to domain-bound organization.

### 5.1 Create Per-Domain API Packages

**Current**:
```
core/pkg/api/
├── grpc/
│   ├── v1/                 # All protos
│   ├── rack.go             # gRPC handler
│   ├── rack.gen.go         # Translator
│   └── channel.go
└── http/
    └── http.go
```

**Target**:
```
core/pkg/api/
├── rack/                   # Transport-agnostic
│   ├── rack.go             # Handler logic, access control
│   └── pb/                 # Only if API type differs from service
├── channel/
│   ├── channel.go
│   └── pb/                 # Channel API type differs
├── grpc/
│   ├── rack/
│   │   └── handler.go      # gRPC binding only
│   └── channel/
│       └── handler.go
└── http/
    └── ...
```

### 5.2 Decision: When Does API Need Its Own pb/?

| Domain | API Type = Service Type? | Needs Own pb/? |
|--------|-------------------------|----------------|
| `rack` | Yes (alias) | No - reuse service/rack/pb |
| `device` | Yes | No |
| `task` | Yes | No |
| `channel` | No (computed fields) | Yes |
| `framer` | No (streaming types) | Yes |
| `range` | No (omitted fields) | Yes |

### 5.3 Translator Strategy

**When API type = Service type**:
- API layer imports `service/{domain}/pb`
- No additional translator needed

**When API type differs**:
- API layer has own `api/{domain}/pb/`
- Translators convert between API type ↔ API proto
- Separate translators convert API type ↔ Service type

---

## Phase 6: Cleanup & Deprecation

**Status**: After all migrations complete
**Parallelizable**: Yes

### 6.1 Remove Old Proto Locations

Once all consumers migrated:
1. Delete `core/pkg/api/grpc/v1/` directory
2. Delete old translator files in `core/pkg/api/grpc/`
3. Update Bazel to remove old proto targets

### 6.2 Update Documentation

- Update `docs/claude/oracle-organization.md` to mark as implemented
- Add migration notes for external consumers
- Update architecture diagrams

### 6.3 CI/CD Updates

- Update proto linting to check new locations
- Update buf configuration
- Verify all Bazel targets build

---

## Detailed Task Specifications

### Task Template

Each migration task should include:

```markdown
## Task: Migrate {domain} to pb/ Structure

### Context
- Domain: {domain}
- Layer: {x/go | service | distribution | api}
- Schema: schemas/{domain}.oracle
- Current proto: {location}

### Prerequisites
- [ ] Phase 1 complete (Oracle tool updates)
- [ ] Dependencies migrated: {list}

### Steps
1. Create pb/ directory
2. Move/create proto file
3. Update proto package and go_package
4. Generate pb.go with buf
5. Update schema with @go pb directive
6. Run oracle sync
7. Update all imports
8. Update Bazel BUILD
9. Run tests
10. Delete old files

### Validation
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `bazel build //...` passes (if applicable)
- [ ] No imports to old locations remain

### Files Modified
- schemas/{domain}.oracle
- {layer}/{domain}/pb/BUILD.bazel (new)
- {layer}/{domain}/pb/{domain}.proto (new/moved)
- {layer}/{domain}/pb/translator.gen.go (generated)
- {consumers that import the proto}
```

---

## Risk Mitigation

### Breaking Changes

**Risk**: Import path changes break external consumers
**Mitigation**:
- Keep old paths working during migration (re-export from old location)
- Deprecation warnings in old locations
- Major version bump when removing old paths

### Proto Compatibility

**Risk**: Proto package changes break wire compatibility
**Mitigation**:
- Proto field numbers unchanged
- Package name change doesn't affect wire format
- Test with existing stored data

### Build System

**Risk**: Bazel/Buf configuration breaks
**Mitigation**:
- Update BUILD files incrementally
- Test each domain before proceeding
- Keep CI green throughout

---

## Success Metrics

### Phase Completion Criteria

- [ ] All `pb/` directories created per target structure
- [ ] No protos remain in `api/grpc/v1/` (except during transition)
- [ ] All translators in `pb/` subdirectories
- [ ] API layer has per-domain packages
- [ ] All tests pass
- [ ] CI/CD green
- [ ] Documentation updated

### Code Health Indicators

- Import paths follow `{domain}/pb` pattern
- No circular dependencies between layers
- Proto packages follow `synnax.{domain}` naming
- Translator files consistently named `translator.gen.go`

---

## Appendix A: Current Schema Files

```
schemas/
├── access.oracle
├── arc.oracle
├── channel.oracle
├── device.oracle
├── framer.oracle
├── group.oracle
├── label.oracle
├── lineplot.oracle
├── log.oracle
├── ontology.oracle
├── rack.oracle
├── range.oracle
├── schematic.oracle
├── status.oracle
├── table.oracle
├── task.oracle
├── telem.oracle
├── user.oracle
└── workspace.oracle
```

## Appendix B: Import Update Script Template

```bash
#!/bin/bash
# Update imports from old to new location for a domain

OLD_IMPORT="github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
NEW_IMPORT="github.com/synnaxlabs/synnax/pkg/service/rack/pb"

find . -name "*.go" -exec sed -i '' \
  "s|${OLD_IMPORT}|${NEW_IMPORT}|g" {} \;
```

## Appendix C: Bazel BUILD Template

```python
# {layer}/{domain}/pb/BUILD.bazel

load("@rules_proto//proto:defs.bzl", "proto_library")
load("@rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "{domain}_proto",
    srcs = ["{domain}.proto"],
    visibility = ["//visibility:public"],
    deps = [
        "//x/go/status/pb:status_proto",  # Common deps
    ],
)

go_proto_library(
    name = "{domain}_go_proto",
    importpath = "github.com/synnaxlabs/synnax/pkg/{layer}/{domain}/pb",
    proto = ":{domain}_proto",
    visibility = ["//visibility:public"],
)
```
