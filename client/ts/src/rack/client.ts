// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, deep, filter } from "@synnaxlabs/x";
import { z } from "zod";

import { type role } from "@/access/role";
import { actions } from "@/actions";
import { group } from "@/group";
import { query } from "@/query";
import { createOf, reduceAll } from "@/rack/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/rack/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  type Payload,
  payloadZ,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Status,
  statusZ,
} from "@/rack/types.gen";
import { LINES } from "@/rack/where";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/rack/where.gen";
import { type status } from "@/status";
import { type task } from "@/task";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const ACTION_CHANNEL_NAME = "sy_rack_action_set";
export const DELETE_CHANNEL_NAME = "sy_rack_delete";

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);
export const rackZ = payloadZ;

const singleRetrieveParamsZ = z.union([
  z
    .object({ key: keyZ, includeStatus: z.boolean().optional() })
    .transform(({ key, includeStatus }): RetrieveRequest => ({
      where: { key: { eq: key } },
      includeStatus,
    })),
  z
    .object({ name: z.string(), includeStatus: z.boolean().optional() })
    .transform(({ name, includeStatus }): RetrieveRequest => ({
      where: { name: { eq: name } },
      includeStatus,
    })),
]);
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

export type RetrieveMultipleParams = RetrieveRequest;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

type SingleQuery =
  { key: Key; includeStatus?: boolean } | { name: string; includeStatus?: boolean };

const createReqZ = z.object({ racks: newZ.array() });
const createResZ = z.object({ racks: payloadZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

/** Drops includeStatus when unset so equivalent queries hash identically. */
const normalizeSingle = (params: RetrieveSingleParams): SingleQuery => {
  const base = "key" in params ? { key: params.key } : { name: params.name };
  return params.includeStatus === true ? { ...base, includeStatus: true } : base;
};

const singleQueryZ = z
  .union([
    z.strictObject({ key: keyZ, includeStatus: z.boolean().optional() }),
    z.strictObject({ name: z.string(), includeStatus: z.boolean().optional() }),
    keyZ.transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

/** The table never holds statuses; the status table is their single home. */
const stripStatus = ({ status: _, ...rack }: Payload): Omit<Payload, "status"> => rack;

export interface ClientConfig {
  unary: UnaryClient;
  tasks: task.Client;
  cache: query.Cache;
  statuses: status.Client;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Omit<Payload, "status">,
  Rack,
  RetrieveSingleParams,
  SingleQuery,
  Key | string
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Omit<Payload, "status">>;
  private readonly dispatcher: actions.Controller<Key, Omit<Payload, "status">, Action>;
  /** Builds requests that move racks between groups. */
  readonly setGroupRequest: group.MoveRequest<Key>;

  constructor(cfg: ClientConfig) {
    const { cache, statuses } = cfg;
    const statusStore = statuses.store;
    const affectedRackKeys = (event: query.TableEvent<status.Key, status.Status>) =>
      statuses.ownerKeys(event, "rack")?.map(Number) ?? null;
    const store = cache.createTable<Key, Omit<Payload, "status">>({
      name: "racks",
      fetch: async (keys) =>
        (await this.fetchThrough(query.keysRequest(keys))).map(stripStatus),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a rack reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Omit<Payload, "status">, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/rack/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    // Composed rows carry any cached status; includeStatus only gates fetch flags.
    const composed = cache.derive<Key, Omit<Payload, "status">, Rack>({
      name: "rack.composed",
      source: store,
      compose: (record) => this.compose(record, true),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      triggers: [query.deriveTrigger(statusStore, (event) => affectedRackKeys(event))],
    });
    statuses.holdOwned(store, resourceID);
    const single = cache.queries<SingleQuery, Rack, Key, Rack>({
      name: "rack",
      table: composed,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => ("key" in q ? q.key : null),
      // The stream and status-free fetches cache racks with no status.
      covers: (r, q) => q.includeStatus !== true || r.status != null,
      matches: (r, q) => ("key" in q ? r.key === q.key : r.name === q.name),
      single: true,
    });
    super(cache, {
      name: "rack",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => (await this.fetchThrough(req)).map(stripStatus),
        matches: (r, req) => matches(LINES, r, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
        triggers: [
          query.trigger(
            statusStore,
            (event, q: RetrieveRequest) => {
              if (q.includeStatus !== true) return null;
              return affectedRackKeys(event);
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
    this.setGroupRequest = group.createMoveRequest("/rack/dispatch", (keys, to) =>
      group.moveCached(store, keys, to),
    );
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/rack/delete",
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
   * Applies the actions to the cached rack until the Core echoes them, and sends them
   * to the Core to apply in one transaction. Actions hold the snake_case form that the
   * Core stores. A refused send drops the actions from the cached rack.
   * @throws {Error} if the cached rack cannot hold an action, before any send.
   * @throws {ValidationError} if the rack cannot hold an action.
   * @throws {NotFoundError} if the rack does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  async create(rack: New): Promise<Rack>;
  async create(racks: New[]): Promise<Rack[]>;
  async create(rack: New | New[]): Promise<Rack | Rack[]> {
    const isSingle = !Array.isArray(rack);
    const { store: statuses } = this.cfg.statuses;
    // A status streamed during the request is newer than the reply.
    const res = await statuses.track(async (since) => {
      const res = await this.cfg.unary.send(
        "/rack/create",
        { racks: array.toArray(rack) },
        createReqZ,
        createResZ,
      );
      res.racks.forEach(({ status: st }) => {
        if (st != null) statuses.ingest(st, { since });
      });
      return res;
    });
    this.store.set(res.racks.map(stripStatus));
    const sugared = this.sugar(res.racks);
    return isSingle ? sugared[0] : sugared;
  }

  sugar(payload: Payload): Rack;
  sugar(payloads: Payload[]): Rack[];
  sugar(payloads: Payload | Payload[]): Rack | Rack[] {
    const isSingle = !Array.isArray(payloads);
    const sugared = array
      .toArray(payloads)
      .map(
        ({ key, name, role, status, integrations, embedded, version, group: g }) =>
          new Rack(
            key,
            name,
            role,
            this.cfg.tasks,
            status,
            integrations,
            embedded,
            version,
            g,
          ),
      );
    return isSingle ? sugared[0] : sugared;
  }

  private compose(cached: Omit<Payload, "status">, includeStatus: boolean): Rack {
    if (!includeStatus) return this.sugar(cached);
    const st = this.statusOf(cached.key);
    if (st == null) return this.sugar(cached);
    return this.sugar({ ...cached, status: st });
  }

  private statusOf(key: Key): Status | undefined {
    const parsed = statusZ.safeParse(this.cfg.statuses.ownedBy(resourceID(key)));
    return parsed.success ? parsed.data : undefined;
  }

  private async execRetrieve(params: RetrieveParams): Promise<Payload[]> {
    const res = await this.cfg.unary.send(
      "/rack/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ,
    );
    return res.racks;
  }

  private async fetchThrough(req: RetrieveParams): Promise<Payload[]> {
    const { store: statuses } = this.cfg.statuses;
    return await this.store.track(
      async (since) =>
        await statuses.track(async (statusSince) => {
          const racks = await this.execRetrieve(req);
          this.store.ingest(racks.map(stripStatus), { since });
          racks.forEach(({ status: st }) => {
            if (st != null) statuses.ingest(st, { since: statusSince });
          });
          return racks;
        }),
    );
  }

  private async fetchSingle(q: SingleQuery): Promise<Rack> {
    // Names are not unique and the table never holds status, so only keyed status-free
    // queries can be served from it.
    if ("key" in q && q.includeStatus !== true) {
      const racks = await this.store.retrieve([q.key]);
      checkForMultipleOrNoResults(
        "rack",
        { where: { key: { eq: q.key } } },
        racks,
        true,
      );
      return this.sugar(racks[0]);
    }
    const racks = await this.fetchThrough(q);
    const where = "key" in q ? { key: { eq: q.key } } : { name: { eq: q.name } };
    checkForMultipleOrNoResults("rack", { where }, racks, true);
    return this.sugar(racks[0]);
  }
}

export class Rack {
  key: Key;
  name: string;
  status?: Status;
  integrations: string[];
  embedded: boolean;
  version?: string;
  group?: group.Key;
  role: role.Key;
  private readonly tasks: task.Client;

  constructor(
    key: Key,
    name: string,
    role: role.Key,
    taskClient: task.Client,
    status?: Status,
    integrations: string[] = [],
    embedded: boolean = false,
    version?: string,
    groupKey?: group.Key,
  ) {
    this.key = key;
    this.name = name;
    this.role = role;
    this.tasks = taskClient;
    this.status = status;
    this.integrations = integrations;
    this.embedded = embedded;
    this.version = version;
    this.group = groupKey;
  }

  async listTasks(): Promise<task.Task[]> {
    return await this.tasks.retrieve({ rack: this.key });
  }

  async createTask(task: task.New): Promise<task.Task>;
  async createTask<Schemas extends task.Schemas = task.Schemas>(
    task: task.New<Schemas>,
    schemas: Schemas,
  ): Promise<task.Task<Schemas>>;

  async createTask<Schemas extends task.Schemas = task.Schemas>(
    task: task.New<Schemas>,
    schemas?: Schemas,
  ): Promise<task.Task<Schemas>> {
    task.rack = this.key;
    return await this.tasks.create(task, schemas as Required<Schemas>);
  }

  async deleteTask(task: task.Key): Promise<void> {
    await this.tasks.delete([task]);
  }

  get payload(): Payload {
    return {
      key: this.key,
      name: this.name,
      status: this.status,
      integrations: this.integrations,
      embedded: this.embedded,
      version: this.version,
      group: this.group,
      role: this.role,
    };
  }
}
