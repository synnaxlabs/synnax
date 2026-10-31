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
  binary,
  type CrudeTimeSpan,
  deep,
  type destructor,
  errors,
  id,
  primitive,
  type record,
  strings,
  sync,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { AccessDeniedError } from "@/errors";
import { type framer } from "@/framer";
import { ontology } from "@/ontology";
import { query } from "@/query";
import {
  type Key as RackKey,
  keyZ as rackKeyZ,
  ontologyID as rackOntologyID,
} from "@/rack/types.gen";
import { type ranger } from "@/ranger";
import { status } from "@/status";
import {
  type Command,
  commandZ,
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Payload,
  type PayloadSchemas as Schemas,
  payloadZ,
  type RetrieveRequest,
  retrieveRequestZ,
  type Status,
  type StatusDetailsZodObject,
  statusZ,
  type Where,
} from "@/task/types.gen";
import { matches, serverComputed } from "@/task/where.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export type { PayloadSchemas as Schemas } from "@/task/types.gen";

export const COMMAND_CHANNEL_NAME = "sy_task_cmd";
export const SET_CHANNEL_NAME = "sy_task_set";
export const DELETE_CHANNEL_NAME = "sy_task_delete";

export const setSignalZ = payloadZ().omit({ config: true, status: true });
export interface SetSignal extends z.infer<typeof setSignalZ> {}

export interface DriftedParams extends Pick<
  Payload,
  "configHash" | "rack" | "status"
> {}

/**
 * Reports whether a task's live instance has drifted from its stored task: the task is
 * running and the stored config or rack differs from what the instance was deployed
 * with. Tasks that are not running never drift. Both hashes are server-assigned, so
 * this compares two given values and never hashes a config. A status with an empty
 * deployed hash never drifts: the deployed config is unknown, not different.
 * @returns True when a redeploy (start) would change the running instance.
 */
export const drifted = (task: DriftedParams): boolean => {
  const details = task.status?.details;
  if (details == null || !details.running || details.configHash === "") return false;
  return details.configHash !== task.configHash || details.rack !== task.rack;
};

// Temporary hack that filters the set of commands that should change the
// status of a task to loading.
// Issue: https://linear.app/synnax/issue/SY-2723/fix-handling-of-non-startstop-commands-loading-indicators-in-tasks
const LOADING_COMMANDS = ["start", "stop"];

// The rack monitor marks a rack whose Driver is alive with one of these.
const HEALTHY_RACK_VARIANTS: status.Variant[] = ["success", "info"];

/** How long a command waits for the Driver before the wait is called off. */
const COMMAND_DEADLINE = TimeSpan.seconds(10);

/** A command awaiting the Driver's reply, with the timer that gives up on it. */
interface CommandDeadline {
  cmd: Command;
  timer: NodeJS.Timeout;
  /** The status this client last wrote for the command, which is never its reply. */
  optimistic: Status;
}

/**
 * The task status schema without a details-data schema. Hoisted because statusZ()
 * builds a fresh schema per call, which costs far more than the parse itself.
 */
export const defaultStatusZ = statusZ();

const retrieveSnapshottedTo = async (taskKey: Key, ontologyClient: ontology.Client) => {
  const parents = await ontologyClient.parents.retrieve({ ids: ontologyID(taskKey) });
  if (parents.length === 0) return null;
  return parents[0];
};

/** One command sent to a task, as issued from the task itself. */
export interface TaskExecuteCommandParams {
  type: string;
  args?: record.Unknown;
}

/** One command sent to a task, as issued from the client. */
export interface ExecuteCommandParams extends TaskExecuteCommandParams {
  task: Key;
}

/** Several commands sent in one round trip. */
export interface ExecuteCommandsParams {
  commands: NewCommand[];
}

/** A command awaited to completion, as issued from the task itself. */
export interface TaskExecuteCommandSyncParams extends TaskExecuteCommandParams {
  /** How long to wait for the status before giving up. */
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

/**
 * A unit of work a rack runs against hardware: its config, its running status, and the
 * commands it accepts. Get one from `client.hardware.tasks` rather than building it.
 */
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
      configHash: this.configHash,
    });
  }

  async executeCommandSync(
    params: TaskExecuteCommandSyncParams,
  ): Promise<Status<S["statusData"]>> {
    return await executeCommandSync<S["statusData"]>({
      ...params,
      frameClient: this.frameClient,
      task: this.key,
      configHash: this.configHash,
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

/** Names one task in the cache: by key, by name, or by type and rack. */
type SingleRequest = { key: Key } | { name: string } | { type: string; rack?: RackKey };

const singleWhere = (q: SingleRequest): Where => {
  if ("key" in q) return { key: { eq: q.key } };
  if ("name" in q) return { name: { eq: q.name } };
  return { type: { eq: q.type }, rack: q.rack == null ? undefined : { eq: q.rack } };
};

const singleRetrieveParamsZ = z.union([
  z
    .object({ key: keyZ, includeStatus: z.boolean().optional() })
    .transform(({ key, includeStatus }): RetrieveRequest => ({
      where: singleWhere({ key }),
      includeStatus,
    })),
  z
    .object({ name: z.string(), includeStatus: z.boolean().optional() })
    .transform(({ name, includeStatus }): RetrieveRequest => ({
      where: singleWhere({ name }),
      includeStatus,
    })),
  z
    .object({ type: z.string(), rack: rackKeyZ.optional() })
    .transform(({ type, rack }): RetrieveRequest => ({
      where: singleWhere({ type, rack }),
    })),
]);
/** Names one task: by key, by name, or by type and rack. */
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

/** Everything a multi-task retrieval can filter on. */
export type RetrieveMultipleParams = RetrieveRequest;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);
/** Params for a task retrieval, single or multiple. */
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

// includeStatus does not change a query's identity: cached fetches always
// request it, so it is stripped before hashing.
const normalizeSingle = (params: RetrieveSingleParams): SingleRequest => {
  if ("key" in params) return { key: params.key };
  if ("name" in params) return { name: params.name };
  return params.rack == null
    ? { type: params.type }
    : { type: params.type, rack: params.rack };
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
const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  retrieveRequestZ.transform(({ includeStatus: _, ...rest }) => rest),
]);

interface RetrieveSchemas<S extends Schemas = Schemas> {
  schemas?: S;
}

const retrieveResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({
    tasks: payloadZ(schemas)
      .array()
      .default(() => []),
  });

const createReqZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ tasks: payloadZ(schemas).array() });
const createResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ tasks: payloadZ(schemas).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const copyReqZ = z.object({ key: keyZ, name: z.string(), snapshot: z.boolean() });
const copyResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ task: payloadZ(schemas) });

const matchesSingle = (t: Omit<Task, "status">, q: SingleRequest): boolean =>
  matches(t, singleWhere(q));

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  framer: framer.Client;
  ontology: ontology.Client;
  ranges: ranger.Client;
  cache: query.Cache;
  statuses: status.Client;
}

/**
 * Creates, reads, deletes, and commands tasks on a Core. Reach it through
 * `client.hardware.tasks`. Reads are served from a cache a change stream keeps current.
 */
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
  private readonly statusesByTask: query.LookupIndex<status.Key, status.Status>;
  private readonly commandDeadlines = new Map<Key, CommandDeadline>();

  constructor(cfg: ClientConfig) {
    const { cache, statuses } = cfg;
    const statusStore = statuses.store;
    const statusesByTask = statusStore.index(
      new query.LookupIndex<status.Key, status.Status>(statusTaskKey),
    );
    const store = cache.createTable<Key, Omit<Task, "status">>({
      name: "tasks",
      equal: (a, b) => deep.equal(a.payload, b.payload),
      fetch: async (keys) => await this.fetchThrough(query.keysRequest(keys)),
      listen: [
        {
          bind: (table) => ({
            channel: SET_CHANNEL_NAME,
            schema: setSignalZ,
            // The set signal omits config: a changed hash makes the cached one stale.
            onChange: async (changed: SetSignal[]) => {
              const stale: Key[] = [];
              const unheld: Key[] = [];
              table.batch(() =>
                changed.forEach((sig) => {
                  const cached = table.get(sig.key);
                  if (cached == null) {
                    unheld.push(sig.key);
                    return;
                  }
                  if (cached.configHash !== sig.configHash) {
                    stale.push(sig.key);
                    return;
                  }
                  table.set(sig.key, (prev) =>
                    prev == null ? undefined : this.sugar({ ...prev.payload, ...sig }),
                  );
                }),
              );
              if (stale.length > 0) await table.retrieve(stale, { refresh: true });
              // The signal reaches users who cannot read every task, so each key is
              // fetched alone and a denied one stays out of the table.
              await Promise.all(
                unheld.map(async (key) => {
                  try {
                    await table.retrieve([key], { refresh: true });
                  } catch (e) {
                    if (!AccessDeniedError.matches(e)) throw errors.fromUnknown(e);
                  }
                }),
              );
            },
          }),
        },
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    cache.listen({
      channel: COMMAND_CHANNEL_NAME,
      schema: commandZ,
      onChange: (commands) =>
        statusStore.batch(() =>
          commands.forEach((changed) => {
            if (!LOADING_COMMANDS.includes(changed.type)) return;
            const key = statusKey(changed.task);
            // Commands reach every client, so one for a task this client neither
            // tracks nor holds a status for is dropped.
            if (store.get(changed.task) == null && statusStore.get(key) == null) return;
            const optimistic = this.optimisticStatus(changed);
            statuses.setLocal(optimistic);
            this.armCommandDeadline(changed, optimistic);
          }),
        ),
    });
    // A reply names its command in details.cmd. The Driver clears cmd on its next
    // status, so the reply is matched on arrival rather than when the deadline fires.
    statusStore.subscribe((event) => {
      if (event.variant !== "set") return;
      const task = statusTaskKey(event.value);
      if (task == null) {
        this.standInForDownRack(event.value);
        return;
      }
      const pending = this.commandDeadlines.get(task);
      if (pending == null || event.value === pending.optimistic) return;
      if (status.detailsOf(event.value)?.cmd !== pending.cmd.key) return;
      clearTimeout(pending.timer);
      this.commandDeadlines.delete(task);
    });
    const composed = cache.derive<Key, Omit<Task, "status">, Task>({
      name: "task.composed",
      source: store,
      compose: (record) => this.compose(record),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      watch: [query.deriveWatch(statusStore, (event) => affectedKeys(event))],
    });
    const single = cache.queries<SingleRequest, Task, Key, Task>({
      name: "task",
      table: composed,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => ("key" in q ? q.key : null),
      matches: matchesSingle,
      single: true,
    });
    super(cache, {
      name: "task",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.fetchThrough(req),
        matches: (t, req) => matches(t, req.where),
        serverComputed: (req) => serverComputed(req.where),
        watch: [query.watch(statusStore, (event) => affectedKeys(event))],
      },
      compose: (record) => this.compose(record),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
    this.statusesByTask = statusesByTask;
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

  /**
   * Forgets the tasks locally without deleting them on the cluster, for callers
   * whose own endpoint already deleted them. Keeps a later retrieve from serving
   * a deleted task while the delete signal is still in flight.
   */
  dropCached(keys: Key | Key[]): void {
    this.dropLocal(array.toArray(keys));
  }

  private dropLocal(keys: Key[]): destructor.Destructor[] {
    return [
      this.cfg.ontology.cache.deleteResources(ontologyID(keys)),
      this.store.delete(keys),
      this.cfg.statuses.store.delete(keys.map((k) => statusKey(k))),
    ];
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => this.dropLocal(keysArr);
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/task/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
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
      commit: async () =>
        await this.cfg.unary.send("/task/rename", { key, name }, renameReqZ, emptyResZ),
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
    const isSingle = z.validate(singleRetrieveParamsZ, params);
    // Schema-parametrized retrieves validate config/status for one caller;
    // their results are not shared through the cache.
    if (schemas != null) {
      const sugared = await this.execRetrieve<S>(params, schemas);
      checkForMultipleOrNoResults("Task", params, sugared, isSingle);
      return isSingle ? sugared[0] : sugared;
    }
    if (isSingle) return (await super.retrieve(params)) as Task<S>;
    return (await super.retrieve(params)) as unknown as Task<S>[];
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

  async list(rack?: RackKey): Promise<Task[]> {
    return await this.retrieve({
      where: {
        internal: { eq: false },
        rack: rack == null ? undefined : { eq: rack },
      },
    });
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

  // Every domain's statuses share one store, so a rack's status sits beside its
  // tasks' under the rack's ontology ID.
  private downRackStatus(rack: RackKey | undefined): status.Status | undefined {
    if (rack == null || primitive.isZero(rack)) return undefined;
    const stat = this.cfg.statuses.store.get(ontology.idToString(rackOntologyID(rack)));
    if (stat == null || HEALTHY_RACK_VARIANTS.includes(stat.variant)) return undefined;
    return stat;
  }

  /**
   * Builds the status shown while a command is in flight. A rack whose Driver is down
   * answers nothing, so its warning stands in for a wait that would never end.
   */
  private optimisticStatus(cmd: Command): Status {
    // Carry the last known deploy info forward: zeroing it would make this
    // optimistic status claim the task deployed with an empty config/rack.
    const latest = this.latestStatusOf(cmd.task);
    const key = statusKey(cmd.task);
    const details = {
      task: cmd.task,
      cmd: "",
      configHash: latest?.details.configHash ?? "",
      rack: latest?.details.rack ?? 0,
      data: {},
    };
    const cached = this.store.get(cmd.task);
    const name = cached?.name ?? latest?.name ?? "";
    const down = this.downRackStatus(cached?.rack);
    if (down == null)
      return status.create<StatusDetailsZodObject>({
        key,
        name,
        variant: "loading",
        message: `Sending ${cmd.type} command`,
        details: { ...details, running: true },
      });
    return status.create<StatusDetailsZodObject>({
      key,
      name,
      variant: "warning",
      message: down.message,
      description: down.description,
      details: { ...details, cmd: cmd.key, running: false },
    });
  }

  // A waiting command shows each new problem of its rack, since the Core copies an
  // outage onto running tasks only.
  private standInForDownRack(rackStatus: status.Status): void {
    for (const [task, pending] of this.commandDeadlines) {
      const down = this.downRackStatus(this.store.get(task)?.rack);
      if (down?.key !== rackStatus.key) continue;
      // A batch stores a reply before it delivers the reply's event.
      const answered = this.statusesByTask
        .get(task)
        .some(
          (s) =>
            s !== pending.optimistic && status.detailsOf(s)?.cmd === pending.cmd.key,
        );
      if (answered) continue;
      pending.optimistic = this.optimisticStatus(pending.cmd);
      this.cfg.statuses.setLocal(pending.optimistic);
    }
  }

  // Only a reply ends the wait sooner. The problem of a down rack outlasts the wait.
  private armCommandDeadline(cmd: Command, optimistic: Status): void {
    const { statuses } = this.cfg;
    const key = statusKey(cmd.task);
    clearTimeout(this.commandDeadlines.get(cmd.task)?.timer);
    // A reply disarms the deadline on arrival, so firing means nothing answered.
    const timer = setTimeout(() => {
      this.commandDeadlines.delete(cmd.task);
      if (statuses.store.get(key) == null) return;
      if (this.downRackStatus(this.store.get(cmd.task)?.rack) != null) return;
      statuses.setLocal(
        status.create<StatusDetailsZodObject>({
          key,
          name: optimistic.name,
          variant: "warning",
          message: `No response to the ${cmd.type} command`,
          description: `The Driver did not respond within ${COMMAND_DEADLINE.toString()}.`,
          // Nothing answered, so the command is taken to have had no effect.
          details: {
            ...optimistic.details,
            cmd: cmd.key,
            running: cmd.type === "stop",
          },
        }),
      );
    }, COMMAND_DEADLINE.milliseconds);
    this.commandDeadlines.set(cmd.task, { cmd, timer, optimistic });
  }

  // A task's status may live under the "task:<key>" row or under any status whose
  // details reference the task; the freshest wins, ties going to the more severe. Rows
  // are parsed because the status table holds every domain's statuses generically.
  private latestStatusOf(key: Key): Status | undefined {
    const direct = this.cfg.statuses.store.get(statusKey(key));
    const rows = this.statusesByTask.get(key);
    // A task status names its task in details too, so the index usually holds it.
    if (direct != null && !rows.includes(direct)) rows.push(direct);
    const candidates = rows
      .map((s) => defaultStatusZ.safeParse(s))
      .filter((p) => p.success)
      .map((p) => p.data);
    if (candidates.length === 0) return undefined;
    return candidates.reduce((latest, s) => {
      const time = new TimeStamp(s.time);
      const latestTime = new TimeStamp(latest.time);
      if (!time.equals(latestTime)) return time.after(latestTime) ? s : latest;
      return status.moreSevere(s.variant, latest.variant) ? s : latest;
    });
  }

  private writeThrough(task: Task): void {
    this.store.set(task);
    if (task.status != null) this.cfg.statuses.store.set(task.status);
  }

  private hydrate(task: Task): void {
    this.store.ingest(task);
    if (task.status != null) this.cfg.statuses.store.ingest(task.status);
  }

  private async fetchThrough(req: RetrieveRequest): Promise<Task[]> {
    // A status that streams in during the fetch is newer than the fetched one.
    const since = this.cfg.statuses.store.stamp();
    const tasks = await this.execRetrieve({ ...req, includeStatus: true });
    tasks.forEach((t) => {
      if (t.status != null) this.cfg.statuses.store.ingest(t.status, { since });
    });
    return tasks;
  }

  private async fetchSingle(q: SingleRequest): Promise<Task> {
    // Task names are not unique, so only a key can answer from the cache.
    const cached = "key" in q ? this.store.get(q.key) : undefined;
    // A cached task without a cached status is ambiguous: the task may have
    // no status or the status may not have synced. Only both count as a hit.
    if (cached != null && this.cfg.statuses.store.has(statusKey(cached.key)))
      return this.compose(cached);
    if ("key" in q) {
      // The refresh covers a cached task whose status has not synced.
      const tasks = await this.store.retrieve([q.key], { refresh: true });
      checkForMultipleOrNoResults("Task", q, tasks, true);
      return this.compose(tasks[0]);
    }
    const tasks = await this.execRetrieve({
      where: singleWhere(q),
      includeStatus: true,
    });
    checkForMultipleOrNoResults("Task", q, tasks, true);
    this.hydrate(tasks[0]);
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

  /**
   * Fills a command's config hash from the cached task row. A command that carries
   * the hash lets the Driver reuse its live instance instead of redeploying.
   */
  private stampConfigHash(cmd: NewCommand): NewCommand {
    if (cmd.configHash != null) return cmd;
    return { ...cmd, configHash: this.store.get(cmd.task)?.configHash };
  }

  async executeCommand(params: ExecuteCommandParams): Promise<string>;

  async executeCommand(params: ExecuteCommandsParams): Promise<string[]>;

  async executeCommand(
    params: ExecuteCommandParams | ExecuteCommandsParams,
  ): Promise<string | string[]> {
    if ("commands" in params)
      return await executeCommands({
        commands: params.commands.map((c) => this.stampConfigHash(c)),
        frameClient: this.cfg.framer,
      });
    return await executeCommand({
      ...this.stampConfigHash(params),
      frameClient: this.cfg.framer,
    });
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
        const ts = await this.retrieve(query.keysRequest(commands.map((t) => t.task)));
        return ts.map((t) => t.name);
      };
      return await executeCommandsSync({
        ...params,
        commands: params.commands.map((c) => this.stampConfigHash(c)),
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
      configHash: params.configHash ?? this.store.get(params.task)?.configHash,
    });
  }
}

/** @returns the key of the status that reports on the given task. */
export const statusKey = (key: Key): string => ontology.idToString(ontologyID(key));

/**
 * @returns the key of the task the status reports on, or null when it names none.
 * Guarded before the key parse: most statuses in the table belong to other domains,
 * and a failed zod parse costs orders of magnitude more than the miss is worth.
 */
const statusTaskKey = (s: status.Status): Key | null => {
  const task = status.detailsOf(s)?.task;
  if (typeof task !== "string") return null;
  return z.validate(keyZ, task) ? task : null;
};

/**
 * @returns the keys of the tasks a status change affects, or null when none. A status
 * names its task in `details.task`, with the "task:<key>" status key as a fallback.
 */
export const affectedKeys = (
  event: query.TableEvent<status.Key, status.Status>,
): Key[] | null => {
  const keys: Key[] = [];
  if (event.variant === "set") {
    const task = statusTaskKey(event.value);
    if (task != null) keys.push(task);
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
  configHash?: string;
}

const executeCommand = async ({
  frameClient,
  task,
  type,
  args,
  configHash,
}: ExecuteCommandInternalParams): Promise<string> =>
  (
    await executeCommands({
      frameClient,
      commands: [{ args, task, type, configHash }],
    })
  )[0];

/** A command queued for a task. */
export interface NewCommand {
  task: Key;
  type: string;
  args?: {};
  /**
   * The config hash the sender wants running. Left unset, the client fills it from
   * its cached task row.
   */
  configHash?: string;
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
  const cmds = commands.map((c) => ({
    ...c,
    configHash: c.configHash ?? "",
    key: id.create(),
  }));
  await w.write(COMMAND_CHANNEL_NAME, cmds);
  await w.close();
  return cmds.map((c) => c.key);
};

interface ExecuteCommandSyncInternalParams<StatusData extends z.ZodType = z.ZodNever>
  extends
    Omit<ExecuteCommandsSyncInternalParams<StatusData>, "commands">,
    TaskExecuteCommandSyncParams {
  task: Key;
  configHash?: string;
}

const executeCommandSync = async <StatusData extends z.ZodType = z.ZodNever>({
  frameClient,
  task,
  type,
  timeout,
  name: taskName,
  statusDataZ,
  args,
  configHash,
}: ExecuteCommandSyncInternalParams<StatusData>): Promise<Status<StatusData>> =>
  (
    await executeCommandsSync({
      frameClient,
      commands: [{ args, task, type, configHash }],
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
  timeout = COMMAND_DEADLINE,
  statusDataZ,
  name: taskName,
}: ExecuteCommandsSyncInternalParams<StatusData>): Promise<Status<StatusData>[]> => {
  if (frameClient == null) throw new Error("Task not created");
  const streamer = await frameClient.openStreamer(status.SET_CHANNEL_NAME);
  const cmdKeys = await executeCommands({ frameClient, commands });
  const parsedTimeout = new TimeSpan(timeout);
  const collect = async (): Promise<Status<StatusData>[]> => {
    let states: Status<StatusData>[] = [];
    while (true) {
      const frame = await streamer.read();
      // A frame can hold statuses for other tasks and racks, which have a different
      // shape and fail to parse.
      for (const str of frame.get(status.SET_CHANNEL_NAME).toStrings()) {
        const res = statusZ(statusDataZ).safeParse(binary.JSON_CODEC.decodeString(str));
        if (!res.success) continue;
        const state = res.data;
        if (state.details.cmd == null || !cmdKeys.includes(state.details.cmd)) continue;
        states = [...states.filter((s) => s.key !== state.key), state];
        if (states.length === cmdKeys.length) return states;
      }
    }
  };
  try {
    return await sync.withTimeout(
      collect(),
      parsedTimeout,
      async () =>
        await formatTimeoutError(
          taskName,
          parsedTimeout,
          commands.map((c) => c.task),
        ),
    );
  } finally {
    streamer.close();
  }
};

const formatTimeoutError = async (
  name: string | string[] | (() => Promise<string | string[]>),
  timeout: TimeSpan,
  key: Key | Key[],
): Promise<Error> => {
  const formattedTimeout = timeout.toString();
  try {
    let names: string[];
    if (typeof name === "string") names = [name];
    else if (Array.isArray(name)) names = name;
    else names = array.toArray(await name());
    const formattedName = strings.naturalLanguageJoin(names);
    return new Error(`Command to ${formattedName} timed out after ${formattedTimeout}`);
  } catch (e) {
    console.error("Failed to retrieve task name for timeout error:", e);
    return new Error(
      `Command to task with key ${strings.naturalLanguageJoin(array.toArray(key).map((k) => k.toString()))} timed out after ${formattedTimeout}`,
    );
  }
};
