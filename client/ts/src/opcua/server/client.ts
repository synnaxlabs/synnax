// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, filter } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Server,
  serverZ,
  statusZ,
  type Update,
  updateZ,
} from "@/opcua/server/types.gen";
import {
  compareCursors,
  cursor,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/opcua/server/where.gen";
import { query } from "@/query";
import type { status } from "@/status";
import { util } from "@/util";

export const SET_CHANNEL_NAME = "sy_opcua_server_set";
export const DELETE_CHANNEL_NAME = "sy_opcua_server_delete";

type Stored = Omit<Server, "status">;

const createReqZ = z.object({ servers: serverZ.array() });
const createResZ = z.object({ servers: serverZ.array() });
const updateReqZ = z.object({ key: keyZ, update: updateZ });
const deleteReqZ = z.object({ keys: keyZ.array() });
const updateResZ = z.object({ server: serverZ });
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

const stripStatus = ({ status: _, ...server }: Server): Stored => server;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  statuses: status.Client;
}

/** Client creates, retrieves, updates, and deletes OPC UA servers. */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Stored,
  Server,
  RetrieveSingleParams,
  SingleQuery
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Stored>;

  constructor(cfg: ClientConfig) {
    const { cache, statuses } = cfg;
    const statusStore = statuses.store;
    const affectedKeys = (event: query.TableEvent<status.Key, status.Status>) =>
      statuses.ownerKeys(event, "opcua_server");
    // The status table is the single home of statuses.
    const store = cache.createTable<Key, Stored>({
      name: "opcua.servers",
      fetch: async (keys) =>
        (
          await this.fetchThrough({ ...query.keysRequest(keys), includeStatus: true })
        ).map(stripStatus),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, serverZ, { value: stripStatus }),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    const composed = cache.derive<Key, Stored, Server>({
      name: "opcua.server.composed",
      source: store,
      compose: (record) => this.compose(record, true),
      triggers: [query.deriveTrigger(statusStore, (event) => affectedKeys(event))],
    });
    statuses.holdOwned(store, resourceID);
    const single = cache.queries<SingleQuery, Server, Key, Server>({
      name: "opcua.server",
      table: composed,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records, q) =>
        q.includeStatus === true ? records[0] : stripStatus(records[0]),
      keyOf: (q) => q.key,
      // The stream and status-free fetches cache servers with no status.
      covers: (d, q) => q.includeStatus !== true || d.status != null,
      single: true,
    });
    super(cache, {
      name: "opcua.server",
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
   * Creates the servers, or replaces the servers whose keys exist. The Core gives each
   * new server an unknown status.
   * @throws {ValidationError} if a server is invalid, or if its rack, password, or
   * certificates do not exist.
   */
  async create(server: New, opts?: query.WriteOptions): Promise<Server>;
  async create(servers: New[], opts?: query.WriteOptions): Promise<Server[]>;
  async create(
    servers: New | New[],
    opts: query.WriteOptions = {},
  ): Promise<Server | Server[]> {
    const isSingle = !Array.isArray(servers);
    const normalized = array.toArray(servers).map((s) => serverZ.parse(s));
    const apply = () => [this.store.set(normalized.map(stripStatus))];
    const { store: statuses } = this.cfg.statuses;
    // A status streamed during the request is newer than the reply.
    const res = await statuses.track(async (since) => {
      const res = await query.optimistic({
        rollbacks: apply(),
        onOptimistic: opts.onOptimistic,
        commit: async () =>
          await this.cfg.unary.send(
            "/opcua/server/create",
            { servers: normalized },
            createReqZ,
            createResZ,
          ),
      });
      res.servers.forEach(({ status: st }) => {
        if (st != null) statuses.ingest(st, { since });
      });
      return res;
    });
    this.store.set(res.servers.map(stripStatus));
    return isSingle ? res.servers[0] : res.servers;
  }

  /**
   * Writes one field group to the server and keeps the other as stored. The update
   * replaces the whole group, so a field it leaves out is cleared, such as the writes
   * of a configuration update that carries only reads.
   * @returns the server as the Core stored it, without its status.
   * @throws {NotFoundError} if the server does not exist.
   * @throws {ValidationError} if the update is invalid.
   */
  async update(key: Key, update: Update): Promise<Server> {
    const { server } = await this.cfg.unary.send(
      "/opcua/server/update",
      { key, update },
      updateReqZ,
      updateResZ,
    );
    // The reply replaces the cached server, even when an older stream set arrived
    // during the request.
    this.store.set([stripStatus(server)]);
    return server;
  }

  /** Deletes the servers with their statuses. */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/opcua/server/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Subscribes to every server set delivered to the cache. Statuses excluded. */
  onSet(handler: (server: Stored) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every server delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private compose(cached: Stored, includeStatus: boolean): Server {
    if (!includeStatus) return cached;
    const parsed = statusZ.safeParse(this.cfg.statuses.ownedBy(resourceID(cached.key)));
    return parsed.success ? { ...cached, status: parsed.data } : cached;
  }

  private async fetchThrough(
    req: RetrieveParams | OneParams | FilterParams,
  ): Promise<Server[]> {
    const { store: statuses } = this.cfg.statuses;
    return await this.store.track(
      async (since) =>
        await statuses.track(async (statusSince) => {
          const { servers } = await this.cfg.unary.send(
            "/opcua/server/retrieve",
            req,
            retrieveParamsZ,
            retrieveResponseZ,
          );
          this.store.ingest(servers.map(stripStatus), { since });
          servers.forEach(({ status: st }) => {
            if (st != null) statuses.ingest(st, { since: statusSince });
          });
          return servers;
        }),
    );
  }

  private async fetchSingle(q: SingleQuery): Promise<Stored> {
    // The refresh covers a cached server whose status has not synced.
    const servers = await this.store.retrieve([q.key], {
      refresh: q.includeStatus === true,
    });
    util.checkForMultipleOrNoResults(
      "OPC UA server",
      { where: { key: { eq: q.key } } },
      servers,
      true,
    );
    return servers[0];
  }
}
