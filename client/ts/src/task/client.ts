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
  id,
  primitive,
  type record,
  strings,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type framer } from "@/framer";
import { ontology } from "@/ontology";
import { query } from "@/query";
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

export const setSignalZ = payloadZ().omit({ config: true, status: true });
export interface SetSignal extends z.infer<typeof setSignalZ> {}

/**
 * Reports whether a task's live instance has drifted from its stored task: the task is
 * running and the stored config or rack differs from what the instance was deployed
 * with. Tasks that are not running never drift. Both hashes are server-assigned, so
 * this compares two given values and never hashes a config. A status with an empty
 * deployed hash never drifts: the deployed config is unknown, not different.
 * @param task - The task payload, including its status.
 * @returns True when a redeploy (start) would change the running instance.
 */
export const drifted = (task: Payload): boolean => {
  const details = task.status?.details;
  if (details == null || !details.running || details.configHash === "") return false;
  return details.configHash !== task.configHash || details.rack !== task.rack;
};

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

const retrieveSnapshottedTo = async (taskKey: Key, ontologyClient: ontology.Client) => {
  const parents = await ontologyClient.parents.retrieve({ ids: ontologyID(taskKey) });
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
  readonly rack: RackKey;
  name: string;
  internal: boolean;
  type: z.infer<S["type"]>;
  snapshot: boolean;
  config: z.infer<S["config"]>;
  readonly configHash: string;
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
    {
      key,
      rack,
      type,
      name,
      config,
      configHash = "",
      internal = false,
      snapshot = false,
      status,
    }: Payload<S>,
    schemas?: S,
    frameClient?: framer.Client,
    ontologyClient?: ontology.Client,
    rangeClient?: ranger.Client,
  ) {
    this.key = key;
    this.rack = rack;
    this.name = name;
    this.type = type;
    this.config = config;
    this.configHash = configHash;
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
      rack: this.rack,
      name: this.name,
      type: this.type,
      config: this.config,
      configHash: this.configHash,
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

type SingleRequest = Partial<
  Pick<z.infer<typeof retrieveReqZ>, "keys" | "names" | "types" | "rack">
>;

// includeStatus does not change a query's identity: cached fetches always
// request it, so it is stripped before hashing.
const normalizeSingle = (params: RetrieveSingleParams): SingleRequest => {
  const parsed = singleRetrieveParamsZ.parse(params);
  if (!("includeStatus" in parsed)) return parsed;
  const { includeStatus: _, ...rest } = parsed;
  return rest;
};

const singleQueryZ = z
  .union([
    z.strictObject({ key: keyZ, includeStatus: z.boolean().optional() }),
    z.strictObject({ name: z.string(), includeStatus: z.boolean().optional() }),
    z.strictObject({ type: z.string(), rack: rackKeyZ.optional() }),
    keyZ.transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

// includeStatus does not change a request's identity either: cached fetches
// always request it, so the schema strips it before hashing.
const cacheRetrieveReqZ = retrieveReqZ.transform(
  ({ includeStatus: _, ...rest }) => rest,
);
const retrieveMultiParamsZ = cacheRetrieveReqZ.or(query.keyListZ(keyZ));

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
    if (req.rack != null && t.rack !== req.rack) return false;
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

const matchesSingle = (t: Omit<Task, "status">, query: SingleRequest): boolean => {
  if (primitive.isNonZero(query.keys)) return t.key === query.keys[0];
  if (primitive.isNonZero(query.names)) return t.name === query.names[0];
  if (primitive.isNonZero(query.types))
    return t.type === query.types[0] && (query.rack == null || t.rack === query.rack);
  return false;
};

export interface ClientConfig {
  unary: UnaryClient;
  framer: framer.Client;
  ontology: ontology.Client;
  ranges: ranger.Client;
  cache: query.Cache;
  statusStore: query.Table<status.Key, status.Status>;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Omit<Task, "status">,
  Task,
  RetrieveSingleParams,
  SingleRequest
> {
  /** The task record table; injected into sibling clients at wiring. */
  readonly store: query.Table<Key, Omit<Task, "status">>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const { cache, statusStore } = cfg;
    const store = cache.createTable<Key, Omit<Task, "status">>({
      name: "tasks",
      equal: (a, b) => deep.equal(a.payload, b.payload),
      fetch: async (keys) => await this.fetchThrough({ keys }),
      listen: [
        // The set signal carries task metadata without config or status. Merge it
        // into the cached row in place: config reaches the store only through this
        // client's own saves or a cache-miss fetch.
        {
          bind: (table) => ({
            channel: SET_CHANNEL_NAME,
            schema: setSignalZ,
            onChange: async (changed: SetSignal) => {
              if (table.has(changed.key))
                table.set(changed.key, (prev) =>
                  prev == null
                    ? undefined
                    : this.sugar({ ...prev.payload, ...changed }),
                );
              else await table.retrieve([changed.key], { refresh: true });
            },
          }),
        },
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    cache.listen({
      channel: COMMAND_CHANNEL_NAME,
      schema: commandZ,
      onChange: (changed) => {
        statusStore.set(statusKey(changed.task), (prev) => {
          if (prev == null || !LOADING_COMMANDS.includes(changed.type)) return prev;
          // Carry the last known deploy info forward: zeroing it would make this
          // optimistic status claim the task deployed with an empty config/rack.
          const latest = this.latestStatusOf(changed.task);
          return status.create<StatusDetailsZodObject>({
            key: statusKey(changed.task),
            name: "Task Status",
            variant: "loading",
            message: `Running ${changed.type} command...`,
            details: {
              task: changed.task,
              running: true,
              cmd: "",
              configHash: latest?.details.configHash ?? "",
              rack: latest?.details.rack ?? 0,
              data: {},
            },
          });
        });
      },
    });
    const composed = cache.derive<Key, Omit<Task, "status">, Task>({
      name: "task.composed",
      source: store,
      compose: (record) => this.compose(record),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      watch: [query.deriveWatch(statusStore, (event) => affectedTaskKeys(event))],
    });
    const single = cache.queries<SingleRequest, Task, Key, Task>({
      name: "task",
      table: composed,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => (primitive.isNonZero(q.keys) ? q.keys[0] : null),
      matches: matchesSingle,
      single: true,
    });
    super(cache, {
      name: "task",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.fetchThrough(req),
        matches: (t, req) => requestFilter(req)(t),
        watch: [query.watch(statusStore, (event) => affectedTaskKeys(event))],
      },
      compose: (record) => this.compose(record),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
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
    const res = await this.cfg.unary.send(
      "/task/create",
      { tasks: array.toArray(task) },
      createReq,
      createRes,
    );
    const sugared = this.sugar<S>(res.tasks as Payload<S>[], schemas);
    sugared.forEach((t) => this.writeThrough(t));
    return isSingle ? sugared[0] : sugared;
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteResources(ontologyID(keysArr)),
      this.store.delete(keysArr),
      this.cfg.statusStore.delete(keysArr.map((k) => statusKey(k))),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/task/delete",
          { keys: keysArr },
          deleteReqZ,
          deleteResZ,
        ),
    });
    drop();
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      this.store.set(key, (p) =>
        p == null ? undefined : this.sugar({ ...p.payload, name }),
      ),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () => {
        const t = await this.retrieve(key);
        await this.create({ ...t.payload, name });
      },
    });
    rename();
  }

  async retrieve<S extends Schemas = Schemas>(
    params: RetrieveSingleParams & RetrieveSchemas<S>,
  ): Promise<Task<S>>;
  async retrieve(params: Key | RetrieveSingleParams): Promise<Task>;
  async retrieve<S extends Schemas = Schemas>(
    params: RetrieveMultipleParams & RetrieveSchemas<S>,
  ): Promise<Task<S>[]>;
  async retrieve(params: RetrieveMultipleParams): Promise<Task[]>;
  async retrieve<S extends Schemas = Schemas>(
    rawParams: Key | (RetrieveParams & RetrieveSchemas<S>),
  ): Promise<Task<S> | Task<S>[]> {
    const { schemas, ...params } =
      typeof rawParams === "string" ? { key: rawParams } : rawParams;
    const isSingle = singleRetrieveParamsZ.safeParse(params).success;
    // Schema-parametrized retrieves validate config/status for one caller;
    // their results are not shared through the cache.
    if (schemas != null) {
      const sugared = await this.execRetrieve<S>(params, schemas);
      checkForMultipleOrNoResults("Task", params, sugared, isSingle);
      return isSingle ? sugared[0] : sugared;
    }
    if (isSingle)
      return (await super.retrieve(params as RetrieveSingleParams)) as Task<S>;
    return (await super.retrieve(
      params as RetrieveMultipleParams,
    )) as unknown as Task<S>[];
  }

  async copy(key: Key, name: string, snapshot: boolean): Promise<Task> {
    const copyRes = copyResZ();
    const response = await this.cfg.unary.send(
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
    if (this.cfg.ontology == null) throw new Error("Task not created");
    return await retrieveSnapshottedTo(taskKey, this.cfg.ontology);
  }

  private async execRetrieve<S extends Schemas = Schemas>(
    params: RetrieveParams,
    schemas?: S,
  ): Promise<Task<S>[]> {
    const res = await this.cfg.unary.send(
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
    const payload = cached.payload;
    if (st == null) return this.sugar(payload);
    return this.sugar({ ...payload, status: st });
  }

  // A task's status may live under the "task:<key>" row or under any status
  // whose details reference the task; the freshest wins. Rows are parsed
  // because the status table holds every domain's statuses generically.
  private latestStatusOf(key: Key): Status | undefined {
    const taskKey = statusKey(key);
    const candidates = this.cfg.statusStore
      .get((s) => s.key === taskKey || status.detailsOf(s)?.task === key)
      .map((s) => statusZ().safeParse(s))
      .filter((p) => p.success)
      .map((p) => p.data);
    if (candidates.length === 0) return undefined;
    return candidates.reduce((latest, s) =>
      new TimeStamp(s.time).afterEq(new TimeStamp(latest.time)) ? s : latest,
    );
  }

  /** Writes a fetched task and its included status. */
  private writeThrough(task: Task): void {
    this.store.set(task);
    if (task.status != null) this.cfg.statusStore.set(task.status);
  }

  /** Fetches tasks with statuses and writes the statuses through. */
  private async fetchThrough(req: RetrieveRequest): Promise<Task[]> {
    const tasks = await this.execRetrieve({ ...req, includeStatus: true });
    tasks.forEach((t) => {
      if (t.status != null) this.cfg.statusStore.set(t.status);
    });
    return tasks;
  }

  private async fetchSingle(q: SingleRequest): Promise<Task> {
    let cached: Omit<Task, "status"> | undefined;
    if (primitive.isNonZero(q.keys)) cached = this.store.get(q.keys[0]);
    else if (primitive.isNonZero(q.names))
      [cached] = this.store.get((t) => t.name === q.names?.[0]);
    // A cached task without a cached status is ambiguous: the task may have
    // no status or the status may not have synced. Only both count as a hit.
    if (cached != null && this.cfg.statusStore.has(statusKey(cached.key)))
      return this.compose(cached);
    const tasks = await this.execRetrieve({ ...q, includeStatus: true });
    checkForMultipleOrNoResults("Task", q, tasks, true);
    this.writeThrough(tasks[0]);
    return tasks[0];
  }

  sugar<S extends Schemas = Schemas>(payloads: Payload<S>[], schemas?: S): Task<S>[];

  sugar<S extends Schemas = Schemas>(payload: Payload<S>, schemas?: S): Task<S>;

  sugar<S extends Schemas = Schemas>(
    payloads: Payload<S> | Payload<S>[],
    schemas?: S,
  ): Task<S>[] | Task<S> {
    const isSingle = !Array.isArray(payloads);
    const res = array.toArray(payloads).map(
      ({ key, rack, name, type, config, configHash, status, internal, snapshot }) =>
        new Task(
          {
            key,
            rack,
            name,
            type,
            config,
            configHash,
            internal,
            snapshot,
            status,
          },
          schemas,
          this.cfg.framer,
          this.cfg.ontology,
          this.cfg.ranges,
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
      return await executeCommands({ ...params, frameClient: this.cfg.framer });
    return await executeCommand({ ...params, frameClient: this.cfg.framer });
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
        frameClient: this.cfg.framer,
        name: retrieveNames,
      });
    }
    const retrieveName = async () => {
      const { task } = params;
      const t = await this.retrieve(task);
      return t.name;
    };
    return await executeCommandSync({
      frameClient: this.cfg.framer,
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
  event: query.TableEvent<status.Key, status.Status>,
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
