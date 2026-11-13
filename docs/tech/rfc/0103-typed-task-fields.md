# 103 Strongly typed task fields

- **Author**: Patrick Dotson
- **Date**: 2026-11-06
- **Related**: [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [PR #4594 - SY-4579: Add a fill value for missing HTTP read fields](https://github.com/synnaxlabs/synnax/pull/4594)

## 0 Summary

HTTP, Kafka, and MQTT tasks describe a JSON value with flat fields and an `any` value.
The Core stores any mix, and each runtime handles a bad one differently. Numbers are
`float64`, so they lose digits past 2^53 on 64-bit channels. HTTP rejects JSON, UUID,
and bytes channels. OPC UA and Modbus have flat fields too, and many config integers are
wider or signed when they cannot be negative.

Each field becomes a union on a shared `Kind` in a new X package, `payload`. Each config
field takes its narrowest correct type (§6).

`core/v0.61.0` shipped every version that this RFC changes, so each change mints a new
version with a §7 migration, and releases do not hold.

## 1 Schema

The current `payload` version, with docs and tags cut:

```
Kind enum { float integer unsigned string boolean timestamp json uuid bytes }
TimeFormat enum { iso8601 unix_sec unix_ms unix_us unix_ns }
Float    float64 { @go hand  @ts hand  @py hand  @cpp hand }
Integer  int64  { @ts type "string"  @py hand }
Unsigned uint64 { @ts type "string"  @py hand }
FloatEntry    struct { label string  value Float }
IntegerEntry  struct { label string  value Integer }
UnsignedEntry struct { label string  value Unsigned }
StringEntry   struct { label string  value string }
BooleanEntry  struct { label string  value bool }
FloatRead     struct { data_type telem.DataType?  enum_values FloatEntry[]  fill Float? }
IntegerRead   struct { data_type telem.DataType?  enum_values IntegerEntry[]  fill Integer? }
UnsignedRead  struct { data_type telem.DataType?  enum_values UnsignedEntry[]  fill Unsigned? }
FloatOptions    struct { enum_values FloatEntry[]  fill Float? }
IntegerOptions  struct { enum_values IntegerEntry[]  fill Integer? }
UnsignedOptions struct { enum_values UnsignedEntry[]  fill Unsigned? }
StringRead    struct { enum_values StringEntry[]  fill string? }
BooleanRead   struct { enum_values BooleanEntry[]  fill bool? }
TimestampRead struct { time_format TimeFormat  fill telem.TimeStamp? }
WriteField union on type Kind extends BaseWriteField {
    float { value Float }  integer { value Integer }  unsigned { value Unsigned }
    string { value string }  boolean { value bool }  json { value string }  uuid {}
    timestamp { time_format TimeFormat = iso8601 }  bytes { value string }
}
```

`BaseWriteField` holds `key` and `pointer`. `timestamp` and `uuid` write fields generate
a value per message. `json` holds JSON text, because `record` holds only objects, and
`bytes` holds base64 text. Phases 7, 9, and 12 extend the numeric `*Read` types, whose
`data_type` sizes a new channel. Phase 27 adds the three `*Options` types without it as
new types, with no new payload version, and Phases 28 to 31 extend them.

Phase 27 also adds `Binding` to `schemas/synnax/task.oracle`, in task v3 with no new
version, because `x` cannot import `channel.Key` or its `@ref`. Integration version
files then pin task v3:

```
Binding union on type {
    bound { key channel.Key  @ref kind use  @ref on_delete restrict }
    unbound { name string = ""  data_type telem.DataType }
}
```

Each integration declares two unions on its own base:

```
ReadField union on type payload.Kind extends BaseReadField {
    float extends payload.FloatOptions {}  integer extends payload.IntegerOptions {}
    unsigned extends payload.UnsignedOptions {}  string extends payload.StringRead {}
    boolean extends payload.BooleanRead {}  timestamp extends payload.TimestampRead {}
    json {}  uuid {}  bytes {}
}

ChannelField union on type payload.Kind extends BaseChannelField {
    float { enum_values payload.FloatEntry[] }
    integer { enum_values payload.IntegerEntry[] }
    unsigned { enum_values payload.UnsignedEntry[] }
    string { enum_values payload.StringEntry[] }
    boolean { enum_values payload.BooleanEntry[] }
    timestamp { time_format payload.TimeFormat }  json {}  uuid {}  bytes {}
}
```

`BaseReadField` holds `key`, `disabled`, `pointer`, and `channel task.Binding`, and
`BaseChannelField` holds `pointer` and `channel task.Binding`. The MQTT Sparkplug read
entry replaces its `channel`, `name`, and `data_type` with `channel task.Binding` too. A
channel field writes a matching entry's label, or else `string` writes text and
`boolean` a boolean.

A field is bound to a channel, or unbound with the name and data type of the channel to
create. No field then holds a `data_type` that the Core must ignore. `unbound` holds a
full data type on every kind, so one union serves read and channel fields, and §4 checks
it as it checks a channel. Kafka write channels, Sparkplug write targets, and edge tags
keep their own channel fields, and this RFC does not bind them.

OPC UA v2 replaces `array_mode` and `array_size` with `sampling Sampling = unary`.
Modbus v3 replaces `string_length`, and its `WriteChannel` stays an alias.

```
Sampling union on type { unary {}  array { size uint32 = 1 } }
RegisterString struct { length uint16  bytes_swapped bool? }
ReadChannel gains holding_register_string and input_register_string extends RegisterString
```

## 2 Numbers

A raw `int64` does not survive `msgpack.EncodedJSON`, a `map[string]any`, where a JSON
number becomes a `float64`. So `Integer` and `Unsigned` are distinct types stored as
decimal strings, like `telem.TimeStamp` and `telem.Alignment`. A negative `uint64` and a
value above `int64` range are shape errors. `Float` is a `float64` stored as a JSON
number when finite, and as `"NaN"`, `"Infinity"`, or `"-Infinity"` otherwise, as Arc IR
values are in #4699 (`types.DecodeValue`, C++ `to_sample_value`). NaN and infinities fit
a `float32` channel, so a non-finite fill needs a float channel by construction.

All three have hand codecs: Go in `x/go/payload`, C++ through `Parser`, which reads
digits from a string (`json.h:591`), TypeScript as a decimal `string` and a `number`,
and Python as an `int` subclass and a `float` that dump text. The Console checks integer
range with `BigInt`, and its float fill input in `console/src/platform/task/payload`
takes the three texts.

JSON numbers parse exactly today in Go and C++. Phase 3 fixes the three inexact places:
`xjson.Enum.Value` and `x::json::EnumMap` hold a float, `Encoder.Append` compares enums
through `n.float()`, and `x::json::from_timestamp` goes through `double`. Each entry
becomes one encoded sample. Integer, unsigned, string, and boolean entries compare by
exact value. A float entry compares by numeric equality, so `-0.0` matches `0` and an
infinity matches itself, but a `"NaN"` entry matches any NaN sample by `isnan`. On read,
a `"NaN"` entry's label decodes to NaN. On write, entries match before the non-finite
rule in §5, so a matched sample writes its label. Phase 3 does this in both runtimes. A
read field's labels are unique and its values may repeat, so `ON` and `HIGH` can both
mean 1. A channel field's values are unique under this rule, after narrowing, and its
labels may repeat, since two values can share a label. Phase 3 also adds string and
boolean write enums: `Encoder.Append` quotes a string sample and writes a boolean with
no lookup, and `from_sample_value` checks enums only for a numeric value written as a
string.

## 3 Bytes

With a pointer, bytes are base64 text, as `AppendSampleText` writes. An empty pointer
stays the JSON root value on other kinds. Only on `bytes` is it a raw, non-JSON payload,
and then the field is the only one enabled. On Kafka, a raw write is an empty
`value_pointer`. It is allowed only when every enabled channel is `bytes` and the record
places nothing else: no channel pointers, no timestamp pointer, and no extra record
fields. The one-field rule does not apply to it, and `ValidateEntry` (§5) checks it.

- **`x/go/json`**: `NewDecoder` accepts `BytesT` as base64 text, and `Encoder` writes it
  as base64, where it quotes raw bytes today.
- **Kafka**: `read_task.go:159` and `write_task.go:261` start to allow one on `bytes`.
- **MQTT and HTTP**: A `bytes` field with an empty pointer skips `readTopic.extract` and
  the body parse. `driver/http/read_task.cpp:112` drops its rejection.
- **`x/cpp`**: `to_sample_value` and `Series` gain `BYTES_T`, and `x::base64::decode`
  gets a strict form. `variable_samples` (`driver/opcua/telem/telem.cpp:99`) moves to
  `Series`.

## 4 Kinds

`payload.KindOf` maps a data type to its kind. Each check and correction uses it.

| Channel data type | `KindOf`, read field | Channel field                   |
| ----------------- | -------------------- | ------------------------------- |
| Float             | `float`              | `float`, `string`, `boolean`    |
| Signed integer    | `integer`            | `integer`, `string`, `boolean`  |
| Unsigned integer  | `unsigned`           | `unsigned`, `string`, `boolean` |
| `boolean`         | `boolean`            | `boolean`, `unsigned`, `string` |
| Other             | Same name            | Same name                       |

A numeric kind takes a JSON boolean as 0 or 1, and an enum label. On an unbound field,
`data_type` takes the place of the channel's data type in this table. A field with no
fill is required, and the #4594 absent rule stays: no value, `null`, or `""` on a
non-string channel.

## 5 Rules by layer

**Core write**, in `ResolveEntry` (`task/config/config.go`), through a `Channels`
function that each service builds, as `slavesOf` does. It skips disabled fields and
missing channels, because the reference check on write rejects a missing channel. It
rejects a kind that §4 does not allow, a fill or entry that does not fit the channel's
width, and a fill on an index channel. It also converts each channel field entry to the
channel's type and rejects values that collide only after narrowing, such as `0.1` and
`0.1000000001` on a `float32` channel. It checks an unbound field by the same rules,
against its `data_type`.

**Core config**, in `ValidateEntry`, which has no channel and checks every field: Read
field labels are unique, channel field values are unique on the stored value, so two
`"NaN"` entries or `0` and `-0.0` collide, `json` and `bytes` text is valid, a raw
`bytes` read field is alone, and a raw Kafka write follows §3. Only a static `float`
write value must be finite, because it goes into a payload as a JSON number. Fills and
float entries may be non-finite. Kafka and MQTT check kinds at task configure and gain
fill (Kafka skips an absent value today, `read_task.go:767`).

**Non-finite samples**, when no entry matches: A `float` channel field fails the write.
A `string` one writes `NaN`, `Infinity`, or `-Infinity`, as `payload.Float` does. A
`boolean` one fails on NaN. Phase 3 aligns this: today Go errors, writes `+Inf`, and
writes `true` (`field.go:318`), and C++ writes `null`, `inf`, and `true`
(`convert.cpp:430`).

**Driver**: It keeps the kind against `ch.data_type`, a float for a non-finite fill, the
index rule, and `check_from_sample_value`, because channels drift. It removes time
format, fill-needs-optional, `json_type`, and enum shape errors, and adds `BOOLEAN_T`.
JSON and UUID samples are strings (`driver/opcua/telem/telem.h:36`), so a new
`Series(SampleValue, DataType)` keeps the channel's type, and writes read `uuids()`. The
HTTP Driver reads the key from `bound` and rejects an enabled `unbound` field, where it
rejected channel 0. The Kafka and MQTT tasks in the Core do the same.

**Console**: The data type select sets the kind, and `DataTypeField` shows every type.
On HTTP and MQTT, `onConfigure` binds each enabled unbound field to the channel that the
device stores for it: the pointer's channel for a read field and the target's channel
for a channel field. It binds only when that channel exists, holds the field's
`data_type`, and, if it is fixed-density, is on the index that the source picks below.
Otherwise, and always on Kafka, whose devices store no channels, it creates one, and its
key replaces the stored one. `unbound.name` holds the user's name. A field that becomes
unbound again takes its channel's name without a `_<data_type>` or `_<data_type>_<n>`
suffix whose type is the channel's own, so suffixes do not stack. A user name of that
form loses the suffix too, which changes only the name of the new channel. An empty name
first takes the name the Console generates today. A new channel takes the name when it
is free, else the name with its data type as a suffix, such as `temp_int32`, else that
name with a counter, such as `temp_int32_2`, until a name is free. A taken name never
fails configure, and configure never binds by name. Each source picks one index: the
channel of a bound index field, else the first non-zero index of its bound channels, on
HTTP and MQTT the stored index when it still exists, else a new one. An unbound index
field binds to it, and each new fixed-density channel takes it. On HTTP and MQTT the
device stores it. On HTTP and MQTT, a new variable-density channel (string, JSON, or
bytes) stays virtual, as today. On Kafka it takes the index, as today. `onConfigure`
then saves the field as bound. An adapter keeps the atomic Phase 12 near 400 lines, and
Phase 14 deletes it.

## 6 Field types

Each integer takes the vendor's or protocol's type. With none, it takes the narrowest
unsigned type that holds its bound, or with no bound the unsigned type of its old width.
Floats stay `float64`, because DAQmx and LJM take `double`, except where DAQmx takes an
integer.

| Schema        | Field                             | Old            | New                              | Validation         | Reason                                                                               |
| ------------- | --------------------------------- | -------------- | -------------------------------- | ------------------ | ------------------------------------------------------------------------------------ |
| OPC UA        | `Sampling.array.size`             | `int32 = 1`    | `uint32 = 1`                     | `min 1`            | `ArrayDimensions` is `UInt32`, and 0 samples reads nothing                           |
| Modbus        | `RegisterString.length`           | `int32 = 0`    | `uint16`                         | `min 1`, `max 250` | One request reads 125 registers at most (`driver/modbus/channels.h:71`)              |
| Modbus        | `bytes_swapped`, `words_swapped`  | `bool?`        | `bool?`                          | -                  | Absent inherits the device's order (`driver/modbus/channels.h:97`)                   |
| Modbus device | `swap_bytes`, `swap_words`        | `bool`         | `bytes_swapped`, `words_swapped` | -                  | Booleans are adjectival (`driver/modbus/device/device.h:243`)                        |
| NI            | Channel and `CJC.chan` `port`     | `int32 = 0`    | `uint32 = 0`                     | -                  | A DAQmx name such as `ai3` has no sign                                               |
| NI            | DI and DO `line`                  | `int32 = 0`    | `uint8 = 0`                      | `max 31`           | A DAQmx port has 32 lines at most                                                    |
| NI            | Counter `divisor`                 | `int32 = 4`    | `uint32 = 4`                     | `min 1`            | `DAQmxCreateCIFreqChan` takes `uInt32`, and 0 has no meaning                         |
| NI            | `pulses_per_rev`                  | `float64 = 24` | `uint32 = 24`                    | `min 1`            | DAQmx takes `uInt32`, and the Driver truncates (`driver/ni/channel/channels.h:1419`) |
| NI            | `initial_count`                   | `float64 = 0`  | `uint32 = 0`                     | -                  | DAQmx takes `uInt32`, and the Driver truncates (`driver/ni/channel/channels.h:1589`) |
| NI device     | `Hardware` `port` and `negative?` | `int32`        | `uint32`                         | -                  | The same DAQmx numbers; absent `negative` still means no pair                        |
| NI device     | `DigitalPort.lines`               | `int32`        | `uint8`                          | `max 32`           | A DAQmx port has 32 lines at most                                                    |
| LabJack       | `neg_chan`                        | `int32 = 199`  | `uint16?`                        | `max 127`          | LJM register is `UINT16`; absent writes `LJM_GND` (199); `AIN127` is last            |
| LabJack       | `*_scan_backlog_warn_on_count`    | `uint32 = 0`   | `uint32?`                        | `min 1`            | Absent lets the Driver pick, not an in-band 0                                        |
| LabJack       | `tcp_scan_multiplier`             | `int32 = 10`   | `uint32 = 10`                    | `min 1`            | The Driver rejects 0 and below (`driver/labjack/scan_task.h:36`)                     |
| Kafka device  | `tls`                             | `bool`         | `secure`                         | -                  | Adjectival, and the HTTP and MQTT name                                               |

The rest are correct: MQTT `authority uint8` is `control.Authority`, EtherCAT
`index uint16` and `sub_index uint8` are CoE's, and Modbus `address uint16` and
`bit uint8` (max 15) are the protocol's. A `@validate` bound rejects new writes from the
Console and the clients. The Driver drops each check that a type or bound now makes. The
probe writes NI `Hardware` from DAQmx, so it needs no migration.

Migration first maps the sentinels: a LabJack `neg_chan` of 199 and a backlog count of 0
become absent, so a single-ended channel stays enabled. It then clamps a stored
magnitude, so a task the user wrote keeps running: below the minimum becomes the
minimum, or 0 with none, above the maximum becomes the maximum, and a fraction
truncates, as the Driver's cast does. It logs the task, channel, field, old value, and
new value. An out-of-range identifier (NI `port` and `line`, LabJack `neg_chan`) clamps
the same way, so it names a real channel and passes `@validate`, and its channel also
becomes disabled. The clamped value may not be the wiring the user meant, so re-enabling
is the user's explicit choice. Every migration path (the store migration, `Normalize`,
and imports) applies these rules in this order. Each migration phase tests a clamp at
both bounds, a clamped-and-disabled identifier, and that a stored 199 becomes absent and
the channel stays enabled.

Phases 15, 21, 23, and 24 landed in OPC UA v2, Modbus v3, LabJack v3, and NI v3. Phase
25 waits for, and renames the Modbus device keys in, the RFC 0081 `modbus.Server`
cutover (RFC 0081 §4.8), which already rewrites every Modbus device and types its
writes. The old Python client writes `swap_bytes`, and an old Driver requires it
(`device.h:273`), so that release's notes say to upgrade both with the Core.

## 7 Migration

`MigrateReadField` is pure, for the envelope ladder, and uses the stored `data_type`. It
cannot see channels, so a key it binds can dangle on import (see **Dangling keys**). A
numeric type picks its kind. `boolean` entries become `value != 0`. `string` drops its
entries, as no conversion read them. `timestamp` with no `time_format` takes `iso8601`.

- **Binding**: From each version, from the shipped one up to the one before `Binding`, a
  read field, channel field, or Sparkplug read entry with `channel != 0` becomes `bound`
  and drops `name` and `data_type`. One with `channel == 0` becomes `unbound` with the
  stored `name` and `data_type`. Each shipped default is `float64`, so
  `MigrateReadField`, the first step from a shipped version, gives such an index field
  or field with a `time_format` the `timestamp` kind and keeps its `time_format`. A
  later step takes the data type from the kind when the field stores none: `float64`,
  `int64`, or `uint64` for a numeric kind, and the kind's own type otherwise.
- **Entries**: A `float64` entry converts when it is whole and fits the kind. Others
  drop with a warning, as they failed at run time.
- **#4594 fill**: A number is `value`. An enum label resolves once to its entry's value,
  or is logged and dropped when no entry has it. `optional` with no fill is `"NaN"` on
  `float`. On each other kind the Driver rejected it, so it was never a working config:
  it becomes no fill, the field warns as usual, and the migration logs it.
- **Channel field**: A stored `timestamp` data type wins. `json_type: string` with
  entries, or `number`, becomes the numeric kind. `string` with no entries stays
  `string`.
- **Static write field**: Lift by the stored Go type, not `json_type`, because the HTTP
  Driver sends the value as it is. A whole `float64` within 2^53 becomes `integer`, and
  other numbers become `float`. An `int64` or `uint64` keeps its value. A string or
  boolean keeps its kind. Others, `null` included, become `json` text. A generator
  becomes `uuid` or `timestamp`.

`Reconcile` sets the kind from §4, against the channel of a bound field or the
`data_type` of an unbound field. A bound field holds no `data_type`, so nothing is left
to clear. Between numeric kinds it converts entries and fills by the same rules, so a
non-finite fill becomes no fill, and it drops them on other kinds. It logs each change,
naming a foreign key that matched an unrelated local channel. Migration and `Reconcile`
apply only each side's rule. A read field keeps the first entry of a repeated label, so
a stored `{ON: 1, HIGH: 1}` keeps both. A channel field keeps the first entry of a set
whose values collide, on the stored value or after narrowing. Each drop is logged.

- **Boot**: `versions.ReadMigrations` becomes `NewReadMigrations(channels)`, as
  `ethercat/versions/v2/resolve.go` does. The Channel service opens first
  (`layer.go:339`). `http_write` gets its first migration.
- **Import and the task migration**: `Normalize` gets `tx` (`task/imex.go:116`,
  `task/versions/v3/migrate.go:137`). A new `ReconcileEntry` hook runs in it at each
  version. Without it, the task migration quarantines a mismatch. Client writes do not
  reconcile.
- **Dangling keys**: Export adds a `$channels` object, with the exporting Core's
  `cluster` key and a `keys` map from each bound key to its channel's `name` and
  `data_type`, next to `type` and `name` in the envelope body (`task/imex.go`). A new
  `Store.BoundKeys(config)` gives export the keys. No config field can hold `$channels`,
  and import removes it and passes it to `Store.Decode`. For any version before
  `Binding`, legacy input included, each integration's store builds the map from the raw
  body after the legacy rewrite and before the ladder runs, with the timestamp rule of
  **Binding**. `Decoded` carries the map to `ReconcileEntry`, which now runs before
  `DropDangling` and turns each bound key into `unbound` from the map when no local
  channel has that key, or when `cluster` is another Core's, because two Cores hand out
  the same channel keys. From another Core, a Sparkplug write target's `channel` and an
  edge tag's `command_channel` become 0, so configure creates them again. Each takes its
  channel name from `$channels`, whose keys include these channels, or else the name
  that the Console derives today, such as `<tag>_cmd`. A Kafka write channel or an edge
  tag's published `channel` becomes disabled, because configure cannot create it. The
  import logs each one, with any name it set. A map built from a version before
  `Binding` holds no real channel values, so it unbinds only a missing key. With no map
  entry for a missing key, `ReconcileEntry` removes the field, or for a channel field
  its target or endpoint, and logs what it removed. `ReconcileEntry` skips a missing key
  that is not a `Binding`, such as a Kafka write channel, and `DropDangling` removes it
  as today. `bound.key` 0 is invalid. Phase 27 adds `BoundKeys`, the `Decode` and hook
  arguments, the order, and a `cluster.Identity` input to the task service. Phases 28 to
  31 apply the rule.

OPC UA v1 `array_mode: true` lifts to `array`. A Modbus `string` data type lifts to the
string variant. Each field in §6 clamps and logs, and an identifier also disables its
channel. A released Core keeps reading v2, v3, and v0.58 to v0.59 envelopes.

## 8 Implementation phases

#4594 merges before Phase 2. Each phase is one PR. A phase that changes a stored shape
in place mints the next version if a release shipped that version first. A new type may
join a shipped version. Phase 13 removes `EnumEntry`, `JSONType`, and `GeneratorType`.

No schema or test uses a qualified variant `extends` or borrowed discriminator. Phase 1
tests both, with `integer payload.IntegerRead` and one `Kind` per integration as
fallbacks.

The embedded Driver (`core/pkg/driver`) always matches the Core. A standalone Driver can
lag, and fails silently on three shapes: an absent `array_mode` runs an array task as
unary, an absent `json_type` sends a number for a `string` or `boolean` channel field,
and an enum entry past 2^53 rounds. The rest fail loudly. Upgrade standalone Drivers
with the Core, and say so in the notes of each release that ships a new shape. PR #4493
would reject a lagging Driver, but this RFC does not depend on it. Every path that
carries a config keeps the `Float` text form; check C++ `to_json`.

| Phase | Scope                                                 | Touches                                                                                                                                                                             | Depends on                     |
| ----- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| 1     | Add the `payload` schema with today's shared types    | `schemas/x`, `x/go/payload`, `x/ts/src/payload`, `x/py/x/payload`, `x/cpp/payload`, `oracle/analyzer`                                                                               | -                              |
| 2     | Point Kafka, MQTT, and HTTP v4 at `payload`           | `schemas/synnax`, `core/pkg/service`, `client`, `console/src/platform/task/payload`                                                                                                 | 1                              |
| 3     | Key JSON enums by sample and keep 64-bit values exact | `x/go/json`, `x/cpp/json`                                                                                                                                                           | -                              |
| 4     | Convert bytes as base64 in `x/go/json`                | `x/go/json`                                                                                                                                                                         | -                              |
| 5     | Pass `tx` to `Normalize` and add `ReconcileEntry`     | `core/pkg/service/task`                                                                                                                                                             | -                              |
| 6     | Inject channels into the HTTP, Kafka, and MQTT stores | `x/go/payload`, `core/pkg/service`                                                                                                                                                  | 1, 5                           |
| 7     | Retype Kafka read fields and rename `tls` to `secure` | `schemas/synnax`, `core/pkg/service/kafka`, `console/src/feature/kafka`, `client`                                                                                                   | 2, 3, 4, 6                     |
| 8     | Retype Kafka write channels                           | `schemas/synnax/versions/kafka`, `core/pkg/service/kafka`, `x/go/json`, `console/src/feature/kafka`, `client`                                                                       | 2, 3, 4, 6                     |
| 9     | Retype MQTT read fields                               | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`                                                                                       | 2, 3, 4, 6                     |
| 10    | Retype MQTT write channel fields                      | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`                                                                                       | 8                              |
| 11    | Share a typed `WriteField` and migrate `http_write`   | `schemas`, `core/pkg/service`, `driver/http`, `console/src/platform/task/payload`, `client`                                                                                         | 8, 10                          |
| 12    | Cut over HTTP read fields                             | `schemas/synnax/versions/http`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`                                                                   | 2, 3, 6                        |
| 13    | Cut over the HTTP `ChannelField`                      | `schemas`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`                                                                                        | 7, 9, 11, 12                   |
| 14    | Retype the HTTP task forms and delete the adapter     | `console/src/feature/http/task`                                                                                                                                                     | 12, 13                         |
| 15    | Replace OPC UA array fields with a `Sampling` union   | `schemas/synnax/versions/opcua`, `core/pkg/service/opcua`, `driver/opcua`, `console/src/feature/opcua/task`, `client`                                                               | -                              |
| 16    | Convert JSON, UUID, and bytes samples in the Driver   | `x/cpp/json`, `x/cpp/base64`, `x/cpp/telem`, `driver/opcua/telem`                                                                                                                   | -                              |
| 17    | Read JSON, UUID, and bytes channels on HTTP           | `driver/http`, `console/src/feature/http/task`                                                                                                                                      | 14, 16                         |
| 18    | Write JSON, UUID, and bytes channels on HTTP          | `driver/http`, `console/src/feature/http/task`                                                                                                                                      | 14, 16                         |
| 19    | Read and write raw bytes payloads on Kafka and MQTT   | `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `console/src/feature/kafka`                                                                                                      | 7, 8, 9, 10                    |
| 20    | Read and write raw bytes payloads on HTTP             | `driver/http`                                                                                                                                                                       | 17, 18                         |
| 21    | Add Modbus v3 string register variants                | `schemas/synnax/versions/modbus`, `core/pkg/service/modbus`, `driver/modbus`, `console/src/feature/modbus`, `client`                                                                | -                              |
| 22    | Remove config-only checks from the Driver             | `driver/http`                                                                                                                                                                       | 12, 13                         |
| 23    | Narrow LabJack v3 config field types                  | `schemas/synnax`, `core/pkg/service/labjack`, `driver/labjack`, `console/src/feature/labjack`, `client`                                                                             | -                              |
| 24    | Narrow NI config field types                          | `schemas/synnax`, `core/pkg/service/ni`, `driver/ni`, `console/src/feature/ni`, `client`                                                                                            | -                              |
| 25    | Rename the Modbus device swap flags                   | `schemas/synnax`, `core/pkg/service/modbus`, `driver/modbus/device`, `console/src/feature/modbus`, `client`                                                                         | 21                             |
| 26    | Update the task docs for typed fields                 | `site/docs/src/pages/reference/hardware`                                                                                                                                            | 14, 15, 19, 20, 21, 23, 24, 25 |
| 27    | Add `task.Binding` and the payload `*Options` types   | `schemas/x`, `schemas/synnax/task.oracle`, `schemas/synnax/versions/task`, `x/go/payload`, `x/ts/src/payload`, `x/py/x/payload`, `x/cpp/payload`, `core/pkg/service/task`, `client` | 11                             |
| 28    | Bind Kafka read fields to `task.Binding`              | `schemas/synnax`, `core/pkg/service/kafka`, `core/pkg/service/task`, `console/src/feature/kafka`, `client`, `site/docs/src/pages/reference/hardware`                                | 7, 27                          |
| 29    | Bind MQTT read fields and Sparkplug entries           | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`, `site/docs/src/pages/reference/hardware`                                             | 9, 27                          |
| 30    | Bind MQTT write channel fields                        | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`, `site/docs/src/pages/reference/hardware`                                             | 10, 27                         |
| 31    | Bind HTTP read fields and the `ChannelField`          | `schemas/synnax/versions/http`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`, `site/docs/src/pages/reference/hardware`                         | 12, 13, 14, 27                 |
