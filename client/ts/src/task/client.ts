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
  deep,
  type destructor,
  id,
  primitive,
  type record,
  strings,
  TimeSpan,
  TimeStamp,
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

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

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
  private readonly clients?: {
    frame: framer.Client;
    ontology: ontology.Client;
    range: ranger.Client;
  };

  get frameClient(): framer.Client {
    if (this.clients == null) throw new Error("Task not created");
    return this.clients.frame;
  }

  get ontologyClient(): ontology.Client {
    if (this.clients == null) throw new Error("Task not created");
    return this.clients.ontology;
  }

  get rangeClient(): ranger.Client {
    if (this.clients == null) throw new Error("Task not created");
    return this.clients.range;
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
    if (frameClient != null && ontologyClient != null && rangeClient != null)
      this.clients = {
        frame: frameClient,
        ontology: ontologyClient,
        range: rangeClient,
      };
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

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

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
 * Client-side matching for a request: key, name, type, rack, and flag
 * filters. Server-computed shapes (search, pagination) never reach this
 * filter; they refetch instead.
 */
const requestFilter = (
  req: RetrieveRequest,
): ((t: Omit<Task, "status">) => boolean) => {
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
  /** The task record table; injected into sibling clients at wiring. */
  readonly store: cache.Table<Key, Omit<Task, "status">>;
  private readonly client: UnaryClient;
  private readonly frameClient: framer.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly rangeClient: ranger.Client;
  private readonly statusStore: cache.Table<status.Key, status.Status>;
  private readonly ontology: ontology.Stores;
  private readonly answers: {
    single: cache.Answers<SingleRequest, Task, Key, Omit<Task, "status">>;
    request: cache.Answers<RetrieveRequest, Task[], Key, Omit<Task, "status">>;
  };

  constructor(
    client: UnaryClient,
    frameClient: framer.Client,
    ontologyClient: ontology.Client,
    rangeClient: ranger.Client,
    engine: cache.Cache,
    statusStore: cache.Table<status.Key, status.Status>,
  ) {
    this.client = client;
    this.frameClient = frameClient;
    this.ontologyClient = ontologyClient;
    this.rangeClient = rangeClient;
    this.statusStore = statusStore;
    this.ontology = ontologyClient.stores;
    this.store = this.createTable(engine, statusStore);
    const matchesSingle = (t: Omit<Task, "status">, query: SingleRequest): boolean => {
      if (primitive.isNonZero(query.keys)) return t.key === query.keys[0];
      if (primitive.isNonZero(query.names)) return t.name === query.names[0];
      if (primitive.isNonZero(query.types))
        return (
          t.type === query.types[0] &&
          (query.rack == null || rackKey(t.key) === query.rack)
        );
      return false;
    };
    this.answers = {
      single: engine.answers({
        name: "task",
        table: this.store,
        fetch: async (query) => [(await this.fetchSingle(query)).key],
        compose: ([record]) => this.compose(record),
        keyOf: (query) => (primitive.isNonZero(query.keys) ? query.keys[0] : null),
        matches: matchesSingle,
        single: true,
        watch: [cache.watch(this.statusStore, (event) => affectedTaskKeys(event))],
      }),
      request: engine.answers({
        name: "tasks",
        table: this.store,
        fetch: async (query) => (await this.fetchRequest(query)).map((t) => t.key),
        compose: (records) => records.map((t) => this.compose(t)),
        matches: (t, query) => requestFilter(query)(t),
        serverFields: SERVER_FIELDS,
        watch: [cache.watch(this.statusStore, (event) => affectedTaskKeys(event))],
      }),
    };
  }

  private createTable(
    engine: cache.Cache,
    statuses: cache.Table<status.Key, status.Status>,
  ): cache.Table<Key, Omit<Task, "status">> {
    const table = engine.createTable<Key, Omit<Task, "status">>({
      name: "tasks",
      equal: (a, b) => deep.equal((a as Task).payload, (b as Task).payload),
    });
    const setListener: cache.ChannelListener<typeof keyZ> = {
      channel: SET_CHANNEL_NAME,
      schema: keyZ,
      onChange: async (key) =>
        table.set(key, await this.retrieve({ key, includeStatus: true })),
    };
    const deleteListener: cache.ChannelListener<typeof keyZ> = {
      channel: DELETE_CHANNEL_NAME,
      schema: keyZ,
      onChange: (changed) => table.delete(changed),
    };
    const commandListener: cache.ChannelListener<typeof commandZ> = {
      channel: COMMAND_CHANNEL_NAME,
      schema: commandZ,
      onChange: (changed) => {
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
    engine.addListeners(table, setListener, deleteListener, commandListener);
    return table;
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
    rollback.add(ontology.deleteCachedResources(this.ontology, ontologyID(keysArr)));
    rollback.add(this.store.delete(keysArr));
    rollback.add(this.statusStore.delete(keysArr.map((k) => statusKey(k))));
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
    this.store.delete(keysArr);
    this.statusStore.delete(keysArr.map((k) => statusKey(k)));
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const rollback = new cache.Rollback();
    rollback.add(
      this.store.set(key, (p) =>
        p == null ? undefined : this.sugar({ ...(p as Task).payload, name }),
      ),
    );
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
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
    if (schemas != null) {
      const sugared = await this.execRetrieve<S>(params, schemas);
      checkForMultipleOrNoResults("Task", params, sugared, isSingle);
      return isSingle ? sugared[0] : sugared;
    }
    if (isSingle)
      return (await this.answers.single.retrieve(
        normalizeSingle(params as RetrieveSingleParams),
      )) as Task<S>;
    return (await this.answers.request.retrieve(
      normalizeRequest(params),
    )) as unknown as Task<S>[];
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a task; every other shape delivers the matching tasks.
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
    const { request, single } = this.answers;
    if (singleRetrieveParamsZ.safeParse(params).success)
      return single.onChange(
        normalizeSingle(params as RetrieveSingleParams),
        handler as cache.ChangeHandler<Task>,
      );
    return request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Task[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Task> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Task[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Task> | cache.Cached<Task[]> | undefined {
    const { request, single } = this.answers;
    if (singleRetrieveParamsZ.safeParse(params).success)
      return single.getCached(normalizeSingle(params as RetrieveSingleParams));
    return request.getCached(normalizeRequest(params));
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
    const st = this.latestStatusOf(cached.key);
    const payload = (cached as Task).payload;
    if (st == null) return this.sugar(payload);
    return this.sugar({ ...payload, status: st as unknown as Status });
  }

  // A task's status may live under the "task:<key>" row or under any status
  // whose details reference the task; the freshest wins.
  private latestStatusOf(key: Key): status.Status | undefined {
    const taskKey = statusKey(key);
    const candidates = this.statusStore.get(
      (s) =>
        s.key === taskKey ||
        (s as { details?: { task?: unknown } }).details?.task === key,
    );
    if (candidates.length === 0) return undefined;
    return candidates.reduce((latest, s) =>
      new TimeStamp(s.time).afterEq(new TimeStamp(latest.time)) ? s : latest,
    );
  }

  /** Writes a fetched task and its included status. */
  private writeThrough(task: Task): void {
    this.store.set(task.key, task);
    if (task.status != null) this.statusStore.set(task.status.key, task.status);
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
      const cached = this.store.get(key);
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
    if (primitive.isNonZero(query.keys)) cached = this.store.get(query.keys[0]);
    else if (primitive.isNonZero(query.names))
      [cached] = this.store.get((t) => t.name === query.names?.[0]);
    if (cached != null && this.statusStore.has(statusKey(cached.key)))
      return this.compose(cached);
    const tasks = await this.execRetrieve({ ...query, includeStatus: true });
    checkForMultipleOrNoResults("Task", query, tasks, true);
    this.writeThrough(tasks[0]);
    return tasks[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Task[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const tasks = await this.execRetrieve({ ...query, includeStatus: true });
    tasks.forEach((t) => this.writeThrough(t));
    return tasks;
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
// the details, with the "task:<key>" status key as a fallback.
const affectedTaskKeys = (
  event: cache.TableEvent<status.Key, status.Status>,
): Key[] | null => {
  const keys: Key[] = [];
  if (event.variant === "set") {
    const parsed = taskStatusZ.safeParse(event.value);
    if (parsed.success) keys.push(parsed.data.details.task);
  }
  const [type, key] = event.key.split(":");
  if (type === "task" && primitive.isNonZero(key) && !keys.includes(key))
    keys.push(key);
  return keys.length === 0 ? null : keys;
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
