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
the schema. HTTP released two shapes for headers, query params, and enum values: records
through 0.53.1 and lists from 0.53.2 on. `http_read.json` freezes the record era,
`http_read_list.json` the list era, and `http_write.json` the write shape at its 0.53.1
introduction (record headers and query params, list enum values); the legacy rewrite's
`RecordToList` covers the record shapes and passes list shapes through untouched.

A task type with more than one released shape carries one extra fixture per older shape,
suffixed with the shape's distinguishing trait:

- `ni_analog_read_config_device.json` freezes the earlier analog read shape: the device
  at the config level instead of on each channel, the kebab-case `strainConfig` and
  lowercase `strain` units Consoles 0.36 through 0.44 wrote, and the
  `ai_frequency_voltage` type name. That spelling has driver provenance, not Console: no
  released Console or Python client wrote it, only the released Driver's parser accepted
  it. It also exercises the `Chan` and `BuiltIn` cold-junction sources, which the newer
  fixture does not carry.
- `labjack_write_cmd_key.json` freezes the earlier write shape, which keyed channels
  with `cmdKey` and `stateKey` instead of `cmdChannel` and `stateChannel`.

The released Python client was a second writer of stored configs, distinct from Console
exports: snake_case keys, no version stamp, and channel types the Console never had.
Fixtures with a `_py` suffix freeze that dialect as `model_dump()` wrote it:

- `ni_analog_read_py.json` carries the Python-only analog types (`ai_charge` with the
  released `uC` units, `ai_freq_voltage`, both thermistors, `ai_voltage_rms`).
- `ni_counter_read_py.json` carries `ci_frequency` and `ci_period` channels with the
  Python-only `DynAvg` measurement method, plus the `Seconds` frequency units.
- `labjack_read_py_no_scale.json` carries AI and thermocouple channels with no `scale`
  key and `pos_chan` on the analog channel, which only Python wrote.
- `arc_py.json` carries the `auto_start` field, which only Python wrote; the Console
  wrote `arcKey` alone.

The `expected/` directory holds the canonical stored record each fixture imports to.
Regenerate with `UPDATE_LEGACY_GOLDENS=1 ginkgo`, then run Prettier over the output; the
comparison is semantic, and Prettier's formatting is the committed form.
