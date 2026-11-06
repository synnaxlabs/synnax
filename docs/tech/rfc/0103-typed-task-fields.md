# 103 Strongly typed task fields

- **Author**: Patrick Dotson
- **Date**: 2026-11-06
- **Related**: [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [PR #4594 - SY-4579: Add a fill value for missing HTTP read fields](https://github.com/synnaxlabs/synnax/pull/4594)

## 0 Summary

HTTP, Kafka, and MQTT tasks describe a JSON value with flat fields and an `any` value.
Most mixes mean nothing, the Core stores them all, and each runtime handles them
differently. Numbers are `float64`, so they lose digits past 2^53 on 64-bit channels.
HTTP rejects JSON, UUID, and bytes channels. OPC UA `array_size` and Modbus
`string_length` have the same shape problem.

Each field becomes a union on a shared `Kind`, in a new X package, `payload`. The Core
checks each kind against the channel on write. No schema holds `any`.

**Releases hold until Phase 15 lands.** http v4, Kafka v0, MQTT v1, and OPC UA v2 are
not in `core/v0.60.1`, so they change in place, with no v5. A release before Phase 15
ships the flat shapes and owes a migration for each. Modbus v2 shipped, so Modbus gets a
v3 and does not hold releases.

## 1 Schema

`schemas/x/versions/payload/v0.oracle`, with docs and tags cut:

```
Kind enum { float integer unsigned string boolean timestamp json uuid bytes }
TimeFormat enum { iso8601 unix_sec unix_ms unix_us unix_ns }
Integer  int64  { @ts type "string"  @py hand }
Unsigned uint64 { @ts type "string"  @py hand }
FloatEntry    struct { label string  value float64 }
IntegerEntry  struct { label string  value Integer }
UnsignedEntry struct { label string  value Unsigned }
StringEntry   struct { label string  value string }
BooleanEntry  struct { label string  value bool }
FloatFill    union on type { nan {}  value { value float64 }   label { label string } }
IntegerFill  union on type { value { value Integer }   label { label string } }
UnsignedFill union on type { value { value Unsigned }  label { label string } }
FloatRead     struct { data_type telem.DataType?  enum_values FloatEntry[]  fill FloatFill? }
IntegerRead   struct { data_type telem.DataType?  enum_values IntegerEntry[]  fill IntegerFill? }
UnsignedRead  struct { data_type telem.DataType?  enum_values UnsignedEntry[]  fill UnsignedFill? }
StringRead    struct { enum_values StringEntry[]  fill string? }
BooleanRead   struct { enum_values BooleanEntry[]  fill bool? }
TimestampRead struct { time_format TimeFormat  fill telem.TimeStamp? }
WriteField union on type Kind extends BaseWriteField {
    float { value float64 }  integer { value Integer }  unsigned { value Unsigned }
    string { value string }  boolean { value bool }
    timestamp { time_format TimeFormat = iso8601 }
    json { value string }  uuid {}  bytes { value string }
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
    timestamp { time_format payload.TimeFormat }
    string {}  boolean {}  json {}  uuid {}  bytes {}
}
```

`BaseReadField` holds `key`, `name`, `disabled`, `channel`, and `pointer`.
`BaseChannelField` holds `pointer`, `channel`, `name`, and `data_type?`. A numeric
channel field with `enum_values` writes a value's label. `string` writes the value as
text, and `boolean` as true or false.

OPC UA v2 replaces `array_mode` and `array_size`. Modbus v3 replaces `string_length`,
and its `WriteChannel` stays an alias.

```
Sampling union on type { unary {}  array { size int32 = 1 } }
ReadConfig struct extends task.ReadConfig { sampling Sampling = unary ... }
RegisterString struct { length int32  bytes_swapped bool? }
ReadChannel union on type extends v1.BaseReadChannel {
    ...  holding_register_string extends RegisterString {}
    input_register_string extends RegisterString {}
}
```

## 2 Integers

A raw `int64` does not survive: each config passes through `msgpack.EncodedJSON`, a
`map[string]any`, where a JSON number becomes a `float64`. So `Integer` and `Unsigned`
are distinct types with the JSON string form of `telem.TimeStamp` and `telem.Alignment`
(`MarshalStringInt64To`, `MarshalStringUint64To`). A negative `uint64` and a value above
`int64` range are shape errors, and `Unsigned` holds up to 2^64 - 1 at every layer.
`nan` exists only on `FloatFill`.

Go uses `int64` and `uint64` with hand methods in `x/go/payload`. C++ `Parser` reads
digits from a string (`json.h:591`). TypeScript holds a decimal `string`, so forms hold
no `bigint`, and the Console uses a text input that checks digits, then the range with
`BigInt`. Python uses an `int` subclass that dumps a string, as `TimeStamp` does.

JSON numbers parse exactly today in Go and C++. Phase 3 fixes the three inexact places:
`xjson.Enum.Value` and `x::json::EnumMap` hold a float, `Encoder.Append` compares enums
through `n.float()`, and `x::json::from_timestamp` goes through `double`. Each entry
becomes one encoded sample, compared exactly.

## 3 Bytes

With a pointer, bytes are standard base64 text in the document, as `AppendSampleText`
writes today. With an empty pointer, the payload is raw and is not JSON. A raw read
field is the only enabled field of its endpoint, record, or entry. A raw channel field
is the whole payload, and on Kafka this is an empty `value_pointer`.

- **`x/go/json`**: `NewDecoder` rejects `BytesT`, and `Encoder` quotes raw bytes.
- **Kafka**: `read_task.go:159` and `write_task.go:261` reject an empty pointer.
- **MQTT and HTTP**: A raw field skips `readTopic.extract` and the body parse.
  `driver/http/read_task.cpp:112` drops its rejection.
- **`x/cpp`**: `to_sample_value` needs `BYTES_T`, and `x::base64::decode` needs a strict
  form. `Series(string, DataType)` and `at<std::string>` reject `BYTES_T`.
  `variable_samples` (`driver/opcua/telem/telem.cpp:99`) moves to `Series`.

OPC UA has bytes and UUID through `data_type`. Modbus, NI, LabJack, and EtherCAT stay.

## 4 Kinds

`payload.KindOf` maps a data type to its kind. Each check and correction uses it.

| Channel data type | `KindOf`, read field | Channel field                   |
| ----------------- | -------------------- | ------------------------------- |
| Float             | `float`              | `float`, `string`, `boolean`    |
| Signed integer    | `integer`            | `integer`, `string`, `boolean`  |
| Unsigned integer  | `unsigned`           | `unsigned`, `string`, `boolean` |
| `boolean`         | `boolean`            | `boolean`, `unsigned`, `string` |
| Other             | Same name            | Same name                       |

A numeric kind still takes a JSON boolean as 0 or 1, and an enum label. `data_type` is
optional: with channel 0 it states the width for a new channel. A field with no fill is
required. The absent rule stays as #4594 has it: no value at the pointer, `null`, or an
empty string on a non-string channel. `json`, `uuid`, and `bytes` reads take no fill.

## 5 Rules by layer

**Core write**, in `ResolveEntry` (`task/config/config.go:105`), through a `Channels`
function that each service builds, as `slavesOf` does. It skips disabled fields, channel
0, and missing channels, because imports hold foreign keys. It rejects a kind that §4
does not allow, a fill or entry that does not fit the channel's width, and a fill on an
index channel. It clears `data_type`.

**Core config**, in `ValidateEntry`: A `label` fill is in the field's own entries.
Labels and channel field values are unique. `json` text and `bytes` text are valid. An
empty pointer needs `bytes` and no other enabled field.

**Kafka and MQTT** run in the Core. The `ch.DataType` check at task configure becomes a
kind check. Both gain fill: Kafka skips an absent value today (`read_task.go:767`).

**Driver**: It keeps the kind against `ch.data_type`, a float for `nan`, the index rule,
and `check_from_sample_value`, because channels drift. It removes time format errors,
fill-needs-optional, `json_type` errors, and enum shape checks. It adds `BOOLEAN_T`,
which the conversions reject today. JSON and UUID samples are strings
(`driver/opcua/telem/telem.h:36`), so a new `Series(SampleValue, DataType)` keeps the
channel's type. `Series::at` throws on `UUID_T`, so the write path reads `uuids()`.

**Console**: The data type select sets the kind, and each variant shows its own inputs.
`DataTypeField` stops hiding JSON, UUID, and bytes. `Task.configureReadChannels` derives
`dataType` from the kind. Phase 12 is atomic across four languages, so a Console adapter
keeps it near 400 lines, and Phase 14 deletes the adapter.

## 6 Migration

`MigrateReadField` is pure, for the envelope ladder, and uses the stored `data_type`.
Float stays `float`. Signed and unsigned become `integer` and `unsigned`. `boolean`
entries become `value != 0`. `string` drops its entries, as no conversion read them.
`timestamp` with no `time_format` takes `iso8601`. JSON, UUID, and bytes keep their
kind.

- **Entries**: A `float64` entry converts when it is whole and fits the kind. Others
  drop with a warning, as they failed at run time. Digits already lost past 2^53 do not
  return.
- **#4594 fill**: `optional` with no fill is `nan`. A number is `value`, and an enum
  label is `label`.
- **Channel field**: A stored `timestamp` data type wins. `json_type: string` with
  entries, or `number`, becomes the numeric kind. `string` with no entries stays
  `string`.
- **Static write field**: Lift by the stored Go type, not `json_type`, because the HTTP
  Driver sends the value as it is. A whole `float64` within 2^53 becomes `integer`, and
  other numbers become `float`. An `int64` or `uint64` keeps its value. A string or
  boolean keeps its kind. Others, `null` included, become `json` text. A generator
  becomes `uuid` or `timestamp`.

`Reconcile` corrects a field from its channel, because the Driver never read the stored
`data_type`, and the HTTP default is `float64`. It sets the kind from §4 and clears
`data_type`. Between numeric kinds it converts entries and fills, by the same rule. For
other kinds it drops them. It logs one warning per corrected field, which also names a
foreign key that matched an unrelated local channel.

- **Boot**: `versions.ReadMigrations` becomes `NewReadMigrations(channels)`, as
  `ethercat/versions/v2/resolve.go` does. The Channel service opens first
  (`layer.go:339`). `http_write` gets its first migration.
- **Import and the task migration**: `Normalize` gets `tx` (`task/imex.go:116`,
  `task/versions/v3/migrate.go:137`). A new `ReconcileEntry` hook runs in it at each
  version. Without it, the `Write` that follows rejects a mismatch, and the task
  migration quarantines the task. Client writes do not reconcile.

OPC UA v1 `array_mode: true` lifts to `array`. A Modbus `string` data type lifts to the
string variant. A released Core keeps reading v2, v3, and v0.58 to v0.59 envelopes,
because frozen packages do not change. Dev Cores are not owed a migration.

## 7 Implementation phases

#4594 merges before Phase 2. Each phase is one PR. Phases 1 to 15 hold releases, and the
others do not.

No schema or test uses a qualified variant `extends` or borrowed discriminator. Phase 1
tests both, and the fallbacks are `integer payload.IntegerRead` and one `Kind` per
integration. Phase 2 moves Kafka, MQTT, and http v4 together, because a lagging pin
fails `oracle/check/versions_gate.go`.

| Phase | Scope                                                 | Touches                                                                                                               | Depends on         |
| ----- | ----------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- | ------------------ |
| 1     | Add the `payload` schema with today's shared types    | `schemas/x`, `x/go/payload`, `x/ts/src/payload`, `x/py/x/payload`, `x/cpp/payload`, `oracle/analyzer`                 | -                  |
| 2     | Point Kafka, MQTT, and HTTP v4 at `payload`           | `schemas/synnax`, `core/pkg/service`, `client`, `console/src/platform/task/payload`                                   | 1                  |
| 3     | Key JSON enums by sample and keep 64-bit values exact | `x/go/json`, `x/cpp/json`                                                                                             | -                  |
| 4     | Convert bytes as base64 in `x/go/json`                | `x/go/json`                                                                                                           | -                  |
| 5     | Pass `tx` to `Normalize` and add `ReconcileEntry`     | `core/pkg/service/task`                                                                                               | -                  |
| 6     | Inject channels into the HTTP, Kafka, and MQTT stores | `x/go/payload`, `core/pkg/service`                                                                                    | 1, 5               |
| 7     | Retype Kafka read fields                              | `schemas/synnax/versions/kafka`, `core/pkg/service/kafka`, `console/src/feature/kafka`, `client`                      | 2, 3, 4, 6         |
| 8     | Retype Kafka write channels                           | `schemas/synnax/versions/kafka`, `core/pkg/service/kafka`, `x/go/json`, `console/src/feature/kafka`, `client`         | 2, 3, 4, 6         |
| 9     | Retype MQTT read fields                               | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`                         | 2, 3, 4, 6         |
| 10    | Retype MQTT write channel fields                      | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `client`                         | 8                  |
| 11    | Share a typed `WriteField` and migrate `http_write`   | `schemas`, `core/pkg/service`, `driver/http`, `console/src/platform/task/payload`, `client`                           | 8, 10              |
| 12    | Cut over HTTP read fields                             | `schemas/synnax/versions/http`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`     | 2, 3, 6            |
| 13    | Cut over the HTTP `ChannelField`                      | `schemas`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`                          | 11                 |
| 14    | Retype the HTTP task forms and delete the adapter     | `console/src/feature/http/task`                                                                                       | 12, 13             |
| 15    | Replace OPC UA array fields with a `Sampling` union   | `schemas/synnax/versions/opcua`, `core/pkg/service/opcua`, `driver/opcua`, `console/src/feature/opcua/task`, `client` | -                  |
| 16    | Convert JSON, UUID, and bytes samples in the Driver   | `x/cpp/json`, `x/cpp/base64`, `x/cpp/telem`, `driver/opcua/telem`                                                     | -                  |
| 17    | Read JSON, UUID, and bytes channels on HTTP           | `driver/http`, `console/src/feature/http/task`                                                                        | 14, 16             |
| 18    | Write JSON, UUID, and bytes channels on HTTP          | `driver/http`, `console/src/feature/http/task`                                                                        | 14, 16             |
| 19    | Read and write raw bytes payloads on Kafka and MQTT   | `core/pkg/service/kafka`, `core/pkg/service/mqtt`                                                                     | 8, 10              |
| 20    | Read and write raw bytes payloads on HTTP             | `driver/http`                                                                                                         | 17, 18             |
| 21    | Add Modbus v3 string register variants                | `schemas/synnax/versions/modbus`, `core/pkg/service/modbus`, `driver/modbus`, `console/src/feature/modbus/task`       | -                  |
| 22    | Remove config-only checks from the Driver             | `driver/http`                                                                                                         | 12, 13             |
| 23    | Update the task docs for typed fields                 | `site/docs/src/pages/reference/hardware`                                                                              | 14, 15, 19, 20, 21 |
