// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Stream,
  type StreamClient,
  type UnaryClient,
} from "@synnaxlabs/freighter";
import { array, deep, type destructor, errors, filter, zod } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { actions } from "@/actions";
import { createOf, encode, isUndoable, kindOf, reduceAll } from "@/arc/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/arc/actions.gen";
import { directory } from "@/arc/directory";
import { type Performance, performanceZ } from "@/arc/task/types.gen";
import {
  type Arc,
  arcZ,
  type Key,
  keyZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/arc/types.gen";
import { LINES } from "@/arc/where";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/arc/where.gen";
import { query } from "@/query";
import { rack } from "@/rack";
import { type status } from "@/status";
import { task } from "@/task";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_arc_set";
export const DELETE_CHANNEL_NAME = "sy_arc_delete";

// The table caches Arcs without their compiled program.
const cachedRequestZ = retrieveRequestZ.omit({ includeProgram: true });
export type RetrieveMultipleParams = z.infer<typeof cachedRequestZ>;
const retrieveMultiParamsZ = filter.requestInputZ(
  cachedRequestZ,
  retrieveInputZ(cachedRequestZ.omit({ where: true }).shape),
);
const createReqZ = z.object({ dispatchKey: z.string(), arcs: arcZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const createResZ = z.object({ arcs: arcZ.array() });
const emptyResZ = z.object({});

export const lspMessageZ = z.object({ content: z.string() });
export type LSPMessage = z.infer<typeof lspMessageZ>;

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }): RetrieveRequest => ({ where: { key: { eq: key } } }));

const nameRetrieveRequestZ = z
  .object({ name: z.string() })
  .transform(({ name }): RetrieveRequest => ({ where: { name: { eq: name } } }));

export const singleRetrieveParamsZ = z.union([
  keyRetrieveRequestZ,
  nameRetrieveRequestZ,
]);

export type SingleRetrieveParams = z.input<typeof singleRetrieveParamsZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = SingleRetrieveParams | RetrieveMultipleParams;

const singleQueryZ = z.union([
  z.strictObject({ key: keyZ }),
  z.strictObject({ name: z.string() }),
  keyZ.transform((key) => ({ key })),
]);

const updateTaskReqZ = z.object({
  key: keyZ,
  rack: rack.keyZ.optional(),
  performance: performanceZ.optional(),
});
const updateTaskResZ = z.object({ task: task.payloadZ().nullish() });

/** The fields of an Arc's task to change. A field left out stays as it is. */
export interface UpdateTaskParams {
  /**
   * Binds the Arc to the rack, creating its task or moving the existing one. A zero
   * rack unbinds the Arc, deleting its task.
   */
  rack?: rack.Key;
  /** Sets how closely the task's runtime holds timer deadlines. */
  performance?: Performance;
}

/** The type of the task that runs an Arc. */
const TASK_TYPE = "arc";

const taskConfigZ = z.object({ arcKey: keyZ });

/** Returns the key of the Arc that the task runs, or undefined for any other task. */
const arcKeyOf = (t: Pick<task.Task, "type" | "config">): Key | undefined => {
  if (t.type !== TASK_TYPE) return undefined;
  const parsed = taskConfigZ.safeParse(t.config);
  return parsed.success ? parsed.data.arcKey : undefined;
};

export interface ClientConfig {
  unary: UnaryClient;
  stream: StreamClient;
  tasks: task.Client;
  cache: query.Cache;
  statuses: status.Client;
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Arc,
  Arc,
  SingleRetrieveParams,
  SingleRetrieveParams
> {
  /** directories creates, renames, deletes, and moves the directories of Arcs. */
  readonly directories: directory.Client;
  private readonly cfg: ClientConfig;
  private readonly dispatcher: actions.Controller<Key, Arc, Action>;
  private readonly store: query.Table<Key, Arc>;
  private readonly taskAnswers: query.Retrieves<Key, task.Task | null>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    // Fetched copies never clobber a doc holding locally replayed edits: the table
    // hydrates if-absent, and hydrate() decides when a fresh network doc replaces the
    // cached one.
    const store = cache.createTable<Key, Arc>({
      name: "Arcs",
      hydrate: "if-absent",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Arc, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      reduce: reduceAll,
      send: actions.sender(cfg.unary, "/arc/dispatch", dispatchReqZ, dispatchResZ),
      encode,
      isUndoable,
      kindOf,
      createOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    const single = cache.queries<SingleRetrieveParams, Arc, Key, Arc>({
      name: "Arc",
      table: store,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => ("key" in q ? q.key : null),
      matches: (a, q) => ("key" in q ? a.key === q.key : a.name === q.name),
      single: true,
    });
    super(cache, {
      name: "Arc",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) =>
          (await this.execRetrieve(req)).map((a) => this.hydrate(a)),
        matches: (a, req) => matches(LINES, a, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.directories = new directory.Client({ unary: cfg.unary, cache, arcs: store });
    this.store = store;
    this.dispatcher = dispatcher;
    const composedTasks = cache.derive<task.Key, Omit<task.Task, "status">, task.Task>({
      name: "arc.task.composed",
      source: this.cfg.tasks.store,
      compose: (record) => this.composeTask(record),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      triggers: [
        query.deriveTrigger(this.cfg.statuses.store, (event) =>
          this.cfg.statuses.ownerKeys(event, "task"),
        ),
      ],
    });
    this.taskAnswers = cache.queries<Key, task.Task | null, task.Key, task.Task>({
      name: "Arc task",
      table: composedTasks,
      fetch: async (q) => {
        const tsk = await this.fetchTask(q);
        return tsk == null ? [] : [tsk.key];
      },
      compose: (records) => records[0] ?? null,
      matches: (t, q) => arcKeyOf(t) === q,
    });
  }

  async create(arc: New, opts?: query.WriteOptions<Arc[]>): Promise<Arc>;
  async create(arcs: New[], opts?: query.WriteOptions<Arc[]>): Promise<Arc[]>;
  async create(
    arcs: New | New[],
    opts: query.WriteOptions<Arc[]> = {},
  ): Promise<Arc | Arc[]> {
    const isMany = Array.isArray(arcs);
    const optimistic = array
      .toArray(arcs)
      .map((a) => zod.parse(arcZ, a, { label: "Arc" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/arc/create",
              { dispatchKey, arcs: optimistic },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.arcs);
    return isMany ? res.arcs : res.arcs[0];
  }

  /**
   * Changes the Arc's task. Returns the task, or null after an unbind.
   * @throws {ValidationError} when unbinding while the task is running, or when
   * setting the performance of an Arc with no rack.
   */
  async updateTask(key: Key, params: UpdateTaskParams): Promise<task.Task | null> {
    // Drops an unbound task from the cache so it is not served until the delete
    // signal lands.
    const unbound = params.rack === 0 ? await this.retrieveTask(key) : null;
    const res = await this.cfg.unary.send(
      "/arc/update-task",
      { key, ...params },
      updateTaskReqZ,
      updateTaskResZ,
    );
    if (res.task == null) {
      if (unbound != null) this.cfg.tasks.dropCached(unbound.key);
      return null;
    }
    const tsk = this.cfg.tasks.sugar(res.task);
    this.cfg.tasks.store.set(tsk);
    return tsk;
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    // Restores only the name, so the rollback cannot undo edits made after it.
    const rename = (): destructor.Destructor => {
      const prev = this.store.get(key)?.name ?? name;
      query.partialUpdate(this.store, key, { name });
      return () => query.partialUpdate(this.store, key, { name: prev });
    };
    const rollback = rename();
    try {
      await opts.onOptimistic?.();
    } catch (e) {
      rollback();
      throw errors.fromUnknown(e);
    }
    await this.dispatcher.send(key, [set({ path: ["name"], value: name })], {
      rollback,
      reapply: rename,
    });
    rename();
  }

  private async retrieveTask(key: Key): Promise<task.Task | null> {
    return await this.taskAnswers.retrieve(key);
  }

  /**
   * Cached queries for the task deployed for an Arc, keyed by the Arc's key. Resolves
   * null when the Arc has no task.
   */
  get task(): query.Retrieves<Key, task.Task | null> {
    return this.taskAnswers;
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => this.store.delete(keysArr);
    await query.optimistic({
      rollbacks: [drop()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/arc/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Subscribes to every Arc delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  async openLSP(): Promise<Stream<typeof lspMessageZ, typeof lspMessageZ>> {
    return await this.cfg.stream.stream("/arc/lsp", lspMessageZ, lspMessageZ);
  }

  /**
   * Applies actions to the cached Arc and sends them to the server, recording an
   * undoable entry. Returns false without side effects when the Arc isn't cached. Rolls
   * back the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<Arc, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions), opts);
  }

  /**
   * Reverts the Arc's most recent undoable entry. Returns false when nothing is
   * undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key);
  }

  /**
   * Re-applies the Arc's most recently undone entry. Returns false when nothing is
   * redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key);
  }

  /** Whether the Arc has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the Arc has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the Arc's undo/redo stacks. Returns a destructor that
   * unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher.onUndoStateChange(callback, key);
  }

  /** Stages actions committed atomically as one undoable entry. */
  beginTransaction(key: Key, kind?: string): actions.Transaction<Action> {
    return this.dispatcher.transaction(key, kind);
  }

  private async execRetrieve(params: RetrieveParams): Promise<Arc[]> {
    const res = await this.cfg.unary.send(
      "/arc/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ,
    );
    return res.arcs;
  }

  // Dispatch mutates documents server-side (including materialized text), so a cached
  // copy is only as fresh as the streamer. Fetches always hit the network; only the
  // server materializes text.
  private async fetchSingle(q: SingleRetrieveParams): Promise<Arc> {
    const { where } = singleRetrieveParamsZ.parse(q);
    const arcs = await this.execRetrieve({ where });
    checkForMultipleOrNoResults("Arc", { where }, arcs, true);
    return this.hydrate(arcs[0]);
  }

  // Answers reuse the identical store doc so selector references stay stable; a fresher
  // network doc replaces it and answers. While a locally replayed dispatch awaits its
  // echo the replayed doc stays, but the network doc answers: it carries the
  // server-materialized text.
  private hydrate(a: Arc): Arc {
    if (this.dispatcher.hasPending(a.key)) {
      this.store.ingest(a);
      return a;
    }
    const prev = this.store.get(a.key);
    if (prev != null && deep.equal(prev, a)) return prev;
    this.store.ingest(a, { mode: "set" });
    return a;
  }

  // The status is parsed because the status table holds every domain's statuses
  // generically.
  private composeTask(cached: Omit<task.Task, "status">): task.Task {
    const cachedStatus = this.cfg.statuses.ownedBy(task.resourceID(cached.key));
    const payload = cached.payload;
    if (cachedStatus == null) return this.cfg.tasks.sugar(payload);
    const parsed = task.defaultStatusZ.safeParse(cachedStatus);
    if (!parsed.success) return this.cfg.tasks.sugar(payload);
    return this.cfg.tasks.sugar({ ...payload, status: parsed.data });
  }

  private async fetchTask(arcKey: Key): Promise<task.Task | null> {
    const [tsk] = await this.cfg.tasks.retrieve({ arcs: [arcKey] });
    return tsk ?? null;
  }
}
