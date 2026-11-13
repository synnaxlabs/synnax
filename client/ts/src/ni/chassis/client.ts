// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, filter } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Chassis,
  chassisZ,
  type Key,
  keyZ,
  type New,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  statusZ,
  type Update,
  updateZ,
} from "@/ni/chassis/types.gen";
import {
  compareCursors,
  cursor,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/ni/chassis/where.gen";
import { query } from "@/query";
import { type status } from "@/status";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_ni_chassis_set";
export const DELETE_CHANNEL_NAME = "sy_ni_chassis_delete";

type Stored = Omit<Chassis, "status">;

const createReqZ = z.object({ chassis: chassisZ.array() });
const createResZ = z.object({ chassis: chassisZ.array() });
const updateReqZ = z.object({ key: keyZ, update: updateZ });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const filterParamsZ = retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape);
type FilterParams = query.RequestInput<typeof filterParamsZ>;
const requestParamsZ = filter.requestInputZ(retrieveRequestZ, filterParamsZ);

const retrieveMultiParamsZ = z.union([query.keysRequestZ(keyZ), requestParamsZ]);

type OneParams = query.OneInput<typeof retrieveMultiParamsZ>;

const singleRetrieveParamsZ = z
  .strictObject({ key: keyZ, includeStatus: z.boolean().optional() })
  .transform(({ key, includeStatus }): RetrieveRequest => ({
    where: { key: { eq: key } },
    includeStatus,
  }));

export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, requestParamsZ]);

type RetrieveParams = RetrieveSingleParams | RetrieveRequest;

type SingleQuery = { key: Key; includeStatus?: boolean };

const includesStatusZ = z.object({ includeStatus: z.literal(true) });

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

const stripStatus = ({ status: _, ...chassis }: Chassis): Stored => chassis;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  statuses: status.Client;
}

/** Client creates, retrieves, updates, and deletes NI chassis. */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Stored,
  Chassis,
  RetrieveSingleParams,
  SingleQuery
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Stored>;

  constructor(cfg: ClientConfig) {
    const { cache, statuses } = cfg;
    const statusStore = statuses.store;
    const affectedKeys = (event: query.TableEvent<status.Key, status.Status>) =>
      statuses.ownerKeys(event, "ni_chassis");
    // The status table is the single home of statuses.
    const store = cache.createTable<Key, Stored>({
      name: "ni.chassis",
      fetch: async (keys) =>
        (
          await this.fetchThrough({ ...query.keysRequest(keys), includeStatus: true })
        ).map(stripStatus),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, chassisZ, { value: stripStatus }),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    const composed = cache.derive<Key, Stored, Chassis>({
      name: "ni.chassis.composed",
      source: store,
      compose: (record) => this.compose(record, true),
      triggers: [query.deriveTrigger(statusStore, (event) => affectedKeys(event))],
    });
    statuses.holdOwned(store, resourceID);
    const single = cache.queries<SingleQuery, Chassis, Key, Chassis>({
      name: "ni.chassis",
      table: composed,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records, q) =>
        q.includeStatus === true ? records[0] : stripStatus(records[0]),
      keyOf: (q) => q.key,
      // The stream and status-free fetches cache chassis with no status.
      covers: (d, q) => q.includeStatus !== true || d.status != null,
      single: true,
    });
    super(cache, {
      name: "ni.chassis",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => (await this.fetchThrough(req)).map(stripStatus),
        matches: (d, req) => matches(d, req.where),
        serverComputed: (req) => serverComputed(req.where),
        serverFields: SERVER_FIELDS,
        triggers: [
          query.trigger(
            statusStore,
            (event, q: RetrieveRequest) =>
              q.includeStatus === true ? affectedKeys(event) : null,
            { contentOnly: true },
          ),
        ],
      },
      compose: (record, q) =>
        this.compose(record, includesStatusZ.safeParse(q).success),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
  }

  /**
   * Creates the chassis, or replaces the chassis whose keys exist. The Core gives each
   * new chassis an unknown status.
   * @throws {ValidationError} if a chassis is invalid or its rack does not exist.
   */
  async create(chassis: New): Promise<Chassis>;
  async create(chassis: New[]): Promise<Chassis[]>;
  async create(chassis: New | New[]): Promise<Chassis | Chassis[]> {
    const isSingle = !Array.isArray(chassis);
    const { store: statuses } = this.cfg.statuses;
    // A chassis or status streamed during the request is newer than the reply.
    const res = await this.store.track(
      async (since) =>
        await statuses.track(async (statusSince) => {
          const res = await this.cfg.unary.send(
            "/ni/chassis/create",
            { chassis: array.toArray(chassis).map((c) => chassisZ.parse(c)) },
            createReqZ,
            createResZ,
          );
          const stored = res.chassis.map(stripStatus);
          // Only a set revives a key this client deleted.
          const revived = stored.filter(
            ({ key }) => this.store.getTombstone(key) != null,
          );
          this.store.set(revived);
          this.store.ingest(
            stored.filter((c) => !revived.includes(c)),
            { since, mode: "set", changed: true },
          );
          res.chassis.forEach(({ status: st }) => {
            if (st != null) statuses.ingest(st, { since: statusSince });
          });
          return res;
        }),
    );
    return isSingle ? res.chassis[0] : res.chassis;
  }

  /**
   * Writes one field group to the chassis and keeps the other as stored. The update
   * replaces the whole group, so a field it leaves out is cleared, such as the serial
   * of an identity update.
   * @throws {NotFoundError} if the chassis does not exist.
   * @throws {ValidationError} if the update is invalid.
   */
  async update(key: Key, update: Update): Promise<void> {
    await this.cfg.unary.send(
      "/ni/chassis/update",
      { key, update },
      updateReqZ,
      emptyResZ,
    );
    await this.fetchThrough(query.keysRequest([key]));
  }

  /**
   * Deletes the chassis with their statuses and the devices they hold. The devices
   * leave the device cache when their deletes stream in.
   */
  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.cfg.unary.send(
      "/ni/chassis/delete",
      { keys: keysArr },
      deleteReqZ,
      emptyResZ,
    );
    this.store.delete(keysArr);
  }

  /** Subscribes to every chassis set delivered to the cache. Statuses excluded. */
  onSet(handler: (chassis: Stored) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every chassis delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private compose(cached: Stored, includeStatus: boolean): Chassis {
    if (!includeStatus) return cached;
    const parsed = statusZ.safeParse(this.cfg.statuses.ownedBy(resourceID(cached.key)));
    return parsed.success ? { ...cached, status: parsed.data } : cached;
  }

  private async fetchThrough(
    req: RetrieveParams | OneParams | FilterParams,
  ): Promise<Chassis[]> {
    const { store: statuses } = this.cfg.statuses;
    return await this.store.track(
      async (since) =>
        await statuses.track(async (statusSince) => {
          const { chassis } = await this.cfg.unary.send(
            "/ni/chassis/retrieve",
            req,
            retrieveParamsZ,
            retrieveResponseZ,
          );
          this.store.ingest(chassis.map(stripStatus), { since });
          chassis.forEach(({ status: st }) => {
            if (st != null) statuses.ingest(st, { since: statusSince });
          });
          return chassis;
        }),
    );
  }

  private async fetchSingle(q: SingleQuery): Promise<Stored> {
    // The refresh covers a cached chassis whose status has not synced.
    const chassis = await this.store.retrieve([q.key], {
      refresh: q.includeStatus === true,
    });
    checkForMultipleOrNoResults(
      "NI chassis",
      { where: { key: { eq: q.key } } },
      chassis,
      true,
    );
    return chassis[0];
  }
}
