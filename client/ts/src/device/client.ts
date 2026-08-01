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
import { query } from "@/query";
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

const singleQueryZ = z
  .strictObject({ key: keyZ, includeStatus: z.boolean().optional() })
  .transform(normalizeSingle);

/** The table never holds statuses; the status table is their single home. */
const stripStatus = ({ status: _, ...device }: Device): Omit<Device, "status"> =>
  device;

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
  event: query.TableEvent<status.Key, status.Status>,
): Key[] | null => {
  if (event.variant === "set") {
    const parsed = statusZ.safeParse(event.value);
    return parsed.success ? [parsed.data.details.device] : null;
  }
  const [type, key] = event.key.split(":");
  if (type !== "device" || !primitive.isNonZero(key)) return null;
  return [key];
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  statusStore: query.Table<status.Key, status.Status>;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<
  typeof retrieveRequestZ,
  Key,
  Omit<Device, "status">,
  Device,
  RetrieveSingleParams,
  SingleQuery
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Omit<Device, "status">>;

  constructor(cfg: ClientConfig) {
    const { cache, statusStore } = cfg;
    // Explicitly omit 'status' from the device type to make sure we aren't
    // storing two copies of the statuses in the store.
    const store = cache.createTable<Key, Omit<Device, "status">>({
      name: "devices",
      fetch: async (keys) => await this.fetchThrough({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, genericDeviceZ, {
          value: ({ status: _, ...device }) => device,
        }),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    // Composed rows always carry status; includeStatus only gates fetch flags.
    const composed = cache.derive<Key, Omit<Device, "status">, Device>({
      name: "device.composed",
      source: store,
      compose: (record) => this.compose(record, true),
      watch: [query.deriveWatch(statusStore, (event) => affectedDeviceKeys(event))],
    });
    const single = cache.queries<SingleQuery, Device, Key, Device>({
      name: "device",
      table: composed,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => q.key,
      single: true,
    });
    super(cache, {
      name: "device",
      table: store,
      request: {
        schema: retrieveRequestZ,
        fetch: async (req) => (await this.fetchThrough(req)).map(stripStatus),
        matches: (d, req) => requestFilter(req)(d),
        serverFields: SERVER_FIELDS,
        watch: [
          query.watch(statusStore, (event, q: RetrieveRequest) => {
            if (q.includeStatus !== true) return null;
            return affectedDeviceKeys(event);
          }),
        ],
      },
      compose: (record, q) =>
        this.compose(record, (q as { includeStatus?: boolean }).includeStatus === true),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
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
  getCached(params: RetrieveSingleParams): query.Cached<Device> | undefined;
  getCached(params: RetrieveMultipleParams): query.Cached<Device[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): query.Cached<Device> | query.Cached<Device[]> | undefined {
    if ("key" in params) return super.getCached(params);
    return (
      super.getCached(params) ?? this.approximateCached(retrieveRequestZ.parse(params))
    );
  }

  /**
   * Approximates an unfetched filter query's answer from the record table.
   * Server-computed shapes (search, pagination) cannot be approximated.
   */
  private approximateCached(req: RetrieveRequest): query.Cached<Device[]> | undefined {
    if (req.searchTerm != null || req.limit != null || req.offset != null)
      return undefined;
    const matches = this.store.get(requestFilter(req));
    if (matches.length === 0) return undefined;
    return matches;
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
    const res = await this.cfg.unary.send(
      "/device/create",
      { devices: array.toArray(devices) },
      createReqZ(schemas),
      createResZ(schemas),
    );
    this.store.set(res.devices.map(stripStatus));
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteResources(ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/device/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    });
    drop();
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const dev = await this.retrieve({ key });
    const renamed = { ...dev, name };
    await query.optimistic({
      rollbacks: [this.store.set(renamed.key, stripStatus(renamed))],
      onOptimistic: opts.onOptimistic,
      commit: async () => {
        await this.create(renamed);
      },
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
    const st = this.cfg.statusStore.get(statusKey(cached.key));
    if (st == null) return cached;
    const parsed = statusZ.safeParse(st);
    if (!parsed.success) return cached;
    return { ...cached, status: parsed.data };
  }

  /** Writes fetched devices and their included statuses. */
  private writeThrough(devices: Device[]): void {
    this.store.set(devices.map(stripStatus));
    devices.forEach(({ status: st }) => {
      if (st != null) this.cfg.statusStore.set(st);
    });
  }

  private async execRetrieve(
    params: RetrieveParams,
    schemas?: DeviceSchemas,
  ): Promise<Device[]> {
    const res = await this.cfg.unary.send(
      "/device/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ(schemas),
    );
    return res.devices;
  }

  /** Fetches devices and writes their included statuses through the caches. */
  private async fetchThrough(req: RetrieveRequest): Promise<Device[]> {
    const devices = await this.execRetrieve(req);
    this.writeThrough(devices);
    return devices;
  }

  private async fetchSingle(q: SingleQuery): Promise<Device> {
    // A status-bearing hit needs both the record and its status cached.
    if (q.includeStatus !== true) {
      const cached = this.store.get(q.key);
      if (cached != null) return cached;
    }
    const devices = await this.execRetrieve(q);
    checkForMultipleOrNoResults("Device", q, devices, true);
    this.writeThrough(devices);
    return devices[0];
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
