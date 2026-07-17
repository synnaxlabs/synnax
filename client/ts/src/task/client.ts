// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import {
  array,
  caseconv,
  type CrudeTimeSpan,
  type destructor,
  id,
  primitive,
  type record,
  strings,
  TimeSpan,
} from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { type framer } from "@/framer";
import { ontology } from "@/ontology";
import { type Key as RackKey, keyZ as rackKeyZ } from "@/rack/types.gen";
import { type ranger } from "@/ranger";
import { status } from "@/status";
import {
  commandZ,
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Payload,
  type PayloadSchemas as Schemas,
  payloadZ,
  type Status,
  type StatusDetailsZodObject,
  statusZ,
} from "@/task/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export type { PayloadSchemas as Schemas } from "@/task/types.gen";

export const COMMAND_CHANNEL_NAME = "sy_task_cmd";
export const SET_CHANNEL_NAME = "sy_task_set";
export const DELETE_CHANNEL_NAME = "sy_task_delete";

export const STORE_KEY = "tasks";

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

/** Registers the task store on the given engine. */
const bindStore = (engine: cache.Engine, client: Client): void => {
  const store = () => engine.store<Key, Omit<Task, "status">>(STORE_KEY);
  const setListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: SET_CHANNEL_NAME,
    schema: keyZ,
    onChange: async ({ changed: key }) =>
      store().set(key, await client.retrieve({ key, includeStatus: true })),
  };
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => store().delete(changed),
  };
  const commandListener: cache.ChannelListener<{}, typeof commandZ> = {
    channel: COMMAND_CHANNEL_NAME,
    schema: commandZ,
    onChange: ({ changed }) => {
      const statuses = engine.store<status.Key, status.Status>(status.STORE_KEY);
      statuses.set(statusKey(changed.task), (prev) => {
        if (prev == null || !LOADING_COMMANDS.includes(changed.type)) return prev;
        return status.create<StatusDetailsZodObject>({
          key: statusKey(changed.task),
          name: "Task Status",
          variant: "loading",
          message: `Running ${changed.type} command...`,
          details: { task: changed.task, running: true, cmd: "", data: {} },
        });
      });
    },
  };
  engine.registerStore<Key, Omit<Task, "status">>(STORE_KEY, {
    listeners: [setListener, deleteListener, commandListener],
  });
};

export const rackKey = (key: Key): RackKey => Number(BigInt(key) >> 32n);

export const newKey = (rackKey: RackKey, taskKey: number = 0): Key =>
  ((BigInt(rackKey) << 32n) + BigInt(taskKey)).toString();

const retrieveSnapshottedTo = async (taskKey: Key, ontologyClient: ontology.Client) => {
  const parents = await ontologyClient.retrieveParents(ontologyID(taskKey));
  if (parents.length === 0) return null;
  return parents[0];
};

export interface TaskExecuteCommandParams {
  type: string;
  args?: record.Unknown;
}

export interface ExecuteCommandParams extends TaskExecuteCommandParams {
  task: Key;
}

export interface ExecuteCommandsParams {
  commands: NewCommand[];
}

export interface TaskExecuteCommandSyncParams extends TaskExecuteCommandParams {
  timeout?: CrudeTimeSpan;
}

export interface ExecuteCommandsSyncParams<StatusData extends z.ZodType> extends Omit<
  ExecuteCommandsSyncInternalParams<StatusData>,
  "frameClient" | "name"
> {}

export interface ExecuteCommandSyncParams<StatusData extends z.ZodType> extends Omit<
  ExecuteCommandSyncInternalParams<StatusData>,
  "frameClient" | "name"
> {}

export class Task<S extends Schemas = Schemas> {
  readonly key: Key;
  name: string;
  internal: boolean;
  type: z.infer<S["type"]>;
  snapshot: boolean;
  config: z.infer<S["config"]>;
  status?: Status<S["statusData"]>;

  readonly schemas: S;
  private readonly frameClient_?: framer.Client;
  private readonly ontologyClient_?: ontology.Client;
  private readonly rangeClient_?: ranger.Client;

  get frameClient(): framer.Client {
    if (this.frameClient_ == null) throw new Error("Task not created");
    return this.frameClient_;
  }

  get ontologyClient(): ontology.Client {
    if (this.ontologyClient_ == null) throw new Error("Task not created");
    return this.ontologyClient_;
  }

  get rangeClient(): ranger.Client {
    if (this.rangeClient_ == null) throw new Error("Task not created");
    return this.rangeClient_;
  }

  constructor(
    { key, type, name, config, internal = false, snapshot = false, status }: Payload<S>,
    schemas?: S,
    frameClient?: framer.Client,
    ontologyClient?: ontology.Client,
    rangeClient?: ranger.Client,
  ) {
    this.key = key;
    this.name = name;
    this.type = type;
    this.config = config;
    this.schemas =
      schemas ??
      ({
        type: z.string(),
        config: z.unknown(),
        statusData: z.unknown().optional(),
      } as unknown as S);
    this.internal = internal;
    this.snapshot = snapshot;
    this.status = status;
    this.frameClient_ = frameClient;
    this.ontologyClient_ = ontologyClient;
    this.rangeClient_ = rangeClient;
  }

  get payload(): Payload<S> {
    return {
      key: this.key,
      name: this.name,
      type: this.type,
      config: this.config,
      status: this.status,
      internal: this.internal,
      snapshot: this.snapshot,
    };
  }

  get ontologyID(): ontology.ID {
    return ontologyID(this.key);
  }

  async executeCommand(params: TaskExecuteCommandParams): Promise<string> {
    return await executeCommand({
      ...params,
      frameClient: this.frameClient,
      task: this.key,
    });
  }

  async executeCommandSync(
    params: TaskExecuteCommandSyncParams,
  ): Promise<Status<S["statusData"]>> {
    return await executeCommandSync<S["statusData"]>({
      ...params,
      frameClient: this.frameClient,
      task: this.key,
      name: this.name,
      statusDataZ: this.schemas.statusData,
    });
  }

  async start(): Promise<void> {
    await this.executeCommand({ type: "start" });
  }

  async stop(): Promise<void> {
    await this.executeCommand({ type: "stop" });
  }

  async run<T>(fn: () => Promise<T>): Promise<T> {
    await this.start();
    try {
      return await fn();
    } finally {
      await this.stop();
    }
  }

  async snapshottedTo(): Promise<ontology.Resource | null> {
    if (this.ontologyClient == null || this.rangeClient == null)
      throw new Error("Task not created");
    if (!this.snapshot) return null;
    return await retrieveSnapshottedTo(this.key, this.ontologyClient);
  }
}

const retrieveReqZ = z.object({
  rack: rackKeyZ.optional(),
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  types: z.string().array().optional(),
  includeStatus: z.boolean().optional(),
  internal: z.boolean().optional(),
  snapshot: z.boolean().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});

const singleRetrieveParamsZ = z.union([
  z
    .object({ key: keyZ, includeStatus: z.boolean().optional() })
    .transform(({ key, includeStatus }) => ({ keys: [key], includeStatus })),
  z
    .object({ name: z.string(), includeStatus: z.boolean().optional() })
    .transform(({ name, includeStatus }) => ({ names: [name], includeStatus })),
  z
    .object({ type: z.string(), rack: rackKeyZ.optional() })
    .transform(({ type, rack }) => ({ types: [type], rack })),
]);
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

const multiRetrieveParamsZ = retrieveReqZ;
export type RetrieveMultipleParams = z.input<typeof multiRetrieveParamsZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, multiRetrieveParamsZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;

const MOUNT_SCOPE = "task.mounts";

interface SingleRequest extends Partial<
  Pick<z.infer<typeof retrieveReqZ>, "keys" | "names" | "types" | "rack">
> {}

// includeStatus does not change a query's identity: cached fetches always
// request it, so it is stripped before hashing.
const normalizeSingle = (params: RetrieveSingleParams): SingleRequest => {
  const parsed = singleRetrieveParamsZ.parse(params);
  if (!("includeStatus" in parsed)) return parsed;
  const { includeStatus: _, ...rest } = parsed;
  return rest;
};

const normalizeRequest = (
  params: RetrieveMultipleParams,
): z.infer<typeof retrieveReqZ> => {
  const { includeStatus: _, ...rest } = multiRetrieveParamsZ.parse(params);
  return rest;
};

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.names == null &&
  req.types == null &&
  req.rack == null &&
  req.internal == null &&
  req.snapshot == null &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null;

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key, name, type, rack, and flag filters, permissive for server-computed
 * shapes (search), which accept every change and drift toward the server's
 * answer.
 */
const requestFilter = (req: RetrieveRequest): ((t: Task) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
  const typeSet = primitive.isNonZero(req.types) ? new Set(req.types) : undefined;
  return (t) => {
    if (keySet != null && !keySet.has(t.key)) return false;
    if (nameSet != null && !nameSet.has(t.name)) return false;
    if (typeSet != null && !typeSet.has(t.type)) return false;
    if (req.rack != null && rackKey(t.key) !== req.rack) return false;
    if (req.internal != null && t.internal !== req.internal) return false;
    if (req.snapshot != null && t.snapshot !== req.snapshot) return false;
    return true;
  };
};

interface RetrieveSchemas<S extends Schemas = Schemas> {
  schemas?: S;
}

const retrieveResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({
    tasks: payloadZ(schemas)
      .array()
      .default(() => []),
  });

export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

const createReqZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ tasks: payloadZ(schemas).array() });
const createResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ tasks: payloadZ(schemas).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const deleteResZ = z.object({});
const copyReqZ = z.object({ key: keyZ, name: z.string(), snapshot: z.boolean() });
const copyResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ task: payloadZ(schemas) });

export class Client {
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly rangeClient: ranger.Client;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<SingleRequest, Task>;
    request: cache.Queries<RetrieveRequest, Task[]>;
  };

  constructor(
    client: UnaryClient,
    frameClient: framer.Client,
    ontologyClient: ontology.Client,
    rangeClient: ranger.Client,
    engine?: cache.Engine,
  ) {
    this.client = client;
    this.frameClient = frameClient;
    this.ontologyClient = ontologyClient;
    this.rangeClient = rangeClient;
    if (engine == null) return;
    bindStore(engine, this);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "task",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "tasks",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
  }

  async create(task: New): Promise<Task>;
  async create(tasks: New[]): Promise<Task[]>;

  async create<S extends Schemas>(task: New<S>, schemas: S): Promise<Task<S>>;
  async create<S extends Schemas>(tasks: New<S>[], schemas: S): Promise<Task<S>[]>;

  async create<S extends Schemas>(
    task: New<S> | New<S>[],
    schemas?: S,
  ): Promise<Task<S> | Task<S>[]> {
    const isSingle = !Array.isArray(task);
    const createReq = createReqZ(schemas);
    const createRes = createResZ(schemas);
    const res = await this.client.send(
      "/task/create",
      { tasks: array.toArray(task) },
      createReq,
      createRes,
    );
    const sugared = this.sugar<S>(res.tasks as Payload<S>[], schemas);
    sugared.forEach((t) => this.writeThrough(t));
    return isSingle ? sugared[0] : sugared;
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    if (this.engine_ != null) {
      rollback.add(ontology.deleteCachedResources(this.engine_, ontologyID(keysArr)));
      rollback.add(this.taskStore.delete(keysArr));
      rollback.add(this.statusStore.delete(keysArr.map((k) => statusKey(k))));
    }
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/task/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    );
    if (this.engine_ == null) return;
    this.taskStore.delete(keysArr);
    this.statusStore.delete(keysArr.map((k) => statusKey(k)));
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const rollback = new cache.Rollback();
    if (this.engine_ != null) {
      rollback.add(
        this.taskStore.set(key, (p) =>
          p == null ? undefined : this.sugar({ ...(p as Task).payload, name }),
        ),
      );
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      const t = await this.retrieve({ key });
      await this.create({ ...t.payload, name });
    });
  }

  async retrieve<S extends Schemas = Schemas>(
    params: RetrieveSingleParams & RetrieveSchemas<S>,
  ): Promise<Task<S>>;
  async retrieve(params: RetrieveSingleParams): Promise<Task>;
  async retrieve<S extends Schemas = Schemas>(
    params: RetrieveMultipleParams & RetrieveSchemas<S>,
  ): Promise<Task<S>[]>;
  async retrieve(params: RetrieveMultipleParams): Promise<Task[]>;
  async retrieve<S extends Schemas = Schemas>({
    schemas,
    ...params
  }: RetrieveParams & RetrieveSchemas<S>): Promise<Task<S> | Task<S>[]> {
    const isSingle = singleRetrieveParamsZ.safeParse(params).success;
    // Schema-parametrized retrieves validate config/status for one caller;
    // their results are not shared through the cache.
    if (this.queries_ == null || schemas != null) {
      const sugared = await this.execRetrieve<S>(params, schemas);
      checkForMultipleOrNoResults("Task", params, sugared, isSingle);
      return isSingle ? sugared[0] : sugared;
    }
    if (isSingle)
      return (await this.queries_.single.retrieve(
        normalizeSingle(params as RetrieveSingleParams),
      )) as Task<S>;
    return (await this.queries_.request.retrieve(
      normalizeRequest(params),
    )) as unknown as Task<S>[];
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a task; every other shape delivers the matching tasks.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Task>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Task[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Task> | cache.ChangeHandler<Task[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if (singleRetrieveParamsZ.safeParse(params).success)
      return queries.single.onChange(
        normalizeSingle(params as RetrieveSingleParams),
        handler as cache.ChangeHandler<Task>,
      );
    return queries.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Task[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Task> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Task[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Task> | cache.Cached<Task[]> | undefined {
    const queries = this.requireQueries();
    if (singleRetrieveParamsZ.safeParse(params).success)
      return queries.single.getCached(normalizeSingle(params as RetrieveSingleParams));
    return queries.request.getCached(normalizeRequest(params));
  }

  async copy(key: Key, name: string, snapshot: boolean): Promise<Task> {
    const copyRes = copyResZ();
    const response = await this.client.send(
      "/task/copy",
      { key, name, snapshot },
      copyReqZ,
      copyRes,
    );
    const sugared = this.sugar(response.task);
    this.writeThrough(sugared);
    return sugared;
  }

  async list(rack?: number): Promise<Task[]> {
    const params: RetrieveMultipleParams = { internal: false };
    if (rack !== undefined) params.rack = rack;
    return await this.retrieve(params);
  }

  async retrieveSnapshottedTo(taskKey: Key): Promise<ontology.Resource | null> {
    if (this.ontologyClient == null) throw new Error("Task not created");
    return await retrieveSnapshottedTo(taskKey, this.ontologyClient);
  }

  private get taskStore(): cache.UnaryStore<Key, Omit<Task, "status">> {
    return this.requireEngine().store(STORE_KEY);
  }

  private get statusStore(): cache.UnaryStore<status.Key, status.Status> {
    return this.requireEngine().store(status.STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get taskEvents(): cache.UnaryStore<Key, Omit<Task, "status">> {
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

  private async execRetrieve<S extends Schemas = Schemas>(
    params: RetrieveParams,
    schemas?: S,
  ): Promise<Task<S>[]> {
    const res = await this.client.send(
      "/task/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ(schemas),
    );
    return this.sugar(res.tasks as Payload<S>[], schemas);
  }

  /** Rebuilds a cached task with its cached status attached. */
  private compose(cached: Omit<Task, "status">): Task {
    const st = this.statusStore.get(statusKey(cached.key));
    const payload = (cached as Task).payload;
    if (st == null) return this.sugar(payload);
    return this.sugar({ ...payload, status: st as unknown as Status });
  }

  /** Rebuilds a cached task with the given status attached. */
  private composeWith(cached: Omit<Task, "status">, st: status.Status): Task {
    const payload = (cached as Task).payload;
    return this.sugar({ ...payload, status: st as unknown as Status });
  }

  /** Writes a fetched task and its included status. */
  private writeThrough(task: Task): void {
    this.taskStore.set(task.key, task);
    if (task.status != null)
      this.statusStore.set(task.status.key, task.status as unknown as status.Status);
  }

  /**
   * Fetches the given keys, serving composed cached entries and fetching only
   * the misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Task[]> {
    const results: Task[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      // A cached task without a cached status is ambiguous: the task may have
      // no status or the status may not have synced. Only both count as a hit.
      const cached = this.taskStore.get(key);
      if (cached != null && this.statusStore.has(statusKey(key)))
        results.push(this.compose(cached));
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses, includeStatus: true });
      fetched.forEach((t) => this.writeThrough(t));
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (t) => t.key);
  }

  private async fetchSingle(query: SingleRequest): Promise<Task> {
    let cached: Omit<Task, "status"> | undefined;
    if (primitive.isNonZero(query.keys)) cached = this.taskStore.get(query.keys[0]);
    else if (primitive.isNonZero(query.names))
      [cached] = this.taskStore.get((t) => t.name === query.names?.[0]);
    if (cached != null && this.statusStore.has(statusKey(cached.key)))
      return this.compose(cached);
    const tasks = await this.execRetrieve({ ...query, includeStatus: true });
    checkForMultipleOrNoResults("Task", query, tasks, true);
    this.writeThrough(tasks[0]);
    return tasks[0];
  }

  private mountSingle({
    query,
    update,
    remove,
  }: cache.MountParams<SingleRequest, Task>) {
    const matches = (t: Omit<Task, "status">): boolean => {
      if (primitive.isNonZero(query.keys)) return t.key === query.keys[0];
      if (primitive.isNonZero(query.names)) return t.name === query.names[0];
      if (primitive.isNonZero(query.types))
        return (
          t.type === query.types[0] &&
          (query.rack == null || rackKey(t.key) === query.rack)
        );
      return false;
    };
    return [
      this.taskEvents.onSet((task) => {
        if (matches(task)) update(this.compose(task));
      }),
      this.taskEvents.onDelete((key) => {
        const corpse = this.taskStore.getTombstone(key)?.corpse;
        const deleted =
          (primitive.isNonZero(query.keys) && key === query.keys[0]) ||
          (corpse != null && matches(corpse));
        if (deleted) remove(corpse == null ? undefined : this.compose(corpse));
      }),
      this.statusEvents.onSet((st) => {
        update((prev) => {
          if (prev == null || !statusRefersTo(st, prev.key)) return prev;
          return this.composeWith(prev, st);
        });
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Task[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const tasks = await this.execRetrieve({ ...query, includeStatus: true });
    tasks.forEach((t) => this.writeThrough(t));
    return tasks;
  }

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, Task[]>) {
    const matches = requestFilter(query);
    return [
      this.taskEvents.onSet((task) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((t) => t.key === task.key);
          const merged = this.compose(task);
          if (!matches(merged))
            return existing ? prev.filter((t) => t.key !== task.key) : prev;
          if (existing) return prev.map((t) => (t.key === task.key ? merged : t));
          return [...prev, merged];
        });
      }),
      this.taskEvents.onDelete((key) => {
        update((prev) => prev?.filter((t) => t.key !== key));
      }),
      this.statusEvents.onSet((st) => {
        update((prev) => {
          if (prev == null) return prev;
          return prev.map((t) =>
            statusRefersTo(st, t.key) ? this.composeWith(t, st) : t,
          );
        });
      }),
    ];
  }

  sugar<S extends Schemas = Schemas>(payloads: Payload<S>[], schemas?: S): Task<S>[];

  sugar<S extends Schemas = Schemas>(payload: Payload<S>, schemas?: S): Task<S>;

  sugar<S extends Schemas = Schemas>(
    payloads: Payload<S> | Payload<S>[],
    schemas?: S,
  ): Task<S>[] | Task<S> {
    const isSingle = !Array.isArray(payloads);
    const res = array.toArray(payloads).map(
      ({ key, name, type, config, status, internal, snapshot }) =>
        new Task(
          {
            key,
            name,
            type,
            config,
            internal,
            snapshot,
            status,
          },
          schemas,
          this.frameClient,
          this.ontologyClient,
          this.rangeClient,
        ),
    );
    return isSingle ? res[0] : res;
  }

  async executeCommand(params: ExecuteCommandParams): Promise<string>;

  async executeCommand(params: ExecuteCommandsParams): Promise<string[]>;

  async executeCommand(
    params: ExecuteCommandParams | ExecuteCommandsParams,
  ): Promise<string | string[]> {
    if ("commands" in params)
      return await executeCommands({ ...params, frameClient: this.frameClient });
    return await executeCommand({ ...params, frameClient: this.frameClient });
  }

  async executeCommandSync<StatusData extends z.ZodType = z.ZodNever>(
    params: ExecuteCommandsSyncParams<StatusData>,
  ): Promise<Status<StatusData>[]>;

  async executeCommandSync<StatusData extends z.ZodType = z.ZodNever>(
    params: ExecuteCommandSyncParams<StatusData>,
  ): Promise<Status<StatusData>>;

  async executeCommandSync<StatusData extends z.ZodType = z.ZodNever>(
    params:
      ExecuteCommandsSyncParams<StatusData> | ExecuteCommandSyncParams<StatusData>,
  ): Promise<Status<StatusData> | Status<StatusData>[]> {
    if ("commands" in params) {
      const retrieveNames = async () => {
        const { commands } = params;
        const ts = await this.retrieve({ keys: commands.map((t) => t.task) });
        return ts.map((t) => t.name);
      };
      return await executeCommandsSync({
        ...params,
        frameClient: this.frameClient,
        name: retrieveNames,
      });
    }
    const retrieveName = async () => {
      const { task } = params;
      const t = await this.retrieve({ key: task });
      return t.name;
    };
    return await executeCommandSync({
      frameClient: this.frameClient,
      name: retrieveName,
      ...params,
    });
  }
}

export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));

const taskStatusZ = z.object({ details: z.object({ task: keyZ }) });

// Task statuses may arrive under any status key; the referenced task lives in
// the details.
const statusRefersTo = (st: status.Status, key: Key): boolean => {
  if (st.key === statusKey(key)) return true;
  const parsed = taskStatusZ.safeParse(st);
  return parsed.success && parsed.data.details.task === key;
};

interface ExecuteCommandInternalParams {
  frameClient: framer.Client | null;
  task: Key;
  type: string;
  args?: {};
}

const executeCommand = async ({
  frameClient,
  task,
  type,
  args,
}: ExecuteCommandInternalParams): Promise<string> =>
  (await executeCommands({ frameClient, commands: [{ args, task, type }] }))[0];

export interface NewCommand {
  task: Key;
  type: string;
  args?: {};
}

interface ExecuteCommandsInternalParams {
  frameClient: framer.Client | null;
  commands: NewCommand[];
}

const executeCommands = async ({
  frameClient,
  commands,
}: ExecuteCommandsInternalParams): Promise<string[]> => {
  if (frameClient == null) throw new Error("Task not created");
  const w = await frameClient.openWriter(COMMAND_CHANNEL_NAME);
  const cmds = commands.map((c) => ({ ...c, key: id.create() }));
  await w.write(COMMAND_CHANNEL_NAME, cmds);
  await w.close();
  return cmds.map((c) => c.key);
};

interface ExecuteCommandSyncInternalParams<StatusData extends z.ZodType = z.ZodNever>
  extends
    Omit<ExecuteCommandsSyncInternalParams<StatusData>, "commands">,
    TaskExecuteCommandSyncParams {
  task: Key;
}

const executeCommandSync = async <StatusData extends z.ZodType = z.ZodNever>({
  frameClient,
  task,
  type,
  timeout,
  name: taskName,
  statusDataZ,
  args,
}: ExecuteCommandSyncInternalParams<StatusData>): Promise<Status<StatusData>> =>
  (
    await executeCommandsSync({
      frameClient,
      commands: [{ args, task, type }],
      timeout,
      statusDataZ,
      name: taskName,
    })
  )[0];

interface ExecuteCommandsSyncInternalParams<StatusData extends z.ZodType = z.ZodNever> {
  frameClient: framer.Client | null;
  commands: NewCommand[];
  timeout?: CrudeTimeSpan;
  statusDataZ: StatusData;
  name: string | string[] | (() => Promise<string | string[]>);
}

const executeCommandsSync = async <StatusData extends z.ZodType = z.ZodNever>({
  frameClient,
  commands,
  timeout = TimeSpan.seconds(10),
  statusDataZ,
  name: taskName,
}: ExecuteCommandsSyncInternalParams<StatusData>): Promise<Status<StatusData>[]> => {
  if (frameClient == null) throw new Error("Task not created");
  const streamer = await frameClient.openStreamer(status.SET_CHANNEL_NAME);
  const cmdKeys = await executeCommands({ frameClient, commands });
  const parsedTimeout = new TimeSpan(timeout);
  let states: Status<StatusData>[] = [];
  let timeoutID: NodeJS.Timeout | undefined;
  const timeoutPromise = new Promise<never>((_, reject) => {
    timeoutID = setTimeout(() => {
      void (async () => {
        const taskKeys = commands.map((c) => c.task);
        reject(await formatTimeoutError("command", taskName, parsedTimeout, taskKeys));
      })();
    }, parsedTimeout.milliseconds);
  });
  try {
    while (true) {
      const frame = await Promise.race([streamer.read(), timeoutPromise]);
      const parseResult = statusZ(statusDataZ).safeParse(
        frame.at(-1)[status.SET_CHANNEL_NAME],
      );
      if (!parseResult.success) continue;
      const state = parseResult.data;
      if (state.details.cmd == null || !cmdKeys.includes(state.details.cmd)) continue;
      states = [...states.filter((s) => s.key !== state.key), state];
      if (states.length === cmdKeys.length) return states;
    }
  } finally {
    clearTimeout(timeoutID);
    streamer.close();
  }
};

const formatTimeoutError = async (
  type: string,
  name: string | string[] | (() => Promise<string | string[]>),
  timeout: TimeSpan,
  key: Key | Key[],
): Promise<Error> => {
  const formattedType = caseconv.capitalize(type);
  const formattedTimeout = timeout.toString();
  try {
    let names: string[];
    if (typeof name === "string") names = [name];
    else if (Array.isArray(name)) names = name;
    else names = array.toArray(await name());
    const formattedName = strings.naturalLanguageJoin(names);
    return new Error(
      `${formattedType} command to ${formattedName} timed out after ${formattedTimeout}`,
    );
  } catch (e) {
    console.error("Failed to retrieve task name for timeout error:", e);
    return new Error(
      `${formattedType} command to task with key ${strings.naturalLanguageJoin(array.toArray(key).map((k) => k.toString()))} timed out after ${formattedTimeout}`,
    );
  }
};
