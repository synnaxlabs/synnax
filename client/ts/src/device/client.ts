// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, filter, type record, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { createOf, reduceAll } from "@/device/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/device/actions.gen";
import {
  type Device,
  type DeviceSchemas,
  deviceZ,
  type Key,
  keyZ,
  type New,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  statusZ,
} from "@/device/types.gen";
import { LINES } from "@/device/where";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/device/where.gen";
import { group } from "@/group";
import { query } from "@/query";
import { type status } from "@/status";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const ACTION_CHANNEL_NAME = "sy_device_action_set";
export const DELETE_CHANNEL_NAME = "sy_device_delete";

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

const filterParamsZ = retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape);
type FilterParams = query.RequestInput<typeof filterParamsZ>;
const requestParamsZ = filter.requestInputZ(retrieveRequestZ, filterParamsZ);

const retrieveMultiParamsZ = z.union([query.keysRequestZ(keyZ), requestParamsZ]);

type OneParams = query.OneInput<typeof retrieveMultiParamsZ>;

const isOneParams = (params: object): params is RetrieveSingleParams | OneParams =>
  query.addressesOne(params);

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

const retrieveParamsZ = z.union([singleRetrieveParamsZ, requestParamsZ]);

export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

type SingleQuery = { key: Key; includeStatus?: boolean };

type RetrieveSchemas<
  Properties extends z.ZodType<record.Unknown>,
  Make extends z.ZodType<string>,
  Model extends z.ZodType<string>,
> = { schemas: DeviceSchemas<Properties, Make, Model> };

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "after"] as const;

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
 * Removes the status of a device. A create of the result stores the default status of a
 * new device, not a status read from the Core.
 */
export const toUpdate = <D extends Pick<Device, "status">>({
  status: _,
  ...device
}: D): Omit<D, "status"> => device;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  statuses: status.Client;
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
  private readonly dispatcher: actions.Controller<Key, Omit<Device, "status">, Action>;
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
      fetch: async (keys) =>
        (
          await this.fetchThrough({ ...query.keysRequest(keys), includeStatus: true })
        ).map(stripStatus),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a device reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Omit<Device, "status">, Action>({
      store,
      listed: (action) =>
        action.type === "set_rack" || actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/device/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    // Composed rows carry any cached status; includeStatus only gates fetch flags.
    const composed = cache.derive<Key, Omit<Device, "status">, Device>({
      name: "device.composed",
      source: store,
      compose: (record) => this.compose(record, true),
      triggers: [
        query.deriveTrigger(statusStore, (event) => affectedDeviceKeys(event)),
      ],
    });
    statuses.holdOwned(store, resourceID);
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
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => (await this.fetchThrough(req)).map(stripStatus),
        matches: (d, req) => matches(LINES, d, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
        serverFields: SERVER_FIELDS,
        triggers: [
          query.trigger(
            statusStore,
            (event, q: RetrieveRequest) => {
              if (q.includeStatus !== true) return null;
              return affectedDeviceKeys(event);
            },
            { contentOnly: true },
          ),
        ],
      },
      compose: (record, q) =>
        this.compose(record, (q as { includeStatus?: boolean }).includeStatus === true),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
    this.setGroupRequest = group.createMoveRequest("/device/dispatch", (keys, to) =>
      group.moveCached(store, keys, to),
    );
  }

  async retrieve<
    Properties extends z.ZodType<record.Unknown>,
    Make extends z.ZodType<string>,
    Model extends z.ZodType<string>,
  >(
    params: (RetrieveSingleParams | OneParams) &
      RetrieveSchemas<Properties, Make, Model>,
  ): Promise<Device<Properties, Make, Model>>;

  async retrieve(params: Key | RetrieveSingleParams | OneParams): Promise<Device>;

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
      | Key
      | Key[]
      | ((RetrieveParams | OneParams | FilterParams) & { schemas?: DeviceSchemas }),
  ): Promise<Device | Array<Device>> {
    if (Array.isArray(rawParams)) return await super.retrieve(rawParams);
    const params = typeof rawParams === "string" ? { key: rawParams } : rawParams;
    const { schemas, ...rest } = params;
    const isSingle = isOneParams(rest);
    // Schemas are not hashable, so schema-typed retrieves bypass the query
    // cache and hit the network, still writing records through.
    if (schemas != null) {
      const request = retrieveParamsZ.parse(query.dropOne(rest));
      const devices = await this.fetchThrough(request, schemas);
      checkForMultipleOrNoResults("device", request, devices, isSingle);
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
    const { store: statuses } = this.cfg.statuses;
    // A status streamed during the request is newer than the reply.
    const res = await statuses.track(async (since) => {
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
        if (st != null) statuses.ingest(st, { since });
      });
      return res;
    });
    this.store.set(res.devices.map(stripStatus));
    return isSingle ? res.devices[0] : res.devices;
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
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

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached device until the Core echoes them, and sends them
   * to the Core to apply in one transaction. Actions hold the snake_case form that the
   * Core stores. A refused send drops the actions from the cached device.
   * @throws {Error} if the cached device cannot hold an action, before any send.
   * @throws {ValidationError} if the device cannot hold an action.
   * @throws {NotFoundError} if the device does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
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

  private async execRetrieve(
    params: RetrieveParams | OneParams | FilterParams,
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
    req: RetrieveParams | OneParams | FilterParams,
    schemas?: DeviceSchemas,
  ): Promise<Device[]> {
    const { store: statuses } = this.cfg.statuses;
    return await this.store.track(
      async (since) =>
        await statuses.track(async (statusSince) => {
          const devices = await this.execRetrieve(req, schemas);
          this.store.ingest(devices.map(stripStatus), { since });
          devices.forEach(({ status: st }) => {
            if (st != null) statuses.ingest(st, { since: statusSince });
          });
          return devices;
        }),
    );
  }

  private async fetchSingle(q: SingleQuery): Promise<Omit<Device, "status">> {
    // The refresh covers a cached device whose status has not synced.
    const devices = await this.store.retrieve([q.key], {
      refresh: q.includeStatus === true,
    });
    checkForMultipleOrNoResults(
      "device",
      { where: { key: { eq: q.key } } },
      devices,
      true,
    );
    return devices[0];
  }
}
