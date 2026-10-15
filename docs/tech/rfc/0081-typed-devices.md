# 81 Typed devices

- **Author**: Patrick Dotson
- **Date**: 2026-10-15
- **Related**: [RFC 0017 - General purpose device drivers](0017-drivers.md),
  [RFC 0034 - Gorp indexes](0034-gorp-indexes.md),
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0079 - Kafka integration](0079-kafka-integration.md)

## 0 Summary

The generic `device` table goes away. Each hardware integration defines its own device
resources in its Oracle schema, with typed fields in place of the `properties` blob, its
own Gorp table, ontology type, migrations, and endpoints, owned by that integration's
Core service. A generic Go `device.Service[D]` provides the shared machinery, the way
`config.Service[E]` does for task configs. Every device gets a UUID key; discovered
hardware carries a `serial`. The scanner and the Console write disjoint field groups, so
neither can overwrite the other. NI chassis and EtherCAT masters become resources of
their own.

## 1 Motivation

A device record today is `key`, `rack`, `location`, `make`, `model`, `name`,
`configured`, and a `properties` JSON blob the Core never reads
(`core/pkg/service/device/versions/v1/types.gen.go:55`). The blob is typed three times
by hand, and the copies disagree:

- The Console holds a Zod schema per make under `console/src/feature/*/device/types.ts`,
  with its own property migrators for OPC UA and HTTP. It is the only place a client
  migrates data the Core persists.
- The Driver holds C++ structs for four of the six makes
  (`driver/opcua/device/device.h`, `driver/modbus/device/device.h:141`,
  `driver/http/device/device.h`, `driver/ethercat/slave/slave.h:72`) and reads nothing
  from the blob for NI and LabJack.
- The Python client builds the blob as an untyped dict in a per-make `Device` subclass.

Two writers share one record. The scanner upserts a whole device after a shallow merge
against its own cache (`driver/common/scan_task.h:40-51,506-519`); the Console writes
identifiers, index keys, and channel maps through the same `create`. A stale scanner
cache overwrites a Console edit, and EtherCAT hand-protects `enabled` against exactly
that (`driver/ethercat/scan_task.cpp:128-131`).

The generic fields carry per-make meaning. `location` is a DAQmx alias for NI, `USB` or
`TCP` for LabJack, `host:port` for Modbus, an endpoint for OPC UA, a host for HTTP, and
`<master>.Slot <n>` for EtherCAT. `model` is a free string, a three-value enum, or a
constant. `make` drifts: the Driver filters Modbus devices by `modbus` while the Console
and Python write `Modbus`, so Console-created Modbus servers never get health checks
(SY-5129).

Every consumer of the generic record dispatches on `make` before it can do anything:
`console/src/platform/device/Select.tsx:52-64`, the make registry in
`console/src/feature/device/make.tsx`, the tree item at
`console/src/feature/device/tree.tsx:56,99`, and the Driver's scan cache
(`driver/common/scan_task.h:140-147`). The abstraction hides nothing.

## 2 Vocabulary

- **Device resource**: One typed record for one piece of hardware or one server, stored
  in a table that belongs to a single integration.
- **Discovered resource**: A resource the Driver's scanner creates from hardware it
  finds: NI chassis and devices, LabJack devices, EtherCAT masters and slaves.
- **Connected resource**: A resource a user creates from a connect form: OPC UA, Modbus,
  and HTTP servers.
- **Identity group**: The fields of a discovered resource that the scanner owns: serial,
  model, alias, simulation flag, bus topology.
- **Configuration group**: The fields a user owns: identifier, index channel keys,
  channel maps, enabled flags, connection settings.
- **Identifier**: The short prefix a user chooses for a discovered resource's channel
  names. A discovered resource with an empty identifier is unconfigured.
- **Scanner**: The per-integration scan task in the Driver. It discovers hardware,
  reports health, and answers test-connection commands.

## 3 Principles

1. **One resource type per kind of hardware**: A chassis, a module, a master, a slave,
   and a server are different things with different fields. Each gets its own struct,
   table, and ontology type. No shared base struct and no union.
2. **Oracle is the only source of a device shape**: Go, TypeScript, Python, and C++ all
   consume generated types. Hand-written Zod, C++ structs, and Python dicts are deleted.
3. **The integration owns its devices**: The Go service that owns an integration's task
   config stores also owns its device tables, endpoints, and migrations
   (`core/pkg/service/ni/service.go:69-176` is the shape).
4. **Shared machinery is generic, not inherited**: `device.Service[D]` takes its
   contract from injected hooks, the way `config.ServiceConfig[E]` does
   (`core/pkg/service/task/config/config.go:66-122`).
5. **One writer per field**: The scanner writes the identity group, the user writes the
   configuration group, and a write path never touches the other group.
6. **Core-minted keys**: Every resource key is a UUID the Core creates. Hardware
   identity is a field with a lookup index, never a key.
7. **Delete refuses, then cascades**: A resource that a task references cannot be
   deleted. A deleted parent takes its children with it.

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

Each resource is a struct in its integration's `.oracle` file, with the file's existing
`@go`, `@ts`, `@py`, and `@cpp` outputs. Every struct carries `@create`, `@retrieve`,
`@search`, `@go marshal`, `@go migrate`, and `@ontology type`. Every struct has
`key uuid = create` with `@key`, `rack rack.Key` with `@filter`, `name string` with
`@filter`, and a `status` of its own `Status` type marked `@go marshal omit`. The three
shared fields are duplicated per struct; a base struct would carry nothing else.

The NI resources, as the pattern for every discovered make:

```
Chassis struct {
    key       Key = create        { @key }
    rack      rack.Key            { @filter }
    name      string              { @filter }
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
    @search
}

Device struct {
    key        Key = create       { @key }
    rack       rack.Key           { @filter }
    name       string             { @filter }
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

`serial` through `chassis` are the identity group. `identifier` and the port maps are
the configuration group. The port maps are the typed form of today's
`analogInput.channels` and friends (`console/src/feature/ni/device/types.ts:18-63`).

Per make, the identity and configuration groups are:

- **LabJack**: identity `serial`, `model` (the existing `LJM_dtT4`, `T7`, `T8` enum),
  `connection` (USB or TCP); configuration `identifier`, `read_index`,
  `thermocouple_index`, `write_state_index`, and the four port channel maps.
- **EtherCAT master**: identity `interface`, `slaves Slave.Key[]` in bus order.
- **EtherCAT slave**: identity `serial`, `vendor_id`, `product_code`, `revision`,
  `master Master.Key`, `pdos`; configuration `identifier`, `enabled`, `read_index`,
  `write_state_index`, and the input and output channel maps.
- **OPC UA server**: `connection` (endpoint, credentials, security mode and policy,
  certificates), `read_indexes`, and the node channel maps, from
  `console/src/feature/opcua/device/types/v1.ts:18-30`.
- **Modbus server**: `host`, `port`, `swap_bytes`, `swap_words`, `read_index`, and the
  register channel maps.
- **HTTP server**: `host`, `secure`, `verify_ssl`, `timeout`, `auth` (the existing
  union), `max_concurrent_requests`, `health_check`, and the per-path read and write
  channel maps, from `console/src/feature/http/device/types.ts:224-239`. The `base_url`
  the Driver injects from `location` today (`driver/http/device/device.cpp:100-106`) is
  derived from `host` and `secure` in generated code.

Connected resources have no identity group and no identifier: they are configured at
creation. `configured` does not exist. A discovered resource is unconfigured while
`identifier` is empty.

Task configs reference a resource by its typed key: `ni.Device.Key` on NI channels
(`schemas/synnax/ni.oracle:489,941,1389`), `labjack.Device.Key` on `labjack.oracle:216`,
and so on. `task.WriteConfig.device device.Key` (`schemas/synnax/task.oracle:197`) moves
down into each integration's write config under its typed key, since no shared type
remains to name.

### 4.2 Core services

`device.Service[D]` is the generic store, in `core/pkg/service/device` once the legacy
service has moved out. Its config mirrors `config.ServiceConfig[E]`:

```go
type ServiceConfig[D gorp.Entry[uuid.UUID]] struct {
    DB              *gorp.DB
    Ontology        *ontology.Ontology
    Search          *search.Service
    Status          *status.Service
    Signals         *signals.Provider
    Type            ontology.ResourceType
    Migrations      []migrate.Migration
    SetEntryKey     func(*D, uuid.UUID)
    RackOf          func(D) rack.Key
    ParentOf        func(D) (ontology.ID, bool)
    ValidateEntry   func(*D) error
    alamos.Instrumentation
}
```

The service opens the Gorp table with the migration chain, registers the ontology and
search services for `Type`, publishes `sy_<type>_set` and `sy_<type>_delete` through
`signals.PublishFromGorp`, and exposes `Create`, `Update`, `Retrieve`, and `Delete`.
`Create` writes the row, an initial unknown status, the ontology resource, and the
`parent_of` edge from `ParentOf` or, when absent, from the rack. `Update` takes a
function over the stored row, so a caller changes one field group and leaves the other
as stored (§4.3). `Delete` refuses with a `validate.ErrValidation` wrap while a referrer
check the integration supplies returns true, then deletes children, status, resource,
and row in one transaction.

Each integration service opens one `device.Service[D]` per resource next to its config
stores. The NI service gains `Chassis *device.Service[Chassis]` and
`Devices *device.Service[Device]`; the EtherCAT service gains `Masters` and `Slaves`.
The referrer check for `ni.Device` scans the five NI config stores for a channel whose
`device` is the key; both tables belong to the same service, so the check needs no
cross-service query.

Device status stays in the status service under `<type>:<key>`. Each resource defines
its own `Status` and `StatusDetails`, since the fields a server reports differ from a
slave's.

### 4.3 Writes

The scanner's write path is `Update` over the identity group. On a scan it resolves each
found serial through the lookup index: a hit updates identity fields and leaves the
configuration group as stored; a miss creates the row with an empty configuration group.
The Console's write path is `Update` over the configuration group. The shallow merge in
`driver/common/scan_task.h:40-51` and the EtherCAT `enabled` guard are deleted.

A resource the scanner no longer sees gets a warning status, as today, and is never
deleted by the scanner.

### 4.4 Hierarchy

`ni.Device.chassis` and `ethercat.Slave.master` are stored keys. The service defines the
`parent_of` edge from them on every write, so the tree renders modules under their
chassis and slaves under their master through ontology children, unchanged. A resource
without a parent key is a child of its rack. When Gorp references land, each becomes
`@ref on_delete cascade acyclic` with no data change.

The master holds its slaves as an ordered `slaves Slave.Key[]`. A slave's position is
its index. Two slaves cannot share a position, a scan writes the bus order as one update
of the master row, and the slave stores no position. A slave with serial zero is matched
across scans by `master.slaves[i]`, vendor, and product. Topology validation
(`driver/ethercat/topology/topology.h:27-58`) reads the master's list. The `network`
string and the `<master>.Slot <n>` location encoding are deleted.

### 4.5 API and clients

Each resource gets its own routes under its integration: `/ni/device/create`,
`/ni/device/retrieve`, `/ni/device/update`, `/ni/device/delete`, and the same for
`/ni/chassis`, `/ethercat/master`, `/opcua/server`, and the rest. Retrieve filters are
the struct's `@filter` fields plus `search_term`, `include_status`, and
`include_parent`. The generic `/device/*` routes are deleted.

Clients expose one namespace per resource: `client.ni.devices`, `client.ni.chassis`,
`client.labjack.devices`, `client.ethercat.masters`, `client.ethercat.slaves`,
`client.opcua.servers`, `client.modbus.servers`, `client.http.servers`. Retrieve returns
the typed struct. The generic `client.devices`, Python `sy.Device`, and
`synnax::device::Client` are deleted. The Python per-make classes keep their keyword
constructors as thin wrappers over the generated types.

Pluto's `device` module keeps its machinery as factories parameterized by a resource
client: the cache table, `createRetrieve`, `createForm`, `SelectSingle`, and
`StatusIndicator` (`pluto/src/device/queries.ts:113-326`). The untyped defaults go away;
every call site passes a resource.

### 4.6 Driver

The C++ `common::ScanTask` becomes a template over the resource type, with the `Scanner`
interface returning `std::vector<D>` and `ClusterAPI` parameterized the same way
(`driver/common/scan_task.h:81-130`). Each integration's read and write tasks take the
generated struct in place of `parser.field` reads. NI's read task keeps using `alias`
for physical channel names, and LabJack opens by `serial`
(`driver/labjack/read_task.h:535` uses the key today).

### 4.7 Console

Each integration's `device` folder consumes its generated types and resource client.
`console/src/feature/device/make.tsx` is deleted: each integration registers its own
tree items, context menu, link handler, and palette commands, the way task layouts
already register per type. The unconfigured-device toast moves into the three discovered
makes, each listening on its own `sy_<type>_set`. The shared `platform/device` layer
keeps `Configure`, `wrapTaskForm`, and the menu items as generics over a resource with
an `identifier` field. The Console property migrators for OPC UA and HTTP are deleted;
the Core's migration chain owns both.

### 4.8 Migration

Release 0.59 shipped the single table. One ordered startup migration cuts over:

1. Read every row of the legacy table. Match `make` case-insensitively, which absorbs
   the `modbus` and `EtherCAT` drift. Decode `properties` into the make's typed struct,
   and NI rows into chassis or device by `is_chassis`.
2. Write each typed row under a fresh UUID. Persist the old key to UUID map in KV. A row
   whose make is unknown or whose properties do not decode is quarantined under
   `sy_device_quarantine/<key>` with a warning, never dropped and never fatal, as task
   v3 does (`core/pkg/service/task/versions/v3/migrate.go:128-155`).
3. Each integration's task config migration rewrites its device references through the
   map.
4. Re-key statuses, ontology resources and edges, and policy object IDs.
5. Delete the legacy table and the map.

Between the typed services landing and this migration landing, a development Core that
already holds 0.59 rows shows them only through the legacy path. Candidate-era writes
through that path are not carried forward; the migration targets the 0.59 shape.

### 4.9 Access control

`ontology.ResourceType` gains the eight types. The builtin role lists that name
`ResourceTypeDevice` (`core/pkg/service/access/rbac/builtin/builtin.go:34,85,123`) name
all eight instead; provisioning rewrites them on the next boot. The migration expands a
custom policy object `{type: "device"}` into the eight typed objects, and a per-key
`device:<key>` object into its typed successor through the map.

### 4.10 Testing

`device.Service[D]` gets its own Ginkgo suite against an in-memory Gorp DB with a test
entry type. Each integration service gets specs for create, identity update, config
update, referrer-refused delete, and cascade. The migration gets a suite seeded with
0.59-shaped rows for every make, including a quarantined row. Driver specs replace the
hand-built JSON devices with generated structs. Console specs for each make's forms and
tree items run against the typed client.

## 5 Prior art

Ignition stores one `DeviceSettingsRecord` row plus one per-driver record, OPC UA DI
subtypes an abstract `DeviceType` per companion spec, and Grafana, Azure IoT Hub, and
Home Assistant keep a generic row with a JSON blob. All of them have runtime-installed
or third-party drivers, so the base table cannot change per driver. Kubernetes alone has
no generic table: every kind is its own typed resource, and cross-kind listing is
discovery plus N lists. Synnax compiles its integrations into the Core and the Driver,
the type set is closed, and no consumer lists devices across makes, so the Kubernetes
shape fits and the shared-row shape buys nothing.

## 6 Implementation phases

Each phase is one pull request into `main`. Typed paths land first and serve empty
tables on an existing Core, so the integration suite, which creates fresh data, stays
green at every step. The migration brings 0.59 data forward near the end.

- **Phase 0: Modbus make.** `MAKE = "Modbus"` in the Driver (SY-5129). Independent of
  this RFC.
- **Phase 1: Legacy move.** The current `core/pkg/service/device` moves to
  `core/pkg/service/device/legacy`. Mechanical, `review/bot`.
- **Phase 2: Generic service.** `device.Service[D]`, `ServiceConfig[D]`, and its suite.
  Nothing uses it yet.
- **Phase 3: Ontology types and roles.** The eight `ResourceType` members, regenerated
  clients, and the builtin role lists.
- **Phases 4 to 9: Schemas and services, one integration each.** NI, LabJack, EtherCAT,
  OPC UA, Modbus, HTTP: the Oracle structs, generated types in four languages, the
  `device.Service[D]` instances in the integration service, and routes. Each lands dark
  with no caller.
- **Phases 10 to 15: Driver, one integration each.** Scanner and tasks on the typed
  client. Phase 10 also turns `ScanTask` into a template.
- **Phases 16 to 21: Console, one integration each.** Forms, tree items, menus, links,
  and palette commands on the typed client. Phase 16 also deletes `make.tsx`.
- **Phase 22: Task config references.** Typed device keys on every config, with the
  config migrations from §4.8 step 3 ready to read the map.
- **Phase 23: Migration.** §4.8 steps 1, 2, 4, and 5 and the policy rewrite.
- **Phase 24: Delete the legacy path.** The legacy service, `/device/*` routes,
  `client.devices`, `sy.Device`, `synnax::device::Client`, and the Console's generic
  device layer.
- **Phase 25: Documentation.** Device pages for each integration and the Python and
  TypeScript examples.

Compatibility: a 0.59 Core upgrades through Phase 23 with every device, status,
reference, and policy carried forward. A client older than Phase 24 talking to a Core at
Phase 24 gets a not-found on `/device/*`; clients and the Core ship in the same train.

## 7 What this RFC does not cover

- Device import and export. The typed structs are shaped for a later `@go imex`.
- A secret store for server credentials.
- Gorp references and the ontology's replacement; the parent keys here become `@ref`
  fields when that lands.
- Kafka clusters (RFC 0079) and any integration that does not exist yet; each follows
  §4.1 when it arrives.

## 8 Resolved decisions

1. **No generic device table**: Every consumer already dispatches on `make`, and the
   only shared behavior is status, which differs per type too. The trade is real: eight
   ontology types instead of one, eight namespaces in every client, and a custom policy
   migration.
2. **No `device.Base` struct**: `key`, `rack`, and `name` are cheap to repeat, and a
   base would force one `Status` shape on every type. The trade is real: three fields
   are declared eight times.
3. **UUID keys everywhere, serial as a field**: Natural keys made the scanner's upsert
   free but put hardware identity in the key, which the pre-0.54 key normalization
   migration shows is fragile. The trade is real: the scanner does one indexed lookup
   per found device.
4. **One record with two field groups, not two records**: A discovered row plus a config
   row would reintroduce the two-table shape of task configs. The trade is real: the
   group boundary is a convention in the write paths, not a table boundary.
5. **`configured` is derived**: It summarized "identifier chosen" and could drift from
   it. Connected resources never had the state.
6. **Chassis and master are resources**: A chassis has no channels and no tasks; a
   master owns the bus order. Folding either into its children left the hierarchy in
   string fields and position uniqueness as a scanner convention.
7. **Position lives on the master**: An ordered list cannot hold two slaves at one
   index. The trade is real: reading a slave's position needs its master row.
8. **Delete refuses on task references and cascades to children**: The rack guard is the
   precedent, and the integration owns both tables.
9. **Quarantine, not failure, on an undecodable row**: Task v3 is the precedent. A
   blocked startup over one bad device is worse than one hidden device with a warning.
10. **Per-resource namespaces, no EtherCAT union**: `masters` and `slaves` share almost
    no fields; a union would restore the generic-plus-discriminator shape one level
    down.
11. **Naming by what the thing is**: `opcua.Server`, not `opcua.Device`. The `model`
    constants `HTTP server` and `OPC UA` existed only to fill a generic column.

## 9 Open questions

- Tree layout: whether the `Devices` root group stays with racks under it, or each rack
  lists typed children directly.
- The home of the generic service while the legacy one still exists: `device/legacy` as
  proposed, or the generic under a new name.
- Whether `rack` on an EtherCAT slave stays as a filter column or is read through the
  master.
- The retention period for quarantined rows.
