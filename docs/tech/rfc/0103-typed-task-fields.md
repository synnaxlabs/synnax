# 103 Strongly typed task fields

- **Author**: Patrick Dotson
- **Date**: 2026-11-06
- **Related**: [RFC 0028 - HTTP driver specification](0028-http-driver.md),
  [RFC 0043 - Oracle support for struct unions](0043-oracle-struct-unions.md),
  [RFC 0044 - Oracle optionality, defaults, and input/output types](0044-oracle-optionality-defaults-mutation.md),
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0058 - Release workflow](0058-release-workflow.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [RFC 0102 - Deployment updates](0102-deployment-updates.md),
  [PR #4594 - SY-4579: Add a fill value for missing HTTP read fields](https://github.com/synnaxlabs/synnax/pull/4594),
  [PR #4493 - SY-5478: Report Driver versions and stop tasks on a mismatch](https://github.com/synnaxlabs/synnax/pull/4493)

## 0 Summary

HTTP, Kafka, and MQTT tasks describe a JSON value with flat fields: `data_type`,
`time_format`, `enum_values`, `json_type`, and an `any` value. Most mixes of these
fields mean nothing, and the Core stores them all. OPC UA `array_size` and Modbus
`string_length` have the same problem.

This RFC makes each field a union on a shared `Kind`, so an invalid mix cannot be
stored. A new X package, `payload`, holds the shared types. The Core checks each kind
against the channel on write. Every JSON integration reads and writes JSON and UUID
channels, and no schema holds `any`.

**Releases hold until Phase 14 lands.** http v4, Kafka v0, MQTT v1, and OPC UA v2 have
not shipped, so they change in place and no v5 exists. A release before Phase 14 ships
the flat shapes, and a migration is then owed for each one. Modbus v2 has shipped, so
Modbus gets a v3 and does not hold releases.

## 1 Motivation

An instrumentation engineer maps a JSON document to channels. Today a wrong field
surfaces late, and each integration handles it in a different way:

- **A wrong `data_type`**: The HTTP Driver ignores it and uses the channel. Kafka
  rejects the task at start (`core/pkg/service/kafka/read_task.go:201`).
- **A static value of the wrong type**: `json_type: number` with the value `"5"`. HTTP
  ignores `json_type` and sends the value (`driver/http/write_task.cpp:218`). MQTT
  rejects it at start. Kafka sends it.
- **Enum values on a string or timestamp field**: The conversion ignores them
  (`x/cpp/json/convert.cpp`, `convert`). No error tells the engineer.
- **A fill value**: #4594 adds `fill_value any?`. The Driver can only check its type at
  configure, against the channel.
- **OPC UA `array_size`**: It has a meaning only when `array_mode` is true.
- **Modbus `string_length`**: It has a meaning only when `data_type` is a string.

`oracle/AGENTS.md` names the fix: when the deciding fact is in the same message, split
the type. "A new client forgets a rule, but cannot forget a shape." When the fact is a
channel data type, the rule belongs in the Core on the write path.

JSON and UUID channels also have gaps. The Core decoder takes them on Kafka and MQTT
(`x/go/json/field.go`), but the Console hides them
(`console/src/platform/task/payload/DataTypeField.tsx:20`). The HTTP Driver rejects them
(`driver/http/read_task.cpp:112`).

## 2 Vocabulary

- **Kind**: The JSON role of a field: `number`, `string`, `boolean`, `timestamp`,
  `json`, or `uuid`. It is the discriminator of each field union.
- **Read field**: A value that a task takes from a document and writes to a channel.
- **Channel field**: A command channel value that a task places in a document.
- **Write field**: A value that a task adds to each document it sends. It is fixed or
  generated.
- **Fill**: The sample that a read field writes when its value is absent. A field with a
  fill is optional. A field with no fill is required.
- **Reconcile**: The correction of a field's kind from its channel.

## 3 Principles

1. **Shape over rule**: A fact in the message selects a variant. A rule is the last
   choice.
2. **The channel owns the data type**: A stored field with a channel holds no second
   data type.
3. **The Core is the guarantee**: Every client passes through the Core, so the Core
   holds each rule that needs a channel. The Console repeats a rule only for a better
   message.
4. **The Driver is the second line**: Channels drift after configure. The Driver checks
   facts about the channel, and drops checks that the shape now makes impossible.
5. **No `any`**: A JSON value in a schema is JSON text in a `string`.
6. **What shipped decides the version**: An unshipped shape changes in place. A shipped
   shape gets a new version and a migration (RFC 0058).

## 4 Design

### 4.0 Placement

| Piece                        | Layer          | Reason                                          |
| ---------------------------- | -------------- | ----------------------------------------------- |
| `Kind`, reads, `WriteField`  | `x/*/payload`  | General purpose. No Synnax resource in them.    |
| `KindOf`                     | `x/go/payload` | A pure function of `telem.DataType`.            |
| `ReadField`, `ChannelField`  | Each service   | They hold `channel.Key`, which X cannot import. |
| Kind, fill, and index checks | The Core       | They need the channel, read in `tx`.            |
| Enum and JSON conversion     | `x/*/json`     | Already the home of sample conversion.          |

The package is `payload`, not more of `x/go/json`. Oracle output there would clash with
the hand-written `TimeFormat` (`x/go/json/time.go`).

### 4.1 Shared schema

The new `schemas/x/versions/payload/v0.oracle` holds these types. Docs and `@go marshal`
tags are cut here.

```
Kind enum { number string boolean timestamp json uuid }

TimeFormat enum { iso8601 unix_sec unix_ms unix_us unix_ns }

NumberEntry  struct { label string  value float64 }
StringEntry  struct { label string  value string }
BooleanEntry struct { label string  value bool }

NumberFill union on type {
    nan {}
    value { value float64 }
    label { label string }
}

NumberRead struct {
    data_type   telem.DataType?
    enum_values NumberEntry[]
    fill        NumberFill?
}

StringRead struct {
    enum_values StringEntry[]
    fill        string?
}

BooleanRead struct {
    enum_values BooleanEntry[]
    fill        bool?
}

TimestampRead struct {
    time_format TimeFormat
    fill        telem.TimeStamp?
}

BaseWriteField struct {
    key     string = create { @key }
    pointer string = ""
}

WriteField union on type Kind extends BaseWriteField {
    number    { value float64 }
    string    { value string }
    boolean   { value bool }
    timestamp { time_format TimeFormat = iso8601 }
    json      { value string }
    uuid      {}
}
```

Each integration declares its own unions, because the base holds `channel.Key`. The
Kafka and MQTT fields differ from these HTTP ones only in the base.

```
ReadField union on type payload.Kind extends BaseReadField {
    number    extends payload.NumberRead {}
    string    extends payload.StringRead {}
    boolean   extends payload.BooleanRead {}
    timestamp extends payload.TimestampRead {}
    json      {}
    uuid      {}
}

ChannelField union on type payload.Kind extends BaseChannelField {
    number    {}
    string    { enum_values payload.NumberEntry[] }
    boolean   {}
    timestamp { time_format payload.TimeFormat }
    json      {}
    uuid      {}
}
```

- `BaseReadField` holds `key`, `name`, `disabled`, `channel`, and `pointer`.
- `BaseChannelField` holds `pointer`, `channel`, `name`, and `data_type?`.
- The enum types are three structs, not a generic. No version file instantiates a
  generic with an argument, and the generic codec falls back to JSON
  (`oracle/plugin/go/marshal/encoder.go:195`).
- `JSONType`, `GeneratorType`, and `EnumEntry` go away in Phase 12.

A shared union cannot hold the variants. The analyzer rejects a union that extends both
a union and a struct base (`oracle/analyzer/analyzer_test.go:4041`). Each variant takes
a mixin instead. §9 records what is not yet proven about this form.

### 4.2 Kinds

`payload.KindOf` maps a data type to its kind. All checks and all corrections use it.

| Channel data type        | `KindOf`    | Read field  | Channel field                 |
| ------------------------ | ----------- | ----------- | ----------------------------- |
| Float, integer, unsigned | `number`    | `number`    | `number`, `string`, `boolean` |
| `boolean`                | `boolean`   | `boolean`   | `number`, `string`, `boolean` |
| `string`                 | `string`    | `string`    | `string`                      |
| `timestamp`              | `timestamp` | `timestamp` | `timestamp`                   |
| `json`                   | `json`      | `json`      | `json`                        |
| `uuid`                   | `uuid`      | `uuid`      | `uuid`                        |
| `bytes`                  | None        | Rejected    | Rejected                      |

A read field's kind must equal `KindOf`. A channel field's kind must equal `KindOf`, or
the channel must be numeric or boolean, because such a value can be written in each of
the three JSON forms.

A `number` field still takes a JSON boolean as 0 or 1 and an enum label, as it does
today. The kind follows the channel, not the JSON token.

`data_type` is optional. With channel 0 it states the width that a `number` cannot
state, which the Console uses to create the channel. The Core clears it at configure
(§4.5). The other kinds imply one data type.

### 4.3 Fill

A fill replaces `optional` and `fill_value` from #4594. The rule for an absent value
stays: the pointer finds nothing, the value is `null`, or the value is an empty string
on a non-string channel.

| #4594 state                    | Typed state             |
| ------------------------------ | ----------------------- |
| `optional` false               | No `fill`               |
| `optional` true, no fill       | `fill: { type: "nan" }` |
| `optional` true, a number      | `value` fill            |
| `optional` true, an enum label | `label` fill            |

`nan` is a variant because JSON cannot hold NaN. The `label` variant keeps the author's
intent, so an edit to an entry's value moves the fill with it.

The timestamp fill is a `telem.TimeStamp`, not text to parse. It holds nanoseconds, and
its JSON form is a string (`x/go/telem/versions/v0/time_stamp.go:44`). So it keeps every
digit through `msgpack.EncodedJSON`, where a JSON number would round to a `float64`. The
index rule stays: an index channel takes no fill.

`json` and `uuid` reads have no fill. A later optional field adds one with no migration.

The shared read types bring fill to Kafka and MQTT. Today Kafka skips a record with an
absent value (`core/pkg/service/kafka/read_task.go:767`). Phases 6 and 8 add the same
absent rule, and encode each fill one time at task configure.

### 4.4 JSON storage

A static `json` write field stores JSON text in a `string`. It is the recommended form.

| Option                | Result                                                              |
| --------------------- | ------------------------------------------------------------------- |
| `value string`        | **Chosen.** Any JSON value, every digit kept, no `any`.             |
| `value record`        | `msgpack.EncodedJSON`: only an object, and `map[string]any` inside. |
| `value bytes`         | Base64 in the config. Nobody can read a stored task.                |
| `msgpack.EncodedJSON` | Not an Oracle type. `record` is how a schema gets it.               |

JSON text is also what a `telem.JSONT` sample holds (`x/go/telem/data_type.go:50`).
`ValidateEntry` checks the text with `jsontext.Value`. The Core places it in the
template as a literal, and the Driver parses it at configure.

This also answers precision. `number { value float64 }` rounds past 2^53. A Core decode
of `any` already gives a `float64`, so this is not a new loss. A value that needs every
digit uses `json` with the text `12345678901234567890`.

### 4.5 Core check

The check runs in `ResolveEntry` (`core/pkg/service/task/config/config.go:105`). It has
`tx` and runs on `Write` and `WriteStrict` only, so stored records stay readable. Each
service builds a `Channels` function from `channel.Service`, as
`core/pkg/service/ethercat/service.go:147` does for slaves. It uses `SkipMissingKeys`
(`core/pkg/service/channel/retrieve.gen.go:220`).

`ResolveEntry` skips disabled fields, channel 0, and missing channels, because imports
hold foreign keys. For the other fields it does this:

- Rejects a kind that the §4.2 table does not allow.
- Rejects a `nan` fill on a non-float channel.
- Rejects a fill on an index channel.
- Rejects `enum_values` on a `string` channel field whose channel is not numeric or
  boolean.
- Clears `data_type`, in one place for all clients.

`ValidateEntry` keeps the rules that need only the config:

- A `label` fill is in the field's own entries.
- Labels are unique, and channel field entry values are unique.
- `json` text is valid JSON.

The Kafka and MQTT tasks run in the Core, so these two integrations have no Driver. The
`ch.DataType != fld.DataType` check becomes a kind check at task configure, as the
second line. RFC 0079 §4.3 said the Core rejects a `data_type` that disagrees. This RFC
replaces that rule.

### 4.6 Migration

The stored paths into a field are the Gorp migration, the envelope ladder, and
`Normalize`.

**Pure lift**: `MigrateReadField` stays pure for the ladder and uses the stored
`data_type`:

| Stored `data_type` | Result                                                 |
| ------------------ | ------------------------------------------------------ |
| Float, integer     | `number`, keeps entries, fill as in §4.3               |
| `boolean`          | `boolean`, entries become `value != 0`                 |
| `string`           | `string`, drops entries, which no conversion read      |
| `timestamp`        | `timestamp`, an unset `time_format` becomes `iso8601`  |
| `json`, `uuid`     | The same kind                                          |
| `bytes`            | `string`. The next write rejects it, as the Driver did |

`MigrateChannelField` maps `json_type`, and a stored `timestamp` data type wins, because
the Driver encodes a timestamp by `time_format` alone. A new `http_write` migration
carries it, the first that record has had.

`MigrateWriteField` maps a static value by its stored dynamic type, not by `json_type`,
because the HTTP Driver sends the value as it is. A string, boolean, or number keeps its
kind. Anything else, `null` included, becomes `json` with its marshalled text. A
generator maps to `uuid` or `timestamp`.

**Reconcile**: Stored fields can disagree with their channels, because the Driver never
read the stored `data_type`. `Reconcile` reads each field's channel, disabled fields
included, and sets the kind from the §4.2 table. A corrected field keeps its base, loses
its entries and fill, and takes a default `time_format`. Dropped entries match today:
the conversion ignored them on that channel. Reconcile also clears `data_type` and logs
one warning per corrected field. It skips channel 0 and missing channels.

**Stored paths, following `core/pkg/service/ethercat/versions/v2/resolve.go`:**

- **Boot**: `versions.ReadMigrations` becomes `NewReadMigrations(channels)`, which lifts
  each record and then runs `Reconcile`. The Channel service opens before HTTP
  (`core/pkg/service/layer.go:339`, `:660`).
- **Import and the task migration**: `Normalize` has no `tx` today. It gets one, and the
  two callers pass theirs (`core/pkg/service/task/imex.go:116`,
  `core/pkg/service/task/versions/v3/migrate.go:137`). A new `ReconcileEntry` hook on
  `config.ServiceConfig` runs in `Normalize`, for each version. Without it, the `Write`
  after `Normalize` rejects the mismatch, and the task migration quarantines the task.
- **Client writes**: These do not reconcile. A client mismatch is an error.

Kafka and MQTT have no ladder, but imports of them still pass through `ReconcileEntry`.

### 4.7 Driver

The Driver work is in HTTP. The generated `ReadField`, `ChannelField`, and `WriteField`
become `std::variant`s, and shared fields come through the generated `base`.

**Removed**, because the shape or the Core now guarantees them: time format errors,
fill-needs-optional, `json_type` errors, and enum shape checks.
`x::json::parse_time_format` stays as the map to the C++ enum.

**Kept**, as `oracle/AGENTS.md` requires: kind against `ch.data_type`, a float for NaN,
the index rule, and `check_from_sample_value`.

**Added**: `BOOLEAN_T` conversion. `number_to_numeric` has no branch for it
(`x/cpp/json/convert.cpp:167`), and `check_from_sample_value` rejects it (`:446`), so a
boolean channel fails on HTTP today.

### 4.8 JSON and UUID by integration

| Integration | Read `json` | Read `uuid` | Write `json`, `uuid` channels | Static `json` |
| ----------- | ----------- | ----------- | ----------------------------- | ------------- |
| HTTP        | Phase 16    | Phase 16    | Phase 17                      | Phase 10      |
| Kafka       | Phase 6     | Phase 6     | Phase 7                       | Phase 10      |
| MQTT        | Phase 8     | Phase 8     | Phase 9                       | Phase 10      |
| OPC UA      | No          | Exists      | Exists for `uuid`             | No            |
| Modbus      | No          | No          | No                            | No            |

**HTTP needs**, all in C++:

- `to_sample_value` gets `JSON_T` (the value's `dump`) and `UUID_T`
  (`x::uuid::UUID::parse`, stored as its 16 bytes). This follows
  `driver/opcua/telem/telem.h:36`, because a `SampleValue` has no UUID.
- `Series(const SampleValue &)` infers `STRING_T` or `UINT8_T`
  (`x/cpp/telem/series.h:703`). The read path needs an overload that takes the channel's
  data type.
- `read_task.cpp:112` drops its rejection of `UUID_T` and `JSON_T`. `BYTES_T` stays
  rejected.
- `Series::at` throws on `UUID_T` (`x/cpp/telem/series.h:1145`). The write path reads it
  through `uuids()`.
- `x::json::Type` gets `JSON` and `UUID`, with `from_sample_value` and
  `check_from_sample_value`.

**Kafka and MQTT** need only an `Encoder` change: a `JSON` type places the sample as raw
JSON, where `case !numeric(dt)` quotes it today (`x/go/json/field.go:314`). The Console
shows each kind in Phases 6 to 9.

**OPC UA** has no JSON document. It maps a `Guid` to `UUID_T` already
(`driver/opcua/telem/telem.cpp:63`), and its fields stay a `data_type`. **Modbus**
registers hold neither type. NI, LabJack, and EtherCAT have no change.

### 4.9 OPC UA

OPC UA v2 has not shipped. `core/v0.60.1` holds only v1, so v2 changes in place.

```
Sampling union on type {
    unary {}
    array { size int32 = 1 }
}

ReadConfig struct extends task.ReadConfig {
    sampling Sampling = unary
    ...
}
```

`array_mode` and `array_size` go away. `MigrateReadConfig` maps v1 `array_mode: true` to
`array`, and a v1 `array_size` with `array_mode` false drops. A union field default has
a precedent in NI: `custom_scale Scale = none`. The Driver selects its source from the
variant (`driver/opcua/factory.cpp:33`). The rule that an array task reads no structure
`field` stays a rule, because it joins the config to each channel.

### 4.10 Modbus

Modbus v2 has shipped. `core/v0.60.1` holds it with `string_length`, so Phase 18 mints
v3. The v2 file on `main` also has bit variants that `core/v0.60.1` does not. They were
added in place, they stay additive, and v3 carries them.

v3 follows `holding_register_bit`: a sibling variant, not a nested one.

```
ReadChannel union on type extends v1.BaseReadChannel {
    coil {}
    discrete_input {}
    holding_register extends RegisterValue {}
    input_register extends RegisterValue {}
    holding_register_string extends RegisterString {}
    input_register_string extends RegisterString {}
    holding_register_bit extends RegisterBit {}
    input_register_bit extends RegisterBit {}
}

RegisterString struct {
    length        int32
    bytes_swapped bool?
}
```

`words_swapped` goes from the string form, because `util::parse_string` does not read it
(`driver/modbus/channels.h:145`). `MigrateReadChannel` is pure: a `string` data type
selects the string variant, with `string_length` as `length`. Write channels hold no
strings, so `WriteChannel` stays an alias. A `RegisterValue` with a `string` data type
stays rejected in the Driver by `is_register_type`.

The check at `driver/modbus/read_task.h:198` moves to `length`. A released Core keeps
reading v1 and v2 through the ladder.

### 4.11 Console and clients

- **Field form**: The data type select sets the kind, and the width for a `number`. Each
  variant shows its own inputs. For an instrumentation engineer, a timestamp field shows
  no enum table. If we are wrong, the engineer loses a mix that worked, and §4.2 is the
  place to widen.
- **Channel creation**: `Task.configureReadChannels` takes a `dataType`. The Console
  derives it from the kind. `string` and `json` channels stay virtual
  (`console/src/platform/task/channels.ts:42`).
- **Timing field**: `isTimingField` reads `timeFormat != null`
  (`console/src/feature/http/task/Read.tsx:60`). It becomes `type === "timestamp"`.
- **Adapter**: Phase 11 flips one schema that generates all four languages, so it must
  be atomic. A Console adapter keeps the form flat and the diff near 400 lines. Phase 13
  deletes it.
- **Python**: The generated types change. `client/py/synnax/http/types.py` and
  `client/py/synnax/mqtt/types.py` and the tests and examples change with their phase.

### 4.12 Compatibility and release hold

**Release hold**: Phases 1 to 14 must land before any stable release of the Core, the
Driver, or the Console. This keeps http v4, Kafka v0, MQTT v1, and OPC UA v2 unshipped
until their shapes are final. Candidates do not count (RFC 0058). The customer need
behind #4594 is not urgent, so the hold has a low cost. A hotfix on `release/*` that
does not hold these schemas can still go out.

| Schema      | In `core/v0.60.1` | Action                 |
| ----------- | ----------------- | ---------------------- |
| http v2, v3 | Yes               | Frozen. Stay readable. |
| http v4     | No                | Edit in place          |
| Kafka v0    | No                | Edit in place          |
| MQTT v1     | No                | Edit in place          |
| OPC UA v1   | Yes               | Frozen                 |
| OPC UA v2   | No                | Edit in place          |
| Modbus v2   | Yes               | Frozen. New v3.        |

The evidence is tree comparison at `core/v0.60.1`, `core/v0.59.3`, `driver/v0.60.1`,
`console/v0.60.1`, `desktop/v1.0.4`, and the remote `release/*` heads. The remote has no
newer `core` or `driver` tag. The clone is shallow, so `git tag --contains` is not
evidence.

**#4594** merges first, as it is. This RFC replaces its fields in http v4.

**#4493** must merge before Phase 10, the first shape a Driver reads. A Driver at v0.60
on a newer Core does this without it:

- Ignores `type` on a read field.
- Fails its float parse on a string enum entry.
- Takes the default for the absent `json_type`.
- Rejects each new `WriteField` variant, because `static` is the variant it knows.
- Reads the absent `array_mode` as false on OPC UA, so an array task runs as unary with
  no error.

A stale client fails with a clear error: a field with no `type` does not decode on the
Core (`unknown type`).

**A released Core** keeps reading v2, v3, and v0.58 to v0.59 envelopes. The frozen
packages do not change. The legacy rewrite enters the ladder at its first version, and
the ladder ends in the new `Migrate` steps and `ReconcileEntry`.

**Dev Cores** that ran http v4, Kafka v0, MQTT v1, or OPC UA v2 are not owed a
migration. They reset their data.

### 4.13 Existence ledger

| Symbol                              | State  | Where                                                 |
| ----------------------------------- | ------ | ----------------------------------------------------- |
| `ResolveEntry`, `ValidateEntry`     | Exists | `core/pkg/service/task/config/config.go:105`          |
| `Store.Normalize`                   | Exists | `core/pkg/service/task/config/config.go:283`          |
| `SkipMissingKeys`                   | Exists | `core/pkg/service/channel/retrieve.gen.go:220`        |
| `Slaves`, `NewReadMigration`        | Exists | `core/pkg/service/ethercat/versions/v2/resolve.go:46` |
| `xjson.Field`, `Decoder`, `Encoder` | Exists | `x/go/json/field.go`                                  |
| Borrowed union discriminator        | Exists | `schemas/synnax/versions/arc/v4.oracle:28`            |
| Union field default                 | Exists | `schemas/synnax/ni.oracle:655`                        |
| `x::uuid::UUID::parse`              | Exists | `x/cpp/uuid/uuid.h:47`                                |
| `Series::uuids`                     | Exists | `x/cpp/telem/series.h:1041`                           |
| `payload.*`                         | NEW    | Analog: `schemas/x/versions/control`                  |
| `payload.KindOf`                    | NEW    | Analog: `numeric`, `x/go/json/number.go:344`          |
| `config.Channel`, `config.Channels` | NEW    | Analog: `v2.Slave`, `v2.Slaves`                       |
| `ServiceConfig.ReconcileEntry`      | NEW    | Analog: `ResolveEntry`                                |
| `Reconcile`, `NewReadMigrations`    | NEW    | Analog: `ResolveReadConfig`, `NewReadMigration`       |
| String and boolean `xjson` enums    | NEW    | Extends `xjson.Enum`, `x/go/json/field.go:64`         |
| `Series(SampleValue, DataType)`     | NEW    | Analog: `Series(NumericType, DataType)`               |
| `opcua.Sampling`                    | NEW    | Analog: `ni.Scale`                                    |
| `modbus.RegisterString`             | NEW    | Analog: `modbus.RegisterBit`                          |

## 5 Prior art

Telegraf and our schema take one side each. Telegraf's `json_v2` parser gives a field a
`type` of `float`, `int`, `uint`, `string`, or `bool`, and has no way to map string
values. Its Modbus input states that `length` is "ONLY valid for STRING type", and it
guards the rule in prose. We align on one declared type per value. We deviate on the
guard: a variant holds `length`, so the rule needs no prose.

## 6 Implementation phases

Before Phase 2, #4594 merges. Before Phase 10, #4493 merges. Each phase below is one PR,
and each leaves `main` green.

- Phases 1 to 3 add the types with no change to behavior. Phase 2 is mechanical and
  `review/bot`. Kafka, MQTT, and http v4 move together, because a pin that lags its
  dependency fails `oracle/check/versions_gate.go`. Frozen v2 and v3 do not change.
- Phases 4 and 5 add the plumbing that the retyping needs.
- Phases 6 to 9 retype Kafka and MQTT, one direction at a time.
- Phases 10 to 13 retype HTTP. Phase 11 is atomic and near 400 lines.
- Phase 14 retypes OPC UA.
- Phases 15 to 20 do not hold releases.

Phases 1 to 14 hold releases (§4.12). Releases resume when Phase 14 lands and #4493 is
in. The phase table format has no column for this, so this list is the mark.

| Phase | Scope                                                 | Touches                                                                                                                                                                                           | Depends on         |
| ----- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| 1     | Add the `payload` schema with today's shared types    | `schemas/x`, `x/go/payload`, `x/ts/src/payload`, `x/py/x/payload`, `x/cpp/payload`, `oracle/analyzer`                                                                                             | -                  |
| 2     | Point Kafka, MQTT, and HTTP v4 at `payload`           | `schemas/synnax`, `core/pkg/service/http`, `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `client`, `console/src/platform/task/payload`                                                       | 1                  |
| 3     | Add string and boolean enums to JSON conversion       | `x/go/json`, `x/cpp/json`                                                                                                                                                                         | -                  |
| 4     | Pass `tx` to `Normalize` and add `ReconcileEntry`     | `core/pkg/service/task/config`, `core/pkg/service/task/imex.go`, `core/pkg/service/task/versions/v3`                                                                                              | -                  |
| 5     | Inject channels into the HTTP, Kafka, and MQTT stores | `x/go/payload`, `core/pkg/service/http`, `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `core/pkg/service/layer.go`                                                                           | 1, 4               |
| 6     | Retype Kafka read fields                              | `schemas/synnax/versions/kafka`, `core/pkg/service/kafka`, `console/src/feature/kafka`, `console/src/platform/task/payload`, `client`                                                             | 2, 3, 5            |
| 7     | Retype Kafka write channels                           | `schemas/synnax/versions/kafka`, `core/pkg/service/kafka`, `x/go/json`, `console/src/feature/kafka`, `console/src/platform/task/payload`, `client`                                                | 2, 3, 5            |
| 8     | Retype MQTT read fields                               | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `console/src/platform/task/payload`, `client`                                                                | 2, 3, 5            |
| 9     | Retype MQTT write channel fields                      | `schemas/synnax/versions/mqtt`, `core/pkg/service/mqtt`, `console/src/feature/mqtt`, `console/src/platform/task/payload`, `client`                                                                | 7                  |
| 10    | Share a typed `WriteField` and migrate `http_write`   | `schemas/x/versions/payload`, `schemas/synnax/versions`, `core/pkg/service/http`, `core/pkg/service/kafka`, `core/pkg/service/mqtt`, `driver/http`, `console/src/platform/task/payload`, `client` | 5, 7, 9            |
| 11    | Cut over HTTP read fields                             | `schemas/synnax/versions/http`, `core/pkg/service/http`, `driver/http`, `x/cpp/json`, `console/src/feature/http/task`, `client`                                                                   | 2, 3, 5            |
| 12    | Cut over the HTTP `ChannelField`                      | `schemas/synnax/versions/http`, `schemas/x/versions/payload`, `core/pkg/service/http`, `driver/http`, `console/src/feature/http/task`, `client`                                                   | 10                 |
| 13    | Retype the HTTP task forms and delete the adapter     | `console/src/feature/http/task`                                                                                                                                                                   | 11, 12             |
| 14    | Replace OPC UA array fields with a `Sampling` union   | `schemas/synnax/versions/opcua`, `core/pkg/service/opcua`, `driver/opcua`, `console/src/feature/opcua/task`, `client`                                                                             | -                  |
| 15    | Convert JSON and UUID samples in the Driver           | `x/cpp/json`, `x/cpp/telem`                                                                                                                                                                       | -                  |
| 16    | Read JSON and UUID channels on HTTP                   | `driver/http`, `console/src/feature/http/task`                                                                                                                                                    | 13, 15             |
| 17    | Write JSON and UUID channels on HTTP                  | `driver/http`, `console/src/feature/http/task`                                                                                                                                                    | 13, 15             |
| 18    | Add Modbus v3 string register variants                | `schemas/synnax/versions/modbus`, `core/pkg/service/modbus`, `driver/modbus`, `console/src/feature/modbus/task`, `client`                                                                         | -                  |
| 19    | Remove config-only checks from the Driver             | `driver/http`                                                                                                                                                                                     | 11, 12             |
| 20    | Update the task docs for typed fields                 | `site/docs/src/pages/reference/hardware`                                                                                                                                                          | 13, 14, 16, 17, 18 |

## 7 What this RFC does not cover

- **OPC UA `stream_rate`**: An array task ignores it, but it comes from
  `task.ReadConfig`, which all hardware reads share. A second RFC can move it.
- **Sparkplug B entries**: They hold a `data_type` and no JSON payload.
- **OPC UA structures as JSON**: A `json` kind there is a new feature.
- **NI, LabJack, and EtherCAT**: They are unions already, and a sweep of the live
  schemas found no other `any` or conditional field in a task config.
- **`nan_fill_enabled`**: It stays a task flag for values that fail to convert.

## 8 Resolved decisions

1. **Hold releases, do not mint v5**: The alternative ships flat shapes and then pays a
   migration, and a Reconcile, for each. The cost of the hold is a delay to all
   releases. The trade is real. The need behind #4594 is not urgent, so the delay is
   cheap.
2. **`data_type` stays optional, not removed**: Removing it loses the width for a new
   channel. A separate `int32` variant per width gives ten variants that differ in a
   name. The cost is one field that exists only before configure.
3. **JSON text in a `string`**: See §4.4. The cost is one parse at configure, and a
   schema that says `string` for a value with more structure.
4. **Every integration that holds a JSON document takes JSON and UUID**: The cost is
   four C++ changes on HTTP.
5. **The package is `payload`**: `x/go/json` clashes on `TimeFormat`, and the types
   describe a document's fields, not JSON in general.
6. **A timestamp fill is a `telem.TimeStamp`**: A string to parse moves an error to
   configure and asks each reader for a time format.
7. **A kind mismatch on import is corrected, not failed**: A file from another Core must
   load. The trade is real: a foreign key can match an unrelated local channel, and
   Reconcile then changes that field to the local kind. The warning names it.
8. **One `Kind` borrowed by all three unions**: Separate `<Union>Type` enums repeat six
   names per integration, and `KindOf` could not compare to them.
9. **`ChannelField` has six variants**: With four, a JSON channel is written as a quoted
   string, and `Kind` cannot be borrowed.
10. **Static write values lift by stored type**: `json_type` was not what the HTTP
    Driver sent, so a lift by `json_type` changes a running task's output.
11. **Separate entry types, not a generic**: See §4.1.
12. **A sibling Modbus variant, not a nested union**: `holding_register_bit` set the
    style, and `driver/modbus` already branches per variant.

## 9 Open questions

These items are not verified. No Go ran for this RFC, so all Oracle claims come from
reading the code.

- **Cross-namespace variant mixin**: The grammar takes a qualified name in a variant
  `extends`, and `collectInlineVariantDecl` treats it as a struct-level `extends` does.
  No schema or test uses one. Phase 1 adds the test. The fallback is a named variant:
  `number payload.NumberRead`.
- **Cross-namespace borrowed discriminator**: The grammar allows it, and no schema uses
  one. Phase 1 adds the test. The fallback is one `Kind` per integration.
- **Generated `Validate` depth**: `ReadEndpoint.Validate` does not reach its fields
  today. Phase 6 confirms that a bare enum in a variant reaches `IsValid`.
- **Field paths in Console errors**: A Core rejection reaches the field only after
  SY-5385. Until then it shows at the task level.
- **`value` fill range**: A `value` that does not fit the channel stays a task configure
  error. `checkFits` could move it to `ResolveEntry`.
- **PR #4504** changes `core/pkg/service/task/config/config.go`. Phase 4 merges with it.
