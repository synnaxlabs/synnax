# Legacy task config fixtures

These files are frozen copies of the task file shape that the released Console exported:
`JSON.stringify({ ...task.config, type: task.type })`, with the config in the camelCase
in-memory form that each task type's zod schema declared. They were derived by hand from
the schemas at Git ref `origin/main` commit `0d092325a152250ce4c729ff08fdbcd92470161d`,
and every task type in this directory existed at that commit. Each fixture sets every
field the schema accepts to a distinctive non-default value so a lossy import path fails
a diff. Do not regenerate these files from newer schemas; they must stay byte-stable as
proof of what released Consoles wrote.

The fixtures were validated mechanically against the zod schemas at that commit: every
fixture parses, and every one except the HTTP pair round-trips byte-identical through
the schema. The HTTP fixtures freeze the older record shape for headers, query params,
and enum values, which that schema still accepted and upgraded to the list shape in
memory; the legacy rewrite's `RecordToList` covers the record shape and passes the
list shape through untouched.

A task type with more than one released shape carries one extra fixture per older
shape, suffixed with the shape's distinguishing trait:

- `ni_analog_read_config_device.json` freezes the earlier analog read shape: the
  device at the config level instead of on each channel, and the
  `ai_frequency_voltage` type name. It also exercises the `Chan` and `BuiltIn`
  cold-junction sources, which the newer fixture does not carry.
- `labjack_write_cmd_key.json` freezes the earlier write shape, which keyed channels
  with `cmdKey` and `stateKey` instead of `cmdChannel` and `stateChannel`.
