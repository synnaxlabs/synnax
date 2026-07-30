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
import { array, deep, type destructor, errors, id, primitive } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { actions } from "@/actions";
import { createOf, isUndoable, kindOf, reduceAll } from "@/arc/actions";
import {
  type Action,
  dispatchReqZ,
  rename as renameAction,
  scopedActionZ,
} from "@/arc/actions.gen";
import { type Arc, arcZ, type Key, keyZ, type New, ontologyID } from "@/arc/types.gen";
import { NotFoundError } from "@/errors";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { status } from "@/status";
import { task } from "@/task";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_arc_set";
export const DELETE_CHANNEL_NAME = "sy_arc_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
  ignoreNotFoundError: z.boolean().optional(),
});
const createReqZ = z.object({ arcs: arcZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ arcs: arcZ.array().default(() => []) });
const createResZ = z.object({ arcs: arcZ.array() });
const emptyResZ = z.object({});

export const lspMessageZ = z.object({ content: z.string() });
export type LSPMessage = z.infer<typeof lspMessageZ>;

export type RetrieveRequest = z.input<typeof retrieveReqZ>;

const keyRetrieveRequestZ = z
  .object({
    key: keyZ,
    includeStatus: z.boolean().optional(),
  })
  .transform(({ key, includeStatus }) => ({ keys: [key], includeStatus }));

const nameRetrieveRequestZ = z
  .object({
    name: z.string(),
    includeStatus: z.boolean().optional(),
  })
  .transform(({ name, includeStatus }) => ({ names: [name], includeStatus }));

export const singleRetrieveParamsZ = z.union([
  keyRetrieveRequestZ,
  nameRetrieveRequestZ,
]);

export type SingleRetrieveParams = z.input<typeof singleRetrieveParamsZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveReqZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

const singleQueryZ = z.union([
  z.strictObject({ key: keyZ, includeStatus: z.boolean().optional() }),
  z.strictObject({ name: z.string(), includeStatus: z.boolean().optional() }),
]);

export interface CreateParams extends New {
  /** Rack to deploy the arc on. Ensures a deployment task exists for it. */
  rack?: number;
}

const TASK_TYPE = "arc";

const taskStatusDataZ = z.null().optional();

const configuringStatus = (taskKey: task.Key): task.Status<typeof taskStatusDataZ> =>
  status.create<ReturnType<typeof task.statusDetailsZ<typeof taskStatusDataZ>>>({
    key: task.statusKey(taskKey),
    name: "Configuring task",
    variant: "loading",
    message: "Configuring task...",
    details: { task: taskKey, running: false, cmd: "", data: undefined },
  });

const TASK_SCHEMAS = {
  type: z.literal(TASK_TYPE),
  config: z.object({ arcKey: z.string() }),
  statusData: taskStatusDataZ,
} as const satisfies task.Schemas;

/**
 * Client-side matching for a request: key and name sets. Server-computed
 * shapes (search, pagination) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((a: Arc) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
  return (a) => {
    if (keySet != null && !keySet.has(a.key)) return false;
    if (nameSet != null && !nameSet.has(a.name)) return false;
    return true;
  };
};

const isTaskChild = (rel: ontology.Relationship, arcKey: Key): boolean =>
  ontology.matchRelationship(rel, {
    from: ontologyID(arcKey),
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "task" },
  });

const taskStatusZ = z.object({ details: z.object({ task: task.keyZ }) });

// Task statuses may arrive under any status key; the referenced task lives in
// the details, with the "task:<key>" status key as a fallback.
const affectedTaskKeys = (
  event: query.TableEvent<status.Key, status.Status>,
): task.Key[] | null => {
  const keys: task.Key[] = [];
  if (event.variant === "set") {
    const parsed = taskStatusZ.safeParse(event.value);
    if (parsed.success) keys.push(parsed.data.details.task);
  }
  const [type, key] = event.key.split(":");
  if (type === "task" && primitive.isNonZero(key) && !keys.includes(key))
    keys.push(key);
  return keys.length === 0 ? null : keys;
};

export interface ClientParams {
  unary: UnaryClient;
  stream: StreamClient;
  ontology: ontology.Client;
  tasks: task.Client;
  cache: query.Cache;
  statusStore: query.Table<status.Key, status.Status>;
}

export class Client extends query.Retriever<
  typeof retrieveReqZ,
  Key,
  Arc,
  Arc,
  SingleRetrieveParams,
  SingleRetrieveParams
> {
  private readonly cfg: ClientParams;
  private readonly dispatcher: actions.Controller<Key, Arc, Action>;
  private readonly store: query.Table<Key, Arc>;
  private readonly taskAnswers: query.Retrieves<Key, task.Task | null>;

  constructor(cfg: ClientParams) {
    const { cache } = cfg;
    // Fetched copies never clobber a doc holding locally replayed edits: the
    // table hydrates if-absent, and hydrate() decides when a fresh network
    // doc replaces the cached one.
    const store = cache.createTable<Key, Arc>({
      name: "arcs",
      hydrate: "if-absent",
      fetch: async (keys) =>
        await this.execRetrieve({ keys, ignoreNotFoundError: true }),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Arc, Action>({
      store,
      onError: cache.onError,
      reduce: reduceAll,
      isUndoable,
      kindOf,
      createOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    const single = cache.queries<SingleRetrieveParams, Arc, Key, Arc>({
      name: "arc",
      table: store,
      fetch: async (q) => [(await this.fetchSingle(q)).key],
      compose: (records) => records[0],
      keyOf: (q) => ("key" in q ? q.key : null),
      matches: (a, q) => ("key" in q ? a.key === q.key : a.name === q.name),
      single: true,
    });
    super(cache, {
      name: "arc",
      table: store,
      request: {
        schema: retrieveReqZ,
        fetch: async (req) =>
          (await this.execRetrieve(req)).map((a) => this.hydrate(a)),
        matches: (a, req) => requestFilter(req)(a),
      },
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
    const composedTasks = cache.derive<task.Key, Omit<task.Task, "status">, task.Task>({
      name: "arc.task.composed",
      source: this.cfg.tasks.store,
      compose: (record) => this.composeTask(record),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      watch: [
        query.deriveWatch(this.cfg.statusStore, (event) => affectedTaskKeys(event)),
      ],
    });
    this.taskAnswers = cache.queries<Key, task.Task | null, task.Key, task.Task>({
      name: "arc task",
      table: composedTasks,
      fetch: async (q) => {
        const tsk = await this.fetchTask(q);
        return tsk == null ? [] : [tsk.key];
      },
      compose: (records) => records[0] ?? null,
      matches: (t, q) =>
        this.cfg.ontology.cache.relationships.has(
          ontology.relationshipToString({
            from: ontologyID(q),
            type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
            to: task.ontologyID(t.key),
          }),
        ),
      watch: [
        query.watch(this.cfg.ontology.cache.relationships, (event, q: Key) => {
          const rel =
            event.variant === "set"
              ? event.value
              : ontology.relationshipZ.parse(event.key);
          if (!isTaskChild(rel, q)) return null;
          return [rel.to.key];
        }),
      ],
    });
  }

  async create(arc: CreateParams, opts?: query.WriteOptions<Arc[]>): Promise<Arc>;
  async create(arcs: CreateParams[], opts?: query.WriteOptions<Arc[]>): Promise<Arc[]>;
  async create(
    arcs: CreateParams | CreateParams[],
    opts: query.WriteOptions<Arc[]> = {},
  ): Promise<Arc | Arc[]> {
    const isMany = Array.isArray(arcs);
    const params = array.toArray(arcs);
    // Resolve the deployment task for each arc targeting a rack. A task
    // previously deployed to a different rack is deleted and recreated.
    const taskKeys = new Map<number, task.Key>();
    for (let i = 0; i < params.length; i++) {
      const { rack, key } = params[i];
      if (rack == null) continue;
      let taskKey = task.newKey(rack, 0);
      if (key != null) {
        const tsk = await this.retrieveTask(key);
        if (tsk != null)
          if (task.rackKey(tsk.key) !== rack) await this.cfg.tasks.delete([tsk.key]);
          else taskKey = tsk.key;
      }
      taskKeys.set(i, taskKey);
    }
    const optimistic = params.map((a) => arcZ.parse(a));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.cfg.unary.send(
          "/arc/create",
          { arcs: optimistic },
          createReqZ,
          createResZ,
        ),
    });
    this.store.set(res.arcs);
    for (const [i, taskKey] of taskKeys) {
      const created = res.arcs[i];
      const newTsk = await this.cfg.tasks.create(
        {
          key: taskKey,
          name: created.name,
          type: TASK_TYPE,
          config: { arcKey: created.key },
          status: configuringStatus(taskKey),
        },
        TASK_SCHEMAS,
      );
      await this.cfg.ontology.addChildren(
        ontologyID(created.key),
        task.ontologyID(newTsk.key),
      );
    }
    return isMany ? res.arcs : res.arcs[0];
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const tsk = await this.retrieveTask(key);
    if (tsk != null) await this.cfg.tasks.rename(tsk.key, name);
    const rename = () => query.partialUpdate(this.store, key, { name });
    await query.optimistic({
      rollbacks: [rename()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.sendDispatch(key, id.create(), [renameAction({ name })]),
    });
    rename();
  }

  private async retrieveTask(key: Key): Promise<task.Task | null> {
    return await this.taskAnswers.retrieve(key);
  }

  /**
   * Cached queries for the task deployed for an arc, keyed by the arc's key.
   * Resolves null when the arc has no task.
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

  /** Subscribes to every arc delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  async openLSP(): Promise<Stream<typeof lspMessageZ, typeof lspMessageZ>> {
    return await this.cfg.stream.stream("/arc/lsp", lspMessageZ, lspMessageZ);
  }

  /**
   * Applies actions to the cached arc and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * arc isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<Arc, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the arc's most recent undoable entry. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key, this.dispatchSender(key));
  }

  /**
   * Re-applies the arc's most recently undone entry. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key, this.dispatchSender(key));
  }

  /** Whether the arc has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the arc has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the arc's undo/redo stacks. Returns a
   * destructor that unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher.onUndoStateChange(callback, key);
  }

  /**
   * Stages actions committed atomically as one undoable entry.
   */
  beginTransaction(key: Key, kind?: string): actions.Transaction<Action> {
    return this.dispatcher.transaction(key, this.dispatchSender(key), kind);
  }

  private dispatchSender(key: Key): actions.SendDispatch<Action> {
    return async (actions, dispatchKey) =>
      await this.sendDispatch(key, dispatchKey, actions);
  }

  private async sendDispatch(
    key: Key,
    dispatchKey: string,
    actions: Action[],
  ): Promise<void> {
    await this.cfg.unary.send(
      "/arc/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  private async execRetrieve(params: RetrieveParams): Promise<Arc[]> {
    const res = await this.cfg.unary.send(
      "/arc/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.arcs;
  }

  // Dispatch mutates documents server-side (including materialized text), so
  // a cached copy is only as fresh as the streamer. Fetches always hit the
  // network; only the server materializes text.
  private async fetchSingle(q: SingleRetrieveParams): Promise<Arc> {
    const arcs = await this.execRetrieve(q);
    checkForMultipleOrNoResults("Arc", q, arcs, true);
    return this.hydrate(arcs[0]);
  }

  // Answers reuse the identical store doc so selector references stay
  // stable; a fresher network doc replaces it and answers. While a locally
  // replayed dispatch awaits its echo the replayed doc stays, but the
  // network doc answers: it carries the server-materialized text.
  private hydrate(a: Arc): Arc {
    if (this.dispatcher.hasOutstanding(a.key) === true) {
      this.store.ingest(a);
      return a;
    }
    const prev = this.store.get(a.key);
    if (prev != null && deep.equal(prev, a)) return prev;
    this.store.set(a);
    return a;
  }

  /**
   * Rebuilds a cached task with its cached status attached. The status is
   * parsed because the status table holds every domain's statuses generically.
   */
  private composeTask(cached: Omit<task.Task, "status">): task.Task {
    const cachedStatus = this.cfg.statusStore.get(task.statusKey(cached.key));
    const payload = cached.payload;
    if (cachedStatus == null) return this.cfg.tasks.sugar(payload);
    const parsed = task.statusZ().safeParse(cachedStatus);
    if (!parsed.success) return this.cfg.tasks.sugar(payload);
    return this.cfg.tasks.sugar({ ...payload, status: parsed.data });
  }

  private async fetchTask(arcKey: Key): Promise<task.Task | null> {
    let children: ontology.Resource[];
    try {
      children = await this.cfg.ontology.children.retrieve({
        ids: ontologyID(arcKey),
        types: ["task"],
      });
    } catch (e) {
      // An arc that does not exist cannot have a task.
      if (NotFoundError.matches(e)) return null;
      throw errors.fromUnknown(e);
    }
    const child = children[0];
    if (child == null) return null;
    return await this.cfg.tasks.retrieve({ key: child.id.key });
  }
}
