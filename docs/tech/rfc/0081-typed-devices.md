# 81 Typed devices

- **Author**: Patrick Dotson
- **Date**: 2026-10-15
- **Related**: [RFC 0017 - General purpose device drivers](0017-drivers.md),
  [RFC 0034 - Gorp indexes](0034-gorp-indexes.md),
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md),
  [RFC 0091 - Modbus driver](0091-modbus-driver.md)

## 0 Summary

The generic `device` table goes away. Each integration defines its device resources in
its own Oracle schema: typed fields in place of the `properties` blob, its own Gorp
table, ontology type, migrations, and endpoints, owned by that integration's Core
service as concrete code. Every device gets a UUID key; discovered hardware carries a
`serial`. The scanner and the user write disjoint field groups. NI chassis and EtherCAT
masters become resources.

## 1 Motivation

A device today is `key`, `rack`, `location`, `make`, `model`, `name`, `configured`, and
a `properties` blob the Core never reads
(`core/pkg/service/device/versions/v1/types.gen.go:55`). The blob is typed by hand three
times, and the copies disagree:

- The Console: a Zod schema per make under `console/src/feature/*/device/types.ts`, plus
  property migrators for OPC UA and HTTP. No other client migrates data the Core
  persists.
- The Driver: C++ structs for four makes (`driver/opcua/device/device.h`,
  `driver/modbus/device/device.h:141`, `driver/http/device/device.h`,
  `driver/ethercat/slave/slave.h:72`). NI and LabJack read nothing from the blob.
- The Python client: an untyped dict in each per-make `Device` subclass.

Two writers share one record. The scanner upserts the whole device after a shallow merge
against its own cache (`driver/common/scan_task.h:40-51,506-519`). The Console writes
identifiers, index keys, and channel maps through the same `create`. A stale scanner
cache overwrites a Console edit; EtherCAT guards `enabled` against exactly that
(`driver/ethercat/scan_task.cpp:128-131`).

The generic fields mean something different per make. `location` is a DAQmx alias for
NI, `USB` or `TCP` for LabJack, `host:port` for Modbus, an endpoint for OPC UA, a host
for HTTP, and `<master>.Slot <n>` for EtherCAT. `model` is a free string, a three-value
enum, or a constant. `make` drifts: the Driver filters Modbus servers by `modbus`, the
Console and Python write `Modbus`, so Console-created Modbus servers never get a health
check (SY-5129).

Every consumer dispatches on `make` before it can act:
`console/src/platform/device/Select.tsx:52-64`, the registry in
`console/src/feature/device/make.tsx`, the tree item at
`console/src/feature/device/tree.tsx:56,99`, and the Driver's scan cache
(`driver/common/scan_task.h:140-147`). The abstraction hides nothing.

### 1.1 Why the task shape does not transfer

Tasks keep a generic row with a per-type config table (RFC 0051), and the obvious move
is to shape devices the same way. The task row works because every field on it is true
of every task. `rack`, `name`, `type`, `config_hash`, `internal`, `snapshot`, and a
status with `running` and `cmd` describe a Modbus read task and an Arc task alike
(`schemas/synnax/task.oracle:61-127`). Every task shares one lifecycle: the Driver
starts, stops, and reconfigures each through the same commands, and the Console lists
tasks of every type with the same columns. The config table holds only what differs.

The device row has no such core. Three of its seven fields are true of every resource:
`key`, `rack`, and `name`. The other four hold a different meaning per make, or none:

| Resource       | `location`          | `model`                | `configured`     |
| -------------- | ------------------- | ---------------------- | ---------------- |
| NI device      | DAQmx alias         | product name           | user has set it  |
| NI chassis     | DAQmx alias         | product name           | never meaningful |
| LabJack device | `USB` or `TCP`      | `T4`, `T7`, `T8`       | user has set it  |
| EtherCAT slave | `<master>.Slot <n>` | always `slave`         | user has set it  |
| OPC UA server  | endpoint URL        | always `OPC UA server` | always `true`    |
| Modbus server  | `host:port`         | always `Modbus`        | always `true`    |
| HTTP server    | host                | always `HTTP server`   | always `true`    |

`location` holds six different things, and only NI's matches its doc string, "the
physical location or address of the device"; the three connect forms copy a connection
field into it (`console/src/feature/opcua/device/useConnectModal.tsx:63`,
`modbus/device/useConnectModal.tsx:57`, `http/device/useConnectModal.tsx:80`). `model`
is a constant for four of the seven (`driver/ethercat/ethercat.h:36`). `configured`
cannot be false for a server and means nothing for a chassis, which has no channels to
name. `make` is a dispatch tag; no reader uses the row without switching on it first. A
base row kept under the task shape would carry three real fields and four that each
integration fills with a placeholder, and every reader would join two tables to render
one thing.

The remaining case for the row is listing devices across makes. Two screens do that
today. The Devices group in the resource tree renders ontology resources and reads
`make` off each to choose its icon and menu
(`console/src/feature/device/tree.tsx:56,99`); under §4.4 it lists the rack's children
and needs no device table. The device select always narrows to one make before it lists
(`console/src/platform/device/Select.tsx:40,50`), so it is a per-resource select
already. No screen shows a server beside a slave with a shared column set, because there
is no shared column: a server has no location and a slave has no model. Status is the
one shared behavior, and it stays in the status service (§4.2).

## 2 Vocabulary

- **Resource**: One typed record for one piece of hardware or one server, in a table
  that belongs to one integration.
- **Discovered resource**: A resource the scanner creates: NI chassis and devices,
  LabJack devices, EtherCAT masters and slaves.
- **Connected resource**: A resource a user creates from a connect form: OPC UA, Modbus,
  and HTTP servers.
- **Identity group**: The fields of a discovered resource that the scanner owns: rack,
  serial, model, alias, simulation flag, bus topology.
- **Configuration group**: The fields a user owns: name, identifier, index channel keys,
  channel maps, enabled flags, connection settings.
- **Identifier**: The prefix a user chooses for a discovered resource's channel names. A
  discovered resource with an empty identifier is unconfigured.
- **Scanner**: An integration's scan task in the Driver. It discovers hardware, reports
  health, and answers test-connection commands.

## 3 Principles

1. **One resource type per kind of hardware**: A chassis, a module, a master, a slave,
   and a server have different fields. Each gets its own struct, table, and ontology
   type. No base struct, no union.
2. **Oracle is the only source of a device shape**: Go, TypeScript, Python, and C++
   consume generated types. Hand-written Zod, C++ structs, and Python dicts are deleted.
3. **The integration owns its devices**: The Go service that owns an integration's task
   config stores owns its device tables, endpoints, and migrations
   (`core/pkg/service/ni/service.go:69-176` is the shape).
4. **Repeat, then deduplicate across every resource**: Device services repeat the writer
   code that racks and schematics also carry. A later project removes that overlap for
   all resources, not for devices alone.
5. **One writer per field**: The scanner writes the identity group and the user writes
   the configuration group. Neither write path touches the other group.
6. **Core-minted keys**: Every key is a UUID the Core creates. Hardware identity is an
   indexed field, never a key.
7. **Delete refuses, then cascades**: A resource a task references cannot be deleted. A
   deleted parent takes its children.

## 4 Design

### 4.0 Resources

| Integration | Resource          | Ontology type     | Class      |
| ----------- | ----------------- | ----------------- | ---------- |
| NI          | `ni.Chassis`      | `ni_chassis`      | discovered |
| NI          | `ni.Device`       | `ni_device`       | discovered |
| LabJack     | `labjack.Device`  | `labjack_device`  | discovered |
| EtherCAT    | `ethercat.Master` | `ethercat_master` | discovered |
| EtherCAT    | `ethercat.Slave`  | `ethercat_slave`  | discovered |
| OPC UA      | `opcua.Server`    | `opcua_server`    | connected  |
| Modbus      | `modbus.Server`   | `modbus_server`   | connected  |
| HTTP        | `http.Server`     | `http_server`     | connected  |

An `ni.Device` is a module in a chassis or a standalone USB or PCIe device; the two
differ only in whether `chassis` is set. A chassis carries identity only: no task
references it, so it has no identifier and no channel maps.

### 4.1 Schemas

Each resource is a struct in its own `.oracle` file with its own package, such as
`ni_device.oracle` and `core/pkg/service/ni/device`, because Oracle allows one
`@retrieve` per output. Each file has `@go`, `@ts`, `@py`, and `@cpp` outputs. Every
struct carries `@create`, `@retrieve`, `@go marshal`, `@go migrate`, and
`@ontology type`, and declares `key uuid = create` with `@key`, `rack rack.Key` with
`@filter`, `name string` with `@filter` and `@search`, and a `status` of its own
`Status` type with `@go marshal omit`. A base struct would carry nothing else, so the
three fields repeat.

Gorp names a table by its Go type, so each resource sets `CustomTypeName`. Otherwise
`ni.Device` shares a table with the legacy `Device`.

The NI resources are the pattern for every discovered make:

```
Chassis struct {
    key       Key = create        { @key }
    rack      rack.Key            { @filter }
    name      string              { @filter @search }
    serial    string              { @index lookup }
    model     string              { @filter }
    alias     string
    simulated bool = false
    status    Status?             { @go marshal omit }

    @create
    @go marshal
    @go migrate
    @ontology type "ni_chassis"
    @retrieve
}

Device struct {
    key        Key = create       { @key }
    rack       rack.Key           { @filter }
    name       string             { @filter @search }
    serial     string             { @index lookup }
    model      string             { @filter }
    alias      string
    simulated  bool = false
    chassis    Chassis.Key?
    identifier string = ""
    analog_input    AnalogInputPorts
    analog_output   AnalogOutputPorts
    counter_input   CounterInputPorts
    digital_input   DigitalPorts
    digital_output  DigitalPorts
    status     Status?            { @go marshal omit }
    ...
}
```

`rack` and `serial` through `chassis` are the identity group. `name`, `identifier`, and
the port maps are the configuration group. The port maps are the typed form of today's
`analogInput.channels` and friends (`console/src/feature/ni/device/types.ts:18-63`).

Field groups per make:

- **LabJack**: identity `serial`, `model` (the existing `LJM_dtT4`, `T7`, `T8` enum),
  `connection` (USB or TCP); configuration `identifier`, `read_index`,
  `thermocouple_index`, `write_state_index`, four port channel maps.
- **EtherCAT master**: identity `interface`, `slaves Slave.Key[]` in bus order.
- **EtherCAT slave**: identity `serial`, `vendor_id`, `product_code`, `revision`,
  `master Master.Key`, `pdos`; configuration `identifier`, `enabled`, `read_index`,
  `write_state_index`, input and output channel maps.
- **OPC UA server**: `connection` (endpoint, credentials, security mode and policy,
  certificates), `read_indexes`, node channel maps
  (`console/src/feature/opcua/device/types/v1.ts:18-30`).
- **Modbus server**: `connection` and `requests` (RFC 0091 §2.0), `read_index`, register
  channel maps. Its `StatusDetails` hold the request counts as `stats` (RFC 0091 §2.7).
- **HTTP server**: `host`, `secure`, `verify_ssl`, `timeout`, `auth` (the existing
  union), `max_concurrent_requests`, `health_check`, per-path read and write channel
  maps (`console/src/feature/http/device/types.ts:224-239`). The `base_url` the Driver
  injects from `location` (`driver/http/device/device.cpp:100-106`) is derived from
  `host` and `secure` in generated code.

A connected resource has no identity group: the user owns every field, `rack` included,
and sets them in the connect form. `configured` does not exist. A discovered resource is
unconfigured while `identifier` is empty.

Task configs reference a resource by its typed key: `ni.Device.Key` on NI channels
(`schemas/synnax/ni.oracle:489,941,1389`), `labjack.Device.Key` on `labjack.oracle:216`,
and so on. `task.WriteConfig.device device.Key` (`schemas/synnax/task.oracle:197`) moves
into each integration's write config under its typed key.

### 4.2 Core services

Each integration service owns its resources beside its config stores: the NI service
gains `Chassis` and `Devices`, the EtherCAT service `Masters` and `Slaves`. Each
resource has a concrete writer and generated retrieve, in the shape of
`core/pkg/service/rack`. No generic device store exists.

Opening a resource opens its Gorp table with its migration chain and Oracle-generated
indexes, registers its ontology service, and publishes `sy_<type>_set` and
`sy_<type>_delete` through `signals.PublishFromGorp`. Its writer exposes `Create`,
`Update`, and `Delete`. `Create` writes the row, an initial unknown status, the ontology
resource, and the `parent_of` edge to its parent key, or to the rack when there is no
parent. `Update` takes a function over the stored row, so a caller changes one field
group and leaves the other as stored. `Delete` refuses with a `validate.ErrValidation`
wrap while a task references the resource, then deletes children, status, resource, and
row in one transaction.

The referrer check for `ni.Device` scans the five NI config stores for a channel whose
`device` is the key. Both tables belong to one service, so no cross-service query
exists.

Status stays in the status service under `<type>:<key>`. Each resource defines its own
`Status` and `StatusDetails`; a server reports different fields from a slave.

### 4.3 Writes

The scanner creates a resource it has not seen with both groups filled: the identity
group from the hardware, and the configuration group with today's defaults (`name` is
`NI <model>` or `<type>-<last4 of serial>`, `identifier` is empty). After creation the
scanner writes only the identity group, through `Update`. When hardware appears on
another rack, the scanner moves it; `name` stays as the user left it. The user's write
path is `Update` over the configuration group. The shallow merge in
`driver/common/scan_task.h:40-51` and the EtherCAT `enabled` guard are deleted.

Each scan matches found hardware to stored rows by a per-resource rule:

- **NI chassis, NI device, LabJack device**: `serial` through the lookup index. A vendor
  serial is unique within its make. A simulated NI resource has no serial, so the
  scanner stores its NI-MAX resource name in `serial`, as it does for the key today
  (`driver/ni/scan_task.cpp:49-64,106`).
- **EtherCAT slave**: `serial` through the lookup index, then `vendor_id` and
  `product_code` must match; today's key already carries all three
  (`driver/ethercat/scan_task.cpp:190-201`). A slave with serial zero matches by
  `master.slaves[i]`, `vendor_id`, and `product_code`.
- **EtherCAT master**: `interface` on the rack.

A resource the scanner no longer sees gets a warning status, as today. The scanner never
deletes.

### 4.4 Hierarchy

`ni.Device.chassis` and `ethercat.Slave.master` are stored keys. The service defines the
`parent_of` edge from them on every write, so the tree shows modules under their chassis
and slaves under their master through ontology children, as now. A resource with no
parent key is a child of its rack. When Gorp references land, each key becomes
`@ref on_delete cascade acyclic` with no data change.

The master holds an ordered `slaves Slave.Key[]`. A slave's position is its index, so
two slaves cannot share one, and a scan writes the bus order as one update of the master
row. The slave stores no position. Topology validation
(`driver/ethercat/topology/topology.h:27-58`) reads the master's list. The `network`
string and the `<master>.Slot <n>` location encoding are deleted.

### 4.5 API and clients

Each resource gets routes under its integration: `/ni/device/create`,
`/ni/device/retrieve`, `/ni/device/update`, `/ni/device/delete`, and the same for
`/ni/chassis`, `/ethercat/master`, `/opcua/server`, and the rest. `update` is one route
that dispatches on a union over the identity and configuration groups, so one request
writes one group. Retrieve filters are the struct's `@filter` fields plus `search_term`,
`include_status`, and `include_parent`. The generic `/device/*` routes are deleted.

Clients expose one namespace per resource: `client.ni.devices`, `client.ni.chassis`,
`client.labjack.devices`, `client.ethercat.masters`, `client.ethercat.slaves`,
`client.opcua.servers`, `client.modbus.servers`, `client.http.servers`. Retrieve returns
the typed struct. The generic `client.devices`, Python `sy.Device`, and
`synnax::device::Client` are deleted. The Python per-make classes keep their keyword
constructors as thin wrappers over the generated types.

The cache table is the TS client's, not Pluto's
(`client/ts/src/device/client.ts:184-218`), and each resource client owns its own, as
every client resource does. Pluto's `device` module exists to bind Flux to
`client.devices` and to re-parse untyped cached records through a make's Zod schemas
(`pluto/src/device/queries.ts:46-110`). Both jobs end: a resource client returns typed
records, and each integration binds `Flux.createRetrieve`, `createList`, and
`createForm` to its own client in the Console, as EtherCAT already does
(`console/src/feature/ethercat/device/queries.ts`). `StatusIndicator` reads only
`variant` and `message`, so it becomes a status-level component. The module is deleted
in Phase 23.

### 4.6 Driver

`common::ScanTask` becomes a template over the resource type; `Scanner` returns
`std::vector<D>` and `ClusterAPI` is parameterized the same way
(`driver/common/scan_task.h:81-130`). Read and write tasks take the generated struct in
place of `parser.field` reads. NI keeps `alias` for physical channel names. LabJack
opens by `serial` (`driver/labjack/read_task.h:535` uses the key today).

### 4.7 Console

Each integration's Console code consumes its generated types and resource client, and
registers its own tree items, context menu, link handler, and palette commands, as task
layouts do. Modules are named for the resource: `server` for OPC UA, Modbus, and HTTP,
`master` and `slave` for EtherCAT, `chassis` and `device` for NI.
`console/src/feature/device/make.tsx` loses one entry per integration moved and is
deleted with the last one. The unconfigured-device toast moves into the three discovered
makes, each on its own `sy_<type>_set`. `platform/device` keeps `Configure`,
`wrapTaskForm`, and the menu items as generics over a resource with an `identifier`. The
Console property migrators for OPC UA and HTTP are deleted; the Core's migration chain
owns both.

### 4.8 Migration

Release 0.59 shipped the single table. One ordered startup migration cuts over:

1. Read every legacy row. Match `make` case-insensitively, which absorbs the `modbus`
   and `EtherCAT` drift. Decode `properties` into the make's struct; NI rows become a
   chassis or a device by `is_chassis`.
2. Write each typed row under a fresh UUID. Persist the old-key-to-UUID map in KV. A row
   with an unknown make or undecodable properties is quarantined under
   `sy_device_quarantine/<key>` with a warning, never dropped and never fatal, as task
   v3 does (`core/pkg/service/task/versions/v3/migrate.go:128-155`). The legacy table
   holds no EtherCAT masters; each slave row carries the master key in `network` and its
   bus order in `position` (`console/src/feature/ethercat/device/types.ts:50-51`). Slave
   rows grouped by `rack` and `network` yield one `ethercat.Master` per group, `rack`
   and `interface` from the group, `slaves` ordered by `position`, and each slave's
   `master` set to the new key.
3. Re-key statuses, ontology resources and edges, and policy object IDs.
4. Each integration's task config migration rewrites its device references through the
   map.
5. Delete the legacy table and the map.

Steps 1 to 3 are one migration, step 4 is one per integration, and step 5 runs last.
They are ordered by version, so a 0.59 Core runs all of them on its first start.

Between the typed services landing and this migration landing, a development Core with
0.59 rows shows them only through the legacy path. Writes through that path in the
candidate era are not carried forward; the migration targets the 0.59 shape.

### 4.9 Access control

`ontology.ResourceType` gains the eight types. The builtin role lists that name
`ResourceTypeDevice` (`core/pkg/service/access/rbac/builtin/builtin.go:34,85,123`) name
all eight; provisioning rewrites them on the next boot. The migration expands a custom
policy object `{type: "device"}` into the eight typed objects and maps a per-key
`device:<key>` object to its typed successor.

### 4.10 Testing

Each integration service gets specs for create, identity update, configuration update,
refused delete, and cascade. The migration suite seeds 0.59-shaped rows for every make,
including one that quarantines. Driver specs build devices from generated structs.
Console specs for each make's forms and tree items run against the typed client.

## 5 Prior art

Ignition stores one `DeviceSettingsRecord` row plus one per-driver record; OPC UA DI
subtypes an abstract `DeviceType` per companion spec; Grafana, Azure IoT Hub, and Home
Assistant keep a generic row with a JSON blob. All of them load drivers at runtime, so
the base table cannot change per driver. Kubernetes alone has no generic table: every
kind is its own typed resource, and cross-kind listing is discovery plus N lists. Synnax
compiles its integrations into the Core and the Driver, the type set is closed, and no
consumer lists devices across makes, so the Kubernetes shape fits.

## 6 Implementation phases

Each phase is one pull request into `main`. Typed paths land first and serve empty
tables on an existing Core, so the integration suite, which creates fresh data, stays
green at every step. The migration brings 0.59 data forward near the end.

- **Phase 0: Modbus make.** `MAKE = "Modbus"` in the Driver (SY-5129). Independent of
  this RFC.
- **Phase 1: Legacy move.** `core/pkg/service/device` moves to
  `core/pkg/service/legacy/device` (SY-5190). The package name, proto package, and C++
  namespace stay `device`, so only import paths change. Mechanical, `review/bot`.
- **Phase 2: Ontology types and roles.** The eight `ResourceType` members, regenerated
  clients, and the builtin role lists.
- **Phases 3 to 8: Schemas and services, one integration each.** NI, LabJack, EtherCAT,
  OPC UA, Modbus, HTTP: the Oracle structs, generated types in four languages, the
  writers, and routes. Each lands dark.
- **Phases 9 to 14: Driver, one integration each.** Scanner and tasks on the typed
  client. Phase 9 also makes `ScanTask` a template.
- **Phases 15 to 20: Console, one integration each.** Forms, tree items, menus, links,
  and palette commands on the typed client. Each phase removes its make from `make.tsx`;
  Phase 20 deletes the file.
- **Phase 21: Migration.** §4.8 steps 1 to 3 and the policy rewrite. The map now exists
  for the next phase.
- **Phase 22: Task config references.** Typed device keys on every config, with the
  config migrations of §4.8 step 4 reading the map.
- **Phase 23: Delete the legacy path.** §4.8 step 5, the legacy service, `/device/*`
  routes, `client.devices`, `sy.Device`, `synnax::device::Client`, `pluto/src/device`,
  and the Console's generic device layer.
- **Phase 24: Documentation.** Device pages per integration and the Python and
  TypeScript examples.

Compatibility: a 0.59 Core upgrades through Phase 23 with every device, status,
reference, and policy carried forward. A client older than Phase 23 gets a not-found on
`/device/*` from a Core at Phase 23; clients and the Core ship in one train.

## 7 What this RFC does not cover

- Device import and export. The typed structs are shaped for a later `@go imex`.
- A secret store for server credentials.
- Gorp references and the ontology's replacement; the parent keys become `@ref` fields
  when that lands.
- Kafka clusters (RFC 0079) and integrations that do not exist yet; each follows §4.1.

## 8 Resolved decisions

1. **No generic device table**: Only `key`, `rack`, and `name` mean the same thing for
   every resource; the rest are placeholders for most makes, and every consumer
   dispatches on `make` (§1.1). The trade is real: eight ontology types instead of one,
   eight client namespaces, and a custom policy migration.
2. **No `device.Base` struct**: `key`, `rack`, and `name` are cheap to repeat, and a
   base would force one `Status` shape on every type. The trade is real: three fields
   are declared eight times.
3. **UUID keys, serial as a field**: Natural keys made the scanner's upsert free but put
   hardware identity in the key, which the pre-0.54 key normalization migration shows is
   fragile. The trade is real: one indexed lookup per found device.
4. **One record with two field groups, not two records**: A discovered row plus a
   configuration row would reintroduce the two-table shape of task configs. The trade is
   real: the group boundary is a convention in the write paths, not a table boundary.
5. **`configured` is derived**: It summarized "identifier chosen" and could drift from
   it. Connected resources never had the state.
6. **Chassis and master are resources**: A chassis has no channels and no tasks; a
   master owns the bus order. Folding either into its children left the hierarchy in
   string fields and position uniqueness as a scanner convention.
7. **Position lives on the master**: An ordered list cannot hold two slaves at one
   index. The trade is real: a slave's position needs its master row.
8. **Delete refuses on task references and cascades to children**: The rack guard is the
   precedent, and the integration owns both tables.
9. **Quarantine, not failure, on an undecodable row**: Task v3 is the precedent. A
   blocked startup over one bad device is worse than one hidden device with a warning.
10. **Per-resource namespaces, no EtherCAT union**: `masters` and `slaves` share almost
    no fields; a union would restore the generic-plus-discriminator shape one level
    down.
11. **Naming by what the thing is**: `opcua.Server`, not `opcua.Device`. The `model`
    constants `HTTP server` and `OPC UA` existed only to fill a generic column.
12. **No generic device store**: A `device.Service[D]` needed 8 hooks, did not fit
    racks, and left the same code in every other resource. The trade is real: eight
    hand-written writers until a project deduplicates all resources.

## 9 Open questions

- Tree layout: the `Devices` root group with racks under it, or typed children directly
  under each rack.
- Whether `rack` on an EtherCAT slave stays a filter column or is read through the
  master.
- The retention period for quarantined rows.
