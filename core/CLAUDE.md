Go development rules for this module: @../docs/claude/toolchains/go.md

# Core Server

The Synnax server: composes Cesium (time-series) and Aspen (distributed KV) into a
clustered telemetry engine, exposed over Freighter transports.

## Layered Architecture (`core/pkg/`)

Four strictly-ordered layers, each with a `layer.go` composition root. Dependencies flow
downward only — a layer never imports one above it:

1. **`storage/`** (lowest) — engine management: Cesium (`ts/`) for telemetry frames +
   Pebble KV. Single-node, no cluster awareness.
2. **`distribution/`** — makes storage cluster-transparent: channels, framer
   (reads/writes routed between nodes), node membership via Aspen. Data here is
   addressable cluster-wide.
3. **`service/`** — business logic on top of distribution: ontology (resource graph),
   group, search, auth, access, user, ranger, hardware (task/rack/device), arc,
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

## Placement Rule

New functionality goes in the lowest layer that can implement it: needs only local
storage → storage; needs cluster routing → distribution; needs Synnax business concepts
→ service; only shapes requests/responses → api. An api handler must stay thin — logic
in api that belongs in service is a defect.
