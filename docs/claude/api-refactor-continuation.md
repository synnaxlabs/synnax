# API Layer Refactoring - Continuation Instructions

## Context

This is Phase 5 of the Oracle Migration Plan. We're transforming the API layer from
transport-bound to domain-bound organization.

**Master Plan**: `docs/claude/oracle-migration-plan.md`

## What Was Completed

### Session Summary

1. **Removed Provider Pattern**
   - Deleted `core/pkg/api/provider.go`
   - Provider structs were unnecessary indirection

2. **Created Config Package**
   - New file: `core/pkg/api/config/config.go`
   - Contains `Config` struct with `Service` and `Distribution` layers

3. **Updated All Domain Service Constructors**
   - All 22 services now take `config.Config` directly
   - Pattern: `func NewService(cfg config.Config) *Service`
   - Services extract their dependencies: `cfg.Distribution.DB`, `cfg.Service.RBAC`, etc.

4. **Moved Token Middleware**
   - Created: `core/pkg/api/auth/middleware.go`
   - Contains `TokenMiddleware()` and `GetSubject()` functions
   - Deleted old `token_middleware.go`

5. **Renamed api.go → layer.go**
   - Removed duplicate Config (now uses `config.Config`)
   - Updated `New()` to call domain constructors with `config.Config`
   - Updated `BindTo()` to use `auth.TokenMiddleware()`

6. **Fixed Import Aliases**
   - `layer.go`: Uses clean names like `rack`, `device` (not `apirack`, `apidevice`)
   - `http/http.go`: Updated to import domain packages directly
   - `grpc/grpc.go`: Updated to import domain packages directly

7. **Deleted Old Files**
   - `core/pkg/api/types.gen.go`
   - `core/pkg/api/provider.go`
   - `core/pkg/api/token_middleware.go`
   - `core/pkg/api/api.go` (replaced by `layer.go`)

## Current State

### Directory Structure

```
core/pkg/api/
├── layer.go                    # Main API layer (renamed from api.go)
├── config/
│   └── config.go               # Config struct
├── auth/
│   ├── auth.go                 # Auth service
│   └── middleware.go           # TokenMiddleware, GetSubject
├── {domain}/
│   └── {domain}.go             # Each domain has its own package
├── grpc/
│   ├── grpc.go                 # Transport setup
│   ├── rack/handler.go         # ✅ Already migrated
│   ├── device/handler.go       # ✅ Already migrated
│   ├── task/handler.go         # ✅ Already migrated
│   ├── auth/handler.go         # ✅ Already migrated
│   ├── channel.go              # ❌ Needs migration to channel/handler.go
│   ├── framer.go               # ❌ Needs migration to framer/handler.go
│   ├── connectivity.go         # ❌ Needs migration
│   ├── ranger.go               # ❌ Needs migration
│   ├── status.go               # ❌ Needs migration
│   ├── arc.go                  # ❌ Needs migration
│   ├── types.gen.go            # ❌ To delete after migration
│   ├── rack.gen.go             # ❌ To delete (rack migrated)
│   └── v1/                     # ❌ Old proto location, to clean up
└── http/
    └── http.go                 # HTTP transport
```

### Service Constructor Pattern

All domain services now follow this pattern:

```go
package rack

import (
    "github.com/synnaxlabs/synnax/pkg/api/auth"
    "github.com/synnaxlabs/synnax/pkg/api/config"
    "github.com/synnaxlabs/synnax/pkg/service/access/rbac"
    svcRack "github.com/synnaxlabs/synnax/pkg/service/rack"
    "github.com/synnaxlabs/x/gorp"
)

type Service struct {
    db       *gorp.DB
    access   *rbac.Service
    internal *svcRack.Service
}

func NewService(cfg config.Config) *Service {
    return &Service{
        db:       cfg.Distribution.DB,
        access:   cfg.Service.RBAC,
        internal: cfg.Service.Rack,
    }
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
    if err := s.access.Enforce(ctx, access.Request{
        Subject: auth.GetSubject(ctx),
        Action:  access.ActionCreate,
        Objects: ...,
    }); err != nil {
        return CreateResponse{}, err
    }
    return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
        // ...
    })
}
```

## Remaining Work

### Priority 1: Move Remaining gRPC Handlers to Subdirectories

Move these files to their own subdirectories following the rack/device/task pattern:

| Current File | New Location |
|-------------|--------------|
| `grpc/channel.go` | `grpc/channel/handler.go` |
| `grpc/framer.go` | `grpc/framer/handler.go` |
| `grpc/connectivity.go` | `grpc/connectivity/handler.go` |
| `grpc/ranger.go` | `grpc/ranger/handler.go` |
| `grpc/status.go` | `grpc/status/handler.go` |
| `grpc/arc.go` | `grpc/arc/handler.go` |

Each handler should:
1. Be in its own package (e.g., `package channel`)
2. Export a `New(a *api.Transport) fgrpc.BindableTransport` function
3. Import translators from `service/{domain}/pb` instead of `api/grpc/v1`

### Priority 2: Update grpc/grpc.go

After moving handlers, update `grpc/grpc.go` to import from subdirectories:

```go
import (
    channelgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/channel"
    framergrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/framer"
    // etc.
)

transports := fgrpc.CompoundBindableTransport{
    channelgrpc.New(&a),
    framergrpc.New(&a, channelSvc),
    // etc.
}
```

### Priority 3: Clean Up Old Files

After all handlers migrated:
1. Delete `grpc/types.gen.go`
2. Delete `grpc/*.gen.go` translator files
3. Clean up `grpc/v1/` - protos should come from `service/{domain}/pb`

### Priority 4: Phase 4 - Distribution Layer

Channel and framer in distribution layer need their own `pb/` subdirectories.

## Key Patterns to Follow

### Import Naming

- In `api` package: Use clean names (`rack`, `device`, not `apirack`)
- In `grpc` subpackages: Alias if needed to avoid conflicts with distribution packages

### When API Type = Service Type

Domains like rack, device, task reuse service types directly:
- Import `service/{domain}/pb` for translators
- No need for separate `api/{domain}/pb/`

### When API Type ≠ Service Type

Domains like channel, framer have computed/transformed fields:
- May need `api/{domain}/pb/` with their own translators
- Or use manual translation in handler

## Files to Reference

- `core/pkg/api/layer.go` - Main entry point
- `core/pkg/api/config/config.go` - Config struct
- `core/pkg/api/auth/middleware.go` - TokenMiddleware pattern
- `core/pkg/api/grpc/rack/handler.go` - Example migrated handler
- `core/pkg/service/rack/pb/` - Example service pb/ directory

## Validation Commands

```bash
# Build API packages
cd core && go build ./pkg/api/...

# Run API tests
cd core && ginkgo -r ./pkg/api/...

# Check for old import references
grep -r "api/grpc/v1" core/pkg/api/
```
