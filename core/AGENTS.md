Go development rules for this module: @../docs/claude/toolchains/go.md

# Core server

The Synnax server: composes Cesium (time-series) and Gorp (typed KV) into a telemetry
engine, exposed over Freighter transports.

## Layered architecture (`core/pkg/`)

Five strictly-ordered layers, each with a `layer.go` composition root. Dependencies flow
downward only — a layer never imports one above it:

1. **`storage/`** (lowest) — engine management: Cesium (`ts/`) for telemetry frames +
   Pebble KV. Single-node, no cluster awareness.
2. **`distribution/`** — turns storage into the data plane: the Core's identity,
   channels with node-aware keys, and the framer (reads, writes, and streams).
3. **`service/`** — business logic on top of distribution: resource (search and names
   across types), group, auth, access, user, ranger, hardware (task/rack/device), arc,
   workspace items (schematic, lineplot, log, table, panel, view), label, status.
4. **`api/`** (highest) — transport-agnostic client interface: request/response types,
   validation, freighter-compatible service definitions.

Supporting packages outside the stack:

- `transport/` — concrete Freighter bindings (`grpc/`, `http/`) for the api layer.
- `server/` — serves gRPC + HTTP multiplexed on one port (`branch.go`).
- `security/` — TLS/cert provider (secure + insecure modes).
- `driver/` — embeds and manages the C++ driver binary lifecycle.
- `console/` — serves the console web build.
- `version/` — build version stamping.

## Placement rule

New functionality goes in the lowest layer that can implement it: needs only local
storage → storage; needs channel keys or frame flow → distribution; needs Synnax
business concepts → service; only shapes requests/responses → api. An api handler must
stay thin — logic in api that belongs in service is a defect.
