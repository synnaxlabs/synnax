# Remaining Work: KV and Alias API Split

This document tracks the remaining work for splitting KV and Alias from the Range API into separate top-level services.

## Completed Work

### Core Backend (Done)

- [x] Added `@pb` to `schemas/kv.oracle` and `schemas/alias.oracle`
- [x] Created `core/pkg/api/kv/kv.go` with handwritten request/response types
- [x] Created `core/pkg/api/alias/alias.go` with handwritten request/response types
- [x] Created `core/pkg/api/grpc/v1/kv.proto` with gRPC service definitions
- [x] Created `core/pkg/api/grpc/v1/alias.proto` with gRPC service definitions
- [x] Created `core/pkg/api/grpc/kv/handler.go` with translators
- [x] Created `core/pkg/api/grpc/alias/handler.go` with translators
- [x] Updated `core/pkg/api/layer.go` (Transport, Layer, BindTo, New)
- [x] Updated `core/pkg/api/grpc/grpc.go` to register new transports
- [x] Cleaned up `core/pkg/api/ranger/ranger.go` - removed KV/Alias code
- [x] Cleaned up `core/pkg/api/grpc/v1/ranger.proto` - removed KV/Alias services
- [x] Cleaned up `core/pkg/api/grpc/ranger/handler.go` - removed KV/Alias translators

---

## Remaining Work

### 1. Build and Test

- [ ] Run `go build ./...` in `core/` to verify compilation
- [ ] Run protobuf generation (`buf generate` or equivalent)
- [ ] Fix any compilation errors
- [ ] Run tests to verify existing functionality

### 2. TypeScript Client Updates

**Create new modules:**

- [ ] `client/ts/src/kv/` - KV client module
  - `client.ts` - KV client class
  - `index.ts` - exports
- [ ] `client/ts/src/alias/` - Alias client module
  - `client.ts` - Alias client class
  - `index.ts` - exports

**Update endpoint paths:**

| Old Path | New Path |
|----------|----------|
| `/range/kv/get` | `/kv/get` |
| `/range/kv/set` | `/kv/set` |
| `/range/kv/delete` | `/kv/delete` |
| `/range/alias/set` | `/alias/set` |
| `/range/alias/resolve` | `/alias/resolve` |
| `/range/alias/delete` | `/alias/delete` |
| `/range/alias/list` | `/alias/list` |
| `/range/alias/retrieve` | `/alias/retrieve` |

**Update ranger client:**

- [ ] `client/ts/src/ranger/` - Remove KV and Alias code from ranger client
- [ ] Update `client/ts/src/index.ts` to export new KV and Alias modules

### 3. Python Client Updates

**Create new modules:**

- [ ] `client/py/synnax/kv/` - KV client module
  - `__init__.py`
  - `client.py` - KV client class
- [ ] `client/py/synnax/alias/` - Alias client module
  - `__init__.py`
  - `client.py` - Alias client class

**Update endpoint paths:** (same as TypeScript)

**Update ranger client:**

- [ ] `client/py/synnax/ranger/` - Remove KV and Alias code
- [ ] Update `client/py/synnax/__init__.py` to export new modules

### 4. Integration Tests

- [ ] Update any integration tests that use KV/Alias APIs
- [ ] Verify end-to-end functionality with new endpoint paths

### 5. Documentation

- [ ] Update API documentation if applicable
- [ ] Delete this file when complete

---

## API Structure Summary

### Before (Unified)

```
core/pkg/api/ranger/
├── ranger.go      # Range + KV + Alias
└── ...

Transport:
  RangeKVGet, RangeKVSet, RangeKVDelete
  RangeAliasSet, RangeAliasResolve, RangeAliasDelete, RangeAliasList, RangeAliasRetrieve
```

### After (Split)

```
core/pkg/api/
├── ranger/
│   └── ranger.go  # Range only (Create, Retrieve, Delete, Rename)
├── kv/
│   └── kv.go      # KV only (Get, Set, Delete)
└── alias/
    └── alias.go   # Alias only (Set, Resolve, Delete, List, Retrieve)

Transport:
  RangeCreate, RangeRetrieve, RangeDelete, RangeRename
  KVGet, KVSet, KVDelete
  AliasSet, AliasResolve, AliasDelete, AliasList, AliasRetrieve
```

---

## Notes

- **Breaking Change**: This is a breaking change for clients. Old endpoint paths will not work.
- **Range-Scoped**: KV and Alias still require a Range UUID in their requests - they are logically scoped to ranges but are now separate API services.
- **Handwritten Types**: Request/response types are handwritten in Go (not Oracle-generated) per project convention.
