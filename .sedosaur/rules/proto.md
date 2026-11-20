+++
description = "Protocol buffer field numbers"
paths = ["**/*.proto"]
+++

A Synnax protocol buffer is only a wire, and wire skew between versions, patches
included, does not matter here: Synnax supports a client only with a Core of its major
and minor version, and stores no data as a protocol buffer. So do not report a
renumbered, reused, or reordered field or enum value, or a removed one that is not
`reserved`. Oracle numbers proto fields in schema order, so a new field goes where it
reads best. The rule covers proto numbers only: a stored shape in an Oracle schema still
needs a new version and a migration. It does not cover the Sparkplug B protocol in
`core/pkg/service/mqtt/sparkplug/`, which other programs read, or the messages that a
client of another version reads to find and report the mismatch: the session messages in
`core/pkg/transport/grpc/auth/auth.proto`, `CheckResponse` in
`core/pkg/transport/grpc/connectivity/connectivity.proto`, status set, and each
`RetrieveRequest` in `core/pkg/transport/grpc/` whose old fields are `reserved` so that
the Core rejects a request from an older client.
