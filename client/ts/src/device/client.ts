// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive, type record, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import {
  type Device,
  type DeviceSchemas,
  deviceZ,
  type Key,
  keyZ,
  type New,
  ontologyID,
  statusZ,
} from "@/device/types.gen";
import { ontology } from "@/ontology";
import { keyZ as rackKeyZ } from "@/rack/types.gen";
import { type status } from "@/status";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_device_set";
export const DELETE_CHANNEL_NAME = "sy_device_delete";

const genericDeviceZ = deviceZ();

const createReqZ = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: DeviceSchemas<Properties, Make, Model>,
) => z.object({ devices: zod.toArray(deviceZ(schemas)) });

const createResZ = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: DeviceSchemas<Properties, Make, Model>,
) => z.object({ devices: deviceZ(schemas).array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  makes: z.string().array().optional(),
  models: z.string().array().optional(),
  locations: z.string().array().optional(),
  racks: rackKeyZ.array().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
  includeParent: z.boolean().optional(),
});

const retrieveResZ = <
  Properties extends z.ZodType<record.Unknown> = z.ZodType<record.Unknown>,
  Make extends z.ZodType<string> = z.ZodString,
  Model extends z.ZodType<string> = z.ZodString,
>(
  schemas?: DeviceSchemas<Properties, Make, Model>,
) =>
  z.object({
    devices: deviceZ(schemas)
      .array()
      .default(() => []),
  });

const singleRetrieveParamsZ = z
  .object({
    key: keyZ,
    includeStatus: z.boolean().optional(),
  })
  .transform(({ key, includeStatus }) => ({
    keys: [key],
    includeStatus,
  }));

export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

type SingleQuery = { key: Key; includeStatus?: boolean };

type RetrieveSchemas<
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
> = { schemas: DeviceSchemas<Properties, Make, Model> };

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset", "includeParent"] as const;

/** Drops includeStatus when unset so equivalent queries hash identically. */
const normalizeSingle = ({ key, includeStatus }: RetrieveSingleParams): SingleQuery =>
  includeStatus === true ? { key, includeStatus: true } : { key };

/** The table never holds statuses; the status table is their single home. */
const stripStatus = ({ status: _, ...device }: Device): Omit<Device, "status"> =>
  device;

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) &&
  req.names == null &&
  req.makes == null &&
  req.models == null &&
  req.locations == null &&
  req.racks == null &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null &&
  req.includeStatus !== true &&
  req.includeParent !== true;

/**
 * Client-side matching for a request: key, name, make, model, location, and
 * rack sets. Server-computed shapes (search, pagination, parent resolution)
 * never reach this filter; they refetch instead.
 */
const requestFilter = (
  req: RetrieveRequest,
): ((d: Omit<Device, "status">) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
  const makeSet = primitive.isNonZero(req.makes) ? new Set(req.makes) : undefined;
  const modelSet = primitive.isNonZero(req.models) ? new Set(req.models) : undefined;
  const locationSet = primitive.isNonZero(req.locations)
    ? new Set(req.locations)
    : undefined;
  const rackSet = primitive.isNonZero(req.racks) ? new Set(req.racks) : undefined;
  return (d) => {
    if (keySet != null && !keySet.has(d.key)) return false;
    if (nameSet != null && !nameSet.has(d.name)) return false;
    if (makeSet != null && !makeSet.has(d.make)) return false;
    if (modelSet != null && !modelSet.has(d.model)) return false;
    if (locationSet != null && !locationSet.has(d.location)) return false;
    if (rackSet != null && !rackSet.has(d.rack)) return false;
    return true;
  };
};

// Device statuses live in the status table under the "device:<key>" status
// key, carrying the device key in their details.
const affectedDeviceKeys = (
  event: cache.TableEvent<status.Key, status.Status>,
): Key[] | null => {
  if (event.variant === "set") {
    const parsed = statusZ.safeParse(event.value);
    return parsed.success ? [parsed.data.details.device] : null;
  }
  const [type, key] = event.key.split(":");
  if (type !== "device" || !primitive.isNonZero(key)) return null;
  return [key];
};

const createTable = (engine: cache.Cache): cache.Table<Key, Omit<Device, "status">> => {
  // Explicitly omit 'status' from the device type to make sure we aren't storing two
  // copies of the statuses in the store.
  const table = engine.createTable<Key, Omit<Device, "status">>({ name: "devices" });
  const set: cache.ChannelListener<typeof genericDeviceZ> = {
    channel: SET_CHANNEL_NAME,
    schema: genericDeviceZ,
    onChange: ({ status: _, ...device }) => table.set(device),
  };
  const del: cache.ChannelListener<typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: (changed) => table.delete(changed),
  };
  engine.addListeners(table, set, del);
  return table;
};

export class Client extends cache.Reader<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  SingleQuery,
  RetrieveRequest,
  Device
> {
  private readonly client: UnaryClient;
  private readonly store: cache.Table<Key, Omit<Device, "status">>;
  private readonly statusStore: cache.Table<status.Key, status.Status>;
  private readonly ontology: ontology.Stores;

  constructor(
    client: UnaryClient,
    engine: cache.Cache,
    statusStore: cache.Table<status.Key, status.Status>,
    ontologyStores: ontology.Stores,
  ) {
    const store = createTable(engine);
    const statusWatch = <Q extends { includeStatus?: boolean }>() =>
      cache.watch(statusStore, (event, query: Q) => {
        if (query.includeStatus !== true) return null;
        return affectedDeviceKeys(event);
      });
    super({
      single: engine.answers({
        name: "device",
        table: store,
        fetch: async (query) => [(await this.fetchSingle(query)).key],
        compose: ([record], query) =>
          this.compose(record, query.includeStatus === true),
        keyOf: (query) => query.key,
        single: true,
        watch: [statusWatch<SingleQuery>()],
      }),
      request: engine.answers({
        name: "devices",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((d) => d.key),
        compose: (records, query) =>
          records.map((d) => this.compose(d, query.includeStatus === true)),
        matches: (d, query) => requestFilter(query)(d),
        serverFields: SERVER_FIELDS,
        watch: [statusWatch<RetrieveRequest>()],
      }),
      isSingle: (params) => "key" in params,
      normalizeSingle,
      normalizeRequest: (params) => retrieveRequestZ.parse(params),
    });
    this.client = client;
    this.statusStore = statusStore;
    this.ontology = ontologyStores;
    this.store = store;
  }

  async retrieve<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    params: RetrieveSingleParams & RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>>;

  async retrieve(params: RetrieveSingleParams): Promise<Device>;

  async retrieve<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    params: RetrieveMultipleParams & RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Array<Device<Properties, Make, Model>>>;

  async retrieve(params: RetrieveMultipleParams): Promise<Array<Device>>;

  async retrieve(
    params: RetrieveParams & { schemas?: DeviceSchemas },
  ): Promise<Device | Array<Device>> {
    const { schemas, ...rest } = params;
    const isSingle = "key" in rest;
    // Schemas are not hashable, so schema-typed retrieves bypass the query
    // cache and hit the network, still writing records through.
    if (schemas != null) {
      const devices = await this.execRetrieve(rest, schemas);
      checkForMultipleOrNoResults("Device", rest, devices, isSingle);
      this.writeThrough(devices);
      return isSingle ? devices[0] : devices;
    }
    if (isSingle) return await super.retrieve(rest);
    return await super.retrieve(rest);
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached. Unfetched filter queries
   * are approximated from the record table when possible.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Device> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Device[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Device> | cache.Cached<Device[]> | undefined {
    if ("key" in params) return super.getCached(params);
    return (
      super.getCached(params) ?? this.approximateCached(retrieveRequestZ.parse(params))
    );
  }

  /**
   * Approximates an unfetched filter query's answer from the record table.
   * Server-computed shapes (search, pagination) cannot be approximated.
   */
  private approximateCached(req: RetrieveRequest): cache.Cached<Device[]> | undefined {
    if (req.searchTerm != null || req.limit != null || req.offset != null)
      return undefined;
    const matches = this.store.get(requestFilter(req));
    if (matches.length === 0) return undefined;
    return { variant: "changed", data: matches };
  }

  async create(device: New): Promise<Device>;

  async create(devices: New[]): Promise<Device[]>;

  async create<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    device: New<Properties, Make, Model>,
    schemas: DeviceSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>>;

  async create(device: New, schemas?: DeviceSchemas): Promise<Device>;

  async create<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    devices: New<Properties, Make, Model>[],
    schemas: DeviceSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>[]>;

  async create(devices: New[], schemas?: DeviceSchemas): Promise<Device[]>;

  async create(
    devices: New | New[],
    schemas?: DeviceSchemas,
  ): Promise<Device | Device[]> {
    const isSingle = !Array.isArray(devices);
    const res = await this.client.send(
      "/device/create",
      { devices: array.toArray(devices) },
      createReqZ(schemas),
      createResZ(schemas),
    );
    this.store.set(res.devices.map(stripStatus));
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(ontology.deleteCachedResources(this.ontology, ontologyID(keysArr)));
    rollback.add(this.store.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/device/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    );
    this.store.delete(keysArr);
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const dev = await this.retrieve({ key });
    const renamed = { ...dev, name };
    const rollback = new cache.Rollback();
    rollback.add(this.store.set(renamed.key, stripStatus(renamed)));
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.create(renamed);
    });
  }

  /** Subscribes to every device set delivered to the cache. Statuses excluded. */
  onSet(handler: (device: Omit<Device, "status">) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every device delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  /** Rebuilds a cached device, attaching its cached status when requested. */
  private compose(cached: Omit<Device, "status">, includeStatus: boolean): Device {
    if (!includeStatus) return cached;
    const st = this.statusStore.get(statusKey(cached.key));
    if (st == null) return cached;
    const parsed = statusZ.safeParse(st);
    if (!parsed.success) return cached;
    return { ...cached, status: parsed.data };
  }

  /** Writes fetched devices and their included statuses. */
  private writeThrough(devices: Device[]): void {
    this.store.set(devices.map(stripStatus));
    devices.forEach(({ status: st }) => {
      if (st != null) this.statusStore.set(st);
    });
  }

  private async execRetrieve(
    params: RetrieveParams,
    schemas?: DeviceSchemas,
  ): Promise<Device[]> {
    const res = await this.client.send(
      "/device/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ(schemas),
    );
    return res.devices;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Device[]> {
    const results: Device[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.store.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.writeThrough(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (d) => d.key);
  }

  private async fetchSingle(query: SingleQuery): Promise<Device> {
    // A status-bearing hit needs both the record and its status cached.
    if (query.includeStatus !== true) {
      const cached = this.store.get(query.key);
      if (cached != null) return cached;
    }
    const devices = await this.execRetrieve(query);
    checkForMultipleOrNoResults("Device", query, devices, true);
    this.writeThrough(devices);
    return devices[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Device[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys);
    const devices = await this.execRetrieve(query);
    this.writeThrough(devices);
    return devices;
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
