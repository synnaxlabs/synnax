# 103 Strongly typed task fields

- **Author**: Patrick Dotson
- **Date**: 2026-11-06
- **Related**: [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [PR #4594 - SY-4579: Add a fill value for missing HTTP read fields](https://github.com/synnaxlabs/synnax/pull/4594)

## 0 Summary

HTTP, Kafka, and MQTT tasks describe a JSON value with flat fields and an `any` value.
The Core stores any mix, and each runtime handles a bad one differently. Numbers are
`float64`, so they lose digits past 2^53 on 64-bit channels. HTTP rejects JSON, UUID,
and bytes channels. OPC UA and Modbus have flat fields too.

Each field becomes a union on a shared `Kind` in a new X package, `payload`.

**Releases hold until every phase from 1 to 15 has landed.** http v4, Kafka v0, MQTT v1,
and OPC UA v2 are not in `core/v0.60.1`, so they change in place, with no v5. A release
before then owes a migration for each. `CONTRIBUTING.md` says `main` "is always
releasable" and that "unfinished work ships dark behind a flag", but a stored shape
cannot ship dark: once a stable release writes it, compatibility is owed. Modbus v2
shipped, so Modbus gets a v3 and does not hold releases.

## 1 Schema

`schemas/x/versions/payload/v0.oracle`, with docs and tags cut:

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
`bytes` holds base64 text. Each integration declares two unions, because its base holds
`channel.Key`:

```
ReadField union on type payload.Kind extends BaseReadField {
    float extends payload.FloatRead {}  integer extends payload.IntegerRead {}
    unsigned extends payload.UnsignedRead {}  string extends payload.StringRead {}
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

`BaseReadField` holds `key`, `name`, `disabled`, `channel`, and `pointer`, and
`BaseChannelField` holds `pointer`, `channel`, `name`, and `data_type?`. A channel field
writes a matching entry's label, or else `string` writes text and `boolean` a boolean.

OPC UA v2 replaces `array_mode` and `array_size` with `sampling Sampling = unary`.
Modbus v3 replaces `string_length`, and its `WriteChannel` stays an alias.

```
Sampling union on type { unary {}  array { size int32 = 1 } }
RegisterString struct { length int32  bytes_swapped bool? }
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
becomes one encoded sample, compared exactly. Phase 3 also adds string and boolean write
enums: `Encoder.Append` quotes a string sample and writes a boolean with no lookup, and
`from_sample_value` checks enums only for a numeric value written as a string.

## 3 Bytes

With a pointer, bytes are base64 text, as `AppendSampleText` writes. An empty pointer
stays the JSON root value on other kinds. Only on `bytes` is it a raw, non-JSON payload,
and then the field is the only one enabled. On Kafka, a raw write is an empty
`value_pointer`.

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

A numeric kind takes a JSON boolean as 0 or 1, and an enum label. With channel 0,
`data_type` states the width for a new channel. A field with no fill is required, and
the #4594 absent rule stays: no value, `null`, or `""` on a non-string channel.

## 5 Rules by layer

**Core write**, in `ResolveEntry` (`task/config/config.go:105`), through a `Channels`
function that each service builds, as `slavesOf` does. It skips disabled fields, channel
0, and missing channels, because imports hold foreign keys. It rejects a kind that §4
does not allow, a fill or entry that does not fit the channel's width, and a fill on an
index channel. It clears `data_type`.

**Core config**, in `ValidateEntry`: Labels and channel field values are unique, `json`
and `bytes` text is valid, a raw `bytes` field is alone, and a static `float` is finite.
Fills and entries are never JSON numbers. Kafka and MQTT check kinds at task configure
and gain fill (Kafka skips an absent value today, `read_task.go:767`).

**Non-finite samples**: A `float` channel field fails the write. A `string` one writes
`NaN`, `Infinity`, or `-Infinity`, as `payload.Float` does. A `boolean` one fails on
NaN. Phase 3 aligns this: today Go errors, writes `+Inf`, and writes `true`
(`field.go:318`), and C++ writes `null`, `inf`, and `true` (`convert.cpp:430`).

**Driver**: It keeps the kind against `ch.data_type`, a float for a non-finite fill, the
index rule, and `check_from_sample_value`, because channels drift. It removes time
format, fill-needs-optional, `json_type`, and enum shape errors, and adds `BOOLEAN_T`.
JSON and UUID samples are strings (`driver/opcua/telem/telem.h:36`), so a new
`Series(SampleValue, DataType)` keeps the channel's type, and writes read `uuids()`.

**Console**: The data type select sets the kind, and `DataTypeField` shows every type.
An adapter keeps the atomic Phase 12 near 400 lines, and Phase 14 deletes it.

## 6 Migration

`MigrateReadField` is pure, for the envelope ladder, and uses the stored `data_type`. A
numeric type picks its kind. `boolean` entries become `value != 0`. `string` drops its
entries, as no conversion read them. `timestamp` with no `time_format` takes `iso8601`.

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

`Reconcile` sets the kind from §4 and clears `data_type`, because the Driver never read
it and the HTTP default is `float64`. Between numeric kinds it converts entries and
fills by the same rules, so a non-finite fill becomes no fill, and it drops them on
other kinds. It logs each change, naming a foreign key that matched an unrelated local
channel.

- **Boot**: `versions.ReadMigrations` becomes `NewReadMigrations(channels)`, as
  `ethercat/versions/v2/resolve.go` does. The Channel service opens first
  (`layer.go:339`). `http_write` gets its first migration.
- **Import and the task migration**: `Normalize` gets `tx` (`task/imex.go:116`,
  `task/versions/v3/migrate.go:137`). A new `ReconcileEntry` hook runs in it at each
  version. Without it, the task migration quarantines a mismatch. Client writes do not
  reconcile.

OPC UA v1 `array_mode: true` lifts to `array`. A Modbus `string` data type lifts to the
string variant. A released Core keeps reading v2, v3, and v0.58 to v0.59 envelopes.

## 7 Implementation phases

#4594 merges before Phase 2. Each phase is one PR. Releases hold until every phase from
1 to 15 has landed. Phase 13 removes `EnumEntry`, `JSONType`, and `GeneratorType`.

No schema or test uses a qualified variant `extends` or borrowed discriminator. Phase 1
tests both, with `integer payload.IntegerRead` and one `Kind` per integration as
fallbacks.

The embedded Driver ships inside the Core (`core/pkg/driver`) and always matches. A
standalone Driver on a remote rack can lag, and fails silently on three shapes: an
absent `array_mode` runs an array task as unary, an absent `json_type` sends a number
for a `string` or `boolean` channel field, and an enum entry past 2^53 rounds. The rest
fail loudly. Upgrade standalone Drivers with the Core, and say so in the notes of the
release that ends the hold. PR #4493 would reject a lagging Driver, but this RFC does
not depend on it. Every path that carries a config keeps the `Float` text form; check
C++ `to_json`.

| Phase | Scope                                                 | Touches                                                                                                                   | Depends on         |
| ----- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| 1     | Add the `payload` schema with today's shared types    | `schemas/x`, `x/go/payload`, `x/ts/src/payload`, `x/py/x/payload`, `x/cpp/payload`, `oracle/analyzer`                     | -                  |
| 2     | Point Kafka, MQTT, and HTTP v4 at `payload`           | `schemas/synnax`, `core/pkg/service`, `client`, `console/src/platform/task/payload`                                       | 1                  |
| 3     | Key JSON enums by sample and keep 64-bit values exact | `x/go/json`, `x/cpp/json`                                                                                                 | -                  |
| 4     | Convert bytes as base64 in `x/go/json`                | `x/go/json`                                                                                                               | -                  |
| 5     | Pass `tx` to `Normalize` and add `ReconcileEntry`     | `core/pkg/service/task`                                                                                                   | -                  |
| 6     | Inject channels into the HTTP, Kafka, and MQTT stores | `x/go/payload`, `core/pkg/service`                                                                                        | 1, 5               |
| 7     | Retype Kafka read fields                              | `schemas/synnax/versions/kafka`, `core/pkg/service/kafka`, `console/src/feature/kafka`, `client`                          | 2, 3, 4, 6         |
| 8     | Retype Kafka write channels                           | `schemas/synnax/versions/kafka`, `core/pkg/service/kafka`, `x/go/json`, `console/src/feature/kafka`, `client`             | 2, 3, 4, 6         |
| 9     | Retype MQTT read fields                               | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`                             | 2, 3, 4, 6         |
| 10    | Retype MQTT write channel fields                      | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`                             | 8                  |
| 11    | Share a typed `WriteField` and migrate `http_write`   | `schemas`, `core/pkg/service`, `driver/http`, `console/src/platform/task/payload`, `client`                               | 8, 10              |
| 12    | Cut over HTTP read fields                             | `schemas/synnax/versions/http`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`         | 2, 3, 6            |
| 13    | Cut over the HTTP `ChannelField`                      | `schemas`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`                              | 7, 9, 11, 12       |
| 14    | Retype the HTTP task forms and delete the adapter     | `console/src/feature/http/task`                                                                                           | 12, 13             |
| 15    | Replace OPC UA array fields with a `Sampling` union   | `schemas/synnax/versions/opcua`, `core/pkg/service/opcua`, `driver/opcua`, `console/src/feature/opcua/task`, `client`     | -                  |
| 16    | Convert JSON, UUID, and bytes samples in the Driver   | `x/cpp/json`, `x/cpp/base64`, `x/cpp/telem`, `driver/opcua/telem`                                                         | -                  |
| 17    | Read JSON, UUID, and bytes channels on HTTP           | `driver/http`, `console/src/feature/http/task`                                                                            | 14, 16             |
| 18    | Write JSON, UUID, and bytes channels on HTTP          | `driver/http`, `console/src/feature/http/task`                                                                            | 14, 16             |
| 19    | Read and write raw bytes payloads on Kafka and MQTT   | `core/pkg/service/kafka`, `core/pkg/service/mqtt`                                                                         | 7, 8, 9, 10        |
| 20    | Read and write raw bytes payloads on HTTP             | `driver/http`                                                                                                             | 17, 18             |
| 21    | Add Modbus v3 string register variants                | `schemas/synnax/versions/modbus`, `core/pkg/service/modbus`, `driver/modbus`, `console/src/feature/modbus/task`, `client` | -                  |
| 22    | Remove config-only checks from the Driver             | `driver/http`                                                                                                             | 12, 13             |
| 23    | Update the task docs for typed fields                 | `site/docs/src/pages/reference/hardware`                                                                                  | 14, 15, 19, 20, 21 |
