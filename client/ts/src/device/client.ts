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
  resourceID,
  statusZ,
} from "@/device/types.gen";
import { group } from "@/group";
import { type ontology } from "@/ontology";
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
const emptyResZ = z.object({});
const renameReqZ = z.object({ key: keyZ, name: z.string() });

const retrieveRequestZ = group.memberFilterZ.extend({
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
  /** Keeps only the devices held by any of these devices. */
  parents: keyZ.array().optional(),
  /**
   * Keeps only the devices held by another device when true, and only the devices that
   * sit directly on their rack when false.
   */
  nested: z.boolean().optional(),
});
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));

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
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

/** Drops includeStatus when unset so equivalent queries hash identically. */
const normalizeSingle = ({ key, includeStatus }: RetrieveSingleParams): SingleQuery =>
  includeStatus === true ? { key, includeStatus: true } : { key };

const singleQueryZ = z
  .union([
    z.strictObject({ key: keyZ, includeStatus: z.boolean().optional() }),
    keyZ.transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

/** The table never holds statuses; the status table is their single home. */
const stripStatus = ({ status: _, ...device }: Device): Omit<Device, "status"> =>
  device;

/**
 * Removes the status of a device. Use it to send a retrieved device or a create reply
 * back to {@link Client.create} without replacing the status the Core holds.
 */
export const toUpdate = <D extends Pick<Device, "status">>({
  status: _,
  ...device
}: D): Omit<D, "status"> => device;

/**
 * Client-side matching for a request: key, name, make, model, location, rack, parent,
 * nesting, and group sets. Server-computed shapes (search and pagination) never reach
 * this filter; they refetch instead.
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
  const parentSet = primitive.isNonZero(req.parents) ? new Set(req.parents) : undefined;
  const inGroup = group.createMemberMatcher(req);
  return (d) => {
    if (keySet != null && !keySet.has(d.key)) return false;
    if (nameSet != null && !nameSet.has(d.name)) return false;
    if (makeSet != null && !makeSet.has(d.make)) return false;
    if (modelSet != null && !modelSet.has(d.model)) return false;
    if (locationSet != null && !locationSet.has(d.location)) return false;
    if (rackSet != null && !rackSet.has(d.rack)) return false;
    if (parentSet != null && (d.parent == null || !parentSet.has(d.parent)))
      return false;
    if (req.nested != null && (d.parent != null) !== req.nested) return false;
    return inGroup(d);
  };
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  statuses: status.Client;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Omit<Device, "status">,
  Device,
  RetrieveSingleParams,
  SingleQuery
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Omit<Device, "status">>;
  /**
   * Builds requests that move devices between groups. The group must sit under each
   * device's rack.
   */
  readonly setGroupRequest: group.MoveRequest<Key>;

  constructor(cfg: ClientConfig) {
    const { cache, statuses } = cfg;
    const statusStore = statuses.store;
    const affectedDeviceKeys = (event: query.TableEvent<status.Key, status.Status>) =>
      statuses.ownerKeys(event, "device");
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
    // Composed rows carry any cached status; includeStatus only gates fetch flags.
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
      // The query gates whether the answer keeps the row's status, mirroring the
      // request space.
      compose: (records, q) =>
        q.includeStatus === true ? records[0] : stripStatus(records[0]),
      keyOf: (q) => q.key,
      // The stream and status-free fetches cache devices with no status.
      covers: (d, q) => q.includeStatus !== true || d.status != null,
      single: true,
    });
    super(cache, {
      name: "device",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
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
    this.setGroupRequest = group.createMoveRequest("/device/set-group", (keys, to) =>
      group.moveCached(store, keys, to),
    );
  }

  async retrieve<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    params: RetrieveSingleParams & RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>>;

  async retrieve(params: Key | RetrieveSingleParams): Promise<Device>;

  async retrieve<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    params: RetrieveMultipleParams & RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Array<Device<Properties, Make, Model>>>;

  async retrieve(params: RetrieveMultipleParams): Promise<Array<Device>>;

  async retrieve(
    rawParams: Key | (RetrieveParams & { schemas?: DeviceSchemas }),
  ): Promise<Device | Array<Device>> {
    const params = typeof rawParams === "string" ? { key: rawParams } : rawParams;
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

  async create<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    device: New<Properties, Make, Model>,
    schemas: DeviceSchemas<Properties, Make, Model>,
    opts?: query.WriteOptions,
  ): Promise<Device<Properties, Make, Model>>;

  async create(
    device: New,
    schemas?: DeviceSchemas,
    opts?: query.WriteOptions,
  ): Promise<Device>;

  async create<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    devices: New<Properties, Make, Model>[],
    schemas: DeviceSchemas<Properties, Make, Model>,
    opts?: query.WriteOptions,
  ): Promise<Device<Properties, Make, Model>[]>;

  async create(
    devices: New[],
    schemas?: DeviceSchemas,
    opts?: query.WriteOptions,
  ): Promise<Device[]>;

  async create(
    devices: New | New[],
    schemas?: DeviceSchemas,
    opts: query.WriteOptions = {},
  ): Promise<Device | Device[]> {
    const isSingle = !Array.isArray(devices);
    // Filling the schema defaults up front hands the cache the same records the request
    // carries. It is the schema createReqZ already applies.
    const normalized = array
      .toArray(devices)
      .map((device) => deviceZ(schemas).parse(device) as Device);
    const apply = () => [this.store.set(normalized.map(stripStatus))];
    // A status streamed during the request is newer than the reply.
    const since = this.cfg.statuses.store.stamp();
    const res = await query.optimistic({
      rollbacks: apply(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/device/create",
          { devices: normalized },
          createReqZ(schemas),
          createResZ(schemas),
        ),
    });
    res.devices.forEach(({ status: st }) => {
      if (st != null) this.cfg.statuses.store.ingest(st, { since });
    });
    this.store.set(res.devices.map(stripStatus));
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteResources(resourceID(keysArr)),
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
          emptyResZ,
        ),
    });
    drop();
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(resourceID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/device/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
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

  private compose(cached: Omit<Device, "status">, includeStatus: boolean): Device {
    if (!includeStatus) return cached;
    const st = this.cfg.statuses.ownedBy(resourceID(cached.key));
    if (st == null) return cached;
    const parsed = statusZ.safeParse(st);
    if (!parsed.success) return cached;
    return { ...cached, status: parsed.data };
  }

  private writeThrough(devices: Device[]): void {
    this.store.ingest(devices.map(stripStatus));
    devices.forEach(({ status: st }) => {
      if (st != null) this.cfg.statuses.store.ingest(st);
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

  private async fetchThrough(req: RetrieveRequest): Promise<Device[]> {
    const devices = await this.execRetrieve(req);
    this.writeThrough(devices);
    return devices;
  }

  private async fetchSingle(q: SingleQuery): Promise<Device> {
    // Status-bearing queries bypass the table, which never holds status.
    if (q.includeStatus !== true) {
      const devices = await this.store.retrieve([q.key]);
      checkForMultipleOrNoResults("Device", q, devices, true);
      return devices[0];
    }
    const devices = await this.execRetrieve(q);
    checkForMultipleOrNoResults("Device", q, devices, true);
    this.writeThrough(devices);
    return devices[0];
  }
}
