// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { ontology } from "@/ontology";
import { bindStore, STORE_KEY } from "@/rack/store";
import {
  type Key,
  keyZ,
  type New,
  newZ,
  ontologyID,
  type Payload,
  payloadZ,
  type Status,
  statusZ,
} from "@/rack/types.gen";
import { status } from "@/status";
import { type task } from "@/task";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  integration: z.string().optional(),
  searchTerm: z.string().optional(),
  embedded: z.boolean().optional(),
  hostIsNode: z.boolean().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
});
const retrieveResZ = z.object({ racks: payloadZ.array().default(() => []) });
export const rackZ = payloadZ;

const singleRetrieveParamsZ = z.union([
  z
    .object({
      key: keyZ,
      includeStatus: z.boolean().optional(),
    })
    .transform(({ key, includeStatus }) => ({ keys: [key], includeStatus })),
  z
    .object({
      name: z.string(),
      includeStatus: z.boolean().optional(),
    })
    .transform(({ name, includeStatus }) => ({ names: [name], includeStatus })),
]);
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

const multiRetrieveParamsZ = retrieveReqZ;

export type RetrieveMultipleParams = z.input<typeof multiRetrieveParamsZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, multiRetrieveParamsZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

type SingleQuery =
  { key: Key; includeStatus?: boolean } | { name: string; includeStatus?: boolean };

const createReqZ = z.object({ racks: newZ.array() });
const createResZ = z.object({ racks: payloadZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});

const MOUNT_SCOPE = "rack.mounts";

/** Drops includeStatus when unset so equivalent queries hash identically. */
const normalizeSingle = (params: RetrieveSingleParams): SingleQuery => {
  const base = "key" in params ? { key: params.key } : { name: params.name };
  return params.includeStatus === true ? { ...base, includeStatus: true } : base;
};

/** The store never holds statuses; the status store is their single home. */
const stripStatus = ({ status: _, ...rack }: Payload): Omit<Payload, "status"> => rack;

const isSingleParams = (params: RetrieveParams): params is RetrieveSingleParams =>
  "key" in params || "name" in params;

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.names == null &&
  req.integration == null &&
  req.searchTerm == null &&
  req.embedded == null &&
  req.hostIsNode == null &&
  req.limit == null &&
  req.offset == null &&
  req.includeStatus !== true;

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key and name sets and payload-held flags, permissive for server-computed
 * shapes (search, host resolution), which accept every change and drift
 * toward the server's answer.
 */
const requestFilter = (
  req: RetrieveRequest,
): ((r: Omit<Payload, "status">) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
  return (r) => {
    if (keySet != null && !keySet.has(r.key)) return false;
    if (nameSet != null && !nameSet.has(r.name)) return false;
    if (req.integration != null && !r.integrations.includes(req.integration))
      return false;
    if (req.embedded != null && r.embedded !== req.embedded) return false;
    return true;
  };
};

export class Client {
  private readonly client: UnaryClient;
  private readonly tasks: task.Client;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<SingleQuery, Rack>;
    request: cache.Queries<RetrieveRequest, Rack[]>;
  };

  constructor(client: UnaryClient, taskClient: task.Client, engine?: cache.Engine) {
    this.client = client;
    this.tasks = taskClient;
    if (engine == null) return;
    bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "rack",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "racks",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
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
          "/rack/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    );
    this.writes?.delete(keysArr);
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(cache.partialUpdate(writes, key, { name }));
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      const r = await this.retrieve({ key });
      await this.create({ ...r.payload, name });
    });
  }

  async create(rack: New): Promise<Rack>;
  async create(racks: New[]): Promise<Rack[]>;
  async create(rack: New | New[]): Promise<Rack | Rack[]> {
    const isSingle = !Array.isArray(rack);
    const res = await this.client.send(
      "/rack/create",
      { racks: array.toArray(rack) },
      createReqZ,
      createResZ,
    );
    this.writes?.set(res.racks.map(stripStatus));
    const sugared = this.sugar(res.racks);
    return isSingle ? sugared[0] : sugared;
  }

  async retrieve(params: RetrieveSingleParams): Promise<Rack>;
  async retrieve(params: RetrieveMultipleParams): Promise<Rack[]>;
  async retrieve(params: RetrieveParams): Promise<Rack | Rack[]> {
    const isSingle = isSingleParams(params);
    if (this.queries_ == null) {
      const sugared = this.sugar(await this.execRetrieve(params));
      checkForMultipleOrNoResults("Rack", params, sugared, isSingle);
      return isSingle ? sugared[0] : sugared;
    }
    if (isSingleParams(params))
      return await this.queries_.single.retrieve(normalizeSingle(params));
    return await this.queries_.request.retrieve(multiRetrieveParamsZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a rack; every other shape delivers the matching racks.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Rack>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Rack[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Rack> | cache.ChangeHandler<Rack[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if (isSingleParams(params))
      return queries.single.onChange(
        normalizeSingle(params),
        handler as cache.ChangeHandler<Rack>,
      );
    return queries.request.onChange(
      multiRetrieveParamsZ.parse(params),
      handler as cache.ChangeHandler<Rack[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Rack> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Rack[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Rack> | cache.Cached<Rack[]> | undefined {
    const queries = this.requireQueries();
    if (isSingleParams(params))
      return queries.single.getCached(normalizeSingle(params));
    return queries.request.getCached(multiRetrieveParamsZ.parse(params));
  }

  sugar(payload: Payload): Rack;
  sugar(payloads: Payload[]): Rack[];
  sugar(payloads: Payload | Payload[]): Rack | Rack[] {
    const isSingle = !Array.isArray(payloads);
    const sugared = array
      .toArray(payloads)
      .map(
        ({ key, name, status, integrations, taskCounter, embedded }) =>
          new Rack(key, name, this.tasks, status, integrations, taskCounter, embedded),
      );
    return isSingle ? sugared[0] : sugared;
  }

  private get writes(): cache.UnaryStore<Key, Omit<Payload, "status">> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get rackStore(): cache.UnaryStore<Key, Omit<Payload, "status">> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get rackEvents(): cache.UnaryStore<Key, Omit<Payload, "status">> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private get statusEvents(): cache.UnaryStore<status.Key, status.Status> {
    return this.requireEngine().store(status.STORE_KEY, MOUNT_SCOPE);
  }

  /** Fires for streamed statuses that parse as rack statuses. */
  private onRackStatusSet(apply: (changed: Status) => void): destructor.Destructor {
    return this.statusEvents.onSet((changed) => {
      const parsed = statusZ.safeParse(changed);
      if (parsed.success) apply(parsed.data);
    });
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

  private async execRetrieve(params: RetrieveParams): Promise<Payload[]> {
    const res = await this.client.send(
      "/rack/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.racks;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Rack[]> {
    const results: Rack[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.rackStore.get(key);
      if (cached != null) results.push(this.sugar(cached));
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.rackStore.set(fetched.map(stripStatus));
      results.push(...this.sugar(fetched));
    }
    return cache.orderByKeys(keys, results, (r) => r.key);
  }

  private async fetchSingle(query: SingleQuery): Promise<Rack> {
    // Names are not unique, so only key queries can be served from the store.
    // Status-bearing answers cannot be served from the status-less store.
    if ("key" in query && query.includeStatus !== true) {
      const cached = this.rackStore.get(query.key);
      if (cached != null) return this.sugar(cached);
    }
    const racks = await this.execRetrieve(query);
    checkForMultipleOrNoResults("Rack", query, racks, true);
    this.rackStore.set(racks.map(stripStatus));
    return this.sugar(racks[0]);
  }

  private mountSingle({ query, update, remove }: cache.MountParams<SingleQuery, Rack>) {
    const matches = (r: Omit<Payload, "status">) =>
      "key" in query ? r.key === query.key : r.name === query.name;
    const listeners = [
      this.rackEvents.onSet((rack) => {
        // Store events carry no status; merging keeps a status-bearing answer.
        if (matches(rack)) update((prev) => this.sugar({ ...prev?.payload, ...rack }));
      }),
      this.rackEvents.onDelete((key) => {
        const corpse = this.rackStore.getTombstone(key)?.corpse;
        const deleted =
          "key" in query ? key === query.key : corpse != null && matches(corpse);
        if (deleted) remove(corpse == null ? undefined : this.sugar(corpse));
      }),
    ];
    if (query.includeStatus === true)
      listeners.push(
        this.onRackStatusSet((changed) =>
          update((prev) => {
            if (prev == null || prev.key !== changed.details.rack) return prev;
            return this.sugar({ ...prev.payload, status: changed });
          }),
        ),
      );
    return listeners;
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Rack[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const racks = await this.execRetrieve(query);
    this.rackStore.set(racks.map(stripStatus));
    return this.sugar(racks);
  }

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, Rack[]>) {
    const matches = requestFilter(query);
    const listeners = [
      this.rackEvents.onSet((rack) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.find((r) => r.key === rack.key);
          if (!matches(rack))
            return existing == null ? prev : prev.filter((r) => r.key !== rack.key);
          const merged = this.sugar({ ...existing?.payload, ...rack });
          if (existing != null)
            return prev.map((r) => (r.key === rack.key ? merged : r));
          return [...prev, merged];
        });
      }),
      this.rackEvents.onDelete((key) => {
        update((prev) => prev?.filter((r) => r.key !== key));
      }),
    ];
    if (query.includeStatus === true)
      listeners.push(
        this.onRackStatusSet((changed) =>
          update((prev) =>
            prev?.map((r) =>
              r.key === changed.details.rack
                ? this.sugar({ ...r.payload, status: changed })
                : r,
            ),
          ),
        ),
      );
    return listeners;
  }
}

export class Rack {
  key: Key;
  name: string;
  status?: Status;
  integrations: string[];
  taskCounter: number;
  embedded: boolean;
  private readonly tasks: task.Client;

  constructor(
    key: Key,
    name: string,
    taskClient: task.Client,
    status?: Status,
    integrations: string[] = [],
    taskCounter: number = 0,
    embedded: boolean = false,
  ) {
    this.key = key;
    this.name = name;
    this.tasks = taskClient;
    this.status = status;
    this.integrations = integrations;
    this.taskCounter = taskCounter;
    this.embedded = embedded;
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
    task.key = (
      (BigInt(this.key) << 32n) +
      (BigInt(task.key ?? 0) & 0xffffffffn)
    ).toString();
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
      taskCounter: this.taskCounter,
      embedded: this.embedded,
    };
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));
