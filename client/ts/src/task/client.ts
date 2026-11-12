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
  filter,
  id,
  primitive,
  type record,
  strings,
  TimeSpan,
} from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { type framer } from "@/framer";
import { query } from "@/query";
import {
  type Key as RackKey,
  resourceID as rackResourceID,
  statusDetailsZ as rackStatusDetailsZ,
  statusZ as rackStatusZ,
} from "@/rack/types.gen";
import { resource } from "@/resource";
import { status } from "@/status";
import { createOf, encode, reduceAll } from "@/task/actions";
import {
  type Action,
  actionZ,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/task/actions.gen";
import {
  type Command,
  commandZ,
  type Key,
  keyZ,
  type New,
  type Payload,
  type PayloadSchemas as Schemas,
  payloadZ,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  type Status,
  type StatusDetailsZodObject,
  statusZ,
  type Where,
} from "@/task/types.gen";
import { LINES } from "@/task/where";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/task/where.gen";
import { checkForMultipleOrNoResults, singleQueryError } from "@/util/retrieve";

export type { PayloadSchemas as Schemas } from "@/task/types.gen";

export const COMMAND_CHANNEL_NAME = "sy_task_cmd";
export const ACTION_CHANNEL_NAME = "sy_task_action_set";
export const DELETE_CHANNEL_NAME = "sy_task_delete";

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

// A rack is healthy while its Driver runs and reports no error.
const isRackHealthy = (stat: status.Status): boolean =>
  rackStatusDetailsZ.safeParse(status.detailsOf(stat)).data?.running === true &&
  stat.variant !== "error";

/** How long a command waits for the Driver before the wait is called off. */
const COMMAND_DEADLINE = TimeSpan.seconds(10);

/**
 * How long an acknowledged command waits for its next status. Each loading status
 * restarts the wait, so this limit ends only a wait whose command was lost, for example
 * to a Driver restart.
 */
const FINAL_STATUS_TIMEOUT = TimeSpan.minutes(2);

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

/** One command sent to a task, as issued from the task itself. */
export interface TaskExecuteCommandParams {
  type: string;
  args?: record.Unknown;
}

/** One command sent to a task, as issued from the client. */
export interface ExecuteCommandParams
  extends TaskExecuteCommandParams, Pick<NewCommand, "configHash" | "pinned"> {
  task: Key;
}

/** Several commands sent in one round trip. */
export interface ExecuteCommandsParams {
  commands: NewCommand[];
}

/** A command awaited to completion, as issued from the task itself. */
export interface TaskExecuteCommandSyncParams extends TaskExecuteCommandParams {
  /**
   * How long to wait for the Driver to acknowledge the command. Once it does, each
   * loading status restarts the wait for the next status, which lasts two minutes or
   * the timeout, whichever is longer.
   */
  timeout?: CrudeTimeSpan;
}

export interface ExecuteCommandsSyncParams<StatusData extends z.ZodType> extends Omit<
  ExecuteCommandsSyncInternalParams<StatusData>,
  "frameClient" | "name" | "racks"
> {}

export interface ExecuteCommandSyncParams<StatusData extends z.ZodType> extends Omit<
  ExecuteCommandSyncInternalParams<StatusData>,
  "frameClient" | "name" | "racks"
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
  config: z.infer<S["config"]>;
  readonly configHash: string;
  status?: Status<S["statusData"]>;

  readonly schemas: S;
  private readonly frame?: framer.Client;

  get frameClient(): framer.Client {
    if (this.frame == null) throw new Error("Task not created");
    return this.frame;
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
      status,
    }: Payload<S>,
    schemas?: S,
    frameClient?: framer.Client,
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
    this.status = status;
    this.frame = frameClient;
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
    };
  }

  get resourceID(): resource.ID {
    return resourceID(this.key);
  }

  async executeCommand(params: TaskExecuteCommandParams): Promise<string> {
    return await executeCommand({
      ...params,
      frameClient: this.frameClient,
      task: this.key,
      configHash: this.configHash,
    });
  }

  /**
   * Sends a command and waits for the reply of the task.
   * @throws once the timeout passes, or with the status of the rack once the Driver
   * that runs the task goes down.
   */
  async executeCommandSync(
    params: TaskExecuteCommandSyncParams,
  ): Promise<Status<S["statusData"]>> {
    return await executeCommandSync<S["statusData"]>({
      ...params,
      frameClient: this.frameClient,
      task: this.key,
      configHash: this.configHash,
      name: this.name,
      racks: [this.rack],
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
}

/** Names one task in the cache: by key or by name. */
type SingleRequest = { key: Key } | { name: string };

const singleWhere = (q: SingleRequest): Where =>
  "key" in q ? { key: { eq: q.key } } : { name: { eq: q.name } };

const singleRetrieveParamsZ = z.union([
  z
    .strictObject({ key: keyZ, includeStatus: z.boolean().optional() })
    .transform(({ key, includeStatus }): RetrieveRequest => ({
      where: singleWhere({ key }),
      includeStatus,
    })),
  z
    .strictObject({ name: z.string(), includeStatus: z.boolean().optional() })
    .transform(({ name, includeStatus }): RetrieveRequest => ({
      where: singleWhere({ name }),
      includeStatus,
    })),
]);
/** Names one task: by key or by name. */
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;

/** Everything a multi-task retrieval can filter on. */
export type RetrieveMultipleParams = RetrieveRequest;

const filterParamsZ = retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape);
type FilterParams = query.RequestInput<typeof filterParamsZ>;
const requestParamsZ = filter.requestInputZ(retrieveRequestZ, filterParamsZ);

const retrieveParamsZ = z.union([singleRetrieveParamsZ, requestParamsZ]);
/** Params for a task retrieval, single or multiple. */
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

// includeStatus does not change a query's identity: cached fetches always
// request it, so it is stripped before hashing.
const normalizeSingle = (params: RetrieveSingleParams): SingleRequest =>
  "key" in params ? { key: params.key } : { name: params.name };

const singleQueryZ = z
  .union([
    z.strictObject({ key: keyZ, includeStatus: z.boolean().optional() }),
    z.strictObject({ name: z.string(), includeStatus: z.boolean().optional() }),
    keyZ.transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

// includeStatus does not change a request's identity either: cached fetches
// always request it, so the schema strips it before hashing.
const retrieveMultiParamsZ = requestParamsZ.transform(
  ({ includeStatus: _, ...rest }) => rest,
);

type OneParams = query.OneInput<typeof retrieveMultiParamsZ>;

const isOneParams = (params: object): params is OneParams => query.addressesOne(params);

interface RetrieveSchemas<S extends Schemas = Schemas> {
  schemas?: S;
}

const retrieveResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({
    tasks: payloadZ(schemas)
      .array()
      .default(() => []),
  });

// JSON has no non-finite number, so a config holds one as text, as payload.Float does.
const nonFiniteToText = (value: unknown): unknown => {
  if (typeof value === "number") return Number.isFinite(value) ? value : String(value);
  if (Array.isArray(value)) return value.map(nonFiniteToText);
  if (typeof value !== "object" || value == null) return value;
  if ("toJSON" in value && typeof value.toJSON === "function")
    return nonFiniteToText(value.toJSON());
  return Object.fromEntries(
    Object.entries(value).map(([k, v]) => [k, nonFiniteToText(v)]),
  );
};

// Every dispatch goes through here, undo and redo included.
const dispatchTextReqZ = dispatchReqZ.transform((req) => ({
  ...req,
  actions: req.actions.map((a) => actionZ.parse(nonFiniteToText(a))),
}));

const createReqZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({
    tasks: payloadZ(schemas)
      .transform((t) => ({ ...t, config: nonFiniteToText(t.config) }))
      .array(),
    dispatchKey: z.string(),
  });
const createResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ tasks: payloadZ(schemas).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});
const copyReqZ = z.object({ key: keyZ, name: z.string() });
const copyResZ = <S extends Schemas = Schemas>(schemas?: S) =>
  z.object({ task: payloadZ(schemas) });

const matchesSingle = (t: Omit<Task, "status">, q: SingleRequest): boolean =>
  matches(LINES, t, singleWhere(q));

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  framer: framer.Client;
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
  private readonly dispatcher: actions.Controller<Key, Omit<Task, "status">, Action>;
  private readonly commandDeadlines = new Map<Key, CommandDeadline>();

  constructor(cfg: ClientConfig) {
    const { cache, statuses } = cfg;
    const statusStore = statuses.store;
    const affectedKeys = (event: query.TableEvent<status.Key, status.Status>) =>
      statuses.ownerKeys(event, "task");
    const store = cache.createTable<Key, Omit<Task, "status">>({
      name: "tasks",
      equal: (a, b) => deep.equal(a.payload, b.payload),
      fetch: async (keys) => await this.fetchThrough(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a task reaches the client as actions, the config hash included.
    const dispatcher = new actions.Controller<Key, Omit<Task, "status">, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/task/dispatch", dispatchTextReqZ, dispatchResZ),
      reduce: (task, acts) => {
        const payload = task.payload;
        const { next, inverse, targets } = reduceAll(payload, acts);
        return { next: next === payload ? task : this.sugar(next), inverse, targets };
      },
      encode: (task, action) => encode(task.payload, action),
      createOf: (action) => {
        const created = createOf(action);
        if (created == null) return undefined;
        // The status streams on its own channel and can land after the task, so the
        // status the create carries fills in until it does.
        const carried =
          action.type === "create" ? action.create.task.status : undefined;
        if (carried != null)
          statusStore.ingest(carried, { mode: "if-absent", changed: true });
        return this.sugar(created);
      },
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    cache.listen({
      channel: COMMAND_CHANNEL_NAME,
      schema: commandZ,
      onChange: (commands) =>
        statusStore.batch(() =>
          commands.forEach((changed) => {
            if (!LOADING_COMMANDS.includes(changed.type)) return;
            // Commands reach every client, and only the status the cache holds for the
            // task has a known key, so a command for any other task shows nothing until
            // the Driver answers.
            const held = statuses.ownedBy(resourceID(changed.task));
            if (held == null) return;
            const optimistic = this.optimisticStatus(changed, held.key);
            statuses.setLocal(optimistic);
            this.armCommandDeadline(changed, optimistic);
          }),
        ),
    });
    // A reply names its command in details.cmd. The Driver clears cmd on its next
    // status, so the reply is matched on arrival rather than when the deadline fires.
    statusStore.subscribe((event) => {
      if (event.variant !== "set") return;
      const { owner } = event.value;
      if (owner?.type === "rack") this.standInForDownRack(event.value);
      if (owner?.type !== "task") return;
      const task = owner.key;
      const pending = this.commandDeadlines.get(task);
      if (pending == null || event.value === pending.optimistic) return;
      if (status.detailsOf(event.value)?.cmd !== pending.cmd.key) return;
      clearTimeout(pending.timer);
      if (event.value.variant !== "loading") {
        this.commandDeadlines.delete(task);
        return;
      }
      // A loading reply acknowledges the command, and a down rack still stands in
      // until the final reply or until the command is taken as lost.
      pending.timer = setTimeout(
        () => this.commandDeadlines.delete(task),
        FINAL_STATUS_TIMEOUT.milliseconds,
      );
    });
    const composed = cache.derive<Key, Omit<Task, "status">, Task>({
      name: "task.composed",
      source: store,
      compose: (record) => this.compose(record),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      triggers: [query.deriveTrigger(statusStore, (event) => affectedKeys(event))],
    });
    statuses.holdOwned(store, resourceID);
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
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.fetchThrough(req),
        matches: (t, req) => matches(LINES, t, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
        triggers: [
          query.trigger(statusStore, (event) => affectedKeys(event), {
            contentOnly: true,
          }),
        ],
      },
      compose: (record) => this.compose(record),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
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
    const tasks = array.toArray(task);
    const createReq = createReqZ(schemas);
    const createRes = createResZ(schemas);
    const keys = tasks.flatMap(({ key }) => (key == null ? [] : [key]));
    const res = await this.dispatcher.create(
      keys,
      async (dispatchKey) =>
        await this.cfg.unary.send(
          "/task/create",
          { tasks, dispatchKey },
          createReq,
          createRes,
        ),
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
    const statuses = keys.flatMap(
      (k) => this.cfg.statuses.ownedBy(resourceID(k))?.key ?? [],
    );
    return [this.store.delete(keys), this.cfg.statuses.store.delete(statuses)];
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

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatcher.send(key, [set({ path: ["name"], value: name })]);
  }

  /**
   * Applies actions to the cached task and sends them to the Core. Paths and values
   * take the camelCase form of the cached task. Returns false without side effects when
   * the task is not cached.
   * @throws {Error} if the task cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached task drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /**
   * Resolves once the cached task holds no dispatch that the Core has not echoed, so
   * its config hash is the one the Core holds for the edits made so far.
   */
  async settled(key: Key): Promise<void> {
    await this.dispatcher.settled(key);
  }

  /** Stages actions that commit atomically as one dispatch. */
  beginTransaction(key: Key): actions.Transaction<Action> {
    return this.dispatcher.transaction(key);
  }

  async retrieve<S extends Schemas = Schemas>(
    params: (RetrieveSingleParams | OneParams) & RetrieveSchemas<S>,
  ): Promise<Task<S>>;
  async retrieve(params: Key | RetrieveSingleParams | OneParams): Promise<Task>;
  async retrieve<S extends Schemas = Schemas>(
    params: (RetrieveMultipleParams | FilterParams) & RetrieveSchemas<S>,
  ): Promise<Task<S>[]>;
  async retrieve(
    params: Key[] | query.RequestInput<typeof retrieveMultiParamsZ>,
  ): Promise<Task[]>;
  async retrieve<S extends Schemas = Schemas>(
    rawParams:
      Key | Key[] | ((RetrieveParams | OneParams | FilterParams) & RetrieveSchemas<S>),
  ): Promise<Task<S> | Task<S>[]> {
    if (Array.isArray(rawParams))
      return (await super.retrieve(rawParams)) as unknown as Task<S>[];
    if (typeof rawParams === "string")
      return (await super.retrieve(rawParams)) as Task<S>;
    const { schemas, ...params } = rawParams;
    const isSingle = z.validate(singleRetrieveParamsZ, params) || isOneParams(params);
    // Schema-parametrized retrieves validate config/status for one caller;
    // their results are not shared through the cache.
    if (schemas != null) {
      const request = query.dropOne(params);
      if (isOneParams(request) && !z.validate(retrieveParamsZ, request)) {
        const res = singleQueryZ.safeParse(request);
        if (!res.success) throw singleQueryError("task", request, res.error);
      }
      const req = retrieveParamsZ.parse(request);
      const sugared = await this.execRetrieve<S>(req, schemas);
      checkForMultipleOrNoResults("task", req, sugared, isSingle);
      return isSingle ? sugared[0] : sugared;
    }
    if (isSingle) return (await super.retrieve(params)) as Task<S>;
    return (await super.retrieve(params)) as unknown as Task<S>[];
  }

  /** Copies the task with the given key under a new key and name. */
  async copy(key: Key, name: string): Promise<Task> {
    const copyRes = copyResZ();
    const response = await this.cfg.unary.send(
      "/task/copy",
      { key, name },
      copyReqZ,
      copyRes,
    );
    const sugared = this.sugar(response.task);
    this.writeThrough(sugared);
    return sugared;
  }

  private async execRetrieve<S extends Schemas = Schemas>(
    params: RetrieveParams | OneParams | FilterParams,
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

  private downRackStatus(rack: RackKey | undefined): status.Status | undefined {
    if (rack == null || primitive.isZero(rack)) return undefined;
    const stat = this.cfg.statuses.ownedBy(rackResourceID(rack));
    if (stat == null || isRackHealthy(stat)) return undefined;
    return stat;
  }

  /**
   * Builds the status shown under key while a command is in flight. A rack whose Driver
   * is down answers nothing, so its warning stands in for a wait that would never end.
   */
  private optimisticStatus(cmd: Command, key: status.Key): Status {
    // Carry the last known deploy info forward: zeroing it would make this
    // optimistic status claim the task deployed with an empty config/rack.
    const latest = this.latestStatusOf(cmd.task);
    const owner = resourceID(cmd.task);
    const details = {
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
        owner,
        variant: "loading",
        message: `Sending ${cmd.type} command`,
        details: { ...details, running: true },
      });
    return status.create<StatusDetailsZodObject>({
      key,
      name,
      owner,
      variant: "warning",
      message: down.message,
      description: down.description,
      details: { ...details, cmd: cmd.key, running: false },
    });
  }

  // A waiting command shows each new problem of its rack, since the Core's outage copy
  // carries no command key and so never answers the command.
  private standInForDownRack(rackStatus: status.Status): void {
    for (const [task, pending] of this.commandDeadlines) {
      const down = this.downRackStatus(this.store.get(task)?.rack);
      if (down?.key !== rackStatus.key) continue;
      // A batch stores a reply before it delivers the reply's event.
      const held = this.cfg.statuses.ownedBy(resourceID(task));
      const answered =
        held != null &&
        held !== pending.optimistic &&
        held.variant !== "loading" &&
        status.detailsOf(held)?.cmd === pending.cmd.key;
      if (answered) continue;
      pending.optimistic = this.optimisticStatus(pending.cmd, pending.optimistic.key);
      this.cfg.statuses.setLocal(pending.optimistic);
    }
  }

  // Only a reply ends the wait sooner. The problem of a down rack outlasts the wait.
  private armCommandDeadline(cmd: Command, optimistic: Status): void {
    const { statuses } = this.cfg;
    const owner = resourceID(cmd.task);
    clearTimeout(this.commandDeadlines.get(cmd.task)?.timer);
    // A reply disarms the deadline on arrival, so firing means nothing answered.
    const timer = setTimeout(() => {
      this.commandDeadlines.delete(cmd.task);
      const held = statuses.ownedBy(owner);
      if (held == null) return;
      if (this.downRackStatus(this.store.get(cmd.task)?.rack) != null) return;
      statuses.setLocal(
        status.create<StatusDetailsZodObject>({
          key: held.key,
          name: optimistic.name,
          owner,
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

  // Parsed because the status table holds every domain's statuses generically.
  private latestStatusOf(key: Key): Status | undefined {
    const parsed = defaultStatusZ.safeParse(this.cfg.statuses.ownedBy(resourceID(key)));
    return parsed.success ? parsed.data : undefined;
  }

  private writeThrough(task: Task): void {
    this.store.set(task);
    if (task.status != null) this.cfg.statuses.store.set(task.status);
  }

  private hydrate(task: Task, since: query.Stamp, statusSince: query.Stamp): void {
    this.store.ingest(task, { since });
    if (task.status != null)
      this.cfg.statuses.store.ingest(task.status, { since: statusSince });
  }

  private async fetchThrough(req: RetrieveRequest): Promise<Task[]> {
    const { store: statuses } = this.cfg.statuses;
    // A status that streams in during the fetch is newer than the fetched one.
    return await statuses.track(async (since) => {
      const tasks = await this.execRetrieve({ ...req, includeStatus: true });
      tasks.forEach((t) => {
        if (t.status != null) statuses.ingest(t.status, { since });
      });
      return tasks;
    });
  }

  private async fetchSingle(q: SingleRequest): Promise<Task> {
    // Task names are not unique, so only a key can answer from the cache.
    const cached = "key" in q ? this.store.get(q.key) : undefined;
    // A cached task without a cached status is ambiguous: the task may have
    // no status or the status may not have synced. Only both count as a hit.
    if (cached != null && this.cfg.statuses.ownedBy(resourceID(cached.key)) != null)
      return this.compose(cached);
    if ("key" in q) {
      // The refresh covers a cached task whose status has not synced.
      const tasks = await this.store.retrieve([q.key], { refresh: true });
      checkForMultipleOrNoResults(
        "task",
        { where: { key: { eq: q.key } } },
        tasks,
        true,
      );
      return this.compose(tasks[0]);
    }
    const where = singleWhere(q);
    return await this.store.track(
      async (since) =>
        await this.cfg.statuses.store.track(async (statusSince) => {
          const tasks = await this.execRetrieve({ where, includeStatus: true });
          checkForMultipleOrNoResults("task", { where }, tasks, true);
          this.hydrate(tasks[0], since, statusSince);
          return tasks[0];
        }),
    );
  }

  sugar<S extends Schemas = Schemas>(payloads: Payload<S>[], schemas?: S): Task<S>[];

  sugar<S extends Schemas = Schemas>(payload: Payload<S>, schemas?: S): Task<S>;

  sugar<S extends Schemas = Schemas>(
    payloads: Payload<S> | Payload<S>[],
    schemas?: S,
  ): Task<S>[] | Task<S> {
    const isSingle = !Array.isArray(payloads);
    const res = array
      .toArray(payloads)
      .map((payload) => new Task(payload, schemas, this.cfg.framer));
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

  /**
   * Sends commands and waits for the reply of each task.
   * @throws once the timeout passes, or with the status of a rack once the Driver
   * that runs one of the tasks goes down.
   */
  async executeCommandSync<StatusData extends z.ZodType = z.ZodNever>(
    params: ExecuteCommandsSyncParams<StatusData>,
  ): Promise<Status<StatusData>[]>;

  /**
   * Sends a command and waits for the reply of the task.
   * @throws once the timeout passes, or with the status of the rack once the Driver
   * that runs the task goes down.
   */
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
        racks: this.racksOf(params.commands.map((c) => c.task)),
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
      racks: this.racksOf([params.task]),
      configHash: params.configHash ?? this.store.get(params.task)?.configHash,
    });
  }

  // Reads only cached tasks: a fetch would make a command need read access to its task.
  private racksOf(tasks: Key[]): RackKey[] {
    return tasks.flatMap((t) => this.store.get(t)?.rack ?? []);
  }
}

interface ExecuteCommandInternalParams extends NewCommand {
  frameClient: framer.Client | null;
}

const executeCommand = async ({
  frameClient,
  ...command
}: ExecuteCommandInternalParams): Promise<string> =>
  (await executeCommands({ frameClient, commands: [command] }))[0];

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
  /**
   * Whether the Driver refuses a start when configHash is not the hash of the stored
   * config. Pin a start to run only the config that the operator saw.
   */
  pinned?: boolean;
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
    pinned: c.pinned ?? false,
    key: id.create(),
  }));
  await w.write(COMMAND_CHANNEL_NAME, cmds);
  await w.close();
  return cmds.map((c) => c.key);
};

interface ExecuteCommandSyncInternalParams<StatusData extends z.ZodType = z.ZodNever>
  extends
    Omit<ExecuteCommandsSyncInternalParams<StatusData>, "commands">,
    TaskExecuteCommandSyncParams,
    Pick<NewCommand, "configHash" | "pinned"> {
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
  configHash,
  pinned,
  racks,
}: ExecuteCommandSyncInternalParams<StatusData>): Promise<Status<StatusData>> =>
  (
    await executeCommandsSync({
      frameClient,
      commands: [{ args, task, type, configHash, pinned }],
      racks,
      timeout,
      statusDataZ,
      name: taskName,
    })
  )[0];

interface ExecuteCommandsSyncInternalParams<StatusData extends z.ZodType = z.ZodNever> {
  frameClient: framer.Client | null;
  commands: NewCommand[];
  /** The racks of the commanded tasks. The wait fails when one of them goes down. */
  racks: RackKey[];
  timeout?: CrudeTimeSpan;
  statusDataZ: StatusData;
  name: string | string[] | (() => Promise<string | string[]>);
}

const executeCommandsSync = async <StatusData extends z.ZodType = z.ZodNever>({
  frameClient,
  commands,
  racks,
  timeout = COMMAND_DEADLINE,
  statusDataZ,
  name: taskName,
}: ExecuteCommandsSyncInternalParams<StatusData>): Promise<Status<StatusData>[]> => {
  if (frameClient == null) throw new Error("Task not created");
  const rackOwners = new Set(
    racks
      .filter((r) => primitive.isNonZero(r))
      .map((r) => resource.idToString(rackResourceID(r))),
  );
  const streamer = await frameClient.openStreamer(status.SET_CHANNEL_NAME);
  const cmdKeys = await executeCommands({ frameClient, commands });
  const parsedTimeout = new TimeSpan(timeout);
  const finalTimeout = parsedTimeout.greaterThan(FINAL_STATUS_TIMEOUT)
    ? parsedTimeout
    : FINAL_STATUS_TIMEOUT;
  const collect = async (
    acknowledge: (cmd: string, loading: boolean) => void,
  ): Promise<Status<StatusData>[]> => {
    let states: Status<StatusData>[] = [];
    while (true) {
      const frame = await streamer.read();
      // A frame can hold statuses for other tasks and racks. Rack details parse as task
      // details, so the owner tells a rack status apart.
      for (const str of frame.get(status.SET_CHANNEL_NAME).toStrings()) {
        const decoded = binary.JSON_CODEC.decodeString(str);
        const rackRes = rackStatusZ.safeParse(decoded);
        if (
          rackRes.success &&
          rackRes.data.owner != null &&
          rackOwners.has(resource.idToString(rackRes.data.owner))
        ) {
          // A down Driver never replies, so its rack's problem ends the wait.
          if (!isRackHealthy(rackRes.data)) throw status.toError(rackRes.data);
          continue;
        }
        const res = statusZ(statusDataZ).safeParse(decoded);
        if (!res.success) continue;
        const state = res.data;
        if (state.details.cmd == null || !cmdKeys.includes(state.details.cmd)) continue;
        // Any reply acknowledges its command. A loading reply is not the final one.
        acknowledge(state.details.cmd, state.variant === "loading");
        if (state.variant === "loading") continue;
        states = [...states.filter((s) => s.key !== state.key), state];
        if (states.length === cmdKeys.length) return states;
      }
    }
  };
  try {
    return await new Promise<Status<StatusData>[]>((resolve, reject) => {
      const timeOutAfter = (span: TimeSpan): ReturnType<typeof setTimeout> =>
        setTimeout(() => {
          formatTimeoutError(
            taskName,
            span,
            commands.map((c) => c.task),
          ).then(reject, reject);
        }, span.milliseconds);
      let timer = timeOutAfter(parsedTimeout);
      const acknowledged = new Set<string>();
      // A loading status shows the Driver still works on its command, so once every
      // command is acknowledged, it starts the wait again.
      const acknowledge = (cmd: string, loading: boolean): void => {
        const first = !acknowledged.has(cmd);
        acknowledged.add(cmd);
        if (acknowledged.size < cmdKeys.length || (!first && !loading)) return;
        clearTimeout(timer);
        timer = timeOutAfter(finalTimeout);
      };
      collect(acknowledge)
        .then(resolve, reject)
        .finally(() => clearTimeout(timer));
    });
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
