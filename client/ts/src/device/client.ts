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
import { bindStore, STORE_KEY } from "@/device/store";
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
import { status } from "@/status";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

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

const MOUNT_SCOPE = "device.mounts";

/** Drops includeStatus when unset so equivalent queries hash identically. */
const normalizeSingle = ({ key, includeStatus }: RetrieveSingleParams): SingleQuery =>
  includeStatus === true ? { key, includeStatus: true } : { key };

/** The store never holds statuses; the status store is their single home. */
const stripStatus = ({ status: _, ...device }: Device): Omit<Device, "status"> =>
  device;

const isKeysOnly = (req: RetrieveRequest): boolean =>
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
 * Client-side approximation of the server's matching for a request: exact for
 * key, name, make, model, location, and rack sets, permissive for
 * server-computed shapes (search), which accept every change and drift toward
 * the server's answer.
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

export class Client {
  private readonly client: UnaryClient;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<SingleQuery, Device>;
    request: cache.Queries<RetrieveRequest, Device[]>;
  };

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "device",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "devices",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
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
    if (this.queries_ == null || schemas != null) {
      const devices = await this.execRetrieve(rest, schemas);
      checkForMultipleOrNoResults("Device", rest, devices, isSingle);
      this.writes?.set(devices.map(stripStatus));
      return isSingle ? devices[0] : devices;
    }
    if (isSingle) return await this.queries_.single.retrieve(normalizeSingle(rest));
    return await this.queries_.request.retrieve(retrieveRequestZ.parse(rest));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a device; every other shape delivers the matching devices.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Device>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Device[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveSingleParams | RetrieveMultipleParams,
    handler: cache.ChangeHandler<Device> | cache.ChangeHandler<Device[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(
        normalizeSingle(params),
        handler as cache.ChangeHandler<Device>,
      );
    return queries.request.onChange(
      retrieveRequestZ.parse(params),
      handler as cache.ChangeHandler<Device[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached. Unfetched filter queries
   * are approximated from the record store when possible.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Device> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Device[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Device> | cache.Cached<Device[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(normalizeSingle(params));
    const req = retrieveRequestZ.parse(params);
    return queries.request.getCached(req) ?? this.approximateCached(req);
  }

  /**
   * Approximates an unfetched filter query's answer from the record store.
   * Server-computed shapes (search, pagination) cannot be approximated.
   */
  private approximateCached(req: RetrieveRequest): cache.Cached<Device[]> | undefined {
    if (req.searchTerm != null || req.limit != null || req.offset != null)
      return undefined;
    const matches = this.deviceStore.get(requestFilter(req));
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
    this.writes?.set(res.devices.map(stripStatus));
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(ontology.deleteCachedResources(this.engine_, ontologyID(keysArr)));
      rollback.add(writes.delete(keysArr));
    }
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
    this.writes?.delete(keysArr);
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const dev = await this.retrieve({ key });
    const renamed = { ...dev, name };
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (writes != null) rollback.add(writes.set(stripStatus(renamed)));
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.create(renamed);
    });
  }

  private get writes(): cache.UnaryStore<Key, Omit<Device, "status">> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get deviceStore(): cache.UnaryStore<Key, Omit<Device, "status">> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  /** Subscribes to every device set delivered to the cache. Statuses excluded. */
  onSet(handler: (device: Omit<Device, "status">) => void): destructor.Destructor {
    return this.deviceEvents.onSet(handler);
  }

  /** Subscribes to every device delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.deviceEvents.onDelete(handler);
  }

  private get deviceEvents(): cache.UnaryStore<Key, Omit<Device, "status">> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private get statusEvents(): cache.UnaryStore<status.Key, status.Status> {
    return this.requireEngine().store(status.STORE_KEY, MOUNT_SCOPE);
  }

  private requireEngine(): cache.Engine {
    if (this.engine_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.engine_;
  }

  private requireQueries(): NonNullable<typeof this.queries_> {
    if (this.queries_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.queries_;
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
      const cached = this.deviceStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.deviceStore.set(fetched.map(stripStatus));
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (d) => d.key);
  }

  private async fetchSingle(query: SingleQuery): Promise<Device> {
    // Status-bearing answers cannot be served from the status-less store.
    if (query.includeStatus !== true) {
      const cached = this.deviceStore.get(query.key);
      if (cached != null) return cached;
    }
    const devices = await this.execRetrieve(query);
    checkForMultipleOrNoResults("Device", query, devices, true);
    this.deviceStore.set(devices.map(stripStatus));
    return devices[0];
  }

  private mountSingle({
    query,
    update,
    remove,
  }: cache.MountParams<SingleQuery, Device>) {
    const destructors = [
      this.deviceEvents.onSet((device) => {
        // Store events carry no status; merging keeps a status-bearing answer.
        if (device.key === query.key) update((prev) => ({ ...prev, ...device }));
      }),
      this.deviceEvents.onDelete((key) => {
        if (key === query.key) remove(this.deviceStore.getTombstone(key)?.corpse);
      }),
    ];
    if (query.includeStatus === true)
      destructors.push(
        this.statusEvents.onSet((s) => {
          const parsed = statusZ.safeParse(s);
          if (!parsed.success) return;
          update((prev) => (prev == null ? prev : { ...prev, status: parsed.data }));
        }, statusKey(query.key)),
      );
    return destructors;
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Device[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const devices = await this.execRetrieve(query);
    this.deviceStore.set(devices.map(stripStatus));
    return devices;
  }

  private mountRequest({
    query,
    update,
  }: cache.MountParams<RetrieveRequest, Device[]>) {
    const matches = requestFilter(query);
    const destructors = [
      this.deviceEvents.onSet((device) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((d) => d.key === device.key);
          if (!matches(device))
            return existing ? prev.filter((d) => d.key !== device.key) : prev;
          if (existing)
            return prev.map((d) => (d.key === device.key ? { ...d, ...device } : d));
          return [...prev, device];
        });
      }),
      this.deviceEvents.onDelete((key) => {
        update((prev) => prev?.filter((d) => d.key !== key));
      }),
    ];
    if (query.includeStatus === true)
      destructors.push(
        this.statusEvents.onSet((s) => {
          const parsed = statusZ.safeParse(s);
          if (!parsed.success) return;
          update((prev) =>
            prev?.map((d) =>
              d.key === parsed.data.details.device ? { ...d, status: parsed.data } : d,
            ),
          );
        }),
      );
    return destructors;
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
