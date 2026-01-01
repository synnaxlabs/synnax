# Oracle Code Generation Organization Standards

This document defines the long-term organizational standards for Oracle-generated code
across the Synnax codebase.

## Fundamental Principles

These principles are foundational and guide all organizational decisions.

### 1. Layers Over Domains

The codebase is organized primarily by **architectural layer**, not by business domain.
This decision is set in stone.

```
core/pkg/
├── storage/          # Layer 1: Persistence
├── distribution/     # Layer 2: Clustering
├── service/          # Layer 3: Business logic
└── api/              # Layer 4: Interface
```

Domains (rack, channel, framer) exist as **subdirectories within layers**, not the
reverse. A feature like "channel" is distributed across layers because each layer has
distinct architectural responsibilities.

### 2. Layer > Domain > Type

For implementation artifacts, the priority ordering is:

1. **Layer** - Which architectural layer does this belong to?
2. **Domain** - Which business concept does this serve?
3. **Type** - What kind of artifact is this (types, translators, handlers)?

This produces structures like `service/rack/pb/translator.go`:
- Layer: `service/`
- Domain: `rack/`
- Type: `pb/translator.go`

**Rationale**: Within a layer, developers work on features. Everything for "rack within
the service layer" changes together. The artifact type is an implementation detail.

### 3. Source-of-Truth Artifacts Are Centralized

Artifacts that **span multiple layers** are centralized rather than distributed:

| Artifact | Why Centralized |
|----------|-----------------|
| `schemas/*.oracle` | Generates into multiple layers and languages |
| Database migrations | Affect storage layer but reviewed holistically |

A single `rack.oracle` schema generates code into `service/rack/`, `api/rack/`, and
`client/ts/src/rack/`. It cannot belong to any single layer without being misleading.

### 4. Protos Are Layer-Specific

Unlike schemas, **protos belong to specific layers** and follow Layer > Domain > Type:

- `service/rack/pb/rack.proto` — Service layer serialization
- `api/channel/pb/channel.proto` — API layer serialization (when type differs)
- `distribution/channel/pb/channel.proto` — Distribution layer serialization

Protos are **not centralized** because each layer may have different serialization needs.
The API layer exposes a curated subset; the distribution layer includes internal fields.

### 5. Pragmatic Serialization

Domain types may carry serialization struct tags (`json:"key" msgpack:"key"`). This is a
pragmatic Go idiom, not a purity violation. The real serialization boundary is enforced
by:

- The `pb/` subpackage isolating proto concerns
- Freighter's `Translator` interface at transport boundaries
- One-way dependency: `pb/` imports parent, never the reverse

## Operational Principles

These principles guide day-to-day decisions.

### Flat Until It Hurts

Avoid deep nesting. One level of semantic nesting (like `pb/`) is acceptable when it
provides clear separation of concerns.

### Semantic Over Mechanical

Directory structure reflects domain meaning, not tooling artifacts.

### Protos Are Core Infrastructure

Protobufs are used for both transport (gRPC) AND storage serialization. They are not
just a transport concern.

### Language-Appropriate Idioms

Each language follows its own conventions. Go uses `pb/` subpackages; C++ uses inline
methods in headers.

## Directory Structure

### Shared Types (x/)

General-purpose infrastructure that could exist outside Synnax (status, telem, spatial).

```
x/go/status/
├── status.go           # Domain type + behavior
├── types.gen.go        # Oracle-generated type definitions
└── pb/
    ├── status.proto    # Serialization format (transport + storage)
    ├── status.pb.go    # Buf-generated proto types
    └── translator.gen.go  # Oracle-generated ToPB/FromPB functions
```

**Rationale**: The `pb/` subpackage isolates serialization concerns. Domain code in
`status.go` never imports `pb/`. The translator handles conversion between domain types
and proto types.

### Service Layer

Synnax-specific domain types and business logic (rack, task, device, user).

```
core/pkg/service/rack/
├── rack.go             # Domain type definition + behavior
├── service.go          # Service implementation
├── writer.go           # Write operations
├── retrieve.go         # Query operations
├── ontology.go         # Ontology integration
├── types.gen.go        # Oracle-generated types
└── pb/
    ├── rack.proto      # Serialization format
    ├── rack.pb.go      # Buf-generated proto types
    └── translator.gen.go  # Oracle-generated translators
```

**Rationale**: Each service domain gets its own package with the standard files. The
`pb/` subpackage follows the same pattern as x/ types.

### Distribution Layer

Topology-aware types that require cluster awareness (channel, framer).

```
core/pkg/distribution/channel/
├── channel.go          # Distributed channel type
├── service.go          # Cluster-aware service
└── pb/
    ├── channel.proto
    ├── channel.pb.go
    └── translator.gen.go
```

**Rationale**: Same pattern as service layer. Distribution types also need serialization
for cluster communication and storage.

### API Layer

Transport-agnostic handles that curate and regulate service layer functionality.

```
core/pkg/api/
├── rack/                   # Transport-agnostic rack API
│   ├── rack.go             # Handlers, access control, query patterns
│   ├── types.gen.go        # Oracle-generated (only if type differs from service)
│   └── pb/                 # Only if API type differs from service type
│       ├── rack.proto
│       ├── rack.pb.go
│       └── translator.gen.go
│
├── channel/                # Channel API (has different type than service)
│   ├── channel.go
│   ├── types.gen.go
│   └── pb/
│       ├── channel.proto
│       └── ...
│
├── grpc/                   # gRPC transport binding
│   ├── grpc.go             # Server setup
│   ├── rack/
│   │   └── handler.go      # gRPC handlers for rack
│   └── channel/
│       └── handler.go
│
└── http/                   # HTTP transport binding
    └── http.go
```

**Rationale**:

- **Package-per-domain** (`api/rack/`, `api/channel/`) because each domain has
  substantial business logic (handlers, access control) even when types are aliases.
- **API types are case-by-case**: If API type equals service type, use alias and reuse
  service proto. If different, define own type and proto.
- **`api/grpc/rack/`** separates transport binding from transport-agnostic API logic.
- **Protos in `api/rack/pb/`** (not `api/grpc/rack/`) because protos are serialization
  format, not just gRPC-specific.

### Client Libraries

Client libraries have their own organization and don't use the `pb/` subpackage pattern.

```
client/ts/src/rack/
├── client.ts
└── types.gen.ts

client/py/synnax/rack/
├── client.py
└── types_gen.py

client/cpp/rack/
├── client.h
└── types.gen.h
```

**Rationale**: Clients handle serialization differently. The `pb/` pattern is
server-side only.

### Schema Files

Centralized as source-of-truth artifacts that span layers.

```
schemas/
├── rack.oracle
├── task.oracle
├── device.oracle
├── status.oracle
├── channel.oracle
└── ...
```

**Rationale**: Schemas are **meta-artifacts** that generate into multiple layers and
languages. A single `rack.oracle` produces:

- `service/rack/types.gen.go` (service layer)
- `service/rack/pb/translator.gen.go` (service layer proto)
- `api/rack/types.gen.go` (api layer, if type differs)
- `client/ts/src/rack/types.gen.ts` (TypeScript client)
- `client/py/synnax/rack/types_gen.py` (Python client)

This violates Layer > Domain > Type intentionally—schemas cannot belong to any single
layer. Centralizing them enables:

- **Auditing**: All type definitions in one place for API review
- **Breaking change detection**: Easy to track field additions/removals
- **Cross-layer consistency**: Single source generates all representations

## The pb/ Subpackage

### Contents

```
{domain}/pb/
├── {domain}.proto      # Proto definition (named after domain, not package)
├── {domain}.pb.go      # Buf-generated proto types (committed to git)
└── translator.gen.go   # Oracle-generated ToPB/FromPB functions
```

### Package Naming

- Directory: `service/rack/pb/`
- Package name: `package pb`
- Import with alias when needed: `import rackpb "service/rack/pb"`

**Rationale**: Simple package name, use Go's standard import aliasing for clarity.

### Dependency Direction

The `pb/` package imports its parent to access domain types:

```go
// service/rack/pb/translator.go
package pb

import "service/rack"  // Child imports parent

func ToPB(r rack.Rack) *PBRack { ... }
```

The parent package NEVER imports `pb/`. This is a strict one-way dependency.

**Rationale**: Keeps domain code clean of serialization concerns. If you need to
serialize, you explicitly import `pb/`.

### When API Layer Has Its Own pb/

API layer only has its own `pb/` when the API type differs from the service type:

| Scenario | API pb/ needed? | Example |
|----------|-----------------|---------|
| API type = service type (alias) | No - reuse service pb/ | Device, Task, Rack |
| API type differs (computed fields, omitted fields) | Yes - own pb/ | Channel, Range |

## Proto Organization

### Package Naming

Use product-scoped packages without version suffixes:

```protobuf
syntax = "proto3";
package synnax.rack;

option go_package = "github.com/synnaxlabs/synnax/pkg/service/rack/pb";
```

**Rationale**: Simple, no version management overhead. Synnax doesn't need proto
versioning.

### Message Naming

Use `PB` prefix to distinguish proto types from domain types:

```protobuf
message PBRack {
    uint32 key = 1;
    string name = 2;
}
```

**Rationale**: When proto types and domain types are in different packages (`pb/` vs
parent), the prefix provides clarity. Import as `rackpb.PBRack` vs `rack.Rack`.

### Bazel Integration

Each `pb/` directory needs a BUILD.bazel with:

```python
load("@rules_proto//proto:defs.bzl", "proto_library")
load("@rules_proto_grpc_cpp//:defs.bzl", "cpp_grpc_library")

proto_library(
    name = "rack_proto",
    srcs = ["rack.proto"],
    visibility = ["//visibility:public"],
    deps = ["//x/go/status/pb:status_proto"],
)

cpp_grpc_library(
    name = "rack_grpc",
    output_mode = "NO_PREFIX",
    protos = [":rack_proto"],
    visibility = ["//visibility:public"],
)
```

**Rationale**: Hybrid build system - Buf generates Go code (committed), Bazel generates
C++ code (at build time).

## Generated File Naming

| File | Purpose |
|------|---------|
| `types.gen.go` | Oracle-generated struct definitions |
| `translator.gen.go` | Oracle-generated ToPB/FromPB functions |
| `{domain}.proto` | Proto definition (named after domain) |
| `{domain}.pb.go` | Buf-generated proto types |

**Rationale**: Purpose-oriented naming with `.gen.` suffix for Oracle output. Proto
files named after domain for clarity.

## C++ Considerations

C++ follows different idioms:

### Translator Pattern

C++ uses inline constructors and methods, not separate translator files:

```cpp
// x/cpp/status/status.h
struct Status {
    std::string key;
    std::string message;

    // Constructor from proto
    explicit Status(const PBStatus &pb)
        : key(pb.key()), message(pb.message()) {}

    // Conversion to proto
    void to_proto(PBStatus *pb) const {
        pb->set_key(key);
        pb->set_message(message);
    }
};
```

**Rationale**: This is idiomatic C++. Each language follows its own conventions.

### Include Paths

C++ includes reference Go directory paths (where Bazel generates headers):

```cpp
#include "x/go/status/pb/status.pb.h"
#include "service/rack/pb/rack.pb.h"
```

## Oracle Schema Conventions

### Per-Struct Output Directives

Each struct specifies where it belongs:

```oracle
Rack struct {
    key         uint32
    name        string
    task_counter uint32

    @go output "core/pkg/service/rack"
    @ts output "client/ts/src/rack"
    @py output "client/py/synnax/rack"
}
```

**Rationale**: Fine-grained control for multi-layer schemas where different structs
target different packages.

### Multi-Layer Schemas

A single schema can define types for multiple layers:

```oracle
// Service layer canonical type
Rack struct {
    key         uint32
    name        string
    task_counter uint32

    @go output "core/pkg/service/rack"
}

// API layer type (if it differs)
RackPayload struct extends Rack {
    -task_counter    // Omit internal field

    @go output "core/pkg/api/rack"
}

// Or define completely separate if delta is large
ChannelPayload struct {
    key         uint32
    name        string
    density     float64    // Computed field not in service type

    @go output "core/pkg/api/channel"
}
```

## Import Conventions

Use idiomatic Go with import aliases:

```go
import (
    // Domain types
    "github.com/synnaxlabs/synnax/pkg/service/rack"
    "github.com/synnaxlabs/x/status"

    // Proto types (aliased for clarity)
    rackpb "github.com/synnaxlabs/synnax/pkg/service/rack/pb"
    statuspb "github.com/synnaxlabs/x/status/pb"

    // API types (aliased to avoid collision)
    apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
)
```

## What Belongs Where

### x/ (General Purpose)

Types that could exist as standalone libraries:

- `status` - Status messages with variants
- `telem` - Time, timestamps, series, frames
- `spatial` - XY, bounds, dimensions
- `label` - Labeling/tagging infrastructure

### service/ (Synnax Domain)

Synnax-specific business entities:

- `rack` - Hardware rack management
- `task` - Task execution
- `device` - Device configuration
- `user` - User management

### distribution/ (Topology-Aware)

Types requiring cluster awareness:

- `channel` - Distributed channel management
- `framer` - Distributed frame reading/writing

### api/ (Curated Exposure)

Transport-agnostic API layer:

- Handlers with access control
- Query pattern enforcement
- Request/response types
- Type aliases or transformations from service layer

## Summary

| Location | Contains | pb/ subpackage? |
|----------|----------|-----------------|
| `x/go/{type}/` | Shared types | Yes |
| `service/{domain}/` | Domain types + logic | Yes |
| `distribution/{domain}/` | Cluster-aware types | Yes |
| `api/{domain}/` | Handlers + API types | Only if type differs |
| `api/grpc/{domain}/` | gRPC handlers | No (uses api/{domain}/pb/) |
| `api/http/` | HTTP handlers | No |
| `client/{lang}/` | Client libraries | No |
| `schemas/` | Oracle schemas | N/A |

## Decision Record

Key architectural decisions and their rationale.

### Layers Over Domains (Decided)

**Decision**: Organize by architectural layer first, domain second.

**Alternatives Considered**:
- Domain-first (vertical slices): `rack/service/`, `rack/api/`, `rack/storage/`
- Hybrid: Some domains vertical, some horizontal

**Rationale**: Layer-first enforces architectural boundaries and dependency direction.
The 4-layer architecture (storage → distribution → service → api) is fundamental to
Synnax's distributed design. Domain-first would blur these boundaries.

### Layer > Domain > Type (Decided)

**Decision**: Within layers, organize by domain, then by artifact type.

**Alternatives Considered**:
- Layer > Type > Domain: `service/pb/rack/`, `service/types/rack/`

**Rationale**: Developers work on features. When modifying "rack in service layer," all
related artifacts (types, translators, handlers) should be co-located. Type-first would
scatter a single feature's code across multiple directories.

### Centralized Schemas (Decided)

**Decision**: Oracle schemas live in `schemas/`, not distributed with layers.

**Alternatives Considered**:
- Distributed: `service/rack/rack.oracle`
- Per-layer: `service/schemas/rack.oracle`, `api/schemas/rack.oracle`

**Rationale**: Schemas generate into multiple layers and languages. They are
meta-artifacts that define the source of truth. Centralizing enables holistic review
of API changes and breaking change detection.

### Distributed Protos (Decided)

**Decision**: Proto files live with their layer (`service/rack/pb/rack.proto`), not
centralized.

**Alternatives Considered**:
- Centralized: `proto/service/rack.proto`, `proto/api/rack.proto`

**Rationale**: Protos are layer-specific. The API layer's proto is a curated public
interface; the distribution layer's proto includes internal fields. Each layer owns
its serialization format. Centralizing would fight the layer structure.
