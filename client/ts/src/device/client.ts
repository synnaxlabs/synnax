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
  type RetrieveRequest,
  retrieveRequestZ,
  statusZ,
} from "@/device/types.gen";
import { matches, retrieveInputZ, serverComputed } from "@/device/where.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { type status } from "@/status";
import { checkForMultipleOrNoResults, singleQueryError } from "@/util/retrieve";

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

const filterParamsZ = retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape);
type FilterParams = query.RequestInput<typeof filterParamsZ>;

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  retrieveRequestZ,
  filterParamsZ,
]);

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
  .strictObject({ key: keyZ, includeStatus: z.boolean().optional() })
  .transform(({ key, includeStatus }): RetrieveRequest => ({
    where: { key: { eq: key } },
    includeStatus,
  }));

export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = RetrieveRequest;

const retrieveParamsZ = z.union([
  singleRetrieveParamsZ,
  retrieveRequestZ,
  filterParamsZ,
]);

export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

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
  .union([
    z.strictObject({ key: keyZ, includeStatus: z.boolean().optional() }),
    keyZ.transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

/** The table never holds statuses; the status table is their single home. */
const stripStatus = ({ status: _, ...device }: Device): Omit<Device, "status"> =>
  device;

/**
 * Removes the fields that make the Core write a device's status and parent. Use it to
 * send a retrieved device or a create reply back to {@link Client.create}: a reply
 * carries a parent, and the Core writes the status of any device sent with one.
 */
export const toUpdate = <D extends Pick<Device, "status" | "parent">>({
  status: _status,
  parent: _parent,
  ...device
}: D): Omit<D, "status" | "parent"> => device;

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
  typeof retrieveMultiParamsZ,
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
      fetch: async (keys) => await this.fetchThrough(query.keysRequest(keys)),
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
        matches: (d, req) => matches(d, req.where),
        serverComputed: (req) => serverComputed(req.where),
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

  async retrieve(params: Key | RetrieveSingleParams): Promise<Device>;

  async retrieve<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    params: (RetrieveMultipleParams | FilterParams) &
      RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Array<Device<Properties, Make, Model>>>;

  async retrieve(
    params: Key[] | query.RequestInput<typeof retrieveMultiParamsZ>,
  ): Promise<Array<Device>>;

  async retrieve(
    rawParams:
      Key | Key[] | ((RetrieveParams | FilterParams) & { schemas?: DeviceSchemas }),
  ): Promise<Device | Array<Device>> {
    if (Array.isArray(rawParams)) return await super.retrieve(rawParams);
    const params = typeof rawParams === "string" ? { key: rawParams } : rawParams;
    const { schemas, ...rest } = params;
    const isSingle = z.validate(singleRetrieveParamsZ, rest);
    // Schemas are not hashable, so schema-typed retrieves bypass the query
    // cache and hit the network, still writing records through.
    if (schemas != null) {
      if (!isSingle && "key" in rest && typeof rest.key === "string") {
        const res = singleQueryZ.safeParse(rest);
        if (!res.success) throw singleQueryError("device", rest, res.error);
      }
      const devices = await this.fetchThrough(rest, schemas);
      checkForMultipleOrNoResults("Device", rest, devices, isSingle);
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
    const since = this.cfg.statusStore.stamp();
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
      if (st != null) this.cfg.statusStore.ingest(st, { since });
    });
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
          emptyResZ,
        ),
    });
    drop();
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
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
    const st = this.cfg.statusStore.get(statusKey(cached.key));
    if (st == null) return cached;
    const parsed = statusZ.safeParse(st);
    if (!parsed.success) return cached;
    return { ...cached, status: parsed.data };
  }

  private async execRetrieve(
    params: RetrieveParams | FilterParams,
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

  private async fetchThrough(
    req: RetrieveParams | FilterParams,
    schemas?: DeviceSchemas,
  ): Promise<Device[]> {
    const since = this.store.stamp();
    const statusSince = this.cfg.statusStore.stamp();
    const devices = await this.execRetrieve(req, schemas);
    this.store.ingest(devices.map(stripStatus), { since });
    devices.forEach(({ status: st }) => {
      if (st != null) this.cfg.statusStore.ingest(st, { since: statusSince });
    });
    return devices;
  }

  private async fetchSingle(q: SingleQuery): Promise<Device> {
    // Status-bearing queries bypass the table, which never holds status.
    if (q.includeStatus !== true) {
      const devices = await this.store.retrieve([q.key]);
      checkForMultipleOrNoResults("Device", q, devices, true);
      return devices[0];
    }
    const devices = await this.fetchThrough(q);
    checkForMultipleOrNoResults("Device", q, devices, true);
    return devices[0];
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
