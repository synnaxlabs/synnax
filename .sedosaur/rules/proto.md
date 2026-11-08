+++
description = "Protocol buffer field numbers"
paths = ["**/*.proto", "schemas/**"]
+++

A Synnax protocol buffer is only a wire between programs of one release: a client needs
a Core of the same major and minor version, and Synnax stores no data as a protocol
buffer. So field and enum value numbers can change. Do not report a renumbered, reused,
or reordered field or enum value, or a removed one that is not `reserved`. Oracle
numbers fields in schema order, so a new field goes where it reads best. This does not
apply to the Sparkplug B protocol in `core/pkg/service/mqtt/sparkplug/`, which other
programs read.
